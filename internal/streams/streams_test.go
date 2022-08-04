package streams

import (
	"time"

	"github.com/stretchr/testify/require"
)

// streamIs validates that a stream outputs values equal to the ones wanted in the specified order
func streamIs[T any](req *require.Assertions, s Stream[T], want []T) {
	n := 0
	req.Eventually(func() bool {
		select {
		case val, ok := <-s:
			if !ok {
				req.Equal(len(want), n, "not all wanted values found on stream")
				return true
			}

			req.Less(n, len(want), "got more values than wanted on stream")
			req.Equal(want[n], val)
			n++
		default:
		}
		return false
	}, time.Second*5, time.Millisecond*50)
}
