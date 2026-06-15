package ai

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"mechat/internal/ai/agent"
	"mechat/internal/ai/chain"
	"mechat/internal/chat"
	"mechat/internal/user"
	"mechat/internal/ws"
	"mechat/pkg/mq"

	"go.uber.org/zap"
)

var ErrVIPRequired = errors.New("该功能仅 VIP 用户可用，请先升级")

type Service struct {
	userRepo *user.Repository
	chatRepo *chat.Repository
	aiRepo   *Repository
	invoker  *chain.Invoker
	logger   *zap.Logger

	registry *agent.Registry // Agent 工具注册表 可为 nil
	runner   *agent.Runner   // Agent 执行器

	// 异步 AI 任务依赖 asyncEnabled=false 降级同步
	pub          mq.Publisher
	hub          *ws.Hub
	asyncEnabled bool
}

// 创建 AI 服务
func NewService(
	userRepo *user.Repository,
	chatRepo *chat.Repository,
	aiRepo *Repository,
	invoker *chain.Invoker,
	logger *zap.Logger,
) *Service {
	return &Service{
		userRepo: userRepo, chatRepo: chatRepo, aiRepo: aiRepo,
		invoker: invoker, logger: logger,
	}
}

// SetAgent 注入工具注册表并建 Runner
func (s *Service) SetAgent(reg *agent.Registry) {
	s.registry = reg
	s.runner = agent.NewRunner(reg, s.invoker, s.logger)
}

// EnableAsync 注入异步依赖 enabled=false 降级同步
func (s *Service) EnableAsync(pub mq.Publisher, hub *ws.Hub, enabled bool) {
	s.pub = pub
	s.hub = hub
	s.asyncEnabled = enabled
}

// isVIP 是否有效 VIP
func (s *Service) isVIP(ctx context.Context, userID uint64) bool {
	u, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return false
	}
	return u.IsVIP()
}

// checkVIP 校验 VIP
func (s *Service) checkVIP(ctx context.Context, userID uint64) error {
	if !s.isVIP(ctx, userID) {
		return ErrVIPRequired
	}
	return nil
}

// GetQuota 返回 VIP 状态
func (s *Service) GetQuota(ctx context.Context, userID uint64) (*QuotaInfo, error) {
	return &QuotaInfo{VIPUser: s.isVIP(ctx, userID)}, nil
}

const chatSystemPrompt = "你是 MeChat 的 AI，贴吧老哥风格，说话直接不废话。知道的直接答，不知道就说不知道，别整那些没营养的客套话。"

const agentSystemPrompt = `你是 MeChat 的 AI，贴吧老哥风格，有话直说，能调工具帮用户操作 App。
【第一铁律】只要用户要”发送消息 / 发布帖子 / 添加好友”，你就必须调用对应写操作工具（send_message / create_post / send_friend_request）来完成；不调用工具操作就不会发生，谎称”已完成”属于严重错误。工具会弹确认框由用户确认，你只管调用、不要替用户犹豫。
工具使用规则（必须严格遵守）：
- 需要会话、好友、动态、用户、某人发过的帖子、帖子内容等实时数据时，必须调用相应查询工具获取，不要凭空编造。
- 任何“发送/发布/添加”类操作，**必须通过调用对应的写操作工具完成（send_message / create_post / send_friend_request）**。严禁在没有调用工具的情况下，用文字声称“已发送/已发布/已添加”——那是错误的，操作不会真正执行。
- 正确做法：先用一句话说明你要做什么，然后立即调用对应写操作工具。工具会弹出确认框，用户确认后才执行。
- 给好友发消息：直接调用 send_message，用 to_nickname 传入对方昵称即可（无需先查好友列表）；发群聊消息用 conversation_id（可先用 get_conversations 获取群的 conversation_id）。
- 添加好友时若只知道昵称，先用 search_user 找到 user_id，再调用 send_friend_request。
- 查看某篇帖子内容时，直接调用 get_post_detail，用 title 传入帖子标题即可（无需先搜索）。
- 只要你的回复提到某篇帖子，就必须用 Markdown 链接给出可点击的帖子地址：[《标题》](/post/帖子ID)。这些链接已包含在工具返回结果中，直接照抄即可，禁止省略。
示例（务必照做，不要只用文字假装完成）：
- “给张三发消息说今晚一起吃饭” → 调用 send_message(to_nickname="张三", text="今晚一起吃饭")。
- “帮我发条帖子聊聊天气” → 调用 create_post(title=..., content=...)。
- “加李四为好友” → 先 search_user(keyword="李四")，再 send_friend_request(to_user_id=...)。
- “看看《周末爬山》讲了啥” → 调用 get_post_detail(title="周末爬山")。
回复直接说重点，别废话，能用 Markdown 排版的用，需要表格用标准 Markdown 表格，使用中文。`

