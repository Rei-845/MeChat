package snowflake

import (
	"sync"

	"github.com/bwmarrin/snowflake"
)

var (
	node *snowflake.Node
	once sync.Once
)

// 初始化雪花节点
func Init(nodeID int64) error {
	var initErr error
	once.Do(func() {
		n, err := snowflake.NewNode(nodeID)
		if err != nil {
			initErr = err
			return
		}
		node = n
	})
	return initErr
}

// 生成雪花 ID
func NextID() uint64 {
	return uint64(node.Generate().Int64())
}
