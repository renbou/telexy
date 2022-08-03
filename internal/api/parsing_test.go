package api

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/goleak"
)

func TestUpdateType_String(t *testing.T) {
	tests := []struct {
		s string
		u UpdateType
	}{
		{s: "message", u: UpdateMessage},
		{s: "edited_message", u: UpdateEditedMessage},
		{s: "channel_post", u: UpdateChannelPost},
		{s: "edited_channel_post", u: UpdateEditedChannelPost},
		{s: "inline_query", u: UpdateInlineQuery},
		{s: "chosen_inline_result", u: UpdateChosenInlineResult},
		{s: "callback_query", u: UpdateCallbackQuery},
		{s: "shipping_query", u: UpdateShippingQuery},
		{s: "pre_checkout_query", u: UpdatePreCheckoutQuery},
		{s: "poll", u: UpdatePoll},
		{s: "poll_answer", u: UpdatePollAnswer},
		{s: "my_chat_member", u: UpdateMyChatMember},
		{s: "chat_member", u: UpdateChatMember},
		{s: "chat_join_request", u: UpdateChatJoinRequest},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.s, func(t *testing.T) {
			t.Parallel()
			assert.Equal(t, tt.s, tt.u.String())
		})
	}
}

func Test_getUpdatesResponseConsumer(t *testing.T) {
	type update struct {
		UpdateInfo
		Value string
	}
	tests := []struct {
		name    string
		data    string
		updates []update
		wantErr bool
	}{
		{
			name:    "non-array response",
			data:    `{"a": "b"}`,
			wantErr: true,
		},
		{
			name:    "invalid response contents",
			data:    `["string", 1]`,
			wantErr: true,
		},
		{
			name:    "update_id field isn't first",
			data:    `[{"message":{}, "update_id": 2}]`,
			wantErr: true,
		},
		{
			name:    "invalid update_id value",
			data:    `[{"update_id":true,"message":{}}]`,
			wantErr: true,
		},
		{
			name:    "invalid update structure",
			data:    `[{"update_id", 1, "message": {}}]`,
			wantErr: true,
		},
		{
			name:    "update without contents",
			data:    `[{"update_id":1}]`,
			wantErr: true,
		},
		{
			name: "valid responses with known and unknown updates",
			data: `[{"update_id":1, "unknown": {}},
			{"update_id": 2, "message": {"text":"testtext"}}, {"update_id": 3,
	"unk": 1}, { "update_id": 4, "poll": {"id":"pollid"}}]`,
			updates: []update{
				{UpdateInfo: UpdateInfo{ID: 2, Type: UpdateMessage}, Value: `{"text":"testtext"}`},
				{UpdateInfo: UpdateInfo{ID: 4, Type: UpdatePoll}, Value: `{"id":"pollid"}`},
			},
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := require.New(t)

			var updates []update
			consumer := getUpdatesResponseConsumer(func(ui UpdateInfo, d Decoder) error {
				var m json.RawMessage
				if err := d.Decode(&m); err != nil {
					return err
				}
				updates = append(updates, update{UpdateInfo: ui, Value: string(m)})
				return nil
			})

			r := strings.NewReader(tt.data)
			d := newDecoder(json.NewDecoder(r))

			err := consumer(d)
			if tt.wantErr {
				req.Error(err)
			} else {
				req.NoError(err)
				req.Equal(tt.updates, updates)
			}
		})
	}
}

func TestMain(m *testing.M) {
	goleak.VerifyTestMain(m)
}
