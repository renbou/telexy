package api

import (
	"fmt"
	"io"
	"sync"

	jsoniter "github.com/json-iterator/go"
)

// Default size of jsoniter decoding buffer - 256 Kb
const DefaultDecodeBufferSize = 256 << 10

// Custom iterator pool which preallocates a single buffer to be used by all later operations
var iteratorPool = sync.Pool{
	New: func() any {
		// Maybe switch to ConfigDefault if issues ever arise
		return jsoniter.Parse(jsoniter.ConfigFastest, nil, DefaultDecodeBufferSize)
	},
}

func borrowIterator(r io.Reader) *jsoniter.Iterator {
	it := iteratorPool.Get().(*jsoniter.Iterator)
	it.Reset(r)
	return it
}

func returnIterator(it *jsoniter.Iterator) {
	// Avoid keeping any references
	it.Error = nil
	it.Attachment = nil
	it.Reset(nil)
	iteratorPool.Put(it)
}

type responseConsumer func(*jsoniter.Iterator) error

// readResponse reads a single API response from the reader and calls the consumer
// once the response metadata has been read and validated ("ok", "description", etc).
// The approach of passing a consumer via args was chosen in order to leverage the JSON
// stream decoding API without leaving the possibility of an unclosed reader.
func readResponse(r io.ReadCloser, consumer responseConsumer) error {
	defer r.Close()
	it := borrowIterator(r)
	defer returnIterator(it)

	var resp Response
	for key := it.ReadObject(); key != "" && it.Error == nil; key = it.ReadObject() {
		// Ignore invalid and possibly unknown fields
		if len(key) < len("ok") {
			continue
		}

		// Check if we received a result, cause if we did, then "ok" *should* be true
		// and we can go right ahead with reading the result
		if key[0] == 'r' {
			resp.Ok = true
			break
		}

		// Parse known fields while relying on their correctness and skip unknown ones
		switch key[0] {
		case 'o': // ok
			resp.Ok = it.ReadBool()
		case 'd': // description
			resp.Description = it.ReadString()
		case 'e': // error_code
			resp.ErrorCode = it.ReadInt()
		default:
			it.Skip()
		}
	}

	if it.Error != nil {
		return fmt.Errorf("parsing telegram api response: %w", it.Error)
	} else if !resp.Ok {
		return fmt.Errorf("non-ok telegram api response: %q (code %d)", resp.Description, resp.ErrorCode)
	}
	return consumer(it)
}

// parseUpdateType parses the update type string by looking at the minimum amount
// of characters required to distinguish different update types. This relies on the
// update type string being correct, and as such this function should only be used
// on responses from the Telegram API.
func parseUpdateType(s string) (UpdateType, bool) {
	if len(s) < len("poll") {
		return 0, false
	}

	switch s[0] {
	case 'm':
		switch len(s) {
		case len("message"):
			return UpdateMessage, true
		case len("my_chat_member"):
			return UpdateMyChatMember, true
		}
	case 'e':
		switch len(s) {
		case len("edited_message"):
			return UpdateEditedMessage, true
		case len("edited_channel_post"):
			return UpdateEditedChannelPost, true
		}
	case 'c':
		switch len(s) {
		case len("channel_post"):
			return UpdateChannelPost, true
		case len("chosen_inline_result"):
			return UpdateChosenInlineResult, true
		case len("callback_query"):
			return UpdateCallbackQuery, true
		case len("chat_member"):
			return UpdateChatMember, true
		case len("chat_join_request"):
			return UpdateChatJoinRequest, true
		}
	case 'i':
		return UpdateInlineQuery, true
	case 's':
		return UpdateShippingQuery, true
	case 'p':
		switch len(s) {
		case len("pre_checkout_query"):
			return UpdatePreCheckoutQuery, true
		case len("poll"):
			return UpdatePoll, true
		case len("poll_answer"):
			return UpdatePollAnswer, true
		}
	}
	return 0, false
}

// getUpdatesResponseConsumer returns a consumer for reading a getUpdates response
// using the given update consumer. It calls the consumer once for each update encountered
// in the getUpdates response.
func getUpdatesResponseConsumer(consumer func(UpdateInfo, *jsoniter.Iterator) error) responseConsumer {
	return func(it *jsoniter.Iterator) error {
		for it.ReadArray() && it.Error == nil {
			// TODO: allow keys in any order (e.g. if update_id comes first, then we parse can pass
			// it to the long poller or smth else straight away, otherwise we read the value and then
			// pass the update_id)
			if key := it.ReadObject(); key != "update_id" {
				if it.Error == nil {
					return fmt.Errorf("expected update_id as the first field, but got: %q", key)
				}
				break
			}
			info := UpdateInfo{
				ID: it.ReadInt(),
			}

			var ok bool
			if info.Type, ok = parseUpdateType(it.ReadObject()); ok {
				// Let the consumer take the value
				if err := consumer(info, it); err != nil {
					return err
				}
			} else {
				// Ignore unknown updates
				it.Skip()
			}

			if key := it.ReadObject(); key != "" {
				if it.Error == nil {
					return fmt.Errorf("getUpdates contains excess field: %q", key)
				}
				break
			}
		}

		if it.Error != nil {
			return fmt.Errorf("parsing getUpdates response: %w", it.Error)
		}
		return nil
	}
}
