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

// Streamer is an interface implemented by various stream providers and consists of a single
// function which returns a pair of value and error streams. The streamer should close the
// returned streams when the context is canceled or times out, as well as when an error occurs.
// When the streaming is stopped via the context, a nil error is sent to the error stream.
type Streamer[T any] interface {
	Stream(ctx context.Context) (Stream[T], Stream[error])
}

// UpdateDecoder is a type commonly used by the default streamers for stream-like parsing of the incoming
// updates, which allows to reduce memory allocations and the CPU overhead of constantly copying values
type UpdateDecoder[T any] func(api.UpdateInfo, *jsoniter.Iterator) (T, error)

// Update represents an update with already parsed info and valid json contents.
// This type can be used when the actual contents of the update don't matter (i.e. for routing).
type Update struct {
	api.UpdateInfo
	Content jsoniter.Any
}

// AsUpdate is an UpdateDecoder which simply copies the update contents
// allowing for further processing somewhere else.
func AsUpdate(info api.UpdateInfo, it *jsoniter.Iterator) (Update, error) {
	update := Update{
		UpdateInfo: info,
		Content:    it.ReadAny(),
	}

	var err error
	if it.Error != nil {
		err = it.Error
	} else if update.Content.LastError() != nil {
		err = update.Content.LastError()
	}

	if err != nil {
		return Update{}, fmt.Errorf("reading update contents: %w", err)
	}
	return update, nil
}

// AsTgBotAPI is an UpdateDecoder which provides updates in the format of the tgbotapi package.
func AsTgBotAPI(info api.UpdateInfo, it *jsoniter.Iterator) (tgbotapi.Update, error) {
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
