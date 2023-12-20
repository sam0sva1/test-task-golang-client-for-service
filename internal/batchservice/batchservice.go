package batchservice

import (
	"context"
	"errors"
	"fmt"
	"time"
)

var ErrBlocked = errors.New("blocked")

type ButchServiceConfig struct {
	amount      uint64
	period      time.Duration
	testHandler func(batch Batch)
}

func WithNumber(number uint64) func(*ButchServiceConfig) {
	return func(c *ButchServiceConfig) {
		c.amount = number
	}
}

func WithPeriod(period time.Duration) func(*ButchServiceConfig) {
	return func(c *ButchServiceConfig) {
		c.period = period
	}
}

func WithTestHandler(testHandler func(batch Batch)) func(*ButchServiceConfig) {
	return func(c *ButchServiceConfig) {
		c.testHandler = testHandler
	}
}

type ButchService struct {
	start       time.Time
	isFrozen    bool
	amount      uint64
	period      time.Duration
	freeze      time.Duration
	testHandler func(batch Batch)
}

func New(options ...func(config *ButchServiceConfig)) Service {
	cfg := &ButchServiceConfig{
		amount:      70,
		period:      100 * time.Millisecond,
		testHandler: nil,
	}

	for _, opt := range options {
		opt(cfg)
	}

	return &ButchService{
		start:       time.Now().Add(-1 * time.Second),
		isFrozen:    false,
		amount:      cfg.amount,
		period:      cfg.period,
		freeze:      5 * time.Second,
		testHandler: cfg.testHandler,
	}
}

func (s *ButchService) GetLimits() (n uint64, p time.Duration) {
	return s.amount, s.period
}

func (s *ButchService) Process(ctx context.Context, batch Batch) error {
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

	fmt.Println("===> Processing: ", len(batch))

	if s.testHandler != nil {
		s.testHandler(batch)
	}
	return nil
}
