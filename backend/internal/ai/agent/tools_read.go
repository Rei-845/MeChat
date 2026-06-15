package agent

import (
	"context"
	"fmt"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// ── get_conversations ──

type getConversationsTool struct{ d *Deps }

func (t *getConversationsTool) Name() string  { return "get_conversations" }
func (t *getConversationsTool) Label() string { return "查询会话列表" }
func (t *getConversationsTool) Desc() string {
	return "获取当前用户的全部会话（私聊/群聊），包含未读数与 conversation_id。当用户问“我有哪些聊天/未读消息”，或需要某个会话的 conversation_id 时调用。"
}
func (t *getConversationsTool) Params() *schema.ParamsOneOf   { return nil }
func (t *getConversationsTool) NeedConfirm() bool             { return false }
func (t *getConversationsTool) Preview(map[string]any) string { return "" }
func (t *getConversationsTool) Execute(ctx context.Context, userID uint64, _ map[string]any) (string, error) {
	convs, err := t.d.Chat.GetConversations(ctx, userID)
	if err != nil {
		return "", err
	}
	if len(convs) == 0 {
		return "当前没有任何会话。", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "共有 %d 个会话：\n", len(convs))
	for i, c := range convs {
		if c.Type == 1 && c.TargetUser != nil {
			status := "离线"
			if c.TargetUser.IsOnline {
				status = "在线"
			}
			fmt.Fprintf(&sb, "%d. [私聊] %s（%s）未读 %d 条 conversation_id=%d\n",
				i+1, c.TargetUser.Nickname, status, c.UnreadCount, c.ID)
		} else if c.Type == 2 && c.GroupInfo != nil {
			fmt.Fprintf(&sb, "%d. [群聊] %s（%d人）未读 %d 条 conversation_id=%d\n",
				i+1, c.GroupInfo.Name, c.GroupInfo.Members, c.UnreadCount, c.ID)
		} else {
			fmt.Fprintf(&sb, "%d. 会话 conversation_id=%d 未读 %d 条\n", i+1, c.ID, c.UnreadCount)
		}
	}
	return sb.String(), nil
}

// ── get_friends ──

type getFriendsTool struct{ d *Deps }

func (t *getFriendsTool) Name() string  { return "get_friends" }
func (t *getFriendsTool) Label() string { return "查询好友列表" }
func (t *getFriendsTool) Desc() string {
	return "获取当前用户的好友列表及其在线状态与 user_id。当需要好友的 user_id（如发消息/发好友申请前）或用户询问好友情况时调用。"
}
func (t *getFriendsTool) Params() *schema.ParamsOneOf   { return nil }
func (t *getFriendsTool) NeedConfirm() bool             { return false }
func (t *getFriendsTool) Preview(map[string]any) string { return "" }
func (t *getFriendsTool) Execute(ctx context.Context, userID uint64, _ map[string]any) (string, error) {
	friends, err := t.d.Friend.GetFriendList(ctx, userID)
	if err != nil {
		return "", err
	}
	if len(friends) == 0 {
		return "你还没有好友。", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "共有 %d 位好友：\n", len(friends))
	for _, f := range friends {
		status := "离线"
		if f.IsOnline {
			status = "在线"
		}
		fmt.Fprintf(&sb, "- %s（%s）user_id=%d\n", f.Nickname, status, f.UserID)
	}
	return sb.String(), nil
}

// ── get_feed ──

type getFeedTool struct{ d *Deps }

func (t *getFeedTool) Name() string  { return "get_feed" }
func (t *getFeedTool) Label() string { return "获取推荐动态" }
func (t *getFeedTool) Desc() string {
	return "获取动态广场推荐的帖子摘要（标题、作者、点赞/评论数、post_id）。当用户想了解最新动态或要对某帖操作时调用。"
}
func (t *getFeedTool) Params() *schema.ParamsOneOf   { return nil }
func (t *getFeedTool) NeedConfirm() bool             { return false }
func (t *getFeedTool) Preview(map[string]any) string { return "" }
func (t *getFeedTool) Execute(ctx context.Context, userID uint64, _ map[string]any) (string, error) {
	posts, _, err := t.d.Feed.GetFeed(ctx, userID, 1, 10, "")
	if err != nil {
		return "", err
	}
	if len(posts) == 0 {
		return "暂时没有推荐动态。", nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "推荐动态（%d 条）：\n", len(posts))
	for i, p := range posts {
		fmt.Fprintf(&sb, "%d. [《%s》](/post/%d) 作者 %s · %d 赞 %d 评论\n",
			i+1, p.Title, p.PostID, p.User.Nickname, p.LikeCount, p.CommentCount)
	}
	return sb.String(), nil
}

// ── get_user_posts ──

type getUserPostsTool struct{ d *Deps }

func (t *getUserPostsTool) Name() string  { return "get_user_posts" }
func (t *getUserPostsTool) Label() string { return "查看用户的帖子" }
func (t *getUserPostsTool) Desc() string {
	return "查看指定用户（含自己）发过的帖子列表，返回标题、点赞/评论数、post_id。需要 user_id（可先用 search_user 或 get_friends 获取；查自己时用当前用户的 user_id）。"
}
func (t *getUserPostsTool) Params() *schema.ParamsOneOf {
	return schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"user_id": {Type: schema.String, Desc: "要查看其帖子的用户 user_id", Required: true},
	})
}
func (t *getUserPostsTool) NeedConfirm() bool             { return false }
func (t *getUserPostsTool) Preview(map[string]any) string { return "" }
func (t *getUserPostsTool) Execute(ctx context.Context, userID uint64, args map[string]any) (string, error) {
	target := argU64(args, "user_id")
	if target == 0 {
		return "请提供有效的 user_id。", nil
	}
	posts, _, err := t.d.Feed.GetUserPosts(ctx, userID, target, 1, 20)
	if err != nil {
		return "", err
	}
	if len(posts) == 0 {
		return fmt.Sprintf("user_id=%d 暂时没有可见的帖子。", target), nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "user_id=%d 共有 %d 篇可见帖子：\n", target, len(posts))
	for i, p := range posts {
		fmt.Fprintf(&sb, "%d. [《%s》](/post/%d) · %d 赞 %d 评论\n",
			i+1, p.Title, p.PostID, p.LikeCount, p.CommentCount)
	}
	return sb.String(), nil
}

