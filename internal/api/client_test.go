package api

import (
	"math/rand"
	"net/http"
	"path"
	"testing"

	"github.com/stretchr/testify/require"
)

var benchmarkCommonMethods = [...]string{"getUpdates", "sendMessage", "sendSticker", "setMyCommands", "getMe"}

func benchmarkPickMethods(n int) []string {
	rnd := rand.New(rand.NewSource(0))
	methods := make([]string, n)
	for i := range methods {
		methods[i] = benchmarkCommonMethods[rnd.Intn(len(benchmarkCommonMethods))]
	}
	return methods
}

func benchmarkMethodURL(b *testing.B, f func(*Client, string) string) {
	b.Helper()
	c, err := NewClient("https://api.telegram.org", "faketoken", nil)
	if err != nil {
		b.Error(err)
		b.FailNow()
	}
	methods := benchmarkPickMethods(b.N)
	b.StartTimer()
	for _, method := range methods {
		_ = f(c, method)
	}
}

func BenchmarkClient_methodURLCached(b *testing.B) {
	b.StopTimer()
	f := func(c *Client, method string) string {
		return c.methodURL(method)
	}
	benchmarkMethodURL(b, f)
}

func BenchmarkClient_methodURLNaive(b *testing.B) {
	b.StopTimer()
	f := func(c *Client, method string) string {
		urlCopy := *c.endpointURL
		urlCopy.Path = path.Join(urlCopy.Path, method)
		return urlCopy.String()
	}
	benchmarkMethodURL(b, f)
}

func TestNewClient(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		endpoint   string
		token      string
		wantErr    bool
		wantClient *http.Client
	}{
		{
			name:     "invalid api endpoint",
			endpoint: "bad endpoint\n\x00",
			token:    "any",
			wantErr:  true,
		},
		{
			name:       "valid client creation",
			endpoint:   "https://api.telegram.org",
			token:      "bot-token",
			wantClient: http.DefaultClient,
		},
		{
			name:       "valid client creation with custom client",
			endpoint:   "https://api.telegram.org",
			token:      "bot-token",
			wantClient: &http.Client{Transport: &http.Transport{ResponseHeaderTimeout: 1}},
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := require.New(t)

			var opts *ClientOpts
			if tt.wantClient != nil && tt.wantClient != http.DefaultClient {
				opts = &ClientOpts{tt.wantClient}
			}
			gotClient, err := NewClient(tt.endpoint, tt.token, opts)
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
				req.NotNil(gotClient)
				req.Equal(tt.wantClient, gotClient.hc)
			}
		})
	}
}
