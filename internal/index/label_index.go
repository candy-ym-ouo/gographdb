package index

import "sync"

// LabelIndex 将标签映射到节点集合。
type LabelIndex struct {
	mu     sync.RWMutex
	values map[string]map[uint64]struct{}
}

// NewLabelIndex 创建标签索引。
func NewLabelIndex() *LabelIndex { return &LabelIndex{values: map[string]map[uint64]struct{}{}} }

// Add 添加标签记录。
func (i *LabelIndex) Add(label string, id uint64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	set := i.values[label]
	if set == nil {
		set = map[uint64]struct{}{}
		i.values[label] = set
	}
	set[id] = struct{}{}
}

// Remove 删除标签记录。
func (i *LabelIndex) Remove(label string, id uint64) {
	i.mu.Lock()
	defer i.mu.Unlock()
	set := i.values[label]
	delete(set, id)
	if len(set) == 0 {
		delete(i.values, label)
	}
}

// Lookup 返回匹配标签的有序 ID。
func (i *LabelIndex) Lookup(label string) []uint64 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return IDs(i.values[label])
}

// Clear 清空索引。
func (i *LabelIndex) Clear() { i.mu.Lock(); i.values = map[string]map[uint64]struct{}{}; i.mu.Unlock() }

// Stats 返回统计。
func (i *LabelIndex) Stats() Stats {
	i.mu.RLock()
	defer i.mu.RUnlock()
	entries := 0
	for _, set := range i.values {
		entries += len(set)
	}
	return Stats{Name: "vertex_labels", Kind: "label", Entries: entries, Keys: len(i.values)}
}
