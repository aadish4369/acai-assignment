package main

import (
	"context"
	"fmt"
	"log/slog"
	"net/http"

	"github.com/acai-travel/tech-challenge/internal/chat"
	"github.com/acai-travel/tech-challenge/internal/chat/assistant"
	"github.com/acai-travel/tech-challenge/internal/chat/model"
	"github.com/acai-travel/tech-challenge/internal/httpx"
	"github.com/acai-travel/tech-challenge/internal/mongox"
	"github.com/acai-travel/tech-challenge/internal/otelx"
	"github.com/acai-travel/tech-challenge/internal/pb"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	"github.com/twitchtv/twirp"
	"go.opentelemetry.io/otel"
)

const serviceName = "chat-server"

func main() {
	_ = godotenv.Load()

	ctx := context.Background()

	shutdown, err := otelx.Setup(ctx, serviceName)
	if err != nil {
		panic(err)
	}
	defer func() { _ = shutdown(context.Background()) }()

	mongo := mongox.MustConnect()

	repo := model.New(mongo)
	assist := assistant.New()

	server := chat.NewServer(repo, assist)

	metrics, err := httpx.Metrics(otel.Meter(serviceName))
	if err != nil {
		panic(err)
	}

	// Configure handler
	handler := mux.NewRouter()
	handler.Use(
		httpx.Tracing(otel.Tracer(serviceName)),
		metrics,
		httpx.Logger(),
		httpx.Recovery(),
	)

	handler.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = fmt.Fprint(w, "Hi, my name is Clippy!")
	})

	handler.PathPrefix("/twirp/").Handler(pb.NewChatServiceServer(server, twirp.WithServerJSONSkipDefaults(true)))

	// Start the server
	slog.Info("Starting the server...")
	if err := http.ListenAndServe(":8080", handler); err != nil {
		panic(err)
	}
}
