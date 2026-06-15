package agent

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"mechat/internal/feed"
	friendpkg "mechat/internal/friend"

	"github.com/cloudwego/eino/schema"
)

// ── send_message ──

type sendMessageTool struct{ d *Deps }

func (t *sendMessageTool) Name() string  { return "send_message" }
func (t *sendMessageTool) Label() string { return "发送消息" }
func (t *sendMessageTool) Desc() string {
	return "代替用户向某个会话或好友发送一条文本消息。指定收件人三选一：conversation_id（已有会话）/ to_user_id（对方 user_id）/ to_nickname（好友昵称，支持模糊匹配，最简单的方式）。发私聊直接给 to_nickname，发群聊给 conversation_id。"
}
func (t *sendMessageTool) Params() *schema.ParamsOneOf {
	return schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"text":            {Type: schema.String, Desc: "要发送的消息内容", Required: true},
		"to_nickname":     {Type: schema.String, Desc: "收件人好友昵称（支持大小写模糊匹配）"},
		"to_user_id":      {Type: schema.String, Desc: "收件人 user_id"},
		"conversation_id": {Type: schema.String, Desc: "目标会话 ID（私聊或群聊）"},
	})
}
func (t *sendMessageTool) NeedConfirm() bool { return true }

// PreValidate 弹确认前校验收件人可解析
func (t *sendMessageTool) PreValidate(ctx context.Context, userID uint64, args map[string]any) string {
	if argU64(args, "conversation_id") != 0 || argU64(args, "to_user_id") != 0 {
		return "" // 已指定目标
	}
	id, msg := t.resolveRecipient(ctx, userID, argStr(args, "to_nickname"))
	if id == 0 {
		return msg // 返回说明让模型转告
	}
	return ""
}
func (t *sendMessageTool) Preview(args map[string]any) string {
	text := argStr(args, "text")
	if to := argStr(args, "to_nickname"); to != "" {
		return "发送给「" + to + "」：\n" + text
	}
	return text
}
func (t *sendMessageTool) Execute(ctx context.Context, userID uint64, args map[string]any) (string, error) {
	text := argStr(args, "text")
	if text == "" {
		return "", errors.New("消息内容不能为空")
	}
	convID := argU64(args, "conversation_id")
	if convID == 0 {
		toUser := argU64(args, "to_user_id")
		if toUser == 0 {
			resolvedID, suggestion := t.resolveRecipient(ctx, userID, argStr(args, "to_nickname"))
			if resolvedID == 0 {
				// 返回纯文本让模型转告
				return suggestion, nil
			}
			toUser = resolvedID
		}
		conv, err := t.d.Chat.GetOrCreatePrivateConv(ctx, userID, toUser)
		if err != nil {
			return "", err
		}
		convID = conv.ID
	}
	msg, err := t.d.Chat.SendMessage(ctx, userID, convID, 1, map[string]any{"text": text})
	if err != nil {
		return "", err
	}
	// 推给发送方自己刷新页面
	t.d.Chat.NotifySenderMessage(ctx, userID, msg)
	return "消息已发送：" + text, nil
}

