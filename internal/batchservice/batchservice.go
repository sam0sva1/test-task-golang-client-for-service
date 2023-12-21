package batchservice

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrBlocked = errors.New("blocked")

type BatchServiceConfig struct {
	amount      uint64
	period      time.Duration
	testHandler func(batch Batch)
}

func WithNumber(number uint64) func(*BatchServiceConfig) {
	return func(c *BatchServiceConfig) {
		c.amount = number
	}
}

func WithPeriod(period time.Duration) func(*BatchServiceConfig) {
	return func(c *BatchServiceConfig) {
		c.period = period
	}
}

func WithTestHandler(testHandler func(batch Batch)) func(*BatchServiceConfig) {
	return func(c *BatchServiceConfig) {
		c.testHandler = testHandler
	}
}

type BatchService struct {
	start       time.Time
	isFrozen    bool
	amount      uint64
	period      time.Duration
	freeze      time.Duration
	testHandler func(batch Batch)
}

func New(options ...func(config *BatchServiceConfig)) Service {
	cfg := &BatchServiceConfig{
		amount:      70,
		period:      100 * time.Millisecond,
		testHandler: nil,
	}

	for _, opt := range options {
		opt(cfg)
	}

	return &BatchService{
		start:       time.Now().Add(-1 * time.Second),
		isFrozen:    false,
		amount:      cfg.amount,
		period:      cfg.period,
		freeze:      5 * time.Second,
		testHandler: cfg.testHandler,
	}
}

func (s *BatchService) GetLimits() (n uint64, p time.Duration) {
	return s.amount, s.period
}

func (s *BatchService) Process(ctx context.Context, batch Batch) error {
	_ = ctx
	delta := time.Since(s.start)
	s.start = time.Now()

	if s.isFrozen && delta > s.freeze {
		s.isFrozen = false
	}

	if delta < s.period || uint64(len(batch)) > s.amount {
		fmt.Println("**** Frozen: ")
		s.isFrozen = true
		return ErrBlocked
	}

	if s.testHandler != nil {
		s.testHandler(batch)
	}
	return nil
}
