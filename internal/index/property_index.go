package index

import (
	"sync"

	"gographdb/internal/graph"
)

// PropertyIndex 对所有节点属性提供等值索引。
type PropertyIndex struct {
	mu     sync.RWMutex
	values map[string]map[string]map[uint64]struct{}
}

// NewPropertyIndex 创建属性索引。
func NewPropertyIndex() *PropertyIndex {
	return &PropertyIndex{values: map[string]map[string]map[uint64]struct{}{}}
}

// Add 添加属性值。
func (i *PropertyIndex) Add(name string, value graph.PropertyValue, id uint64) {
	key := valueKey(value)
	i.mu.Lock()
	defer i.mu.Unlock()
	byValue := i.values[name]
	if byValue == nil {
		byValue = map[string]map[uint64]struct{}{}
		i.values[name] = byValue
	}
	set := byValue[key]
	if set == nil {
		set = map[uint64]struct{}{}
		byValue[key] = set
	}
	set[id] = struct{}{}
}

// Remove 删除属性值。
func (i *PropertyIndex) Remove(name string, value graph.PropertyValue, id uint64) {
	key := valueKey(value)
	i.mu.Lock()
	defer i.mu.Unlock()
	byValue := i.values[name]
	set := byValue[key]
	delete(set, id)
	if len(set) == 0 {
		delete(byValue, key)
	}
	if len(byValue) == 0 {
		delete(i.values, name)
	}
}

// Lookup 返回等值匹配 ID。
func (i *PropertyIndex) Lookup(name string, value graph.PropertyValue) []uint64 {
	i.mu.RLock()
	defer i.mu.RUnlock()
	return IDs(i.values[name][valueKey(value)])
}

// Clear 清空索引。
func (i *PropertyIndex) Clear() {
	i.mu.Lock()
	i.values = map[string]map[string]map[uint64]struct{}{}
	i.mu.Unlock()
}

// Stats 返回统计。
func (i *PropertyIndex) Stats() Stats {
	i.mu.RLock()
	defer i.mu.RUnlock()
	entries, keys := 0, 0
	for _, byValue := range i.values {
		keys += len(byValue)
		for _, set := range byValue {
			entries += len(set)
		}
	}
	return Stats{Name: "vertex_properties", Kind: "property", Entries: entries, Keys: keys}
}
