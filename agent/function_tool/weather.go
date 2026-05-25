package functiontool

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"google.golang.org/adk/tool"
	adkfunctiontool "google.golang.org/adk/tool/functiontool"
)

type currentWeatherArgs struct {
	City string `json:"city" jsonschema:"The city to get the weather for."`
	Unit string `json:"unit,omitempty" jsonschema:"Temperature unit to use. Default is 'celsius'."`
}

type currentWeatherResult struct {
	City        string  `json:"city"`
	Temperature float64 `json:"temperature"`
	FeelsLike   float64 `json:"feels_like"`
	Description string  `json:"description"`
	Unit        string  `json:"unit"`
}

type weatherForecastArgs struct {
	City string `json:"city" jsonschema:"The city to get the weather forecast for."`
	Unit string `json:"unit,omitempty" jsonschema:"Temperature unit to use. Default is 'celsius'."`
}

type weatherForecastDay struct {
	Date         string  `json:"date"`
	TextDay      string  `json:"text_day"`
	TextNight    string  `json:"text_night"`
	TempMin      float64 `json:"temp_min"`
	TempMax      float64 `json:"temp_max"`
	Humidity     int     `json:"humidity"`
	Precip       float64 `json:"precip"`
	WindDirDay   string  `json:"wind_dir_day"`
	WindScaleDay string  `json:"wind_scale_day"`
}

type weatherForecastResult struct {
	City string               `json:"city"`
	Unit string               `json:"unit"`
	Days []weatherForecastDay `json:"days"`
}

type qweatherConfig struct {
	rootURL string
	apiKey  string
	client  *http.Client
}

type qweatherGeoResponse struct {
	Code     string `json:"code"`
	Location []struct {
		ID      string `json:"id"`
		Name    string `json:"name"`
		Adm1    string `json:"adm1"`
		Adm2    string `json:"adm2"`
		Country string `json:"country"`
	} `json:"location"`
}

type qweatherNowResponse struct {
	Code string `json:"code"`
	Now  struct {
		Temp      string `json:"temp"`
		FeelsLike string `json:"feelsLike"`
		Text      string `json:"text"`
	} `json:"now"`
}

type qweatherForecastResponse struct {
	Code  string `json:"code"`
	Daily []struct {
		FxDate       string `json:"fxDate"`
		TextDay      string `json:"textDay"`
		TextNight    string `json:"textNight"`
		TempMax      string `json:"tempMax"`
		TempMin      string `json:"tempMin"`
		Humidity     string `json:"humidity"`
		Precip       string `json:"precip"`
		WindDirDay   string `json:"windDirDay"`
		WindScaleDay string `json:"windScaleDay"`
	} `json:"daily"`
}

type qweatherErrorResponse struct {
	Error *struct {
		Status int    `json:"status"`
		Title  string `json:"title"`
		Detail string `json:"detail"`
	} `json:"error"`
}

func getCurrentWeather(ctx tool.Context, args *currentWeatherArgs) (*currentWeatherResult, error) {
	if args == nil || strings.TrimSpace(args.City) == "" {
		return nil, fmt.Errorf("city is required")
	}

	cfg, err := loadQWeatherConfig()
	if err != nil {
		return nil, err
	}

	locationID, cityName, err := lookupLocation(cfg.client, cfg.rootURL, cfg.apiKey, strings.TrimSpace(args.City))
	if err != nil {
		return nil, err
	}

	weatherResp, err := fetchCurrentWeather(cfg.client, cfg.rootURL, cfg.apiKey, locationID, toQWeatherUnit(args.Unit))
	if err != nil {
		return nil, err
	}

	temperature, err := strconv.ParseFloat(weatherResp.Now.Temp, 64)
	if err != nil {
		return nil, fmt.Errorf("parse temperature failed: %w", err)
	}
	feelsLike, err := strconv.ParseFloat(weatherResp.Now.FeelsLike, 64)
	if err != nil {
		return nil, fmt.Errorf("parse feels like failed: %w", err)
	}

	description := weatherResp.Now.Text
	if feelsLike := strings.TrimSpace(weatherResp.Now.FeelsLike); feelsLike != "" {
		description = fmt.Sprintf("%s, 体感 %s°", description, feelsLike)
	}

	return &currentWeatherResult{
		City:        cityName,
		Temperature: temperature,
		FeelsLike:   feelsLike,
		Description: description,
		Unit:        normalizeResultUnit(args.Unit),
	}, nil
}

