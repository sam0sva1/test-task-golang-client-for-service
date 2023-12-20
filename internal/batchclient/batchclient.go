package batchclient

import (
	"context"
	"errors"
	"fmt"
	"service-client/internal/batchservice"
	"sync"
	"time"
)

type ButchClient struct {
	channel         chan batchservice.Batch
	done            chan struct{}
	service         batchservice.Service
	itemNumberLimit uint64
	periodLimit     time.Duration
}

var ErrClientAlreadyTerminated = errors.New("sending to terminated client")

var client *ButchClient
var once sync.Once

func Init(ctx context.Context, service batchservice.Service) *ButchClient {
	once.Do(func() {
		number, period := service.GetLimits()
		channel := make(chan batchservice.Batch)
		done := make(chan struct{})

		client = &ButchClient{
			channel,
			done,
			service,
			number,
			period,
		}

		go func(c *ButchClient) {
			for {
				select {
				case batch := <-c.channel:
					part := batch

					for {
						if uint64(len(part)) <= c.itemNumberLimit {
							c.process(ctx, part)
							break
						}

						toSend := part[:c.itemNumberLimit]
						part = part[c.itemNumberLimit:]

						c.process(ctx, toSend)
					}
				case <-ctx.Done():
					return
				default:

				}
			}
		}(client)
	})

	return client
}

func (c *ButchClient) process(ctx context.Context, newBatch batchservice.Batch) {
	after := time.After(c.periodLimit)

	err := c.service.Process(ctx, newBatch)
	if err != nil {
		fmt.Println("error", err)

		if errors.Is(err, batchservice.ErrBlocked) {
			time.Sleep(5 * time.Second)
		}
	}

	<-after
}

func (c *ButchClient) Send(ctx context.Context, newBatch batchservice.Batch) error {
	ch := make(chan error)

	go func() {
		select {
		case <-ctx.Done():
			ch <- ErrClientAlreadyTerminated
			return
		case c.channel <- newBatch:
			ch <- nil
			return
		}
	}()

	return <-ch
}
