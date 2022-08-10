package streams

import (
	"strings"
	"sync"
	"sync/atomic"
	"unicode"

	"github.com/renbou/telexy/internal/api"
)

// SubscriptionOpts specify the various update stream subscription preferences
// for the Mux' Subscribe method. The Updates and Commands options complement each other.
type SubscriptionOpts struct {
	// If this option is set, all others are ignored and all updates
	// are sent via the subscription stream.
	All      bool
	Updates  []api.UpdateType
	Commands []string
}

// Mux is an update multiplexer with dynamic subs/unsubs and concurrent
// processing of incoming updates. Closing of streams/workers is handled automatically
// once the incoming stream is closed.
type Mux struct {
	nWorkers    int
	closed      atomic.Bool
	subCapacity int
	subs        sync.Map
	subID       muxSubID
}

// NewMux creates and starts a new mux with the given number of workers.
// If workers < 1 is passed, it defaults to 1.
func NewMux(stream Stream[Update], workers int) *Mux {
	if workers < 1 {
		workers = 1
	}
	mux := &Mux{
		nWorkers:    workers,
		subCapacity: cap(stream),
	}

	var wg sync.WaitGroup
	wg.Add(workers)

	for i := 0; i < workers; i++ {
		go func(workerID int) {
			mux.process(workerID, stream)
			wg.Done()
		}(i)
	}

	go func() {
		// clean everything up once all workers die (the incoming update stream is closed)
		wg.Wait()
		mux.subs.Range(func(key, value any) bool {
			sub := value.(*subscriberDesc)
			close(sub.stream)

			mux.subs.Delete(key)
			return true
		})
	}()
	return mux
}

// Subscribe creates a new subscriber which receives matching updates
// via the returned stream. A subscription key is also returned and can be
// used to unsubscribe from these updates once needed. If the mux is already closed,
// then an invalid key and a nil stream is returned.
func (m *Mux) Subscribe(opts SubscriptionOpts) (any, Stream[Update]) {
	key := atomic.AddUint64((*uint64)(&m.subID), 1)
	desc := &subscriberDesc{
		all:             opts.All,
		updates:         make(map[api.UpdateType]bool, len(opts.Updates)),
		commands:        make(map[string]bool, len(opts.Commands)),
		done:            make(chan struct{}),
		doneConfirmedBy: make([]atomic.Bool, m.nWorkers),
		stream:          make(chan Update, m.subCapacity),
	}

	if !opts.All {
		for _, u := range opts.Updates {
			desc.updates[u] = true
		}
		for _, cmd := range opts.Commands {
			desc.commands[cmd] = true
		}
	}

	// key should always be unique because it monotonically increases on each call
	m.subs.Store(key, desc)

	// NOW check if the mux is closed since the it might've happened
	// while we were setting everything up
	if m.closed.Load() {
		m.subs.Delete(key)
		return nil, nil
	}
	// There shouldn't be a race condition here... If the mux gets closed after the if,
	// then the new sub is already in the map and will be closed along with the others.
	return key, desc.stream
}

// Unsubscribe removes a subscriber if one exists with the given key. This method
// should only be used if you want to dynamically remove a subscriber, as the rest
// will be removed automatically once the mux' incoming update stream is closed.
// Calling Unsubscribe twice on the same key will panic.
func (m *Mux) Unsubscribe(key any) {
	if val, ok := m.subs.Load(key); ok {
		sub := val.(*subscriberDesc)
		// Notify the workers, they'll close the subscriber as soon as possible
		close(sub.done)
	}
}

type muxSubID uint64

// subscriberDesc is similar SubscriptionOpts however it describes an existing
// subscriber and stores the wanted options as maps for fast lookups
type subscriberDesc struct {
	all bool
	// map values are bool for readability during matching
	updates  map[api.UpdateType]bool
	commands map[string]bool
	// done is closed during Unsubscribe which allows all workers to be notified
	done chan struct{}
	// doneConfirmedN is a counter of how many workers confirmed the unsub.
	// once this is equal to the total number of workers, this subscription is closed for good
	doneConfirmedN atomic.Uint32
	// doneConfirmedBy is a list of bools used by the workers to check
	//  if they've already confirmed the unsub operation
	doneConfirmedBy []atomic.Bool
	// stream is closed once all workers confirm the unsub operation or when the input stream is closed
	stream chan Update
}

// process accepts incoming updates from the source and routes them to the
// active subscribers. process can be called concurrently in order to route
// incoming updates concurrently.
//
// Once the source is closed, process returns. It is safe to close the subscriber
// channels only when all instances of process are finished.
func (m *Mux) process(workerID int, source Stream[Update]) {
	for update := range source {
		m.subs.Range(func(key, value any) bool {
			sub := value.(*subscriberDesc)

			// check if this sub is done before doing anything, since we might
			// be looping over this sub multiple times before the other workers
			// confirm the unsubscription
			select {
			case <-sub.done:
				m.confirmDone(workerID, key, sub)
				return true
			default:
			}

			// continue only if this update matches the subscription
			if !m.match(sub, &update) {
				return true
			}

			// either send the update or confirm the unsubscription
			select {
			case sub.stream <- update:
			case <-sub.done:
				m.confirmDone(workerID, key, sub)
			}
			return true
		})
	}
}

func (m *Mux) match(sub *subscriberDesc, update *Update) bool {
	if sub.all || sub.updates[update.Type] {
		return true
	} else if update.Type == api.UpdateMessage {
		cmd := update.Content.Get("text").ToString()
		// Technically bots can get commands in the middle of a message... Too bad!
		if len(cmd) < 1 || cmd[0] != '/' {
			return false
		}

		// Extract the actual command
		if cmdEnd := strings.IndexFunc(cmd, unicode.IsSpace); cmdEnd != -1 {
			cmd = cmd[:cmdEnd]
		}
		if cmdEnd := strings.IndexByte(cmd, '@'); cmdEnd != -1 {
			cmd = cmd[:cmdEnd]
		}
		return sub.commands[cmd]
	}
	return false
}

func (m *Mux) confirmDone(workerID int, key any, sub *subscriberDesc) {
	if !sub.doneConfirmedBy[workerID].CompareAndSwap(false, true) {
		// didn't swap, meaning the worker has already confirmed the unsub
		return
	}
	if sub.doneConfirmedN.Add(1) == uint32(m.nWorkers) {
		// finally delete the subscriber once all workers have confirmed
		m.subs.Delete(key)
		close(sub.stream)
	}
}