// 记忆窗口 按轮算 一轮=一问一答
const (
	memNormalRounds = 10
	memVIPRounds    = 30
	genTimeout      = 3 * time.Minute
)

// memWindow 喂模型与前端展示的最近消息条数 一轮两条
func memWindow(vip bool) int {
	if vip {
		return memVIPRounds * 2
	}
	return memNormalRounds * 2
}

// History 用户的 AI 对话历史 只返回记忆窗口内的 超出的看不到
func (s *Service) History(ctx context.Context, userID uint64) ([]*AIMessage, error) {
	return s.aiRepo.Recent(ctx, userID, memWindow(s.isVIP(ctx, userID)))
}

// ClearHistory 清空用户的 AI 对话
func (s *Service) ClearHistory(ctx context.Context, userID uint64) error {
	return s.aiRepo.Clear(ctx, userID)
}

// Chat 落库用户消息 后台生成回答 返回事件流供 SSE 转发与待回填消息 id
func (s *Service) Chat(userID uint64, text string) (uint64, <-chan agent.Event, error) {
	ctx := context.Background()
	if s.invoker == nil || !s.invoker.Enabled() || s.runner == nil {
		return 0, nil, errors.New("AI 服务未配置，请在 config.yaml 设置 ai.api_key 后重启服务")
	}
	if err := s.aiRepo.Add(ctx, &AIMessage{UserID: userID, Role: "user", Content: text, Status: "done"}); err != nil {
		return 0, nil, err
	}

	// 记忆窗口与系统提示按 VIP 区分 工具仅 VIP
	vip := s.isVIP(ctx, userID)
	system := chatSystemPrompt
	if vip {
		system = agentSystemPrompt
	}
	// 历史拼成模型上下文 工具注记并入文本喂模型 跳过空轮
	recent, _ := s.aiRepo.Recent(ctx, userID, memWindow(vip))
	turns := make([]agent.Turn, 0, len(recent))
	for _, m := range recent {
		content := strings.TrimSpace(m.Content)
		if m.Note != "" {
			content = strings.TrimSpace(content + "\n" + m.Note)
		}
		if content == "" {
			continue
		}
		turns = append(turns, agent.Turn{Role: m.Role, Content: content})
	}

	// 占位 assistant 刷新时能看到生成中
	pending := &AIMessage{UserID: userID, Role: "assistant", Status: "pending"}
	if err := s.aiRepo.Add(ctx, pending); err != nil {
		return 0, nil, err
	}

	out := make(chan agent.Event, 32)
	go s.generate(userID, pending.ID, system, turns, vip, out)
	return pending.ID, out, nil
}

