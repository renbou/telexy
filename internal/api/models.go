package api

// A single Response from the Telegram Bot API.
// Doesn't contain fields such as "parameters" since currently they aren't by anything in telexy.
type Response struct {
	Ok          bool   `json:"ok,omitempty"`
	Description string `json:"description,omitempty"`
	ErrorCode   int    `json:"error_code,omitempty"`
	Result      any    `json:"result,omitempty"`
}

// UpdateType is an enum of the possible Telegram Bot API update message contents.
type UpdateType int

//go:generate stringer -linecomment -type=UpdateType
const (
	UpdateMessage            UpdateType = iota // message
	UpdateEditedMessage                        // edited_message
	UpdateChannelPost                          // channel_post
	UpdateEditedChannelPost                    // edited_channel_post
	UpdateInlineQuery                          // inline_query
	UpdateChosenInlineResult                   // chosen_inline_result
	UpdateCallbackQuery                        // callback_query
	UpdateShippingQuery                        // shipping_query
	UpdatePreCheckoutQuery                     // pre_checkout_query
	UpdatePoll                                 // poll
	UpdatePollAnswer                           // poll_answer
	UpdateMyChatMember                         // my_chat_member
	UpdateChatMember                           // chat_member
	UpdateChatJoinRequest                      // chat_join_request
)

type UpdateInfo struct {
	ID   int
	Type UpdateType
}

type GetUpdatesRequest struct {
	Offset         int      `json:"offset,omitempty"`
	Limit          int      `json:"limit,omitempty"`
	Timeout        int      `json:"timeout,omitempty"`
	AllowedUpdates []string `json:"allowed_updates,omitempty"`
}
