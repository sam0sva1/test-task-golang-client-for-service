package batchservice

import (
	"context"
	"errors"
	"testing"
	"time"
)

var ctx context.Context
var service Service

func Setup() {
	ctx, _ = context.WithCancel(context.Background())
	service = New(
		WithNumber(5),
		WithPeriod(1*time.Second),
	)
}

func TestBatchService_GetLimits(t *testing.T) {
	t.Run("sets limits and gets correct", func(t *testing.T) {
		t.Parallel()

		number := uint64(1234)
		period := 4321 * time.Millisecond

		localService := New(
			WithNumber(number),
			WithPeriod(period),
		)

		returnedNumber, returnedPeriod := localService.GetLimits()

		if returnedNumber != number {
			t.Fatalf("returned incorrect number")
		}

		if returnedPeriod != period {
			t.Fatalf("returned incorrect period")
		}
	})
}

func TestBatchService_Process(t *testing.T) {
	t.Run("with correct amount", func(t *testing.T) {
		Setup()

		number, _ := service.GetLimits()

		x := int(number)

		var batch Batch
		for i := 0; i < x; i++ {
			batch = append(batch, Item{ID: i})
		}

		err := service.Process(ctx, batch)

		if err != nil {
			t.Fatalf("incorrect cause of block")
		}
	})

	t.Run("with correct timing", func(t *testing.T) {
		Setup()

		_, period := service.GetLimits()

		batch := Batch{
			Item{ID: 1},
		}

		err := service.Process(ctx, batch)
		time.Sleep(period)
		err = service.Process(ctx, batch)

		if err != nil {
			t.Fatalf("incorrect cause of block")
		}
	})

	t.Run("with too many items", func(t *testing.T) {
		Setup()
		number, _ := service.GetLimits()

		x := int(number) + 1

		var batch Batch
		for i := 0; i < x; i++ {
			batch = append(batch, Item{ID: i})
		}

		err := service.Process(ctx, batch)

		if err != ErrBlocked {
			t.Fatalf("cannot catch block cause")
		}
	})

	t.Run("with incorrect timing", func(t *testing.T) {
		Setup()

		batch := Batch{
			Item{ID: 1},
		}

		err := service.Process(ctx, batch)
		err = service.Process(ctx, batch)

		if !errors.Is(err, ErrBlocked) {
			t.Fatalf("incorrect cause of block")
		}
	})

	t.Run("with incorrect timing", func(t *testing.T) {
		Setup()

		batch := Batch{
			Item{ID: 1},
		}

		service.Process(ctx, batch)
		service.Process(ctx, batch)

		time.Sleep(6 * time.Second)
		err := service.Process(ctx, batch)

		if err != nil {
			t.Fatalf("Cannot wake up after freeze")
		}
	})

	t.Run("runs a special test handler if provided", func(t *testing.T) {
		t.Parallel()

		isHandlerActivated := false

		localCtx, _ := context.WithCancel(context.Background())
		service = New(
			WithNumber(5),
			WithPeriod(1*time.Second),
			WithTestHandler(func(batch Batch) {
				isHandlerActivated = true
			}),
		)

		batch := Batch{
			Item{ID: 1},
		}

		service.Process(localCtx, batch)

		if !isHandlerActivated {
			t.Fatalf("Cannot run test handler")
		}
	})
}
