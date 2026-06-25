package assistant

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"strings"

	"github.com/acai-travel/tech-challenge/internal/chat/assistant/tools"
	"github.com/acai-travel/tech-challenge/internal/chat/model"
	"github.com/openai/openai-go/v2"
)

type Assistant struct {
	cli   openai.Client
	tools *tools.ToolRegistry
}

func New() *Assistant {
	return &Assistant{
		cli: openai.NewClient(),
		tools: tools.NewToolRegistry(
			tools.WeatherTool{},
			tools.DateTool{},
			tools.HolidaysTool{},
			tools.CurrencyTool{},
		),
	}
}

func (a *Assistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "An empty conversation", nil
	}

	slog.InfoContext(ctx, "Generating title for conversation", "conversation_id", conv.ID)

	msgs := make([]openai.ChatCompletionMessageParamUnion, len(conv.Messages))

	msgs[0] = openai.AssistantMessage("Generate a concise, descriptive title for the conversation based on the user message. The title should be a single line, no more than 80 characters, and should not include any special characters or emojis.")
	for _, m := range conv.Messages {
		msgs = append(msgs, openai.UserMessage(m.Content))
	}

	resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
		Model:    openai.ChatModelO1,
		Messages: msgs,
	})

	if err != nil {
		return "", err
	}

	if len(resp.Choices) == 0 || strings.TrimSpace(resp.Choices[0].Message.Content) == "" {
		return "", errors.New("empty response from OpenAI for title generation")
	}

	title := resp.Choices[0].Message.Content
	title = strings.ReplaceAll(title, "\n", " ")
	title = strings.Trim(title, " \t\r\n-\"'")

	if len(title) > 80 {
		title = title[:80]
	}

	return title, nil
}

func (a *Assistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	if len(conv.Messages) == 0 {
		return "", errors.New("conversation has no messages")
	}

	slog.InfoContext(ctx, "Generating reply for conversation", "conversation_id", conv.ID)

	msgs := []openai.ChatCompletionMessageParamUnion{
		openai.SystemMessage("You are a helpful, concise AI assistant. Provide accurate, safe, and clear responses."),
	}

	for _, m := range conv.Messages {
		switch m.Role {
		case model.RoleUser:
			msgs = append(msgs, openai.UserMessage(m.Content))
		case model.RoleAssistant:
			msgs = append(msgs, openai.AssistantMessage(m.Content))
		}
	}

	for i := 0; i < 15; i++ {
		resp, err := a.cli.Chat.Completions.New(ctx, openai.ChatCompletionNewParams{
			Model:    openai.ChatModelGPT4_1,
			Messages: msgs,
			Tools:    a.tools.Definitions(),
		})

		if err != nil {
			return "", err
		}

		if len(resp.Choices) == 0 {
			return "", errors.New("no choices returned by OpenAI")
		}

		if message := resp.Choices[0].Message; len(message.ToolCalls) > 0 {
			msgs = append(msgs, message.ToParam())

			for _, call := range message.ToolCalls {
				slog.InfoContext(ctx, "Tool call received", "name", call.Function.Name, "args", call.Function.Arguments)

				result, err := a.tools.Execute(ctx, call.Function.Name, json.RawMessage(call.Function.Arguments))
				if err != nil {
					slog.WarnContext(ctx, "Tool call failed", "name", call.Function.Name, "error", err)
					result = "failed: " + err.Error()
				}

				msgs = append(msgs, openai.ToolMessage(result, call.ID))
			}

			continue
		}

		return resp.Choices[0].Message.Content, nil
	}

	return "", errors.New("too many tool calls, unable to generate reply")
}
