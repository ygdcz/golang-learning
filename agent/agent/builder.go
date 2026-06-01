package appagent

import (
	"context"
	"fmt"
	"log"
	"strings"

	functiontool "github.com/ygdcz/golang-learning/agent/function_tool"
	adkagent "google.golang.org/adk/agent"
	"google.golang.org/adk/agent/llmagent"
	"google.golang.org/adk/model"
	"google.golang.org/adk/model/gemini"
	"google.golang.org/adk/tool"
	"google.golang.org/adk/tool/agenttool"
	"google.golang.org/adk/tool/geminitool"
	"google.golang.org/adk/tool/loadartifactstool"
	"google.golang.org/genai"
)

const (
	defaultRouterModel = "gemini-2.5-flash"
	defaultWorkerModel = "gemma-4-26b-a4b-it"
)

// NewRootAgent builds the router agent and its weather/search sub-agents.
func NewRootAgent(ctx context.Context, apiKey string, envLookup func(string) string) (adkagent.Agent, error) {
	if strings.TrimSpace(apiKey) == "" {
		return nil, fmt.Errorf("GOOGLE_API_KEY is not set")
	}
	if envLookup == nil {
		envLookup = func(string) string { return "" }
	}

	models := loadModelsConfig(envLookup)
	log.Printf("[agent] router candidates: %s", strings.Join(models.Router, ", "))
	log.Printf("[agent] weather candidates: %s", strings.Join(models.Weather, ", "))
	log.Printf("[agent] document candidates: %s", strings.Join(models.Document, ", "))
	log.Printf("[agent] search candidates: %s", strings.Join(models.Search, ", "))

	routerModel, err := newModel(ctx, "router-agent", models.Router, apiKey)
	if err != nil {
		return nil, fmt.Errorf("create router model failed: %w", err)
	}
	weatherModel, err := newModel(ctx, "weather-agent", models.Weather, apiKey)
	if err != nil {
		return nil, fmt.Errorf("create weather model failed: %w", err)
	}
	searchModel, err := newModel(ctx, "search-agent", models.Search, apiKey)
	if err != nil {
		return nil, fmt.Errorf("create search model failed: %w", err)
	}
	documentModel, err := newModel(ctx, "document-agent", models.Document, apiKey)
	if err != nil {
		return nil, fmt.Errorf("create document model failed: %w", err)
	}

	currentWeatherTool, err := functiontool.NewCurrentWeatherTool()
	if err != nil {
		return nil, fmt.Errorf("create current weather tool failed: %w", err)
	}
	weatherForecast7DTool, err := functiontool.NewWeatherForecast7DTool()
	if err != nil {
		return nil, fmt.Errorf("create weather forecast tool failed: %w", err)
	}
	analyzeDocumentTool, err := functiontool.NewAnalyzeDocumentTool()
	if err != nil {
		return nil, fmt.Errorf("create document analysis tool failed: %w", err)
	}

	generateConfig := defaultGenerateConfig()

	weatherAgent, err := llmagent.New(llmagent.Config{
		Name:        "weather-agent",
		Model:       weatherModel,
		Description: "处理天气相关问题，使用真实天气工具查询实时天气或7日预报。",
		Instruction: `你是天气助手。
当用户询问现在天气、当前温度、体感、阴晴、冷热时，使用 get-current-weather 工具。
当用户询问未来几天、本周、7天、七日天气趋势或预报时，使用 get-weather-7d 工具。
先从用户问题里提取城市名；如果缺少城市，就先追问城市。
如果用户指定华氏度，则传入 fahrenheit；否则默认 celsius。
收到实时天气结果后，用自然中文简洁回答，优先包含天气现象、当前温度和体感温度。
收到7日预报结果后，请按日期概括未来天气变化，不要原样复述整段 JSON。`,
		Tools: []tool.Tool{
			currentWeatherTool,
			weatherForecast7DTool,
		},
		GenerateContentConfig:    generateConfig,
		DisallowTransferToParent: true,
		DisallowTransferToPeers:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("create weather agent failed: %w", err)
	}

	searchAgent, err := llmagent.New(llmagent.Config{
		Name:        "search-agent",
		Model:       searchModel,
		Description: "处理通用联网检索问题，使用 Google Search 获取公开网络信息。",
		Instruction: `你是联网搜索助手。
当问题需要公开网页信息、新闻、资料检索、常识补充或实时搜索时，使用 google_search 工具。
优先整理搜索结果后再回答，中文简洁输出。
不要处理天气工具已经能直接完成的城市天气查询。`,
		Tools: []tool.Tool{
			geminitool.GoogleSearch{},
		},
		GenerateContentConfig:    generateConfig,
		DisallowTransferToParent: true,
		DisallowTransferToPeers:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("create search agent failed: %w", err)
	}

	documentAgent, err := llmagent.New(llmagent.Config{
		Name:        "document-agent",
		Model:       documentModel,
		Description: "处理用户上传文档、artifact 内容读取、摘要分析、图片理解和分析结果保存。",
		Instruction: `你是文档与图像分析助手。
当用户直接发来图片、截图、照片，或要求比较图片差异、识别图片内容时，直接用你的视觉能力分析并回答，无需加载 artifact。
当用户提到已上传文件、附件、artifact、文档总结、文档分析、读取文件内容时，优先处理这类请求。
如果用户在问某个已上传文件的内容、摘要或结论，先使用 load_artifacts 工具加载对应 artifact，再根据内容回答。
如果用户明确要求"分析并保存结果"、"总结成新文件"、"生成分析报告"，使用 analyze-document 工具。
如果用户没有提供文件名，但问题明显在指向已上传文件，先追问文件名，或者提示用户先上传文件。
只在文件内容已经可用时再下结论；不要编造未加载的文档内容。`,
		Tools: []tool.Tool{
			loadartifactstool.New(),
			analyzeDocumentTool,
		},
		GenerateContentConfig:    generateConfig,
		DisallowTransferToParent: true,
		DisallowTransferToPeers:  true,
	})
	if err != nil {
		return nil, fmt.Errorf("create document agent failed: %w", err)
	}

	// 包装 documentAgent 为 tool
	documentTool := agenttool.New(documentAgent, &agenttool.Config{
		SkipSummarization: true,
	})

	rootAgent, err := llmagent.New(llmagent.Config{
		Name:        "router-agent",
		Model:       routerModel,
		Description: "根据用户意图调用天气、文档或搜索能力。",
		Instruction: `你是智能助手，根据用户需求调用合适的工具。
当用户询问天气、温度、体感、天气现象、冷热时，转交给 weather-agent。
当用户发来图片、截图、照片，或要求比较图片差异、识别图片内容时，使用 document-agent 工具。
当用户询问已上传文件、附件、artifact、文档总结、文档分析、读取文件内容、保存文档分析结果时，使用 document-agent 工具。
当用户询问公开网页信息、资料检索、新闻、百科、实时网络信息时，转交给 search-agent。
优先只使用一个最合适的工具或子智能体；如果用户问题已经非常简单且不需要联网，可以直接回答。`,
		Tools: []tool.Tool{
			documentTool,
		},
		SubAgents: []adkagent.Agent{
			weatherAgent,
			searchAgent,
		},
		GenerateContentConfig: generateConfig,
		GlobalInstruction:     "请始终使用中文回答用户，并尽量简洁准确。",
	})
	if err != nil {
		return nil, fmt.Errorf("create router agent failed: %w", err)
	}

	return rootAgent, nil
}

type modelsConfig struct {
	Router   []string
	Weather  []string
	Document []string
	Search   []string
}

func loadModelsConfig(envLookup func(string) string) modelsConfig {
	return modelsConfig{
		Router:   resolveModelList(envLookup, "ROUTER_LLM_MODELS", "ROUTER_LLM_MODEL", defaultRouterModel),
		Weather:  resolveModelList(envLookup, "WEATHER_LLM_MODELS", "WEATHER_LLM_MODEL", defaultWorkerModel),
		Document: resolveModelList(envLookup, "DOCUMENT_LLM_MODELS", "DOCUMENT_LLM_MODEL", defaultRouterModel),
		Search:   resolveModelList(envLookup, "SEARCH_LLM_MODELS", "SEARCH_LLM_MODEL", defaultWorkerModel),
	}
}

func newModel(ctx context.Context, label string, modelNames []string, apiKey string) (model.LLM, error) {
	if len(modelNames) == 0 {
		return nil, fmt.Errorf("no model names configured")
	}

	models := make([]model.LLM, 0, len(modelNames))
	for _, modelName := range modelNames {
		llm, err := gemini.NewModel(ctx, modelName, &genai.ClientConfig{
			APIKey: apiKey,
		})
		if err != nil {
			return nil, fmt.Errorf("create model %q failed: %w", modelName, err)
		}
		models = append(models, llm)
	}

	return newFallbackModel(label, models)
}

func defaultGenerateConfig() *genai.GenerateContentConfig {
	temperature := float32(0.7)
	topP := float32(0.9)
	return &genai.GenerateContentConfig{
		Temperature:     &temperature,
		TopP:            &topP,
		MaxOutputTokens: 2048,
	}
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	return ""
}

func resolveModelList(envLookup func(string) string, listKey, singleKey, defaultValue string) []string {
	listValue := firstNonEmpty(envLookup(listKey), envLookup(singleKey), envLookup("LLM_MODEL"), defaultValue)
	return splitModelList(listValue)
}

func splitModelList(raw string) []string {
	parts := strings.Split(raw, ",")
	models := make([]string, 0, len(parts))
	for _, part := range parts {
		if trimmed := strings.TrimSpace(part); trimmed != "" {
			models = append(models, trimmed)
		}
	}
	return models
}
