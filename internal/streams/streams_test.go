package streams

import (
	"testing"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	jsoniter "github.com/json-iterator/go"
	"github.com/renbou/telexy/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// streamIs validates that a stream outputs values equal to the ones wanted in the specified order
func streamIs[T any](req *require.Assertions, s Stream[T], want []T) {
	n := 0
	req.Eventually(func() bool {
		select {
		case val, ok := <-s:
			if !ok {
				req.Equal(len(want), n, "not all wanted values found on stream")
				return true
			}

			req.Less(n, len(want), "got more values than wanted on stream")
			req.Equal(want[n], val)
			n++
		default:
		}
		return false
	}, time.Second*5, time.Millisecond*50)
}

func TestTgBotAPIDecoder(t *testing.T) {
	tests := []struct {
		info      api.UpdateInfo
		data      string
		want      tgbotapi.Update
		assertion assert.ErrorAssertionFunc
	}{
		{
			info: api.UpdateInfo{
				ID:   1,
				Type: api.UpdateMessage,
			},
			data: `{"message_id":1,"text":"message"}`,
			want: tgbotapi.Update{
				UpdateID: 1,
				Message: &tgbotapi.Message{
					MessageID: 1,
					Text:      "message",
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   2,
				Type: api.UpdateEditedMessage,
			},
			data: `{"message_id":2,"text":"edited message"}`,
			want: tgbotapi.Update{
				UpdateID: 2,
				EditedMessage: &tgbotapi.Message{
					MessageID: 2,
					Text:      "edited message",
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   3,
				Type: api.UpdateChannelPost,
			},
			data: `{"message_id":3,"text":"channel post"}`,
			want: tgbotapi.Update{
				UpdateID: 3,
				ChannelPost: &tgbotapi.Message{
					MessageID: 3,
					Text:      "channel post",
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   4,
				Type: api.UpdateEditedChannelPost,
			},
			data: `{"message_id":4,"text":"edited channel post"}`,
			want: tgbotapi.Update{
				UpdateID: 4,
				EditedChannelPost: &tgbotapi.Message{
					MessageID: 4,
					Text:      "edited channel post",
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   5,
				Type: api.UpdateInlineQuery,
			},
			data: `{"id":"inline-query-id","query":"inline query"}`,
			want: tgbotapi.Update{
				UpdateID: 5,
				InlineQuery: &tgbotapi.InlineQuery{
					ID:    "inline-query-id",
					Query: "inline query",
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   6,
				Type: api.UpdateChosenInlineResult,
			},
			data: `{"result_id":"inline-result-update-id","query":"chosen inline result"}`,
			want: tgbotapi.Update{
				UpdateID: 6,
				ChosenInlineResult: &tgbotapi.ChosenInlineResult{
					ResultID: "inline-result-update-id",
					Query:    "chosen inline result",
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   7,
				Type: api.UpdateCallbackQuery,
			},
			data: `{"id":"callback-query-id"}`,
			want: tgbotapi.Update{
				UpdateID: 7,
				CallbackQuery: &tgbotapi.CallbackQuery{
					ID: "callback-query-id",
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   8,
				Type: api.UpdateShippingQuery,
			},
			data: `{"id":"shipping-query-id","invoice_payload":"shipping query"}`,
			want: tgbotapi.Update{
				UpdateID: 8,
				ShippingQuery: &tgbotapi.ShippingQuery{
					ID:             "shipping-query-id",
					InvoicePayload: "shipping query",
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   9,
				Type: api.UpdatePreCheckoutQuery,
			},
			data: `{"id":"precheckout-query-id","invoice_payload":"precheckout query"}`,
			want: tgbotapi.Update{
				UpdateID: 9,
				PreCheckoutQuery: &tgbotapi.PreCheckoutQuery{
					ID:             "precheckout-query-id",
					InvoicePayload: "precheckout query",
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   10,
				Type: api.UpdatePoll,
			},
			data: `{"id":"poll-id","question":"poll question"}`,
			want: tgbotapi.Update{
				UpdateID: 10,
				Poll: &tgbotapi.Poll{
					ID:       "poll-id",
					Question: "poll question",
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   11,
				Type: api.UpdatePollAnswer,
			},
			data: `{"poll_id":"original-poll-id"}`,
			want: tgbotapi.Update{
				UpdateID: 11,
				PollAnswer: &tgbotapi.PollAnswer{
					PollID: "original-poll-id",
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   12,
				Type: api.UpdateMyChatMember,
			},
			data: `{"chat":{"id":123},"old_chat_member":{"custom_title":"bot"}}`,
			want: tgbotapi.Update{
				UpdateID: 12,
				MyChatMember: &tgbotapi.ChatMemberUpdated{
					Chat:          tgbotapi.Chat{ID: 123},
					OldChatMember: tgbotapi.ChatMember{CustomTitle: "bot"},
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   13,
				Type: api.UpdateChatMember,
			},
			data: `{"chat":{"id":321},"old_chat_member":{"custom_title":"admin"}}`,
			want: tgbotapi.Update{
				UpdateID: 13,
				ChatMember: &tgbotapi.ChatMemberUpdated{
					Chat:          tgbotapi.Chat{ID: 321},
					OldChatMember: tgbotapi.ChatMember{CustomTitle: "admin"},
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   14,
				Type: api.UpdateChatJoinRequest,
			},
			data: `{"chat":{"id":111},"bio":"cool chat user"}`,
			want: tgbotapi.Update{
				UpdateID: 14,
				ChatJoinRequest: &tgbotapi.ChatJoinRequest{
					Chat: tgbotapi.Chat{ID: 111},
					Bio:  "cool chat user",
				},
			},
			assertion: assert.NoError,
		},
		{
			info: api.UpdateInfo{
				ID:   15,
				Type: -1,
			},
			data:      `{"type":"unknown"}`,
			want:      tgbotapi.Update{},
			assertion: assert.Error,
		},
		{
			info: api.UpdateInfo{
				ID:   16,
				Type: api.UpdateMessage,
			},
			data:      `{"text":1}`,
			want:      tgbotapi.Update{},
			assertion: assert.Error,
		},
	}
	for _, tt := range tests {
		t.Run(tt.info.Type.String(), func(t *testing.T) {
			t.Parallel()

			it := jsoniter.ConfigFastest.BorrowIterator([]byte(tt.data))
			defer jsoniter.ConfigFastest.ReturnIterator(it)

			got, err := TgBotAPIDecoder(tt.info, it)
			tt.assertion(t, err)
			assert.Equal(t, tt.want, got)
		})
	}
}