// resolveRecipient 好友昵称模糊匹配 返回 user_id 与建议
func (t *sendMessageTool) resolveRecipient(ctx context.Context, userID uint64, nickname string) (uint64, string) {
	if nickname == "" {
		return 0, "缺少收件人，请提供 to_nickname / to_user_id / conversation_id。"
	}
	friends, _ := t.d.Friend.GetFriendList(ctx, userID)
	low := strings.ToLower(nickname)

	// 精确匹配
	for _, f := range friends {
		if f.Nickname == nickname {
			return f.UserID, ""
		}
	}
	// 不区分大小写精确匹配
	for _, f := range friends {
		if strings.ToLower(f.Nickname) == low {
			return f.UserID, ""
		}
	}
	// 昵称包含模糊
	type hitEntry struct {
		id   uint64
		name string
	}
	var hits []hitEntry
	for _, f := range friends {
		if strings.Contains(strings.ToLower(f.Nickname), low) ||
			strings.Contains(low, strings.ToLower(f.Nickname)) {
			hits = append(hits, hitEntry{f.UserID, f.Nickname})
		}
	}
	if len(hits) == 1 {
		return hits[0].id, ""
	}
	if len(hits) > 1 {
		names := make([]string, len(hits))
		for i, h := range hits {
			names[i] = h.name
		}
		return 0, fmt.Sprintf("「%s」匹配到多个好友：%s。请告诉我你要发给哪位？", nickname, strings.Join(names, "、"))
	}
	// 全局搜索兜底
	if users, err := t.d.User.Search(ctx, nickname); err == nil {
		for _, u := range users {
			if strings.ToLower(u.Nickname) == low {
				return u.ID, ""
			}
		}
	}
	// 没找到给建议
	return 0, fmt.Sprintf(
		"在好友列表中没有找到与「%s」匹配的人（已尝试模糊匹配）。可能对方不在你的好友列表，或昵称有差异。你可以用 search_user 工具搜索，或告诉我对方准确昵称。",
		nickname,
	)
}

// ── create_post ──

type createPostTool struct{ d *Deps }

func (t *createPostTool) Name() string  { return "create_post" }
func (t *createPostTool) Label() string { return "发布帖子" }
func (t *createPostTool) Desc() string {
	return "代替用户在动态广场发布一篇帖子（标题必填，正文可选）。调用前请先用一句话告诉用户你要发布的内容。"
}
func (t *createPostTool) Params() *schema.ParamsOneOf {
	return schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"title":   {Type: schema.String, Desc: "帖子标题，60 字以内", Required: true},
		"content": {Type: schema.String, Desc: "帖子正文，可选"},
	})
}
func (t *createPostTool) NeedConfirm() bool { return true }
func (t *createPostTool) Preview(args map[string]any) string {
	title := argStr(args, "title")
	content := argStr(args, "content")
	if content == "" {
		return title
	}
	return title + "\n" + content
}
func (t *createPostTool) Execute(ctx context.Context, userID uint64, args map[string]any) (string, error) {
	title := argStr(args, "title")
	if title == "" {
		return "", errors.New("标题不能为空")
	}
	r := []rune(title)
	if len(r) > 60 {
		title = string(r[:60])
	}
	post, err := t.d.Feed.CreatePost(ctx, userID, &feed.CreatePostReq{
		Title:   title,
		Content: argStr(args, "content"),
	}, "")
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("帖子已发布：[《%s》](/post/%d)", post.Title, post.PostID), nil
}

// ── send_friend_request ──

type sendFriendRequestTool struct{ d *Deps }

func (t *sendFriendRequestTool) Name() string  { return "send_friend_request" }
func (t *sendFriendRequestTool) Label() string { return "发送好友申请" }
func (t *sendFriendRequestTool) Desc() string {
	return "代替用户向某个 user_id 发送好友申请。若只知道昵称，请先用 search_user 找到 user_id。"
}
func (t *sendFriendRequestTool) Params() *schema.ParamsOneOf {
	return schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"to_user_id": {Type: schema.String, Desc: "目标用户的 user_id", Required: true},
		"message":    {Type: schema.String, Desc: "申请附言，可选"},
	})
}
func (t *sendFriendRequestTool) NeedConfirm() bool { return true }
func (t *sendFriendRequestTool) Preview(args map[string]any) string {
	msg := argStr(args, "message")
	if msg == "" {
		return "（无附言）"
	}
	return msg
}
func (t *sendFriendRequestTool) Execute(ctx context.Context, userID uint64, args map[string]any) (string, error) {
	toUser := argU64(args, "to_user_id")
	if toUser == 0 {
		return "", errors.New("缺少 to_user_id")
	}
	err := t.d.Friend.SendRequest(ctx, userID, &friendpkg.SendRequestReq{
		ToUserID: toUser,
		Message:  argStr(args, "message"),
	})
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("已向 user_id=%d 发送好友申请", toUser), nil
}
