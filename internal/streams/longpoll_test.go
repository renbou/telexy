package streams

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	jsoniter "github.com/json-iterator/go"
	"github.com/renbou/telexy/internal/api"
	"github.com/stretchr/testify/require"
	"gopkg.in/telebot.v3"
)

func encodeTestAPIResponse(data any) ([]byte, error) {
	b, err := jsoniter.Marshal(api.Response{Ok: true, Result: data})
	if err != nil {
		return nil, fmt.Errorf("marshaling api response: %w", err)
	}
	return b, nil
}

type longPollTestRoundTripper struct {
	data  []*bytes.Reader
	empty []byte
	i     int64
}

const responseSleep = time.Millisecond * 50

func (rt *longPollTestRoundTripper) RoundTrip(req *http.Request) (*http.Response, error) {
	select {
	case <-req.Context().Done():
		return nil, req.Context().Err()
	default:
	}
	response := func(b *bytes.Reader) *http.Response {
		return &http.Response{
			StatusCode:    http.StatusOK,
			ContentLength: b.Size(),
			Body:          io.NopCloser(b),
		}
	}

	if strings.HasSuffix(req.URL.Path, "getMe") {
		b, err := encodeTestAPIResponse(tgbotapi.User{ID: 1})
		if err != nil {
			return nil, err
		}
		return response(bytes.NewReader(b)), nil
	}

	cur := atomic.AddInt64(&rt.i, 1)
	if cur >= int64(len(rt.data)) {
		time.Sleep(responseSleep) // sleep so that we aren't blatantly wasting cpu cycles
		return response(bytes.NewReader(rt.empty)), nil
	}
	return response(rt.data[cur]), nil
}

// bNumReqs returns the number of batches of requests and number of requests in each batch
func bNumReqs(b *testing.B) (int, int) {
	b.Helper()
	return b.N, DefaultLongPollLimit
}

func newLongPollTestClient(b *testing.B) *http.Client {
	b.Helper()
	n, by := bNumReqs(b)
	data := make([]*bytes.Reader, n)
	for i := range data {
		updates := make([]telebot.Update, by)
		for j := range updates {
			updates[j] = telebot.Update{
				ID:      i*by + j,
				Message: &telebot.Message{Text: "long polling text"},
			}
		}

		raw, err := encodeTestAPIResponse(updates)
		require.NoError(b, err)
		data[i] = bytes.NewReader(raw)
	}

	empty, err := encodeTestAPIResponse([]tgbotapi.Update{})
	require.NoError(b, err)
	return &http.Client{Transport: &longPollTestRoundTripper{data: data, empty: empty, i: -1}}
}

func longPollTestValidate(b *testing.B, stop func(), s Stream[tgbotapi.Update]) {
	b.Helper()
	n, by := bNumReqs(b)
	cnt, end := 0, n*by
	for range s {
		cnt++
		if cnt > end {
			b.Error("cnt > end")
			b.FailNow()
		} else if cnt == end {
			stop()
		}
	}
}

// Benchmark of simple long polling using a single goroutine receiving messages and decoding them
func BenchmarkNaiveLongPoll(b *testing.B) {
	n, by := bNumReqs(b)
	b.Logf("Decoding %d requests by %d messages", n, by)
	api, err := tgbotapi.NewBotAPIWithClient("faketoken", tgbotapi.APIEndpoint, newLongPollTestClient(b))
	require.NoError(b, err)

	b.ResetTimer()
	stream := api.GetUpdatesChan(tgbotapi.UpdateConfig{})
	longPollTestValidate(b, api.StopReceivingUpdates, Stream[tgbotapi.Update](stream))
}

func BenchmarkOptimizedLongPoll(b *testing.B) {
	n, by := bNumReqs(b)
	b.Logf("Decoding %d requests by %d messages", n, by)
	client, err := api.NewClient(telebot.DefaultApiURL, "faketoken", &api.ClientOpts{
		Client: newLongPollTestClient(b),
	})
	require.NoError(b, err)
	streamer := NewLongPollStreamer(client, TgBotAPIDecoder, nil)
	ctx, cancel := context.WithCancel(context.Background())

	b.ResetTimer()
	stream, errstream := streamer.Stream(ctx)
	longPollTestValidate(b, cancel, stream)
	select {
	case err := <-errstream:
		require.NoError(b, err)
	default:
	}
}
