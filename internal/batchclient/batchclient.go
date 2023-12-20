package batchclient

import (
	"context"
	"errors"
	"fmt"
	"service-client/internal/batchservice"
	"sync"
	"time"
)

type ChannelItem struct {
	batch      batchservice.Batch
	reqContext context.Context
}

type ButchClient struct {
	channel         chan struct{}
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
		channel := make(chan struct{})
		done := make(chan struct{})

		client = &ButchClient{
			channel,
			done,
			service,
			number,
			period,
		}
	})

	go func() {
		<-ctx.Done()
		fmt.Println("GLOBAL CANCELED")
		close(client.done)
	}()

	return client
}

func (c *ButchClient) processBatch(chItem *ChannelItem) {
	defer func() {
		c.channel <- struct{}{}
	}()

	part := chItem.batch

	for {
		select {
		case <-chItem.reqContext.Done():
			fmt.Println("CANCELED")
			return

		default:
			if uint64(len(part)) <= c.itemNumberLimit {
				c.processItem(chItem.reqContext, part)
				return
			}

			toSend := part[:c.itemNumberLimit]
			part = part[c.itemNumberLimit:]

			c.processItem(chItem.reqContext, toSend)
		}
	}
}

func (c *ButchClient) processItem(ctx context.Context, newBatch batchservice.Batch) {
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
	select {
	case <-c.done:
		fmt.Println("ALREADY TERMINATED")
		return ErrClientAlreadyTerminated
	default:
		go func() {
			chiItem := &ChannelItem{
				reqContext: ctx,
				batch:      newBatch,
			}
			go c.processBatch(chiItem)
			<-c.channel
		}()

		return nil
	}
}
