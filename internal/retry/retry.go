// Package retry hosts utilities for retrying various calls with logging.
// All functions in this package operate using a predefined set of constants
// for simplicity, which can be changed externally if needed.
package retry

import (
	"errors"
	"fmt"
	"time"

	"github.com/renbou/telexy/tlxlog"
)

type (
	RecoverFunc    func() error
	DelayScheduler func() time.Duration
)

type recoverError struct {
	wrapped error
	msg     string
	kvs     []interface{}
}

func (e recoverError) Error() string {
	return fmt.Sprintf("recoverable error: %s", e.wrapped.Error())
}

func (e recoverError) Unwrap() error {
	return e.wrapped
}

// Recoverable is used to explicitly mark a recovery along with information
// about the error that occurred, which will be logged upon recovery.
func Recoverable(err error, msg string, kvs ...interface{}) error {
	return recoverError{err, msg, kvs}
}

// Recover runs the function using a custom delay scheduler. If the function
// returns an error upon being called, if it is marked as Recoverable it will
// be logged and the call will be retried according to the delay scheduler.
// Otherwise it will be returned (meaning it is nil or unrecoverable).
func Recover(logger tlxlog.Logger, f RecoverFunc, s DelayScheduler) error {
	logger = tlxlog.WithDefault(logger)
	for {
		var re recoverError
		if err := f(); !errors.As(err, &re) {
			return err
		}

		delay := s()
		logger.Error(re, re.msg, append(re.kvs, "delay", delay)...)
		time.Sleep(delay)
	}
}

var (
	DefaultBackoffMinDelay = time.Millisecond * 50
	DefaultBackoffMaxDelay = time.Minute * 10
	DefaultBackoffFactor   = 2
)

// Backoff runs the function using the backoff retry algorithm.
func Backoff(logger tlxlog.Logger, f RecoverFunc) error {
	delay, next := time.Duration(0), DefaultBackoffMinDelay
	return Recover(logger, f, func() time.Duration {
		delay, next = next, next*time.Duration(DefaultBackoffFactor)
		if next > DefaultBackoffMaxDelay {
			next = DefaultBackoffMaxDelay
		}
		return delay
	})
}

var DefaultStaticDelay = time.Second

// Static runs the function using a static retry delay.
func Static(logger tlxlog.Logger, f RecoverFunc) error {
	return Recover(logger, f, func() time.Duration {
		return DefaultStaticDelay
	})
}
