package agent

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"regexp"
	"strings"

	"mechat/internal/ai/chain"

	"github.com/cloudwego/eino/schema"
	"go.uber.org/zap"
)

// 提取帖子 Markdown 链接
var postLinkRe = regexp.MustCompile(`\[《(.+?)》\]\((/post/\d+)\)`)

// seenPost 本轮出现过的帖子
type seenPost struct {
	title  string
	link   string // 形如 /post/123
	forced bool   // 聚焦单帖强制补链
}

// 收集帖子链接
func collectPostLinks(text string, forced bool, seen *[]seenPost) {
	for _, m := range postLinkRe.FindAllStringSubmatch(text, -1) {
		*seen = append(*seen, seenPost{title: m[1], link: m[2], forced: forced})
	}
}

// appendPostFooter 末尾补相关帖子链接
func appendPostFooter(content string, seen []seenPost, emit func(Event)) {
	added := map[string]bool{}
	var parts []string
	for _, p := range seen {
		if added[p.link] || strings.Contains(content, p.link) {
			continue
		}
		if !p.forced && !strings.Contains(content, p.title) {
			continue
		}
		added[p.link] = true
		parts = append(parts, fmt.Sprintf("[《%s》](%s)", p.title, p.link))
	}
	if len(parts) > 0 {
		emit(Event{Type: EventToken, Text: "\n\n📎 相关帖子：" + strings.Join(parts, " · ")})
	}
}

// 工具调用轮数上限 防死循环
const maxSteps = 6

// 模型假称完成写操作时的纠正提示
const writeCorrectionSystemMsg = `严重错误：你刚才声称已经“发送/发布/添加”，但你并没有调用任何写操作工具，因此操作【根本没有执行】。
请立刻调用正确的写操作工具（send_message / create_post / send_friend_request）来真正执行该操作。不要再用文字假装完成。`

// 自称完成写操作的措辞
var writeClaimKeywords = []string{
	"已发送", "已发出", "已发给", "已经发送", "已经发出", "已帮你发", "已为你发",
	"已发布", "已经发布", "已为你发布", "已帮你发布",
	"已添加好友", "已发送好友申请", "已为你添加", "已申请添加",
	"发送成功", "发布成功", "申请已发送",
}

// claimsWriteWithoutAction 是否自称完成写操作
func claimsWriteWithoutAction(content string) bool {
	for _, kw := range writeClaimKeywords {
		if strings.Contains(content, kw) {
			return true
		}
	}
	return false
}

// Turn 一轮历史消息
type Turn struct {
	Role    string // "user" | "assistant"
	Content string
}

// Runner 驱动 Agent 循环
type Runner struct {
	reg     *Registry
	invoker *chain.Invoker
	logger  *zap.Logger
}

// 创建 Runner
func NewRunner(reg *Registry, invoker *chain.Invoker, logger *zap.Logger) *Runner {
	return &Runner{reg: reg, invoker: invoker, logger: logger}
}

// Run 执行一次对话 enableTools=false 退化为普通聊天
func (r *Runner) Run(ctx context.Context, userID uint64, system string, turns []Turn, enableTools bool, emit func(Event)) error {
	msgs := make([]*schema.Message, 0, len(turns)+8)
	if system != "" {
		msgs = append(msgs, schema.SystemMessage(system))
	}
	for _, t := range turns {
		if t.Role == "assistant" {
			msgs = append(msgs, schema.AssistantMessage(t.Content, nil))
		} else {
			msgs = append(msgs, schema.UserMessage(t.Content))
		}
	}

	var toolInfos []*schema.ToolInfo
	if enableTools {
		toolInfos = r.reg.ToolInfos()
	}

	corrected := false  // 是否已纠正过 最多一次
	var seen []seenPost // 本轮出现过的帖子

	for range maxSteps {
		// 流式调用 LLM
		stream, err := r.invoker.StreamMessages(ctx, msgs, toolInfos)
		if err != nil {
			return err
		}

		// 消费 stream 收集 chunk
		chunks := make([]*schema.Message, 0, 16)
		for {
			chunk, recvErr := stream.Recv()
			if errors.Is(recvErr, io.EOF) {
				break
			}
			if recvErr != nil {
				stream.Close()
				return recvErr
			}
			if chunk.Content != "" {
				emit(Event{Type: EventToken, Text: chunk.Content})
			}
			chunks = append(chunks, chunk)
		}
		stream.Close()

		// 拼接得到完整消息
		if len(chunks) == 0 {
			return nil
		}
		full, err := schema.ConcatMessages(chunks)
		if err != nil {
			return err
		}

		// 无工具调用 即最终回答
		if len(full.ToolCalls) == 0 {
			// 假称完成则清掉重答 仅一次
			if enableTools && !corrected && claimsWriteWithoutAction(full.Content) {
				emit(Event{Type: EventReset})
				msgs = append(msgs, full)
				msgs = append(msgs, schema.SystemMessage(writeCorrectionSystemMsg))
				corrected = true
				continue
			}
			// 兜底补帖子链接
			appendPostFooter(full.Content, seen, emit)
			return nil
		}

		// 记录 assistant 消息供回喂
		msgs = append(msgs, full)

		// 逐个处理工具调用
		needConfirm := false
		for _, tc := range full.ToolCalls {
			tool, ok := r.reg.Get(tc.Function.Name)
			if !ok {
				msgs = append(msgs, schema.ToolMessage("错误：未知工具 "+tc.Function.Name, tc.ID))
				continue
			}
			args := parseArgs(tc.Function.Arguments)

			// 写操作 预验证后弹确认
			if tool.NeedConfirm() {
				if v, ok := tool.(PreValidator); ok {
					if failMsg := v.PreValidate(ctx, userID, args); failMsg != "" {
						// 验证失败回喂原因 不弹确认
						emit(Event{Type: EventToolStart, Name: tool.Name(), Label: tool.Label()})
						emit(Event{Type: EventToolDone, Name: tool.Name(), Label: tool.Label(), Summary: firstLine(failMsg)})
						msgs = append(msgs, schema.ToolMessage(failMsg, tc.ID))
						continue
					}
				}
				argsJSON, _ := json.Marshal(args) // 重新序列化保证合法 JSON
				emit(Event{
					Type:    EventConfirm,
					Tool:    tool.Name(),
					Label:   tool.Label(),
					Preview: tool.Preview(args),
					Args:    json.RawMessage(argsJSON),
				})
				needConfirm = true
				continue
			}

			// 只读工具直接执行
			emit(Event{Type: EventToolStart, Name: tool.Name(), Label: tool.Label()})
			result, execErr := tool.Execute(ctx, userID, args)
			if execErr != nil {
				result = "执行失败：" + execErr.Error()
			}
			// 收集帖子链接 单帖强制补链
			collectPostLinks(result, tool.Name() == "get_post_detail", &seen)
			emit(Event{Type: EventToolDone, Name: tool.Name(), Label: tool.Label(), Summary: firstLine(result)})
			msgs = append(msgs, schema.ToolMessage(result, tc.ID))
		}

		if needConfirm {
			return nil
		}
	}

	emit(Event{Type: EventToken, Text: "\n\n（已达到工具调用次数上限，请重新提问）"})
	return nil
}
