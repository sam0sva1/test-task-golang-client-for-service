package batchservice

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"
)

var ErrBlocked = errors.New("blocked")

type BatchServiceConfig struct {
	amountNumber uint64
	period       time.Duration
	testHandler  func(batch Batch)
}

func WithNumber(number uint64) func(*BatchServiceConfig) {
	return func(c *BatchServiceConfig) {
		c.amountNumber = number
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
	start        time.Time
	isFrozen     bool
	amountNumber uint64
	period       time.Duration
	freeze       time.Duration
	testHandler  func(batch Batch)
}

func New(options ...func(config *BatchServiceConfig)) Service {
	cfg := &BatchServiceConfig{
		testHandler: nil,
	}

	for _, opt := range options {
		opt(cfg)
	}

	if cfg.period == 0 {
		log.Fatal("BatchService error: \"period\" param is required ")
	}

	if cfg.amountNumber == 0 {
		log.Fatal("BatchService error: \"number\" param is required ")
	}

	return &BatchService{
		start:        time.Now().Add(-1 * time.Second),
		isFrozen:     false,
		amountNumber: cfg.amountNumber,
		period:       cfg.period,
		freeze:       5 * time.Second,
		testHandler:  cfg.testHandler,
	}
}

func (s *BatchService) GetLimits() (n uint64, p time.Duration) {
	return s.amountNumber, s.period
}

func (s *BatchService) Process(ctx context.Context, batch Batch) error {
	_ = ctx
	delta := time.Since(s.start)
	s.start = time.Now()

	if s.isFrozen && delta > s.freeze {
		s.isFrozen = false
	}

	if delta < s.period || uint64(len(batch)) > s.amountNumber {
		fmt.Println("**** Frozen: ")
		s.isFrozen = true
		return ErrBlocked
	}

	if s.testHandler != nil {
		s.testHandler(batch)
	}
	return nil
}
