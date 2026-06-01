package functiontool

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"os"
	"strings"
	"unicode/utf8"

	"google.golang.org/adk/tool"
	adkfunctiontool "google.golang.org/adk/tool/functiontool"
	"google.golang.org/genai"
)

const defaultDocumentAnalysisModel = "gemini-2.5-flash-lite"

type analyzeDocumentArgs struct {
	FileName string `json:"file_name" jsonschema:"The artifact file name to analyze."`
}

type analyzeDocumentResult struct {
	FileName       string `json:"file_name"`
	ArtifactName   string `json:"artifact_name"`
	Version        int64  `json:"version"`
	Summary        string `json:"summary"`
	CharacterCount int    `json:"character_count"`
	TotalFiles     int    `json:"total_files"`
}

type documentAnalysisConfig struct {
	apiKey    string
	modelName string
}

func makeAnalyzeDocument() func(tool.Context, *analyzeDocumentArgs) (*analyzeDocumentResult, error) {
	return func(ctx tool.Context, args *analyzeDocumentArgs) (*analyzeDocumentResult, error) {
		if args == nil || strings.TrimSpace(args.FileName) == "" {
			return nil, fmt.Errorf("file_name is required")
		}
		if ctx.Artifacts() == nil {
			return nil, fmt.Errorf("artifact service is not initialized")
		}

		fileName := strings.TrimSpace(args.FileName)
		loadResp, err := ctx.Artifacts().Load(ctx, fileName)
		if err != nil {
			return nil, fmt.Errorf("load artifact %q failed: %w", fileName, err)
		}

		documentText, err := extractArtifactText(ctx, loadResp.Part)
		if err != nil {
			return nil, fmt.Errorf("analyze artifact %q failed: %w", fileName, err)
		}

		summary := summarizeDocument(fileName, documentText)
		saveName := buildAnalysisArtifactName(fileName)
		saveResp, err := ctx.Artifacts().Save(ctx, saveName, genai.NewPartFromText(summary))
		if err != nil {
			return nil, fmt.Errorf("save analysis failed: %w", err)
		}

		listResp, err := ctx.Artifacts().List(ctx)
		if err != nil {
			return nil, fmt.Errorf("list artifacts failed: %w", err)
		}

		return &analyzeDocumentResult{
			FileName:       fileName,
			ArtifactName:   saveName,
			Version:        saveResp.Version,
			Summary:        summary,
			CharacterCount: utf8.RuneCountInString(documentText),
			TotalFiles:     len(listResp.FileNames),
		}, nil
	}
}

// NewAnalyzeDocumentTool creates a tool that loads an artifact, summarizes it,
// and saves the analysis result back to the artifact store.
func NewAnalyzeDocumentTool() (tool.Tool, error) {
	return adkfunctiontool.New(
		adkfunctiontool.Config{
			Name:        "analyze-document",
			Description: "读取已上传的文档 artifact，生成摘要并保存为新的分析结果 artifact",
		},
		makeAnalyzeDocument(),
	)
}

func extractArtifactText(ctx context.Context, part *genai.Part) (string, error) {
	if part == nil {
		return "", fmt.Errorf("artifact is empty")
	}
	if text := strings.TrimSpace(part.Text); text != "" {
		return text, nil
	}
	if part.InlineData != nil {
		mimeType := strings.ToLower(strings.TrimSpace(part.InlineData.MIMEType))
		switch {
		case supportsTextMIME(mimeType):
			text := strings.TrimSpace(string(part.InlineData.Data))
			if text == "" {
				return "", fmt.Errorf("artifact content is empty")
			}
			return text, nil
		case mimeType == "application/vnd.openxmlformats-officedocument.wordprocessingml.document":
			return extractDocxText(part.InlineData.Data)
		case supportsGeminiVision(mimeType):
			cfg := loadDocumentAnalysisConfig()
			return extractViaGemini(ctx, part.InlineData.Data, mimeType, cfg)
		default:
			return "", fmt.Errorf("unsupported artifact mime type %q", mimeType)
		}
	}
	if part.FileData != nil {
		mimeType := strings.ToLower(strings.TrimSpace(part.FileData.MIMEType))
		return "", fmt.Errorf("artifact %q uses file_data (%s); please load and analyze text-based uploaded artifacts", part.FileData.FileURI, mimeType)
	}
	return "", fmt.Errorf("artifact does not contain readable text")
}

func supportsTextMIME(mimeType string) bool {
	if mimeType == "" {
		return true
	}
	if strings.HasPrefix(mimeType, "text/") {
		return true
	}
	switch mimeType {
	case "application/json", "application/xml", "application/yaml", "application/x-yaml", "text/csv":
		return true
	default:
		return false
	}
}

func supportsGeminiVision(mimeType string) bool {
	switch mimeType {
	case "application/pdf",
		"image/jpeg", "image/jpg", "image/png", "image/gif",
		"image/webp", "image/heic", "image/heif":
		return true
	default:
		return false
	}
}

