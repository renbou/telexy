package retry

import (
	"fmt"
	"testing"

	"github.com/renbou/telexy/tlxlog"
	"github.com/stretchr/testify/assert"
)

func assertNumCallsFunc(a *assert.Assertions, n int, tmpErr, finalErr error) RecoverFunc {
	ctr := 0
	return func() error {
		a.Less(ctr, n, "should be called %d times at most", n)
		ctr++
		if ctr == n {
			return finalErr
		}
		return tmpErr
	}
}

func TestRetry(t *testing.T) {
	strategies := []struct {
		name string
		f    func(tlxlog.Logger, RecoverFunc) error
	}{
		{"backoff", Backoff},
		{"static", Static},
	}
	recoverable := Recoverable(assert.AnError, "recoverable")

	for n := 1; n < 4; n++ {
		for _, strategy := range strategies {
			t.Run(fmt.Sprintf("retry using %s %d times", strategy.name, n), func(t *testing.T) {
				a := assert.New(t)
				a.NoError(strategy.f(
					tlxlog.Discard(), assertNumCallsFunc(a, n, recoverable, nil),
				))
				a.ErrorIs(strategy.f(
					tlxlog.Discard(), assertNumCallsFunc(a, n, recoverable, assert.AnError),
				), assert.AnError)
			})
		}
	}
}
