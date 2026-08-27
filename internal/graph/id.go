package graph

import "sync"

// IDAllocator 是并发安全的单调 ID 分配器。
type IDAllocator struct {
	mu   sync.Mutex
	next uint64
}

// NewIDAllocator 创建从 start 开始的分配器。
func NewIDAllocator(start uint64) *IDAllocator {
	if start == 0 {
		start = 1
	}
	return &IDAllocator{next: start}
}

// Next 分配下一个 ID。
func (a *IDAllocator) Next() uint64 {
	a.mu.Lock()
	defer a.mu.Unlock()
	id := a.next
	a.next++
	return id
}

// Peek 返回下一个将被分配的 ID。
func (a *IDAllocator) Peek() uint64 { a.mu.Lock(); defer a.mu.Unlock(); return a.next }

// Restore 仅在更大时推进分配器，避免恢复后重复。
func (a *IDAllocator) Restore(next uint64) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if next > a.next {
		a.next = next
	}
}
