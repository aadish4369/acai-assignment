package tools

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/openai/openai-go/v2"
)

type Tool interface {
	Definition() openai.ChatCompletionToolUnionParam
	Execute(ctx context.Context, args json.RawMessage) (string, error)
}

type ToolRegistry struct {
	byName map[string]Tool
	defs   []openai.ChatCompletionToolUnionParam
}

func NewToolRegistry(tools ...Tool) *ToolRegistry {
	r := &ToolRegistry{
		byName: make(map[string]Tool, len(tools)),
		defs:   make([]openai.ChatCompletionToolUnionParam, 0, len(tools)),
	}

	for _, tool := range tools {
		def := tool.Definition()
		name := def.OfFunction.Function.Name

		if _, exists := r.byName[name]; exists {
			panic(fmt.Sprintf("tools: duplicate tool registered: %q", name))
		}

		r.byName[name] = tool
		r.defs = append(r.defs, def)
	}

	return r
}

func (r *ToolRegistry) Definitions() []openai.ChatCompletionToolUnionParam {
	return r.defs
}

func (r *ToolRegistry) Execute(ctx context.Context, name string, args json.RawMessage) (string, error) {
	tool, ok := r.byName[name]
	if !ok {
		return "", fmt.Errorf("unknown tool: %s", name)
	}

	return tool.Execute(ctx, args)
}
