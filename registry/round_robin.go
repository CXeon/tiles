package registry

import (
	"slices"
	"sync/atomic"
)

type RoundRobinBalancer struct {
	index uint64 // 使用 uint64 配合原子操作
}

func NewRoundRobinBalancer() *RoundRobinBalancer {
	return &RoundRobinBalancer{index: 0}
}

// Select 使用原子操作实现线程安全的轮询
func (lb *RoundRobinBalancer) Select(eps []Endpoint) *Endpoint {
	length := len(eps)
	if length == 0 {
		return nil
	}

	// 按权重降序排序
	slices.SortFunc(eps, func(a, b Endpoint) int {
		return int(b.Weight - a.Weight)
	})

	// 使用原子操作获取索引并递增，避免竞态条件
	idx := atomic.AddUint64(&lb.index, 1) - 1
	idx = idx % uint64(length)

	return &eps[idx]
}