// generate 后台生成 用 Background ctx 不随客户端断开取消 完成即落库
func (s *Service) generate(userID, pendingID uint64, system string, turns []agent.Turn, enableTools bool, out chan<- agent.Event) {
	ctx, cancel := context.WithTimeout(context.Background(), genTimeout)
	defer cancel()
	defer close(out)

	var sb strings.Builder
	var confirms []string // 本轮弹过确认框的写操作 用于注记
	var pending string    // 待确认写操作 JSON 供切栏目/刷新后恢复确认框
	emit := func(ev agent.Event) {
		switch ev.Type {
		case agent.EventToken:
			sb.WriteString(ev.Text)
		case agent.EventReset:
			sb.Reset()
		case agent.EventConfirm:
			confirms = append(confirms, ev.Label)
			if b, e := json.Marshal(pendingAction{Tool: ev.Tool, Label: ev.Label, Preview: ev.Preview, Args: ev.Args}); e == nil {
				pending = string(b)
			}
		}
		out <- ev // 阻塞写 由 handler 持续读取或断开后排空
	}

	status := "done"
	if err := s.runner.Run(ctx, userID, system, turns, enableTools, emit); err != nil {
		emit(agent.Event{Type: agent.EventError, Message: err.Error()})
		status = "error"
	}
	// 弹过确认框就记一笔 提醒下一轮模型别当已完成
	note := ""
	if len(confirms) > 0 {
		note = "（系统记录：已就「" + strings.Join(confirms, "」「") + "」弹出确认框，用户未确认前这些写操作尚未执行，请勿当作已完成）"
	}
	if err := s.aiRepo.Finish(context.Background(), pendingID, sb.String(), status, note, pending); err != nil {
		s.logger.Warn("finish ai message", zap.Error(err))
	}
}

// pendingAction 待确认写操作落库结构 与前端确认卡片对应
type pendingAction struct {
	Tool    string          `json:"tool"`
	Label   string          `json:"label"`
	Preview string          `json:"preview"`
	Args    json.RawMessage `json:"args,omitempty"`
}

// ConfirmAction 执行已确认的写操作
func (s *Service) ConfirmAction(ctx context.Context, userID uint64, toolName string, args map[string]any) (string, error) {
	if s.registry == nil {
		return "", errors.New("AI 服务未配置")
	}
	if err := s.checkVIP(ctx, userID); err != nil {
		return "", err
	}
	tool, ok := s.registry.Get(toolName)
	if !ok || !tool.NeedConfirm() {
		return "", errors.New("无效的操作")
	}
	result, err := tool.Execute(ctx, userID, args)
	if err == nil {
		// 已执行 清掉待确认 写最终注记 防止模型下一轮重复发起
		note := "（系统记录：「" + tool.Label() + "」已确认并执行成功，请勿重复执行）"
		if e := s.aiRepo.ResolvePending(ctx, userID, note); e != nil {
			s.logger.Warn("resolve pending", zap.Error(e))
		}
	}
	return result, err
}

// CancelAction 用户取消写操作 清掉待确认 切回来不再弹确认框
func (s *Service) CancelAction(ctx context.Context, userID uint64) error {
	return s.aiRepo.ClearPending(ctx, userID)
}

// streamTokens 消费单轮流并推 token
func (s *Service) streamTokens(ctx context.Context, system, userPrompt string, emit func(agent.Event)) {
	if s.invoker == nil || !s.invoker.Enabled() {
		emit(agent.Event{Type: agent.EventError, Message: "AI 服务未配置，请在 config.yaml 设置 ai.api_key"})
		return
	}
	stream, err := s.invoker.StreamSingleTurn(ctx, system, userPrompt)
	if err != nil {
		emit(agent.Event{Type: agent.EventError, Message: err.Error()})
		return
	}
	defer stream.Close()
	for {
		chunk, recvErr := stream.Recv()
		if errors.Is(recvErr, io.EOF) {
			break
		}
		if recvErr != nil {
			emit(agent.Event{Type: agent.EventError, Message: recvErr.Error()})
			return
		}
		if chunk.Content != "" {
			emit(agent.Event{Type: agent.EventToken, Text: chunk.Content})
		}
	}
}

const draftMessageSystem = `帮用户把要说的意思整成一条消息。直接输出消息正文，不要解释，不要加前缀，说人话，长度适中。`

const draftPostSystem = `根据关键词帮用户整一篇贴吧风格的帖子，包含标题和正文。
严格按照以下格式输出，不要有任何多余内容：
标题：<简短直接的标题，20字以内>
正文：
<帖子正文>

正文要求：
- 说人话，有啥说啥，别整"绝绝子""yyds"或小红书那套"姐妹们冲鸭！！！"
- 学习孙吧老哥的说话风格
- 直接点，废话少说，但要把事情说清楚
- 适当换行，看着舒服点
- 长度200-400字
- 不使用 Markdown 格式`

