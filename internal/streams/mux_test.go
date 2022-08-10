package streams

import (
	"math/rand"
	"runtime"
	"sync"
	"sync/atomic"
	"testing"

	jsoniter "github.com/json-iterator/go"
	"github.com/renbou/telexy/internal/api"
	"github.com/stretchr/testify/require"
)

func randomString(rnd *rand.Rand) string {
	b := make([]byte, 10+rnd.Intn(10))
	for i := range b {
		b[i] = byte('a' + rnd.Intn(26))
	}
	return string(b)
}

func randomTestUpdate(rnd *rand.Rand,
	updateID *atomic.Uint64, opts SubscriptionOpts,
) Update {
	return Update{
		UpdateInfo: api.UpdateInfo{
			ID:   int(updateID.Add(1)),
			Type: api.UpdateMessage,
		},
		Content: jsoniter.Wrap(map[string]string{
			"text": opts.Commands[rnd.Intn(len(opts.Commands))] + " " + randomString(rnd),
		}),
	}
}

func randomTestSubOpts(rnd *rand.Rand) SubscriptionOpts {
	opts := SubscriptionOpts{
		Commands: make([]string, 1+rnd.Intn(3)),
	}
	for i := range opts.Commands {
		opts.Commands[i] = "/" + randomString(rnd)
	}
	return opts
}

func testSubscriber(t *testing.T, seed int64, mux *Mux, input chan Update,
	updateID *atomic.Uint64, wg *sync.WaitGroup,
) {
	t.Helper()

	rnd := rand.New(rand.NewSource(seed))
	opts := randomTestSubOpts(rnd)

	updates := make([]Update, 1+rnd.Intn(200))
	for i := range updates {
		updates[i] = randomTestUpdate(rnd, updateID, opts)
	}
	t.Logf("test subscriber (seed %d) with opts %+v and %d updates", seed, opts, len(updates))

	key, output := mux.Subscribe(opts)

	go func() {
		for _, update := range updates {
			input <- update
		}
		wg.Done()
	}()
	streamContains(require.New(t), output, updates, func() {
		mux.Unsubscribe(key)
	})
}

func TestMux(t *testing.T) {
	n := runtime.GOMAXPROCS(0)
	t.Logf("running mux with %d workers and subscribers", n)

	input := make(chan Update, DefaultLongPollLimit)
	mux := NewMux(input, n)

	var updateID atomic.Uint64
	var inputWG, testWG sync.WaitGroup
	for i := 0; i < n; i++ {
		inputWG.Add(1)
		testWG.Add(1)
		go func(seed int64) {
			defer testWG.Done()
			testSubscriber(t, seed, mux, input, &updateID, &inputWG)
		}(int64(i))
	}

	go func() {
		inputWG.Wait()
		close(input)
	}()
	testWG.Wait()
}
