package batchclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"service-client/internal/batchservice"
	"time"
)

// ChannelItem contains one batch and a context related to it
type ChannelItem struct {
	batch      batchservice.Batch
	reqContext context.Context
}

type ButchClient struct {
	// Replace with logger interface
	logger          *slog.Logger
	channel         chan struct{}
	done            chan struct{}
	service         batchservice.Service
	itemNumberLimit uint64
	periodLimit     time.Duration
}

func Init(logger *slog.Logger, service batchservice.Service) *ButchClient {
	number, period := service.GetLimits()
	channel := make(chan struct{})
	done := make(chan struct{})

	go func() {
		channel <- struct{}{}
	}()

	client := &ButchClient{
		logger,
		channel,
		done,
		service,
		number,
		period,
	}

	return client
}

// processBatch cuts a batch into chunks to comply underlying service restrictions.
// At the end of processing it releases the queue
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

// processItem runs only synchronously because of after channel synchronization
// next call runs only after period limit or process response
func (c *ButchClient) processItem(ctx context.Context, newBatch batchservice.Batch) {
	mark := "batchClient.processItem"

	after := time.After(c.periodLimit)

	err := c.service.Process(ctx, newBatch)
	if err != nil {
		c.logger.Error(fmt.Sprintf("%s: %s", mark, err))

		if errors.Is(err, batchservice.ErrBlocked) {
			c.logger.Error(fmt.Sprintf("%s: freeze accured", mark))
			time.Sleep(5 * time.Second)
		}
	}

	<-after
}

// Send processes batch items one by one avoiding concurrent start
// but not blocking a caller function
// and this way avoiding blockage of underlying service
func (c *ButchClient) Send(ctx context.Context, newBatch batchservice.Batch) {
	go func() {
		<-c.channel
		chiItem := &ChannelItem{
			reqContext: ctx,
			batch:      newBatch,
		}
		go c.processBatch(chiItem)
	}()
}