func getWeatherForecast7D(ctx tool.Context, args *weatherForecastArgs) (*weatherForecastResult, error) {
	if args == nil || strings.TrimSpace(args.City) == "" {
		return nil, fmt.Errorf("city is required")
	}

	cfg, err := loadQWeatherConfig()
	if err != nil {
		return nil, err
	}

	locationID, cityName, err := lookupLocation(cfg.client, cfg.rootURL, cfg.apiKey, strings.TrimSpace(args.City))
	if err != nil {
		return nil, err
	}

	forecastResp, err := fetchWeatherForecast7D(cfg.client, cfg.rootURL, cfg.apiKey, locationID, toQWeatherUnit(args.Unit))
	if err != nil {
		return nil, err
	}

	days := make([]weatherForecastDay, 0, len(forecastResp.Daily))
	for _, day := range forecastResp.Daily {
		tempMin, err := strconv.ParseFloat(day.TempMin, 64)
		if err != nil {
			return nil, fmt.Errorf("parse min temperature failed: %w", err)
		}
		tempMax, err := strconv.ParseFloat(day.TempMax, 64)
		if err != nil {
			return nil, fmt.Errorf("parse max temperature failed: %w", err)
		}
		humidity, err := strconv.Atoi(day.Humidity)
		if err != nil {
			return nil, fmt.Errorf("parse humidity failed: %w", err)
		}
		precip, err := strconv.ParseFloat(day.Precip, 64)
		if err != nil {
			return nil, fmt.Errorf("parse precipitation failed: %w", err)
		}

		days = append(days, weatherForecastDay{
			Date:         day.FxDate,
			TextDay:      day.TextDay,
			TextNight:    day.TextNight,
			TempMin:      tempMin,
			TempMax:      tempMax,
			Humidity:     humidity,
			Precip:       precip,
			WindDirDay:   day.WindDirDay,
			WindScaleDay: day.WindScaleDay,
		})
	}

	return &weatherForecastResult{
		City: cityName,
		Unit: normalizeResultUnit(args.Unit),
		Days: days,
	}, nil
}

// NewCurrentWeatherTool creates the real-time weather tool.
func NewCurrentWeatherTool() (tool.Tool, error) {
	return adkfunctiontool.New(
		adkfunctiontool.Config{
			Name:        "get-current-weather",
			Description: "获取指定城市的实时天气信息",
		},
		getCurrentWeather,
	)
}

// NewWeatherForecast7DTool creates the 7-day forecast tool.
func NewWeatherForecast7DTool() (tool.Tool, error) {
	return adkfunctiontool.New(
		adkfunctiontool.Config{
			Name:        "get-weather-7d",
			Description: "获取指定城市的7日天气预报",
		},
		getWeatherForecast7D,
	)
}

// NewWeatherTool is kept for backward compatibility and points to the real-time tool.
func NewWeatherTool() (tool.Tool, error) {
	return NewCurrentWeatherTool()
}

func loadQWeatherConfig() (*qweatherConfig, error) {
	baseURL := strings.TrimSpace(os.Getenv("weather_api_url"))
	apiKey := strings.TrimSpace(os.Getenv("weather_api_key"))
	if baseURL == "" {
		return nil, fmt.Errorf("weather_api_url is not set")
	}
	if apiKey == "" {
		return nil, fmt.Errorf("weather_api_key is not set")
	}

	rootURL, err := normalizeQWeatherRoot(baseURL)
	if err != nil {
		return nil, err
	}

	return &qweatherConfig{
		rootURL: rootURL,
		apiKey:  apiKey,
		client:  &http.Client{Timeout: 10 * time.Second},
	}, nil
}

func normalizeQWeatherRoot(raw string) (string, error) {
	if raw == "" {
		return "", fmt.Errorf("weather_api_url is empty")
	}

	parsed, err := url.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid weather_api_url: %w", err)
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return "", fmt.Errorf("weather_api_url must include scheme and host")
	}
	return fmt.Sprintf("%s://%s", parsed.Scheme, parsed.Host), nil
}

