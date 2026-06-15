package chain

import (
	"context"
	"fmt"

	"github.com/cloudwego/eino/components/model"
	"github.com/cloudwego/eino/schema"
)

// 封装 Eino ChatModel
type Invoker struct {
	model model.ChatModel
}

// 创建 Invoker
func New(m model.ChatModel) *Invoker {
	return &Invoker{model: m}
}

// 模型是否已配置
func (inv *Invoker) Enabled() bool {
	return inv != nil && inv.model != nil
}

// 流式生成 tools 非空时绑定工具
func (inv *Invoker) StreamMessages(ctx context.Context, msgs []*schema.Message, tools []*schema.ToolInfo) (*schema.StreamReader[*schema.Message], error) {
	if len(tools) > 0 {
		tcm, ok := inv.model.(model.ToolCallingChatModel)
		if !ok {
			return nil, fmt.Errorf("当前模型不支持工具调用")
		}
		bound, err := tcm.WithTools(tools)
		if err != nil {
			return nil, fmt.Errorf("bind tools: %w", err)
		}
		return bound.Stream(ctx, msgs)
	}
	return inv.model.Stream(ctx, msgs)
}

// 单轮流式生成
func (inv *Invoker) StreamSingleTurn(ctx context.Context, system, userPrompt string) (*schema.StreamReader[*schema.Message], error) {
	msgs := []*schema.Message{
		{Role: schema.System, Content: system},
		{Role: schema.User, Content: userPrompt},
	}
	return inv.model.Stream(ctx, msgs)
}

// 单轮生成
func (inv *Invoker) chat(ctx context.Context, system, userPrompt string) (string, error) {
	msgs := []*schema.Message{
		{Role: schema.System, Content: system},
		{Role: schema.User, Content: userPrompt},
	}
	resp, err := inv.model.Generate(ctx, msgs)
	if err != nil {
		return "", fmt.Errorf("llm generate: %w", err)
	}
	return resp.Content, nil
}

// 总结聊天记录
func (inv *Invoker) Summarize(ctx context.Context, messages string) (string, error) {
	system := `把下面的聊天记录总结一下，提取关键信息和决定，寒暄废话直接跳过，不超过200字，用中文，说人话别整格式。`
	return inv.chat(ctx, system, "聊天记录：\n"+messages)
}
