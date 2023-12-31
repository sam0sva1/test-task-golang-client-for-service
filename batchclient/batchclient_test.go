package batchclient

import (
	"context"
	"github.com/sam0sva1/batchservice/v2"
	"log/slog"
	"os"
	"sync/atomic"
	"testing"
	"time"
)

var logger = slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{Level: slog.LevelDebug}))

func TestBatchClient_Send(t *testing.T) {
	t.Run("gets correct final total", func(t *testing.T) {
		t.Parallel()
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
		localClient := Init(logger, service)

		var batch batchservice.Batch
		for i := 0; i < correctTotal; i++ {
			batch = append(batch, batchservice.Item{ID: i})
		}

		localClient.Send(ctx, batch)

		//time.Sleep(10 * time.Minute)
		time.Sleep(3 * time.Second)

		gotTotal := total.Load()
		if gotTotal != uint64(correctTotal) {
			t.Fatalf("Incorrect total. Get %d, expected: %d", gotTotal, correctTotal)
		}
	})

	t.Run("do not panic if correct timing", func(t *testing.T) {
		t.Parallel()
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
		localClient := Init(logger, service)

		var batch batchservice.Batch
		for i := 0; i < correctTotal; i++ {
			batch = append(batch, batchservice.Item{ID: i})
		}

		requestCtx, _ := context.WithCancel(context.Background())
		localClient.Send(requestCtx, batch)
		time.Sleep(3 * time.Second)
	})

	t.Run("stops processing after request ctx.cancel", func(t *testing.T) {
		t.Parallel()
		correctTotal := 70

		service := batchservice.New(
			batchservice.WithNumber(17),
			batchservice.WithPeriod(1*time.Second),
		)

		localClient := Init(logger, service)

		var batch batchservice.Batch
		for i := 0; i < correctTotal; i++ {
			batch = append(batch, batchservice.Item{ID: i})
		}

		requestCtx, requestCancel := context.WithCancel(context.Background())
		localClient.Send(requestCtx, batch)
		time.Sleep(2 * time.Second)
		requestCancel()
	})

	t.Run("works properly with multiple calls", func(t *testing.T) {
		t.Parallel()
		correctTotal := 10000
		iterations := 5
		correctTiming := 50 * time.Millisecond
		var numberOfItems uint64 = 500

		service := batchservice.New(
			batchservice.WithNumber(numberOfItems),
			batchservice.WithPeriod(correctTiming),
		)
		localClient := Init(logger, service)

		requestCtx, _ := context.WithCancel(context.Background())

		for i := 0; i < iterations; i++ {
			batch := make(batchservice.Batch, 0, correctTotal)
			for j := 0; j < correctTotal; j++ {
				batch = append(batch, batchservice.Item{ID: i})
			}

			localClient.Send(requestCtx, batch)
		}

		time.Sleep(7 * time.Second)
	})

	t.Run("sends what remains if nothing get from the channel", func(t *testing.T) {
		t.Parallel()

		remainedAmount := 1
		var timesItGotRemained atomic.Uint64

		correctTotal := 5
		iterations := 5
		correctTiming := 100 * time.Millisecond
		var numberOfItems uint64 = 2

		service := batchservice.New(
			batchservice.WithNumber(numberOfItems),
			batchservice.WithPeriod(correctTiming),
			batchservice.WithTestHandler(func(batch batchservice.Batch) {
				if len(batch) == remainedAmount {
					timesItGotRemained.Add(1)
				}
			}),
		)
		localClient := Init(logger, service)

		requestCtx, _ := context.WithCancel(context.Background())

		for i := 0; i < iterations; i++ {
			batch := make(batchservice.Batch, 0, correctTotal)
			for j := 0; j < correctTotal; j++ {
				batch = append(batch, batchservice.Item{ID: j})
			}

			localClient.Send(requestCtx, batch)

			time.Sleep(500 * time.Millisecond)
		}

		time.Sleep(3 * time.Second)
		timesGotRemained := timesItGotRemained.Load()
		if timesGotRemained != uint64(correctTotal) {
			t.Fatalf("Incorrect timesGotRemained. Get %d, expected: %d", timesGotRemained, correctTotal)
		}
	})

	t.Run("sends only one item in a batch", func(t *testing.T) {
		t.Parallel()

		var timesItGotRemained atomic.Uint64
		var itemsInTheBatch atomic.Int32

		correctTotal := 1
		correctBatchLength := 1
		correctTiming := 100 * time.Millisecond
		var numberOfItems uint64 = 2

		service := batchservice.New(
			batchservice.WithNumber(numberOfItems),
			batchservice.WithPeriod(correctTiming),
			batchservice.WithTestHandler(func(batch batchservice.Batch) {
				itemsInTheBatch.Add(int32(len(batch)))

				timesItGotRemained.Add(1)
			}),
		)
		localClient := Init(logger, service)

		requestCtx, _ := context.WithCancel(context.Background())
		batch := batchservice.Batch{
			{ID: 1},
		}
		localClient.Send(requestCtx, batch)

		time.Sleep(2 * time.Second)

		timesGotRemained := timesItGotRemained.Load()
		if timesGotRemained != uint64(correctTotal) {
			t.Fatalf("Incorrect timesGotRemained. Get %d, expected: %d", timesGotRemained, correctTotal)
		}

		actualItemsLen := itemsInTheBatch.Load()
		if actualItemsLen != int32(correctBatchLength) {
			t.Fatalf("Incorrect actualItemsLen. Get %d, expected: %d", actualItemsLen, correctBatchLength)
		}
	})

	t.Run("sends Zero items in a batch", func(t *testing.T) {
		t.Parallel()

		var timesItGotRemained atomic.Uint64
		var itemsInTheBatch atomic.Int32

		correctTotal := 0
		correctBatchLength := 0
		correctTiming := 100 * time.Millisecond
		var numberOfItems uint64 = 2

		service := batchservice.New(
			batchservice.WithNumber(numberOfItems),
			batchservice.WithPeriod(correctTiming),
			batchservice.WithTestHandler(func(batch batchservice.Batch) {
				itemsInTheBatch.Add(int32(len(batch)))

				timesItGotRemained.Add(1)
			}),
		)
		localClient := Init(logger, service)

		requestCtx, _ := context.WithCancel(context.Background())
		batch := batchservice.Batch{}
		localClient.Send(requestCtx, batch)

		time.Sleep(2 * time.Second)

		timesGotRemained := timesItGotRemained.Load()
		if timesGotRemained != uint64(correctTotal) {
			t.Fatalf("Incorrect timesGotRemained. Get %d, expected: %d", timesGotRemained, correctTotal)
		}

		actualItemsLen := itemsInTheBatch.Load()
		if actualItemsLen != int32(correctBatchLength) {
			t.Fatalf("Incorrect actualItemsLen. Get %d, expected: %d", actualItemsLen, correctBatchLength)
		}
	})
}
