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

	t.Run("panics if incorrect timing", func(t *testing.T) {
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

	t.Run("stops processing after ctx.cancel", func(t *testing.T) {
		Setup()
		once = sync.Once{}
		correctTotal := 50
		expectedTotalAfterStop := 50

		var total atomic.Uint64

		ctx, cancel := context.WithCancel(context.Background())
		service := batchservice.New(
			batchservice.WithNumber(17),
			batchservice.WithPeriod(200*time.Millisecond),
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

		time.Sleep(1 * time.Second)
		cancel()

		time.Sleep(1 * time.Second)
		err1 := localClient.Send(ctx, batch)
		err2 := localClient.Send(ctx, batch)

		time.Sleep(1 * time.Second)

		if err1 != ErrClientAlreadyTerminated {
			t.Fatalf("Cannot prevent writing into already closed channel")
		}

		if err2 != ErrClientAlreadyTerminated {
			t.Fatalf("Cannot prevent writing into already closed channel")
		}

		gotTotal := total.Load()
		if gotTotal != uint64(expectedTotalAfterStop) {
			t.Fatalf("Incorrect total. Get %d, expected: %d", gotTotal, correctTotal)
		}
	})
}
