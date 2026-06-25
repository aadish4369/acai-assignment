package tools

import (
	"context"
	"encoding/json"
	"time"

	"github.com/openai/openai-go/v2"
)

type DateTool struct{}

func (DateTool) Definition() openai.ChatCompletionToolUnionParam {
	return openai.ChatCompletionFunctionTool(openai.FunctionDefinitionParam{
		Name:        "get_today_date",
		Description: openai.String("Get today's date and time in RFC3339 format"),
	})
}

func (DateTool) Execute(ctx context.Context, args json.RawMessage) (string, error) {
	return time.Now().Format(time.RFC3339), nil
}
