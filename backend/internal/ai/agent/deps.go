package agent

import (
	"mechat/internal/ai/chain"
	"mechat/internal/chat"
	"mechat/internal/feed"
	"mechat/internal/friend"
	"mechat/internal/user"
)

// Deps 工具执行所需的业务依赖
type Deps struct {
	Chat    *chat.Service
	Feed    *feed.Service
	Friend  *friend.Service
	User    *user.Service
	Invoker *chain.Invoker
}

// BuildRegistry 注册全部工具
func BuildRegistry(d *Deps) *Registry {
	reg := NewRegistry()
	// 只读工具
	reg.Register(&getConversationsTool{d})
	reg.Register(&getFriendsTool{d})
	reg.Register(&getFeedTool{d})
	reg.Register(&getUserPostsTool{d})
	reg.Register(&getPostDetailTool{d})
	reg.Register(&searchPostsTool{d})
	reg.Register(&searchUserTool{d})
	reg.Register(&summarizeChatTool{d})
	// 写操作工具 需确认
	reg.Register(&sendMessageTool{d})
	reg.Register(&createPostTool{d})
	reg.Register(&sendFriendRequestTool{d})
	return reg
}
