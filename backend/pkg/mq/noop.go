package mq

import "context"

// MQ 不可用时的空实现
type NoopMQ struct{}

func (n *NoopMQ) Publish(_ context.Context, _ string, _ []byte) error { return nil }
func (n *NoopMQ) Consume(_ string, _ func([]byte) error) error        { return nil }
func (n *NoopMQ) Close() error                                        { return nil }
