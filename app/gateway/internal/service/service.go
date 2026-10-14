package service

import "github.com/hellokitty980808-ctrl/im-core/app/gateway/internal/data"

func NewAccountProxy(c *data.Clients) *AccountProxy { return &AccountProxy{Client: c.Accounts} }
func NewConversationProxy(c *data.Clients) *ConversationProxy {
	return &ConversationProxy{Client: c.Conversations}
}
