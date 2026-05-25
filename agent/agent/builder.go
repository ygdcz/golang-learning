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
	"google.golang.org/adk/tool/geminitool"
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

	currentWeatherTool, err := functiontool.NewCurrentWeatherTool()
	if err != nil {
		return nil, fmt.Errorf("create current weather tool failed: %w", err)
	}
	weatherForecast7DTool, err := functiontool.NewWeatherForecast7DTool()
	if err != nil {
		return nil, fmt.Errorf("create weather forecast tool failed: %w", err)
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

	rootAgent, err := llmagent.New(llmagent.Config{
		Name:        "router-agent",
		Model:       routerModel,
		Description: "根据用户意图把任务分发给天气或搜索子智能体。",
		Instruction: `你是路由助手，负责判断应该把请求交给哪个子智能体处理。
如果用户在问某个城市的天气、温度、体感、天气现象、冷热，转交给 weather-agent。
如果用户在问公开网页信息、资料检索、新闻、百科、实时网络信息，转交给 search-agent。
优先只转交给一个最合适的子智能体；如果用户问题已经非常简单且不需要联网，可以直接回答。`,
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
	Router  []string
	Weather []string
	Search  []string
}

func loadModelsConfig(envLookup func(string) string) modelsConfig {
	return modelsConfig{
		Router:  resolveModelList(envLookup, "ROUTER_LLM_MODELS", "ROUTER_LLM_MODEL", defaultRouterModel),
		Weather: resolveModelList(envLookup, "WEATHER_LLM_MODELS", "WEATHER_LLM_MODEL", defaultWorkerModel),
		Search:  resolveModelList(envLookup, "SEARCH_LLM_MODELS", "SEARCH_LLM_MODEL", defaultWorkerModel),
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
