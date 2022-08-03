package api

import (
	"encoding/json"
	"fmt"
	"io"
	"reflect"
)

// ignoreJSON is a struct which can be passed into json decoding functions
// to simply ignore the content. This is better than using json.RawMessage
// because it does not do any copying.
type ignoreJSON struct{}

func (ignoreJSON) UnmarshalJSON([]byte) error {
	return nil
}

// Decoder is a wrapper around json.Decoder to simplify usage
// of the streaming API by providing helpers for token and type matching.
type Decoder struct {
	*json.Decoder
}

func (d Decoder) readErr(what any, err error) error {
	return fmt.Errorf("reading %v from json: %w", what, err)
}

func (d Decoder) expectedErr(what, got any) error {
	return fmt.Errorf("expected %v but got %v (%s)", what, got, reflect.TypeOf(got).String())
}

func (d Decoder) Skip() {
	var ignore ignoreJSON
	_ = d.Decode(&ignore)
}

func (d Decoder) TokenIs(v any) error {
	if t, err := d.Token(); err != nil {
		return d.readErr(v, err)
	} else if t != v {
		return d.expectedErr(v, t)
	}
	return nil
}

func decoderWant[T any](what string, d Decoder) (T, error) {
	var v T
	if t, err := d.Token(); err != nil {
		return v, d.readErr(what, err)
	} else if v, ok := t.(T); !ok {
		return v, d.expectedErr(what, t)
	} else {
		return v, nil
	}
}

func (d Decoder) String() (string, error) {
	return decoderWant[string]("a string", d)
}

func (d Decoder) Bool() (bool, error) {
	return decoderWant[bool]("a boolean", d)
}

func (d Decoder) number(want string) (json.Number, error) {
	return decoderWant[json.Number](want, d)
}

func (d Decoder) Int64() (int64, error) {
	if n, err := d.number("an int"); err != nil {
		return 0, err
	} else if i, err := n.Int64(); err != nil {
		f, _ := n.Float64()
		return int64(f), nil
	} else {
		return i, nil
	}
}

func (d Decoder) Int() (int, error) {
	i64, err := d.Int64()
	return int(i64), err
}

func newDecoder(d *json.Decoder) Decoder {
	d.UseNumber()
	return Decoder{d}
}

// a short description of a response received from Telegram's API.
type response struct {
	Ok          bool   `json:"ok"`
	Description string `json:"description"`
	ErrorCode   int    `json:"error_code"`
}

type responseConsumer func(Decoder) error

// readResponse reads a single API response from the reader and calls the consumer
// once the response metadata has been read and validated ("ok", "description", etc).
// The approach of passing a consumer via args was chosen in order to leverage the JSON
// stream decoding API without leaving the possibility of an unclosed reader.
func readResponse(r io.ReadCloser, consumer responseConsumer) error {
	defer r.Close()
	d := newDecoder(json.NewDecoder(r))

	if err := d.TokenIs(json.Delim('{')); err != nil {
		return err
	}

	// Set response's "ok" to true by default in case we encounter "result" first
	resp := response{Ok: true}
	for d.More() {
		key, err := d.String()
		if err != nil {
			return fmt.Errorf("expected api response field key: %w", err)
		} else if len(key) < len("ok") {
			// Ignore invalid and possibly unknown fields
			continue
		}

		// Check if we received a result, cause if we did, then "ok" *should* be true
		// and we can go right ahead with reading the result
		if key[0] == 'r' {
			break
		}

		// Parse known fields while relying on their correctness and skip unknown ones
		switch key[0] {
		case 'o': // ok
			resp.Ok, err = d.Bool()
		case 'd': // description
			resp.Description, err = d.String()
		case 'e': // error_code
			resp.ErrorCode, err = d.Int()
		default:
			d.Skip()
		}
		if err != nil {
			return fmt.Errorf("expected api response field value: %w", err)
		}
	}

	if !resp.Ok {
		return fmt.Errorf("non-ok telegram api response: %q (code %d)", resp.Description, resp.ErrorCode)
	}
	return consumer(d)
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
func getUpdatesResponseConsumer(consumer func(UpdateInfo, Decoder) error) responseConsumer {
	return func(d Decoder) (err error) {
		for err = d.TokenIs(json.Delim('[')); d.More() && err == nil; err = d.TokenIs(json.Delim('}')) {
			// We currently rely on the update id being the first field in an update
			if err = d.TokenIs(json.Delim('{')); err != nil {
				return err
			} else if err = d.TokenIs("update_id"); err != nil {
				return fmt.Errorf("expected update_id to be the first field: %w", err)
			}

			var info UpdateInfo
			if info.ID, err = d.Int(); err != nil {
				return fmt.Errorf("invalid value specified as update_id: %w", err)
			}

			// The only field left should be the actual value, starting with its type as the object key
			var key string
			if key, err = d.String(); err != nil {
				return fmt.Errorf("expected update type key: %w", err)
			}

			var ok bool
			if info.Type, ok = parseUpdateType(key); !ok {
				// Ignore unknown updates
				d.Skip()
				continue
			}

			// Finally let the consumer take the value
			if err = consumer(info, d); err != nil {
				return err
			}
		}
		return err
	}
}
