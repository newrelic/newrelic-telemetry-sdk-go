package telemetry

import (
	"context"
	"errors"
	"fmt"
	"net/http"
)

var DiscardBatchError = errors.New("Discarding batch recommended")
var RetryWithSplitError = errors.New("Retry with split recommended")
var RetryWithBackOffError = errors.New("Batch retry recommended")

type BatchSender struct {
	factory RequestFactory
	client  *http.Client
}

func NewBatchSender(factory RequestFactory, client *http.Client) *BatchSender {
	return &BatchSender{factory, client}
}

func (b *BatchSender) SendBatch(batches []Batch, ctx context.Context) error {
	reqs, err := BuildSplitRequests(batches, b.factory)
	if err != nil {
		return err
	}

	for _, req := range reqs {
		req := req.WithContext(ctx)

		resp, err := b.client.Do(req)
		if nil != err {
			return fmt.Errorf("error posting data: %v. %w", err, RetryWithBackOffError)
		}
		if err := ctx.Err(); err != nil {
			return err
		}

		switch resp.StatusCode {
		case 200, 202:
			continue
		case 400, 403, 404, 405, 411:
			return DiscardBatchError
		case 413:
			return RetryWithSplitError
		default:
			return RetryWithBackOffError
		}
	}
	return nil
}
