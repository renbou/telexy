package streams

import (
	"context"
	"math"
	"time"

	"github.com/renbou/telexy/internal/api"
)

// LongPollOptions specify various options to use inside the long poll streamer.
// Limit and Timeout specify the values to send to Telegram's getUpdates method.
type LongPollOptions struct {
	Limit   int
	Timeout time.Duration
}

// DefaultLongPollOpts are the default options for the long polling streamer
var DefaultLongPollOpts = LongPollOptions{
	Limit:   100,
	Timeout: time.Second * 60,
}

type longPollStreamer[T any] struct {
	opts   api.GetUpdatesRequest
	parser parseFunc[T]
	client *api.Client
}

func (s *longPollStreamer[T]) Stream(ctx context.Context) (Stream[T], ErrStream) {
	stream, errStream := make(chan T, s.opts.Limit), make(chan error, 1)
	go func() {
		defer close(stream)
		defer close(errStream)

		for {
		}
	}()
	return stream, errStream
}

// NewLongPollStreamer creates a new long polling streamer with the specified options.
// It uses the client's GetUpdates method for long polling.
func NewLongPollStreamer[T any](client *api.Client, parser parseFunc[T], opts *LongPollOptions) Streamer[T] {
	if opts == nil {
		opts = &LongPollOptions{}
	}
	if opts.Limit == 0 {
		opts.Limit = DefaultLongPollOpts.Limit
	}
	if opts.Timeout == 0 {
		opts.Timeout = DefaultLongPollOpts.Timeout
	}

	return &longPollStreamer[T]{
		api.GetUpdatesRequest{
			Limit:   opts.Limit,
			Timeout: int(math.Round(opts.Timeout.Seconds())),
		},
		parser,
		client,
	}
}
