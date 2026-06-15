package mq

import (
	"context"
	"fmt"
	"sync"
	"time"

	amqp "github.com/rabbitmq/amqp091-go"
	"go.uber.org/zap"
)

const (
	dlxName       = "mechat.dlx" 		// 死信交换机
	prefetchCount = 16           		// 未确认消息上限 兼并发上限
	reconnectMin  = 1 * time.Second
	reconnectMax  = 30 * time.Second
)

// 死信队列名
func dlqName(queue string) string { return queue + ".dlq" }

// 已注册消费者 重连后据此重建
type consumerReg struct {
	queue   string
	handler func(body []byte) error
}

// RabbitMQ 带自动重连的客户端
type RabbitMQ struct {
	url    string
	logger *zap.Logger

	mu      sync.RWMutex
	conn    *amqp.Connection
	channel *amqp.Channel

	consumers []consumerReg // 已注册消费者
	closed    bool          // 主动关闭标志
}

// 创建 RabbitMQ 客户端
func NewRabbitMQ(url string, logger *zap.Logger) (*RabbitMQ, error) {
	r := &RabbitMQ{url: url, logger: logger}
	if err := r.connect(); err != nil {
		return nil, err
	}
	return r, nil
}

// 建连 + 通道 + QoS + 拓扑 + 断线监听
func (r *RabbitMQ) connect() error {
	conn, err := amqp.DialConfig(r.url, amqp.Config{
		Heartbeat: 30 * time.Second,                   // 空闲心跳
		Dial:      amqp.DefaultDial(10 * time.Second), // 连接超时
	})
	if err != nil {
		return fmt.Errorf("rabbitmq dial: %w", err)
	}
	ch, err := conn.Channel()
	if err != nil {
		conn.Close()
		return fmt.Errorf("rabbitmq channel: %w", err)
	}
	// 限制未确认消息数
	if err := ch.Qos(prefetchCount, 0, false); err != nil {
		ch.Close()
		conn.Close()
		return fmt.Errorf("rabbitmq qos: %w", err)
	}
	if err := declareTopology(ch); err != nil {
		ch.Close()
		conn.Close()
		return err
	}

	r.mu.Lock()
	r.conn = conn
	r.channel = ch
	r.mu.Unlock()

	go r.watchClose(conn) // 监听关闭触发重连
	return nil
}

// 声明交换机与队列拓扑
func declareTopology(ch *amqp.Channel) error {
	// 主交换机
	if err := ch.ExchangeDeclare(ExchangeName, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare exchange: %w", err)
	}
	// 死信交换机
	if err := ch.ExchangeDeclare(dlxName, "direct", true, false, false, false, nil); err != nil {
		return fmt.Errorf("declare dlx: %w", err)
	}

	for _, q := range []string{QueueAITask} {
		dlq := dlqName(q)
		// 死信队列 + 绑定
		if _, err := ch.QueueDeclare(dlq, true, false, false, false, nil); err != nil {
			return fmt.Errorf("declare dlq %s: %w", dlq, err)
		}
		if err := ch.QueueBind(dlq, dlq, dlxName, false, nil); err != nil {
			return fmt.Errorf("bind dlq %s: %w", dlq, err)
		}
		// 业务队列 失败消息路由到 DLQ
		args := amqp.Table{
			"x-dead-letter-exchange":    dlxName,
			"x-dead-letter-routing-key": dlq,
		}
		if _, err := ch.QueueDeclare(q, true, false, false, false, args); err != nil {
			return fmt.Errorf("declare queue %s: %w", q, err)
		}
		if err := ch.QueueBind(q, q, ExchangeName, false, nil); err != nil {
			return fmt.Errorf("bind queue %s: %w", q, err)
		}
	}
	return nil
}

// 等待关闭并指数退避重连
func (r *RabbitMQ) watchClose(conn *amqp.Connection) {
	reason := <-conn.NotifyClose(make(chan *amqp.Error))

	r.mu.RLock()
	closed := r.closed
	r.mu.RUnlock()
	if closed {
		return // 主动关闭不重连
	}
	r.logger.Warn("rabbitmq connection lost, reconnecting", zap.Any("reason", reason))

	backoff := reconnectMin
	for {
		time.Sleep(backoff)

		r.mu.RLock()
		closed := r.closed
		r.mu.RUnlock()
		if closed {
			return
		}

		if err := r.connect(); err != nil {
			r.logger.Warn("rabbitmq reconnect failed, will retry",
				zap.Duration("backoff", backoff), zap.Error(err))
			if backoff *= 2; backoff > reconnectMax {
				backoff = reconnectMax
			}
			continue
		}

		r.logger.Info("rabbitmq reconnected")
		// 重建消费者
		r.mu.RLock()
		regs := make([]consumerReg, len(r.consumers))
		copy(regs, r.consumers)
		r.mu.RUnlock()
		for _, reg := range regs {
			if err := r.startConsume(reg); err != nil {
				r.logger.Error("rabbitmq re-consume failed", zap.String("queue", reg.queue), zap.Error(err))
			}
		}
		return
	}
}

// 发布消息
func (r *RabbitMQ) Publish(ctx context.Context, routingKey string, body []byte) error {
	r.mu.RLock()
	ch := r.channel
	r.mu.RUnlock()
	if ch == nil {
		return fmt.Errorf("rabbitmq channel unavailable")
	}
	return ch.PublishWithContext(ctx, ExchangeName, routingKey, false, false,
		amqp.Publishing{
			ContentType:  "application/json",
			DeliveryMode: amqp.Persistent,
			Body:         body,
		},
	)
}

// Consume 注册并启动消费者
func (r *RabbitMQ) Consume(queue string, handler func(body []byte) error) error {
	reg := consumerReg{queue: queue, handler: handler}
	r.mu.Lock()
	r.consumers = append(r.consumers, reg)
	r.mu.Unlock()
	return r.startConsume(reg)
}

// 开始消费队列 每条消息独立 goroutine 处理 失败或 panic 进 DLQ
func (r *RabbitMQ) startConsume(reg consumerReg) error {
	r.mu.RLock()
	ch := r.channel
	r.mu.RUnlock()
	if ch == nil {
		return fmt.Errorf("rabbitmq channel unavailable")
	}
	msgs, err := ch.Consume(reg.queue, "", false, false, false, false, nil)
	if err != nil {
		return err
	}

	go func() {
		sem := make(chan struct{}, prefetchCount) // 并发闸门
		for d := range msgs {
			sem <- struct{}{}
			go func(d amqp.Delivery) {
				defer func() { <-sem }()
				defer func() {
					if rec := recover(); rec != nil {
						r.logger.Error("mq handler panic, dead-lettering",
							zap.String("queue", reg.queue), zap.Any("panic", rec))
						d.Nack(false, false)
					}
				}()
				if err := reg.handler(d.Body); err != nil {
					r.logger.Error("mq handler error, dead-lettering",
						zap.String("queue", reg.queue), zap.Error(err))
					d.Nack(false, false)
				} else {
					d.Ack(false)
				}
			}(d)
		}
	}()
	return nil
}

// 关闭连接
func (r *RabbitMQ) Close() error {
	r.mu.Lock()
	r.closed = true
	ch, conn := r.channel, r.conn
	r.mu.Unlock()
	if ch != nil {
		ch.Close()
	}
	if conn != nil {
		return conn.Close()
	}
	return nil
}
