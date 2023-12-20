package batchclient

import (
	"context"
	"service-client/internal/batchservice"
	"sync/atomic"
	"testing"
	"time"
)

func TestButchClient_Send(t *testing.T) {
	t.Run("gets correct final total", func(t *testing.T) {
		correctTotal := 100

		var total atomic.Uint64

		ctx, _ := context.WithCancel(context.Background())
		service := batchservice.New(
			batchservice.WithNumber(17),
			batchservice.WithPeriod(100*time.Millisecond),
			batchservice.WithTestHandler(func(batch batchservice.Batch) {
				total.Add(uint64(len(batch)))
			}),
		)
		localClient := Init(service)

		var batch batchservice.Batch
		for i := 0; i < correctTotal; i += 1 {
			batch = append(batch, batchservice.Item{ID: i})
		}

		localClient.Send(ctx, batch)

		time.Sleep(3 * time.Second)

		gotTotal := total.Load()
		if gotTotal != uint64(correctTotal) {
			t.Fatalf("Incorrect total. Get %d, expected: %d", gotTotal, correctTotal)
		}
	})

	t.Run("do not panic if correct timing", func(t *testing.T) {
		correctTotal := 100
		correctTiming := 100 * time.Millisecond

		start := time.Now().Add(-1 * time.Second)

		service := batchservice.New(
			batchservice.WithNumber(17),
			batchservice.WithPeriod(correctTiming),
			batchservice.WithTestHandler(func(batch batchservice.Batch) {
				if time.Since(start) < correctTiming {
					panic("incorrect timing")
				}
			}),
		)
		localClient := Init(service)

		var batch batchservice.Batch
		for i := 0; i < correctTotal; i += 1 {
			batch = append(batch, batchservice.Item{ID: i})
		}

		requestCtx, _ := context.WithCancel(context.Background())
		localClient.Send(requestCtx, batch)
		time.Sleep(3 * time.Second)
	})

	t.Run("stops processing after request ctx.cancel", func(t *testing.T) {
		correctTotal := 70

		service := batchservice.New(
			batchservice.WithNumber(17),
			batchservice.WithPeriod(1*time.Second),
		)

		localClient := Init(service)

		var batch batchservice.Batch
		for i := 0; i < correctTotal; i += 1 {
			batch = append(batch, batchservice.Item{ID: i})
		}

		requestCtx, requestCancel := context.WithCancel(context.Background())
		localClient.Send(requestCtx, batch)
		time.Sleep(2 * time.Second)
		requestCancel()
	})

	t.Run("do not panic if correct timing", func(t *testing.T) {
		correctTotal := 10000
		iterations := 5
		correctTiming := 50 * time.Millisecond
		var numberOfItems uint64 = 500

		service := batchservice.New(
			batchservice.WithNumber(numberOfItems),
			batchservice.WithPeriod(correctTiming),
		)
		localClient := Init(service)

		requestCtx, _ := context.WithCancel(context.Background())

		for i := 0; i < iterations; i += 1 {
			batch := make(batchservice.Batch, 0, correctTotal)
			for j := 0; j < correctTotal; j += 1 {
				batch = append(batch, batchservice.Item{ID: i})
			}

			localClient.Send(requestCtx, batch)
		}

		time.Sleep(7 * time.Second)
	})
}
