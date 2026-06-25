package chat

import (
	"context"
	"errors"
	"testing"

	"github.com/acai-travel/tech-challenge/internal/chat/model"
	. "github.com/acai-travel/tech-challenge/internal/chat/testing"
	"github.com/acai-travel/tech-challenge/internal/pb"
	"github.com/google/go-cmp/cmp"
	"github.com/twitchtv/twirp"
	"google.golang.org/protobuf/testing/protocmp"
)

func TestServer_DescribeConversation(t *testing.T) {
	ctx := context.Background()
	srv := NewServer(model.New(ConnectMongo()), nil)

	t.Run("describe existing conversation", WithFixture(func(t *testing.T, f *Fixture) {
		c := f.CreateConversation()

		out, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: c.ID.Hex()})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		got, want := out.GetConversation(), c.Proto()
		if !cmp.Equal(got, want, protocmp.Transform()) {
			t.Errorf("DescribeConversation() mismatch (-got +want):\n%s", cmp.Diff(got, want, protocmp.Transform()))
		}
	}))

	t.Run("describe non existing conversation should return 404", WithFixture(func(t *testing.T, f *Fixture) {
		_, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: "08a59244257c872c5943e2a2"})
		if err == nil {
			t.Fatal("expected error for non-existing conversation, got nil")
		}

		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.NotFound {
			t.Fatalf("expected twirp.NotFound error, got %v", err)
		}
	}))
}

type stubAssistant struct {
	title    string
	reply    string
	titleErr error
	replyErr error

	titleCalls int
	replyCalls int
}

func (s *stubAssistant) Title(ctx context.Context, conv *model.Conversation) (string, error) {
	s.titleCalls++
	return s.title, s.titleErr
}

func (s *stubAssistant) Reply(ctx context.Context, conv *model.Conversation) (string, error) {
	s.replyCalls++
	return s.reply, s.replyErr
}

func TestServer_StartConversation(t *testing.T) {
	ctx := context.Background()

	t.Run("creates conversation, populates title, triggers reply", WithFixture(func(t *testing.T, f *Fixture) {
		assist := &stubAssistant{title: "Weather in Barcelona", reply: "It is sunny and 28°C in Barcelona."}
		srv := NewServer(f.Repository, assist)

		out, err := srv.StartConversation(ctx, &pb.StartConversationRequest{Message: "What is the weather like in Barcelona?"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.Cleanup(func() {
			if err := f.DeleteConversation(ctx, out.GetConversationId()); err != nil {
				t.Logf("failed to cleanup conversation %s: %v", out.GetConversationId(), err)
			}
		})

		if out.GetConversationId() == "" {
			t.Error("expected a non-empty conversation id")
		}

		if got, want := out.GetTitle(), "Weather in Barcelona"; got != want {
			t.Errorf("title = %q, want %q", got, want)
		}

		if got, want := out.GetReply(), "It is sunny and 28°C in Barcelona."; got != want {
			t.Errorf("reply = %q, want %q", got, want)
		}

		if assist.titleCalls != 1 {
			t.Errorf("Title called %d times, want 1", assist.titleCalls)
		}

		if assist.replyCalls != 1 {
			t.Errorf("Reply called %d times, want 1", assist.replyCalls)
		}

		desc, err := srv.DescribeConversation(ctx, &pb.DescribeConversationRequest{ConversationId: out.GetConversationId()})
		if err != nil {
			t.Fatalf("describe persisted conversation: %v", err)
		}

		conv := desc.GetConversation()
		if conv.GetTitle() != "Weather in Barcelona" {
			t.Errorf("persisted title = %q, want %q", conv.GetTitle(), "Weather in Barcelona")
		}

		msgs := conv.GetMessages()
		if len(msgs) != 2 {
			t.Fatalf("persisted message count = %d, want 2", len(msgs))
		}

		if got, want := msgs[0].GetContent(), "What is the weather like in Barcelona?"; got != want {
			t.Errorf("user message = %q, want %q", got, want)
		}

		if got, want := msgs[1].GetContent(), "It is sunny and 28°C in Barcelona."; got != want {
			t.Errorf("assistant message = %q, want %q", got, want)
		}
	}))

	t.Run("keeps default title when title generation fails", WithFixture(func(t *testing.T, f *Fixture) {
		assist := &stubAssistant{titleErr: errors.New("title boom"), reply: "Here is your answer."}
		srv := NewServer(f.Repository, assist)

		out, err := srv.StartConversation(ctx, &pb.StartConversationRequest{Message: "Hello there"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		t.Cleanup(func() {
			if err := f.DeleteConversation(ctx, out.GetConversationId()); err != nil {
				t.Logf("failed to cleanup conversation %s: %v", out.GetConversationId(), err)
			}
		})

		if got, want := out.GetTitle(), "Untitled conversation"; got != want {
			t.Errorf("title = %q, want %q", got, want)
		}

		if out.GetReply() != "Here is your answer." {
			t.Errorf("reply = %q, want %q", out.GetReply(), "Here is your answer.")
		}
	}))

	t.Run("fails when reply generation fails", WithFixture(func(t *testing.T, f *Fixture) {
		assist := &stubAssistant{title: "Some title", replyErr: errors.New("reply boom")}
		srv := NewServer(f.Repository, assist)

		_, err := srv.StartConversation(ctx, &pb.StartConversationRequest{Message: "Hello there"})
		if err == nil {
			t.Fatal("expected error when reply generation fails, got nil")
		}

		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.Internal {
			t.Fatalf("expected twirp.Internal error, got %v", err)
		}
	}))

	t.Run("rejects empty message", WithFixture(func(t *testing.T, f *Fixture) {
		srv := NewServer(f.Repository, &stubAssistant{})

		_, err := srv.StartConversation(ctx, &pb.StartConversationRequest{Message: "   "})
		if err == nil {
			t.Fatal("expected error for empty message, got nil")
		}

		if te, ok := err.(twirp.Error); !ok || te.Code() != twirp.InvalidArgument {
			t.Fatalf("expected twirp.InvalidArgument error, got %v", err)
		}
	}))
}
