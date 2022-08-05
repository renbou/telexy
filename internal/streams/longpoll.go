package streams

import (
	"context"
	"errors"
	"fmt"
	"time"

	jsoniter "github.com/json-iterator/go"
	"github.com/renbou/telexy/internal/api"
	"github.com/renbou/telexy/internal/retry"
	"github.com/renbou/telexy/tlxlog"
)

// LongPollOptions specify various options to use inside the long poll streamer.
// Limit and Timeout specify the values to send to Telegram's getUpdates method.
type LongPollOptions struct {
	Limit   int
	Timeout time.Duration
	Logger  tlxlog.Logger
}

const (
	DefaultLongPollLimit   = 100
	DefaultLongPollTimeout = time.Second * 60
)

type longPollStreamer[T any] struct {
	*LongPollOptions
	parser UpdateDecoder[T]
	client *api.Client
}

func (s *longPollStreamer[T]) poll(ctx context.Context, offset int, stream chan T) (int, error) {
	newOffset := offset
	err := retry.Backoff(s.Logger, func() error {
		rctx, cancel := context.WithTimeout(ctx, s.Timeout)
		defer cancel()

		err := s.client.GetUpdates(rctx, api.GetUpdatesRequest{
			Offset:  newOffset,
			Limit:   s.Limit,
			Timeout: int(s.Timeout.Seconds()),
		}, func(ui api.UpdateInfo, it *jsoniter.Iterator) error {
			t, err := s.parser(ui, it)
			if err != nil {
				return err
			}

			// Either wait for the update to go through, or for the context to timeout
			select {
			case stream <- t:
				// Modify the offset only after the update is actually consumed
				newOffset = ui.ID + 1
			case <-ctx.Done():
			}
			return nil
		})

		// Everything went fine or the request/global context has timed out, lets end this attempt
		if err == nil || ctx.Err() != nil || errors.Is(err, context.DeadlineExceeded) {
			return nil
		}
		// Always try to recover in hope of being able to get some updates...
		// TODO: properly handle status codes from the API and don't try to recover on
		// unrecoverable errors such as 401
		return retry.Recoverable(err, "failed to get updates via long polling")
	})
	// Currently no error should be returned since we always retry...
	// But let's not ignore the returned value for good measure
	if err != nil {
		err = fmt.Errorf("critical error while long polling: %w", err)
	}
	return newOffset, err
}

func (s *longPollStreamer[T]) Stream(ctx context.Context) (Stream[T], ErrStream) {
	stream, errStream := make(chan T, s.Limit), make(chan error, 1)
	go func() {
		defer close(stream)
		defer close(errStream)

		var offset int
		for {
			if err := ctx.Err(); err != nil {
				errStream <- nil
				return
			}

			newOffset, err := s.poll(ctx, offset, stream)
			if err != nil {
				errStream <- err
				return
			}
			offset = newOffset
		}
	}()
	return stream, errStream
}

// NewLongPollStreamer creates a new long polling streamer with the specified options.
// It uses the client's GetUpdates method for long polling.
func NewLongPollStreamer[T any](client *api.Client, parser UpdateDecoder[T], opts *LongPollOptions) Streamer[T] {
	if opts == nil {
		opts = &LongPollOptions{}
	}
	if opts.Limit == 0 {
		opts.Limit = DefaultLongPollLimit
	}
	if opts.Timeout == 0 {
		opts.Timeout = DefaultLongPollTimeout
	}
	opts.Logger = tlxlog.WithDefault(opts.Logger)

	return &longPollStreamer[T]{LongPollOptions: opts, parser: parser, client: client}
}
