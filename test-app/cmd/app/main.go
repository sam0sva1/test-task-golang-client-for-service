package main

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/go-chi/chi/v5"
	"github.com/sam0sva1/batchclient/v2"
	"github.com/sam0sva1/batchservice/v2"
	"log"
	"log/slog"
	"net/http"
	"os"
	"time"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	logger := setupLogger()

	service := batchservice.New(
		batchservice.WithNumber(70),
		batchservice.WithPeriod(100*time.Millisecond),
	)
	client := batchclient.Init(logger, service)

	r := chi.NewRouter()

	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Add("Content-Type", "application/json")
		w.Write([]byte("{ \"app\": \"test-app\", \"status\": \"OK\", \"version\": \"0.0.2\" }"))
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
		for i := 0; i < bodyHolder.Number; i++ {
			batch = append(batch, batchservice.Item{ID: i})
		}

		client.Send(ctx, batch)

		w.Write([]byte("done"))
	})

	r.Post("/cancel", func(w http.ResponseWriter, r *http.Request) {
		cancel()
		w.Write([]byte("done"))
	})

	server := &http.Server{
		Addr:              ":3000",
		ReadHeaderTimeout: 3 * time.Second,
		Handler:           r,
	}

	logger.Info("starting the server...")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatal(err)
	}
}

func setupLogger() *slog.Logger {
	// env switch here
	return slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))
}