// ── search_posts ──

type searchPostsTool struct{ d *Deps }

func (t *searchPostsTool) Name() string  { return "search_posts" }
func (t *searchPostsTool) Label() string { return "搜索帖子" }
func (t *searchPostsTool) Desc() string {
	return "按标题关键词搜索动态广场的帖子，返回标题、作者、点赞/评论数、post_id。当用户想找某个主题的帖子时调用。"
}
func (t *searchPostsTool) Params() *schema.ParamsOneOf {
	return schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"keyword": {Type: schema.String, Desc: "帖子标题关键词", Required: true},
	})
}
func (t *searchPostsTool) NeedConfirm() bool             { return false }
func (t *searchPostsTool) Preview(map[string]any) string { return "" }
func (t *searchPostsTool) Execute(ctx context.Context, userID uint64, args map[string]any) (string, error) {
	keyword := argStr(args, "keyword")
	if keyword == "" {
		return "请提供搜索关键词。", nil
	}
	posts, _, err := t.d.Feed.SearchPosts(ctx, userID, keyword, "", 1, 10)
	if err != nil {
		return "", err
	}
	if len(posts) == 0 {
		return fmt.Sprintf("没有找到标题包含「%s」的帖子。", keyword), nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "找到 %d 篇相关帖子：\n", len(posts))
	for i, p := range posts {
		fmt.Fprintf(&sb, "%d. [《%s》](/post/%d) 作者 %s · %d 赞 %d 评论\n",
			i+1, p.Title, p.PostID, p.User.Nickname, p.LikeCount, p.CommentCount)
	}
	return sb.String(), nil
}

// ── get_post_detail ──

type getPostDetailTool struct{ d *Deps }

func (t *getPostDetailTool) Name() string  { return "get_post_detail" }
func (t *getPostDetailTool) Label() string { return "查看帖子详情" }
func (t *getPostDetailTool) Desc() string {
	return "查看一篇帖子的完整内容（标题、正文、作者、点赞/评论数），用于总结或回答关于该帖的问题。可直接用 title 传入帖子标题（会自动查找），或用 post_id。优先用 title 即可，无需先搜索。"
}
func (t *getPostDetailTool) Params() *schema.ParamsOneOf {
	return schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"title":   {Type: schema.String, Desc: "帖子标题（最简单的方式，会自动按标题查找）"},
		"post_id": {Type: schema.String, Desc: "帖子 post_id"},
	})
}
func (t *getPostDetailTool) NeedConfirm() bool             { return false }
func (t *getPostDetailTool) Preview(map[string]any) string { return "" }
func (t *getPostDetailTool) Execute(ctx context.Context, userID uint64, args map[string]any) (string, error) {
	postID := argU64(args, "post_id")
	if postID == 0 {
		// 按标题取最匹配的一篇
		if title := argStr(args, "title"); title != "" {
			if posts, _, e := t.d.Feed.SearchPosts(ctx, userID, title, "", 1, 1); e == nil && len(posts) > 0 {
				postID = posts[0].PostID
			}
		}
	}
	if postID == 0 {
		return "没有找到对应的帖子，请提供帖子标题(title)或 post_id。", nil
	}
	p, err := t.d.Feed.GetPost(ctx, postID, userID)
	if err != nil {
		return "帖子不存在或已被删除。", nil
	}
	content := strings.TrimSpace(p.Content)
	if content == "" {
		content = "（无正文）"
	}
	imgNote := ""
	if len(p.Images) > 0 {
		imgNote = fmt.Sprintf("，含 %d 张图片", len(p.Images))
	}
	return fmt.Sprintf("[《%s》](/post/%d)  作者：%s · 点赞 %d · 评论 %d%s\n\n正文：\n%s",
		p.Title, p.PostID, p.User.Nickname, p.LikeCount, p.CommentCount, imgNote, content), nil
}

