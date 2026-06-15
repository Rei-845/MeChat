// Package agent 实现 AI 助手的工具调用能力
package agent

import (
	"context"
	"encoding/json"
	"strconv"
	"strings"

	"github.com/cloudwego/eino/schema"
)

// PreValidator 写操作工具可选接口
type PreValidator interface {
	PreValidate(ctx context.Context, userID uint64, args map[string]any) string
}

// Tool Agent 工具统一接口
type Tool interface {
	// Name 工具唯一标识
	Name() string
	// Label 展示名
	Label() string
	// Desc 给模型的功能描述
	Desc() string
	// Params 参数 schema
	Params() *schema.ParamsOneOf
	// NeedConfirm 是否写操作
	NeedConfirm() bool
	// Preview 确认卡片预览文本
	Preview(args map[string]any) string
	// Execute 执行工具
	Execute(ctx context.Context, userID uint64, args map[string]any) (string, error)
}

// Registry 工具注册表 只读共享
type Registry struct {
	order []string
	tools map[string]Tool
}

// 创建注册表
func NewRegistry() *Registry {
	return &Registry{tools: make(map[string]Tool)}
}

// Register 注册工具
func (r *Registry) Register(t Tool) {
	if _, exists := r.tools[t.Name()]; !exists {
		r.order = append(r.order, t.Name())
	}
	r.tools[t.Name()] = t
}

// 取工具
func (r *Registry) Get(name string) (Tool, bool) {
	t, ok := r.tools[name]
	return t, ok
}

// ToolInfos 按注册顺序返回工具定义
func (r *Registry) ToolInfos() []*schema.ToolInfo {
	infos := make([]*schema.ToolInfo, 0, len(r.order))
	for _, name := range r.order {
		t := r.tools[name]
		infos = append(infos, &schema.ToolInfo{
			Name:        t.Name(),
			Desc:        t.Desc(),
			ParamsOneOf: t.Params(),
		})
	}
	return infos
}

// ── SSE 事件 ──

const (
	EventToken     = "token"      // 模型输出的文本片段
	EventToolStart = "tool_start" // 开始调用某工具
	EventToolDone  = "tool_done"  // 工具调用完成
	EventConfirm   = "confirm"    // 写操作待用户确认
	EventReset     = "reset"      // 清空已输出文本 重答
	EventError     = "error"      // 出错
	EventDone      = "done"       // 流结束
)

// Event SSE 单条事件
type Event struct {
	Type    string          `json:"type"`
	Text    string          `json:"text,omitempty"`    // token
	Name    string          `json:"name,omitempty"`    // tool 名称
	Label   string          `json:"label,omitempty"`   // tool 展示名
	Summary string          `json:"summary,omitempty"` // tool_done 简要结果
	Tool    string          `json:"tool,omitempty"`    // confirm 的工具名
	Preview string          `json:"preview,omitempty"` // confirm 预览
	Args    json.RawMessage `json:"args,omitempty"`    // confirm 的原始参数
	Message string          `json:"message,omitempty"` // error 信息
}

// 解析参数
func parseArgs(raw string) map[string]any {
	args := map[string]any{}
	if strings.TrimSpace(raw) == "" {
		return args
	}
	_ = json.Unmarshal([]byte(raw), &args)
	return args
}

// argU64 读 uint64 兼容字符串与数字
func argU64(args map[string]any, key string) uint64 {
	switch v := args[key].(type) {
	case float64:
		return uint64(v)
	case json.Number:
		n, _ := v.Int64()
		return uint64(n)
	case string:
		n, _ := strconv.ParseUint(strings.TrimSpace(v), 10, 64)
		return n
	}
	return 0
}

// 读字符串参数
func argStr(args map[string]any, key string) string {
	if v, ok := args[key].(string); ok {
		return strings.TrimSpace(v)
	}
	return ""
}

// firstLine 取首行简要展示
func firstLine(s string) string {
	s = strings.TrimSpace(s)
	if i := strings.IndexByte(s, '\n'); i >= 0 {
		s = s[:i]
	}
	r := []rune(s)
	if len(r) > 40 {
		return string(r[:40]) + "…"
	}
	return s
}
