package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"log/slog"
	"net/http"
	"os"
	"service-client/internal/batchclient"
	"service-client/internal/batchservice"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	logger := setupLogger()

	service := batchservice.New()
	client := batchclient.Init(logger, service)

	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("welcome"))
	})

	r.Post("/send", func(w http.ResponseWriter, r *http.Request) {
		var bodyHolder struct {
			Number int `json:"number"`
		}

		if err := json.NewDecoder(r.Body).Decode(&bodyHolder); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		defer func() {
			if err := r.Body.Close(); err != nil {
				fmt.Println("defer error", err)
			}
		}()

		var batch batchservice.Batch
		for i := 0; i < bodyHolder.Number; i += 1 {
			batch = append(batch, batchservice.Item{ID: i})
		}

		client.Send(ctx, batch)

		w.Write([]byte("done"))
	})

	r.Post("/cancel", func(w http.ResponseWriter, r *http.Request) {
		cancel()
		w.Write([]byte("done"))
	})

	http.ListenAndServe(":3000", r)
}

func setupLogger() *slog.Logger {
	// env switch here
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}