// extractDocxText extracts plain text from a DOCX file by reading word/document.xml
// inside the ZIP archive and collecting all <w:t> element values.
func extractDocxText(data []byte) (string, error) {
	r, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return "", fmt.Errorf("open docx zip: %w", err)
	}

	for _, f := range r.File {
		if f.Name != "word/document.xml" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return "", fmt.Errorf("open word/document.xml: %w", err)
		}
		defer rc.Close()

		var sb strings.Builder
		dec := xml.NewDecoder(rc)
		for {
			tok, err := dec.Token()
			if err == io.EOF {
				break
			}
			if err != nil {
				return "", fmt.Errorf("parse word/document.xml: %w", err)
			}
			if se, ok := tok.(xml.StartElement); ok && se.Name.Local == "t" {
				var text string
				if err := dec.DecodeElement(&text, &se); err != nil {
					continue
				}
				sb.WriteString(text)
			}
		}

		result := strings.TrimSpace(sb.String())
		if result == "" {
			return "", fmt.Errorf("docx document contains no text")
		}
		return result, nil
	}
	return "", fmt.Errorf("word/document.xml not found in docx archive")
}

// extractViaGemini sends binary data (PDF or image) to Gemini for text extraction.
func extractViaGemini(ctx context.Context, data []byte, mimeType string, cfg documentAnalysisConfig) (string, error) {
	if strings.TrimSpace(cfg.apiKey) == "" {
		return "", fmt.Errorf("GOOGLE_API_KEY is required only when analyzing PDF/image artifacts")
	}

	client, err := genai.NewClient(ctx, &genai.ClientConfig{
		APIKey:  cfg.apiKey,
		Backend: genai.BackendGeminiAPI,
	})
	if err != nil {
		return "", fmt.Errorf("create genai client: %w", err)
	}

	prompt := genai.NewPartFromText("请提取并返回这份文档中的所有文字内容，保持原有段落结构，不要添加任何额外说明。")
	doc := genai.NewPartFromBytes(data, mimeType)

	contents := []*genai.Content{
		{Role: "user", Parts: []*genai.Part{doc, prompt}},
	}
	resp, err := client.Models.GenerateContent(ctx, cfg.modelName, contents, nil)
	if err != nil {
		return "", fmt.Errorf("gemini vision request failed: %w", err)
	}

	var sb strings.Builder
	for _, cand := range resp.Candidates {
		if cand.Content == nil {
			continue
		}
		for _, p := range cand.Content.Parts {
			if p.Text != "" {
				sb.WriteString(p.Text)
			}
		}
	}
	result := strings.TrimSpace(sb.String())
	if result == "" {
		return "", fmt.Errorf("gemini returned no text for the document")
	}
	return result, nil
}

func loadDocumentAnalysisConfig() documentAnalysisConfig {
	return documentAnalysisConfig{
		apiKey: strings.TrimSpace(os.Getenv("GOOGLE_API_KEY")),
		modelName: firstNonEmptyString(
			os.Getenv("DOCUMENT_ANALYSIS_MODEL"),
			os.Getenv("DOCUMENT_LLM_MODEL"),
			os.Getenv("LLM_MODEL"),
			defaultDocumentAnalysisModel,
		),
	}
}

func firstNonEmptyString(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func summarizeDocument(fileName, text string) string {
	cleanText := normalizeDocumentText(text)
	runes := []rune(cleanText)
	preview := cleanText
	if len(runes) > 240 {
		preview = strings.TrimSpace(string(runes[:240])) + "..."
	}

	lines := nonEmptyLines(text)
	firstLine := ""
	if len(lines) > 0 {
		firstLine = truncateRunes(lines[0], 80)
	}

	var sections []string
	sections = append(sections, fmt.Sprintf("文档分析结果：%s", fileName))
	if firstLine != "" {
		sections = append(sections, fmt.Sprintf("标题/首行：%s", firstLine))
	}
	sections = append(sections, fmt.Sprintf("字符数：%d", utf8.RuneCountInString(cleanText)))
	sections = append(sections, fmt.Sprintf("摘要：%s", preview))
	return strings.Join(sections, "\n")
}

func buildAnalysisArtifactName(fileName string) string {
	sanitized := strings.NewReplacer("/", "_", "\\", "_", " ", "_").Replace(strings.TrimSpace(fileName))
	return "analysis_" + sanitized + ".txt"
}

func normalizeDocumentText(text string) string {
	return strings.Join(strings.Fields(strings.TrimSpace(text)), " ")
}

func nonEmptyLines(text string) []string {
	rawLines := strings.Split(text, "\n")
	lines := make([]string, 0, len(rawLines))
	for _, line := range rawLines {
		if trimmed := strings.TrimSpace(line); trimmed != "" {
			lines = append(lines, trimmed)
		}
	}
	return lines
}

func truncateRunes(text string, limit int) string {
	runes := []rune(strings.TrimSpace(text))
	if len(runes) <= limit {
		return string(runes)
	}
	return string(runes[:limit]) + "..."
}
