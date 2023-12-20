package batchservice

import (
	"context"
	"time"
)

type Service interface {
	GetLimits() (n uint64, p time.Duration)
	Process(ctx context.Context, batch Batch) error
}

// Batch is a batch of items.
type Batch []Item

// Item is some abstract item.
type Item struct {
	ID int
}