func lookupLocation(client *http.Client, rootURL, apiKey, city string) (string, string, error) {
	endpoint, err := url.Parse(rootURL + "/geo/v2/city/lookup")
	if err != nil {
		return "", "", fmt.Errorf("build geo lookup url failed: %w", err)
	}

	query := endpoint.Query()
	query.Set("location", city)
	query.Set("number", "1")
	endpoint.RawQuery = query.Encode()

	var geoResp qweatherGeoResponse
	if err := doQWeatherRequest(client, endpoint.String(), apiKey, &geoResp); err != nil {
		return "", "", err
	}

	if geoResp.Code != "200" {
		return "", "", fmt.Errorf("city lookup failed, qweather code=%s", geoResp.Code)
	}
	if len(geoResp.Location) == 0 {
		return "", "", fmt.Errorf("city %q not found", city)
	}

	location := geoResp.Location[0]
	cityName := location.Name
	if location.Adm2 != "" && location.Adm2 != cityName {
		cityName = location.Adm2 + " " + cityName
	}
	if location.Adm1 != "" && !strings.Contains(cityName, location.Adm1) {
		cityName = cityName + ", " + location.Adm1
	}
	if location.Country != "" && !strings.Contains(cityName, location.Country) {
		cityName = cityName + ", " + location.Country
	}

	return location.ID, cityName, nil
}

func fetchCurrentWeather(client *http.Client, rootURL, apiKey, locationID, unit string) (*qweatherNowResponse, error) {
	endpoint, err := url.Parse(rootURL + "/v7/weather/now")
	if err != nil {
		return nil, fmt.Errorf("build weather url failed: %w", err)
	}

	query := endpoint.Query()
	query.Set("location", locationID)
	query.Set("unit", unit)
	endpoint.RawQuery = query.Encode()

	var weatherResp qweatherNowResponse
	if err := doQWeatherRequest(client, endpoint.String(), apiKey, &weatherResp); err != nil {
		return nil, err
	}
	if weatherResp.Code != "200" {
		return nil, fmt.Errorf("weather request failed, qweather code=%s", weatherResp.Code)
	}
	return &weatherResp, nil
}

func fetchWeatherForecast7D(client *http.Client, rootURL, apiKey, locationID, unit string) (*qweatherForecastResponse, error) {
	endpoint, err := url.Parse(rootURL + "/v7/weather/7d")
	if err != nil {
		return nil, fmt.Errorf("build forecast url failed: %w", err)
	}

	query := endpoint.Query()
	query.Set("location", locationID)
	query.Set("unit", unit)
	endpoint.RawQuery = query.Encode()

	var forecastResp qweatherForecastResponse
	if err := doQWeatherRequest(client, endpoint.String(), apiKey, &forecastResp); err != nil {
		return nil, err
	}
	if forecastResp.Code != "200" {
		return nil, fmt.Errorf("forecast request failed, qweather code=%s", forecastResp.Code)
	}
	return &forecastResp, nil
}

func doQWeatherRequest(client *http.Client, endpoint, apiKey string, target any) error {
	req, err := http.NewRequest(http.MethodGet, endpoint, nil)
	if err != nil {
		return fmt.Errorf("create request failed: %w", err)
	}
	req.Header.Set("X-QW-Api-Key", apiKey)
	req.Header.Set("Accept", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request qweather failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= http.StatusBadRequest {
		var apiErr qweatherErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&apiErr); err == nil && apiErr.Error != nil {
			return fmt.Errorf("qweather request failed: %s: %s", apiErr.Error.Title, apiErr.Error.Detail)
		}
		return fmt.Errorf("qweather request failed with status %d", resp.StatusCode)
	}

	if err := json.NewDecoder(resp.Body).Decode(target); err != nil {
		return fmt.Errorf("decode qweather response failed: %w", err)
	}
	return nil
}

func toQWeatherUnit(unit string) string {
	switch strings.ToLower(strings.TrimSpace(unit)) {
	case "", "celsius", "metric", "m":
		return "m"
	case "fahrenheit", "imperial", "i":
		return "i"
	default:
		return "m"
	}
}

func normalizeResultUnit(unit string) string {
	switch toQWeatherUnit(unit) {
	case "i":
		return "fahrenheit"
	default:
		return "celsius"
	}
}
