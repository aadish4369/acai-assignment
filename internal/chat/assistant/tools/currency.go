package tools

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/openai/openai-go/v2"
)

const currencyBaseURL = "https://api.frankfurter.app"

var currencyClient = &http.Client{Timeout: 10 * time.Second}

type CurrencyTool struct{}

type currencyArgs struct {
	Amount float64 `json:"amount"`
	From   string  `json:"from"`
	To     string  `json:"to"`
}

type currencyResponse struct {
	Amount float64            `json:"amount"`
	Base   string             `json:"base"`
	Date   string             `json:"date"`
	Rates  map[string]float64 `json:"rates"`
}

func (CurrencyTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        "convert_currency",
		Description: openai.String("Convert an amount of money from one currency to another using current exchange rates."),
		Parameters: openai.FunctionParameters{
			"type": "object",
			"properties": map[string]any{
				"amount": map[string]any{
					"type":        "number",
					"description": "The amount of money to convert. Defaults to 1 if omitted.",
				},
				"from": map[string]string{
					"type":        "string",
					"description": "The source currency as a 3-letter ISO 4217 code, e.g. EUR.",
				},
				"to": map[string]string{
					"type":        "string",
					"description": "The target currency as a 3-letter ISO 4217 code, e.g. USD.",
				},
			},
			"required": []string{"from", "to"},
		},
	})
}

func (CurrencyTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	var payload currencyArgs
	if err := json.Unmarshal(args, &payload); err != nil {
		return "", fmt.Errorf("parse arguments: %w", err)
	}

	from := strings.ToUpper(strings.TrimSpace(payload.From))
	to := strings.ToUpper(strings.TrimSpace(payload.To))
	if from == "" || to == "" {
		return "", fmt.Errorf("currency: both 'from' and 'to' are required")
	}

	amount := payload.Amount
	if amount <= 0 {
		amount = 1
	}

	if from == to {
		return fmt.Sprintf("%.2f %s = %.2f %s (rate 1.0000)", amount, from, amount, to), nil
	}

	query := url.Values{
		"amount": {fmt.Sprintf("%g", amount)},
		"from":   {from},
		"to":     {to},
	}

	endpointURL := currencyBaseURL + "/latest?" + query.Encode()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpointURL, nil)
	if err != nil {
		return "", fmt.Errorf("currency: build request: %w", err)
	}

	resp, err := currencyClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("currency: request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("currency: unexpected status %d", resp.StatusCode)
	}

	var data currencyResponse
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		return "", fmt.Errorf("currency: decode response: %w", err)
	}

	converted, ok := data.Rates[to]
	if !ok {
		return "", fmt.Errorf("currency: no rate returned for %s", to)
	}

	rate := converted / amount
	return fmt.Sprintf("%.2f %s = %.2f %s (rate %.4f, as of %s)", amount, from, converted, to, rate, data.Date), nil
}
