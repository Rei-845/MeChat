package ai

import (
	"context"

	"mechat/pkg/mq"

	"go.uber.org/zap"
)

// Consumer 消费异步 AI 任务队列
type Consumer struct {
	sub    mq.Consumer
	svc    *Service
	logger *zap.Logger
}

// 创建消费者
func NewConsumer(sub mq.Consumer, svc *Service, logger *zap.Logger) *Consumer {
	return &Consumer{sub: sub, svc: svc, logger: logger}
}

// Start 注册消费回调 用 Background context 不随请求取消
func (c *Consumer) Start() error {
	return c.sub.Consume(mq.QueueAITask, func(body []byte) error {
		return c.svc.ProcessAITask(context.Background(), body)
	})
}