// StreamDraftMessage 帮写消息
func (s *Service) StreamDraftMessage(ctx context.Context, _ uint64, req *DraftMessageReq, emit func(agent.Event)) {
	userPrompt := fmt.Sprintf("聊天上下文：%s\n用户描述/草稿：%s\n\n请生成或优化这条消息：", req.Context, req.Draft)
	s.streamTokens(ctx, draftMessageSystem, userPrompt, emit)
}

// StreamDraftPost 帮写帖子
func (s *Service) StreamDraftPost(ctx context.Context, _ uint64, req *DraftPostReq, emit func(agent.Event)) {
	s.streamTokens(ctx, draftPostSystem, "请根据以下关键词生成帖子："+req.Keywords, emit)
}

// aiTaskSummarize 会话总结任务类型
const aiTaskSummarize = "summarize"

// aiTask 异步任务信封
type aiTask struct {
	Kind           string `json:"kind"`
	UserID         uint64 `json:"user_id"`
	ConversationID uint64 `json:"conversation_id"`
	MessageCount   int    `json:"message_count"`
}

// SubmitSummarize 提交总结任务 queued=false 表示已同步执行
func (s *Service) SubmitSummarize(ctx context.Context, userID uint64, req *SummarizeReq) (queued bool, result string, err error) {
	if err = s.checkVIP(ctx, userID); err != nil {
		return false, "", err
	}
	// MQ 不可用同步执行
	if !s.asyncEnabled || s.pub == nil {
		result, err = s.runSummarize(ctx, req.ConversationID, req.MessageCount)
		return false, result, err
	}
	// 入队后立即返回
	body, _ := json.Marshal(aiTask{
		Kind:           aiTaskSummarize,
		UserID:         userID,
		ConversationID: req.ConversationID,
		MessageCount:   req.MessageCount,
	})
	if err = s.pub.Publish(ctx, mq.QueueAITask, body); err != nil {
		return false, "", err
	}
	return true, "", nil
}

// ProcessAITask 消费 AI 任务 结果经 WS 推回 返回 nil 即 ACK
func (s *Service) ProcessAITask(ctx context.Context, body []byte) error {
	var task aiTask
	if err := json.Unmarshal(body, &task); err != nil {
		s.logger.Error("unmarshal ai task", zap.Error(err))
		return nil // 格式错误不重试
	}
	switch task.Kind {
	case aiTaskSummarize:
		result, err := s.runSummarize(ctx, task.ConversationID, task.MessageCount)
		if err != nil {
			s.logger.Warn("summarize task failed", zap.Uint64("user_id", task.UserID), zap.Error(err))
		}
		// 结果经 WS 推回发起者
		errStr := ""
		if err != nil {
			errStr = err.Error()
		}
		if s.hub != nil {
			data := ws.AIResultData{Kind: aiTaskSummarize, ConversationID: task.ConversationID, Result: result, Error: errStr}
			if payload, e := ws.BuildEnvelope(ws.TypeAIResult, 0, data); e == nil {
				s.hub.Deliver(ctx, task.UserID, payload)
			}
		}
	default:
		s.logger.Warn("unknown ai task kind", zap.String("kind", task.Kind))
	}
	return nil
}

// runSummarize 执行会话总结
func (s *Service) runSummarize(ctx context.Context, convID uint64, messageCount int) (string, error) {
	if s.invoker == nil || !s.invoker.Enabled() {
		return "", errors.New("AI 服务未配置，请在 config.yaml 设置 ai.api_key 后重启服务")
	}

	msgs, err := s.chatRepo.GetMessages(ctx, convID, 0, messageCount)
	if err != nil {
		return "", err
	}

	// 按时间正序拼接
	var sb strings.Builder
	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]
		content := ""
		if text, ok := msg.Content["text"]; ok {
			content = fmt.Sprintf("%v", text)
		}
		fmt.Fprintf(&sb, "[用户%d]: %s\n", msg.SenderID, content)
	}

	return s.invoker.Summarize(ctx, sb.String())
}
