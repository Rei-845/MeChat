package mq

import "context"

// Publisher 消息发布接口
type Publisher interface {
	Publish(ctx context.Context, routingKey string, body []byte) error
	Close() error
}

// Consumer 消息消费接口
type Consumer interface {
	Consume(queue string, handler func(body []byte) error) error
	Close() error
}

// 队列名称常量
const (
	// QueueAITask 异步 AI 任务队列
	QueueAITask = "ai_task"

	ExchangeName = "mechat.direct"
)
