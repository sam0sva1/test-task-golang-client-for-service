package batchclient

import (
	"context"
	"errors"
	"fmt"
	"service-client/internal/batchservice"
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

func Init(service batchservice.Service) *ButchClient {
	number, period := service.GetLimits()
	channel := make(chan struct{})
	done := make(chan struct{})

	client := &ButchClient{
		channel,
		done,
		service,
		number,
		period,
	}

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

func (c *ButchClient) Send(ctx context.Context, newBatch batchservice.Batch) {
	go func() {
		chiItem := &ChannelItem{
			reqContext: ctx,
			batch:      newBatch,
		}
		go c.processBatch(chiItem)
		<-c.channel
	}()
}
