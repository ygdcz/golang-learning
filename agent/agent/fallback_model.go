package appagent

import (
	"context"
	"fmt"
	"iter"
	"log"
	"maps"
	"net/http"
	"strings"

	"google.golang.org/adk/model"
	"google.golang.org/genai"
)

type fallbackModel struct {
	label  string
	name   string
	models []model.LLM
}

func newFallbackModel(label string, models []model.LLM) (model.LLM, error) {
	if len(models) == 0 {
		return nil, fmt.Errorf("no models configured")
	}

	return &fallbackModel{
		label:  label,
		name:   models[0].Name(),
		models: models,
	}, nil
}

func (m *fallbackModel) Name() string {
	return m.name
}

func (m *fallbackModel) GetGoogleLLMVariant() genai.Backend {
	if m == nil {
		return genai.BackendUnspecified
	}
	for _, llm := range m.models {
		if googleLLM, ok := llm.(interface{ GetGoogleLLMVariant() genai.Backend }); ok {
			return googleLLM.GetGoogleLLMVariant()
		}
	}
	return genai.BackendUnspecified
}

func (m *fallbackModel) GenerateContent(ctx context.Context, req *model.LLMRequest, stream bool) iter.Seq2[*model.LLMResponse, error] {
	return func(yield func(*model.LLMResponse, error) bool) {
		var lastErr error

		for i, llm := range m.models {
			log.Printf("[model:%s] try %d/%d: %s", m.label, i+1, len(m.models), llm.Name())
			reqCopy := cloneLLMRequest(req)
			reqCopy.Model = llm.Name()
			emitted := false

			for resp, err := range llm.GenerateContent(ctx, reqCopy, stream) {
				if err != nil {
					lastErr = err
					if shouldFallbackToNextModel(err) && !emitted && i < len(m.models)-1 {
						log.Printf("[model:%s] %s hit retryable error, fallback to next model: %v", m.label, llm.Name(), err)
						break
					}
					log.Printf("[model:%s] %s failed: %v", m.label, llm.Name(), err)
					if emitted || !shouldFallbackToNextModel(err) || i == len(m.models)-1 {
						yield(nil, err)
						return
					}
					break
				}

				if !emitted {
					log.Printf("[model:%s] selected: %s", m.label, llm.Name())
				}
				emitted = true
				if !yield(resp, nil) {
					return
				}
			}

			if emitted {
				return
			}
		}

		if lastErr != nil {
			yield(nil, lastErr)
		}
	}
}

func cloneLLMRequest(req *model.LLMRequest) *model.LLMRequest {
	if req == nil {
		return &model.LLMRequest{}
	}

	cloned := *req

	if req.Contents != nil {
		cloned.Contents = append([]*genai.Content(nil), req.Contents...)
	}
	if req.Config != nil {
		configCopy := *req.Config
		if req.Config.HTTPOptions != nil {
			httpOptionsCopy := *req.Config.HTTPOptions
			if req.Config.HTTPOptions.Headers != nil {
				headersCopy := make(http.Header, len(req.Config.HTTPOptions.Headers))
				for key, values := range req.Config.HTTPOptions.Headers {
					headersCopy[key] = append([]string(nil), values...)
				}
				httpOptionsCopy.Headers = headersCopy
			}
			configCopy.HTTPOptions = &httpOptionsCopy
		}
		cloned.Config = &configCopy
	}
	if req.Tools != nil {
		toolsCopy := make(map[string]any, len(req.Tools))
		maps.Copy(toolsCopy, req.Tools)
		cloned.Tools = toolsCopy
	}

	return &cloned
}

func shouldFallbackToNextModel(err error) bool {
	if err == nil {
		return false
	}

	message := strings.ToLower(err.Error())

	neverRetry := []string{
		"displayname",
		"enterprise agent platform",
		"invalid_argument",
		"status: invalid_argument",
		"malformed",
		"unsupported",
	}
	for _, needle := range neverRetry {
		if strings.Contains(message, needle) {
			return false
		}
	}

	retryNeedles := []string{
		"resource_exhausted",
		"quota exceeded",
		"rate limit",
		"too many requests",
		"429",
		"limit exceeded",
		"internal error encountered",
		" status: internal",
		" status: unavailable",
		"service unavailable",
		"backend error",
		"temporarily unavailable",
		"deadline exceeded",
		"502",
		"503",
		"504",
	}
	for _, needle := range retryNeedles {
		if strings.Contains(message, needle) {
			return true
		}
	}
	return false
}
