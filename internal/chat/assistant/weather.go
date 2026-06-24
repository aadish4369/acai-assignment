package assistant

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/acai-travel/tech-challenge/internal/chat/model"
)

const (
	weatherBaseURL     = "https://api.weatherapi.com/v1"
	weatherMaxForecast = 7
)

var weatherClient = &http.Client{Timeout: 10 * time.Second}

// CurrentWeather returns a human-readable summary of the current weather at the
// given location, suitable for passing back to the model as tool output.
func CurrentWeather(ctx context.Context, location string) (string, error) {
	data, err := requestWeather(ctx, "current.json", location, nil)
	if err != nil {
		return "", err
	}

	return formatCurrent(data), nil
}

// ForecastWeather returns a human-readable summary of the current weather plus a
// multi-day forecast for the given location. days is clamped to the range
// supported by the free WeatherAPI plan (1-7).
func ForecastWeather(ctx context.Context, location string, days int) (string, error) {
	if days < 1 {
		days = 1
	}
	if days > weatherMaxForecast {
		days = weatherMaxForecast
	}

	data, err := requestWeather(ctx, "forecast.json", location, url.Values{
		"days": {fmt.Sprint(days)},
	})
	if err != nil {
		return "", err
	}

	return formatCurrent(data) + "\n\n" + formatForecast(data), nil
}

func requestWeather(ctx context.Context, endpoint, location string, extra url.Values) (*model.WeatherData, error) {
	apiKey := os.Getenv("WEATHER_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("weather: WEATHER_API_KEY is not set")
	}

	location = strings.TrimSpace(location)
	if location == "" {
		return nil, fmt.Errorf("weather: location is required")
	}

	query := url.Values{"key": {apiKey}, "q": {location}}
	for k, vs := range extra {
		for _, v := range vs {
			query.Add(k, v)
		}
	}

	endpointURL := weatherBaseURL + "/" + endpoint + "?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
	if err != nil {
		return nil, fmt.Errorf("weather: build request: %w", err)
	}

	resp, err := weatherClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("weather: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		var apiErr struct {
			Error struct {
				Message string `json:"message"`
			} `json:"error"`
		}
		if json.NewDecoder(resp.Body).Decode(&apiErr) == nil && apiErr.Error.Message != "" {
			return nil, fmt.Errorf("weather: API error (%d): %s", resp.StatusCode, apiErr.Error.Message)
		}
		return nil, fmt.Errorf("weather: unexpected status %d", resp.StatusCode)
	}

	var data model.WeatherData
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return nil, fmt.Errorf("weather: decode response: %w", err)
	}

	return &data, nil
}

func formatCurrent(data *model.WeatherData) string {
	loc := data.Location
	place := loc.Name
	for _, part := range []string{loc.Region, loc.Country} {
		if part != "" && part != place {
			place += ", " + part
		}
	}

	c := data.Current
	var b strings.Builder
	fmt.Fprintf(&b, "Current weather in %s", place)
	if loc.Localtime != "" {
		fmt.Fprintf(&b, " (local time %s)", loc.Localtime)
	}
	b.WriteString(":\n")
	fmt.Fprintf(&b, "- Condition: %s\n", c.Condition.Text)
	fmt.Fprintf(&b, "- Temperature: %.1f°C (feels like %.1f°C)\n", c.TempC, c.FeelslikeC)
	fmt.Fprintf(&b, "- Wind: %.1f km/h %s\n", c.WindKph, c.WindDir)
	fmt.Fprintf(&b, "- Humidity: %d%%\n", c.Humidity)
	fmt.Fprintf(&b, "- Precipitation: %.1f mm", c.PrecipMM)

	return b.String()
}

func formatForecast(data *model.WeatherData) string {
	var b strings.Builder
	b.WriteString("Forecast:")
	for _, fd := range data.Forecast.Forecastday {
		d := fd.Day
		fmt.Fprintf(&b, "\n- %s: %s, high %.1f°C / low %.1f°C, wind up to %.1f km/h, precipitation %.1f mm",
			fd.Date, d.Condition.Text, d.MaxtempC, d.MintempC, d.MaxwindKph, d.TotalprecipMM)
	}

	return b.String()
}
