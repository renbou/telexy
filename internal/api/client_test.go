package api

import (
	"math/rand"
	"path"
	"testing"
)

const benchmarkSeed = 0x13337

var benchmarkCommonMethods = [...]string{"getUpdates", "sendMessage", "sendSticker", "setMyCommands", "getMe"}

func benchmarkPickMethods(n int) []string {
	rand.Seed(benchmarkSeed)
	methods := make([]string, n)
	for i := range methods {
		methods[i] = benchmarkCommonMethods[rand.Intn(len(benchmarkCommonMethods))]
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
