package assistant

import (
	"context"
	"os"
	"strings"
	"testing"

	"github.com/acai-travel/tech-challenge/internal/chat/model"
	"github.com/joho/godotenv"
	"go.mongodb.org/mongo-driver/bson/primitive"
)

func TestMain(m *testing.M) {
	for _, path := range []string{".env", "../../.env", "../../../.env"} {
		if err := godotenv.Load(path); err == nil {
			break
		}
	}
	os.Exit(m.Run())
}

func TestAssistant_Title_EmptyConversation(t *testing.T) {
	a := New()

	got, err := a.Title(context.Background(), &model.Conversation{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got != "An empty conversation" {
		t.Errorf("Title() = %q, want %q", got, "An empty conversation")
	}
}

// TestAssistant_Title_Integration exercises the real OpenAI-backed path. It is
// skipped unless OPENAI_API_KEY is set so the unit suite stays offline.
func TestAssistant_Title_Integration(t *testing.T) {
	if os.Getenv("OPENAI_API_KEY") == "" {
		t.Skip("OPENAI_API_KEY not set; skipping OpenAI integration test")
	}

	a := New()

	conv := &model.Conversation{
		ID: primitive.NewObjectID(),
		Messages: []*model.Message{{
			ID:      primitive.NewObjectID(),
			Role:    model.RoleUser,
			Content: "What is the weather like in Barcelona today?",
		}},
	}

	title, err := a.Title(context.Background(), conv)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.TrimSpace(title) == "" {
		t.Error("expected a non-empty title")
	}

	if strings.Contains(title, "\n") {
		t.Errorf("title should be a single line, got %q", title)
	}

	if len(title) > 80 {
		t.Errorf("title length = %d, want <= 80", len(title))
	}
}
