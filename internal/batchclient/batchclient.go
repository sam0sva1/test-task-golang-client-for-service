package batchclient

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"service-client/internal/batchservice"
	"time"
)

// ChannelItem contains one batch and a context related to it.
type ChannelItem struct {
	batch      batchservice.Batch
	reqContext context.Context
}

type BatchClient struct {
	// Replace with logger interface
	logger          *slog.Logger
	queue           chan ChannelItem
	service         batchservice.Service
	itemNumberLimit uint64
	periodLimit     time.Duration
}

func Init(logger *slog.Logger, service batchservice.Service) *BatchClient {
	queue := make(chan ChannelItem)

	client := &BatchClient{
		logger:  logger,
		queue:   queue,
		service: service,
	}

	client.resetLimits()

	go client.startMainLoop()

	return client
}

func (c *BatchClient) resetLimits() {
	number, period := c.service.GetLimits()
	c.itemNumberLimit = number
	c.periodLimit = period
}

// startMainLoop begins to wait for chanItem to process batches.
// During iterations the main loop make sure that a sending batch is full.
// Otherwise, adds more items from the next batch
func (c *BatchClient) startMainLoop() {
	var accum batchservice.Batch
	var lastContext context.Context

	for {
	escapeInnerLoop:

		select {
		case chanItem := <-c.queue:
			{
				accum = append(chanItem.batch)
				lastContext = chanItem.reqContext

				for {
					if uint64(len(accum)) < c.itemNumberLimit {
						break escapeInnerLoop
					}

					batchToSend := accum[:c.itemNumberLimit]
					accum = accum[c.itemNumberLimit:]

					c.processBatch(lastContext, batchToSend)
				}
			}
		default:
			if len(accum) > 0 {
				c.processBatch(lastContext, accum)
				accum = nil
				lastContext = nil
			}
		}
	}
}

// processBatch runs only synchronously because of after-channel synchronization.
// Next call runs only after period limit or process response.
func (c *BatchClient) processBatch(ctx context.Context, newBatch batchservice.Batch) {
	mark := "batchClient.processItem"

	after := time.After(c.periodLimit)

	err := c.service.Process(ctx, newBatch)
	if err != nil {
		c.logger.Error(fmt.Sprintf("%s: %s", mark, err))

		if errors.Is(err, batchservice.ErrBlocked) {
			freezeWait := time.After(5 * time.Second)
			c.logger.Error(fmt.Sprintf("%s: freeze accured", mark))

			// Ensures that limits are up-to-date
			c.resetLimits()

			<-freezeWait
		}
	}

	<-after
}

// Send starts a goroutine that processes batch items one by one avoiding concurrent start
// but not blocking Send function itself
// and this way avoiding blockage of underlying service.
func (c *BatchClient) Send(ctx context.Context, newBatch batchservice.Batch) {
	go func() {
		chiItem := ChannelItem{
			reqContext: ctx,
			batch:      newBatch,
		}

		select {
		case <-ctx.Done():
			return
		case c.queue <- chiItem:
		}
	}()
}
