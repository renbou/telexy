package streams

import (
	"context"
	"encoding/json"
)

// Stream is a readonly channel of some type.
type Stream[T any] <-chan T

// ErrStream is a stream of errors. Usually streaming functions return streams in
// pairs: a value stream, and an error stream. By default returned error streams
// should be monitored for errors as value streams, if not specified otherwise,
// will close as soon as an unhandled (and thus sent via the error stream) error occurs.
type ErrStream Stream[error]

// Streamer is an interface implemented by various stream providers and consists of a single
// function which returns a pair of value and error streams. The returned streams should be closed
// when the context is canceled or times out, as well as when an error occurs.
type Streamer[T any] interface {
	Stream(ctx context.Context) (Stream[T], ErrStream)
}

// parseFunc is a type commonly used by the default streamers for stream-like parsing of the incoming
// updates, which allows to reduce memory allocations and the CPU overhead of constantly copying values
type parseFunc[T any] func(*json.Decoder) (T, error)
