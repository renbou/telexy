package streams

import (
	"context"
	"fmt"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	jsoniter "github.com/json-iterator/go"
	"github.com/renbou/telexy/internal/api"
)

// Stream is a readonly channel of some type.
type Stream[T any] <-chan T

// ErrStream is a stream of errors. Usually streaming functions return streams in
// pairs: a value stream, and an error stream. By default returned error streams
// should be monitored for errors as value streams, if not specified otherwise,
// will close as soon as an unhandled (and thus sent via the error stream) error occurs.
type ErrStream Stream[error]

// Streamer is an interface implemented by various stream providers and consists of a single
// function which returns a pair of value and error streams. The streamer should close the
// returned streams when the context is canceled or times out, as well as when an error occurs.
// When the streaming is stopped via the context, a nil error is sent to the error stream.
type Streamer[T any] interface {
	Stream(ctx context.Context) (Stream[T], ErrStream)
}

// UpdateDecoder is a type commonly used by the default streamers for stream-like parsing of the incoming
// updates, which allows to reduce memory allocations and the CPU overhead of constantly copying values
type UpdateDecoder[T any] func(api.UpdateInfo, *jsoniter.Iterator) (T, error)

// TgBotAPIDecoder is an UpdateDecoder which decodes updates to the format of the tgbotapi package.
func TgBotAPIDecoder(info api.UpdateInfo, it *jsoniter.Iterator) (tgbotapi.Update, error) {
	update := tgbotapi.Update{UpdateID: info.ID}

	// This might seem bulky but is a whole lot faster than decoding via reflection
	var where any
	switch info.Type {
	case api.UpdateMessage:
		where = &update.Message
	case api.UpdateEditedMessage:
		where = &update.EditedMessage
	case api.UpdateChannelPost:
		where = &update.ChannelPost
	case api.UpdateEditedChannelPost:
		where = &update.EditedChannelPost
	case api.UpdateInlineQuery:
		where = &update.InlineQuery
	case api.UpdateChosenInlineResult:
		where = &update.ChosenInlineResult
	case api.UpdateCallbackQuery:
		where = &update.CallbackQuery
	case api.UpdateShippingQuery:
		where = &update.ShippingQuery
	case api.UpdatePreCheckoutQuery:
		where = &update.PreCheckoutQuery
	case api.UpdatePoll:
		where = &update.Poll
	case api.UpdatePollAnswer:
		where = &update.PollAnswer
	case api.UpdateMyChatMember:
		where = &update.MyChatMember
	case api.UpdateChatMember:
		where = &update.ChatMember
	case api.UpdateChatJoinRequest:
		where = &update.ChatJoinRequest
	default:
		return tgbotapi.Update{}, fmt.Errorf(
			"tgbotapi cannot decode unknown update type: %s", info.Type.String(),
		)
	}

	if it.ReadVal(where); it.Error != nil {
		return tgbotapi.Update{}, fmt.Errorf("decoding tgbotapi update: %w", it.Error)
	}
	return update, nil
}
