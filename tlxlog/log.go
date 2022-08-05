// Package tlxlog (telexylog) describes the logging interface used by the client and the proxy
// to provide informational logs when running. Logging is necessary in their cases
// since both the client (i.e. long polling streamer) and the proxy are long-lived,
// attempting to gracefully handle most errors, reconnecting or restarting when possible.
package tlxlog

import (
	"errors"
	"fmt"
	"log"
	"strings"
)

// Logger defines the globally used logging interface. Its methods accept arguments as key-value pairs
// to allow both structured and non-structured logging. One thing worth noting is that this interface
// is implemented by go-logr/logr.Logger, which can be directly passed as a logger into telexy.
type Logger interface {
	Error(err error, msg string, kvs ...interface{})
	Info(msg string, kvs ...interface{})
}

type discard struct{}

func (discard) Error(err error, msg string, kvs ...interface{}) {}
func (discard) Info(msg string, kvs ...interface{})             {}

var discardSingleton = discard{}

// Discard returns a special logger the operations of which do absolutely nothing.
// NB: only use this if you're absolutely sure with letting everything go wrong someday.
func Discard() Logger {
	return discardSingleton
}

type std struct{}

func (s std) format(sb *strings.Builder, msg string, kvs ...interface{}) {
	sb.WriteString(fmt.Sprintf("msg=%q", msg))
	for i := 0; i < len(kvs)-1; i += 2 {
		sb.WriteString(fmt.Sprintf(" %v=%#v", kvs[i], kvs[i+1]))
	}
	if len(kvs)%2 == 1 {
		sb.WriteString(fmt.Sprintf(" %#v", kvs[len(kvs)-1]))
	}
	log.Println(sb.String())
}

func (s std) Error(err error, msg string, kvs ...interface{}) {
	var sb strings.Builder
	sb.WriteString("ERROR: errors=")
	if err == nil {
		sb.WriteString("nil ")
	} else {
		sb.WriteByte('[')
		first := true
		for err != nil {
			if !first {
				sb.WriteString(", ")
			} else {
				first = false
			}

			sb.WriteString(fmt.Sprintf("%q", err.Error()))
			err = errors.Unwrap(err)
		}
		sb.WriteString("] ")
	}
	s.format(&sb, msg, kvs...)
}

func (s std) Info(msg string, kvs ...interface{}) {
	var sb strings.Builder
	sb.WriteString("INFO: ")
	s.format(&sb, msg, kvs...)
}

var stdSingleton = std{}

// Std returns a logger which logs everything to the Go standard library logger
// by calling log.Printf on the received and formatted messages. It is used by
// default when no logger is passed to some component to provide some valid source
// of logs in case something goes wrong.
func Std() Logger {
	return stdSingleton
}

// WithDefault is a utility which either returns the logger passed to it, if it isn't nil,
// or returns the default Std logger which can be used by any part of telexy.
func WithDefault(l Logger) Logger {
	if l != nil {
		return l
	}
	return Std()
}
