package batchclient

import (
	"context"
	"service-client/internal/batchservice"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func Setup() {
	once = sync.Once{}
}

func TestButchClient_Send(t *testing.T) {
	t.Run("gets correct final total", func(t *testing.T) {
		Setup()
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
		localClient := Init(ctx, service)

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
		Setup()
		correctTotal := 100
		correctTiming := 100 * time.Millisecond

		start := time.Now().Add(-1 * time.Second)

		ctx, _ := context.WithCancel(context.Background())
		service := batchservice.New(
			batchservice.WithNumber(17),
			batchservice.WithPeriod(correctTiming),
			batchservice.WithTestHandler(func(batch batchservice.Batch) {
				if time.Since(start) < correctTiming {
					panic("incorrect timing")
				}
			}),
		)
		localClient := Init(ctx, service)

		var batch batchservice.Batch
		for i := 0; i < correctTotal; i += 1 {
			batch = append(batch, batchservice.Item{ID: i})
		}

		localClient.Send(ctx, batch)
		time.Sleep(3 * time.Second)
	})

	t.Run("stops processing after request ctx.cancel", func(t *testing.T) {
		Setup()

		correctTotal := 70

		ctx, _ := context.WithCancel(context.Background())
		service := batchservice.New(
			batchservice.WithNumber(17),
			batchservice.WithPeriod(1*time.Second),
		)

		localClient := Init(ctx, service)

		var batch batchservice.Batch
		for i := 0; i < correctTotal; i += 1 {
			batch = append(batch, batchservice.Item{ID: i})
		}

		requestCtx, requestCancel := context.WithCancel(context.Background())
		localClient.Send(requestCtx, batch)
		time.Sleep(2 * time.Second)
		requestCancel()
	})

	t.Run("stops processing after global ctx.cancel", func(t *testing.T) {
		Setup()

		correctTotal := 5
		var total atomic.Uint64

		globalCtx, globalCancel := context.WithCancel(context.Background())
		service := batchservice.New(
			batchservice.WithNumber(2),
			batchservice.WithPeriod(1*time.Second),
			batchservice.WithTestHandler(func(batch batchservice.Batch) {
				total.Add(uint64(len(batch)))
			}),
		)

		localClient := Init(globalCtx, service)

		batch := batchservice.Batch{
			batchservice.Item{ID: 1},
			batchservice.Item{ID: 2},
			batchservice.Item{ID: 3},
			batchservice.Item{ID: 4},
			batchservice.Item{ID: 5},
		}
		requestCtx, _ := context.WithCancel(context.Background())
		err := localClient.Send(requestCtx, batch)

		time.Sleep(2 * time.Second)
		globalCancel()
		time.Sleep(1 * time.Second)

		err = localClient.Send(requestCtx, batch)
		if err == nil {
			t.Fatalf("cannot catch terminated client")
		}

		gotTotal := total.Load()
		if gotTotal != uint64(correctTotal) {
			t.Fatalf("Incorrect total. Get %d, expected: %d", gotTotal, correctTotal)
		}
	})
}