// ── search_user ──

type searchUserTool struct{ d *Deps }

func (t *searchUserTool) Name() string  { return "search_user" }
func (t *searchUserTool) Label() string { return "搜索用户" }
func (t *searchUserTool) Desc() string {
	return "按昵称关键词搜索用户，返回 user_id、昵称和简介。当只知道对方昵称、需要找到其 user_id 时调用。"
}
func (t *searchUserTool) Params() *schema.ParamsOneOf {
	return schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"keyword": {Type: schema.String, Desc: "用户昵称关键词", Required: true},
	})
}
func (t *searchUserTool) NeedConfirm() bool             { return false }
func (t *searchUserTool) Preview(map[string]any) string { return "" }
func (t *searchUserTool) Execute(ctx context.Context, userID uint64, args map[string]any) (string, error) {
	keyword := argStr(args, "keyword")
	if keyword == "" {
		return "请提供搜索关键词。", nil
	}
	users, err := t.d.User.Search(ctx, keyword)
	if err != nil {
		return "", err
	}
	if len(users) == 0 {
		return fmt.Sprintf("没有找到昵称包含「%s」的用户。", keyword), nil
	}
	var sb strings.Builder
	fmt.Fprintf(&sb, "找到 %d 位用户：\n", len(users))
	for _, u := range users {
		bio := u.Bio
		if bio == "" {
			bio = "暂无简介"
		}
		fmt.Fprintf(&sb, "- %s（%s）user_id=%d\n", u.Nickname, bio, u.ID)
	}
	return sb.String(), nil
}

// ── summarize_chat ──

type summarizeChatTool struct{ d *Deps }

func (t *summarizeChatTool) Name() string  { return "summarize_chat" }
func (t *summarizeChatTool) Label() string { return "总结聊天记录" }
func (t *summarizeChatTool) Desc() string {
	return "总结某个会话最近的聊天记录，提炼关键信息与结论。需要 conversation_id（可先用 get_conversations 获取）。当用户想快速了解某段对话讲了什么时调用。"
}
func (t *summarizeChatTool) Params() *schema.ParamsOneOf {
	return schema.NewParamsOneOfByParams(map[string]*schema.ParameterInfo{
		"conversation_id": {Type: schema.String, Desc: "要总结的会话 ID", Required: true},
		"count":           {Type: schema.Integer, Desc: "总结最近多少条消息，默认 50，最大 200"},
	})
}
func (t *summarizeChatTool) NeedConfirm() bool             { return false }
func (t *summarizeChatTool) Preview(map[string]any) string { return "" }
func (t *summarizeChatTool) Execute(ctx context.Context, userID uint64, args map[string]any) (string, error) {
	convID := argU64(args, "conversation_id")
	if convID == 0 {
		return "请提供有效的 conversation_id。", nil
	}
	count := int(argU64(args, "count"))
	if count <= 0 {
		count = 50
	}
	if count > 200 {
		count = 200
	}
	// 复用 GetMessages 含成员校验防越权
	msgs, _, err := t.d.Chat.GetMessages(ctx, userID, convID, 0, count)
	if err != nil {
		return "", err
	}
	if len(msgs) == 0 {
		return "该会话还没有消息可供总结。", nil
	}
	var sb strings.Builder
	// GetMessages 倒序 按时间正序拼接
	for i := len(msgs) - 1; i >= 0; i-- {
		m := msgs[i]
		text := ""
		if v, ok := m.Content["text"]; ok {
			text = fmt.Sprintf("%v", v)
		}
		if text == "" {
			continue
		}
		name := m.SenderNickname
		if name == "" {
			name = fmt.Sprintf("用户%d", m.SenderID)
		}
		fmt.Fprintf(&sb, "%s: %s\n", name, text)
	}
	if sb.Len() == 0 {
		return "该会话暂无文本消息可供总结。", nil
	}
	return t.d.Invoker.Summarize(ctx, sb.String())
}
