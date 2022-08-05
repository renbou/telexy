package tlxlog

import (
	"bytes"
	"errors"
	"fmt"
	"log"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStdLogger(t *testing.T) {
	var buf bytes.Buffer
	testOutput := func(t *testing.T, expected string) {
		t.Helper()
		s, _ := buf.ReadString('\x00')
		assert.Equal(t, expected, s)
	}

	logger := Std()
	log.SetFlags(0)
	log.SetOutput(&buf)

	tests := []struct {
		name     string
		withErr  bool
		err      error
		msg      string
		kvs      []any
		expected string
	}{
		{
			name:     "info log with two values",
			msg:      "test message",
			kvs:      []any{"key", "value"},
			expected: `INFO: msg="test message" key="value"` + "\n",
		},
		{
			name:     "info log with one value",
			msg:      "test msg",
			kvs:      []any{"val"},
			expected: `INFO: msg="test msg" "val"` + "\n",
		},
		{
			name:     "error log with nil error",
			withErr:  true,
			err:      nil,
			msg:      "err msg",
			kvs:      []any{"time", time.Second.String()},
			expected: `ERROR: errors=nil msg="err msg" time="1s"` + "\n",
		},
		{
			name:     "error log with actual errors",
			withErr:  true,
			err:      fmt.Errorf("error wrapper: %w", errors.New("another error")),
			msg:      "error chain",
			kvs:      nil,
			expected: `ERROR: errors=["error wrapper: another error", "another error"] msg="error chain"` + "\n",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			if tt.withErr {
				logger.Error(tt.err, tt.msg, tt.kvs...)
			} else {
				logger.Info(tt.msg, tt.kvs...)
			}
			testOutput(t, tt.expected)
		})
	}
}
