// Package index 实现标签、属性等值和有序范围索引。
package index

import (
	"sort"

	"gographdb/internal/graph"
)

// Stats 描述一个索引的规模。
type Stats struct {
	Name    string `json:"name"`
	Kind    string `json:"kind"`
	Entries int    `json:"entries"`
	Keys    int    `json:"keys"`
}

// IDs 将 ID 集合转为有序切片。
func IDs(set map[uint64]struct{}) []uint64 {
	out := make([]uint64, 0, len(set))
	for id := range set {
		out = append(out, id)
	}
	sort.Slice(out, func(i, j int) bool { return out[i] < out[j] })
	return out
}

// cloneSet 复制集合。
func cloneSet(src map[uint64]struct{}) map[uint64]struct{} {
	out := make(map[uint64]struct{}, len(src))
	for id := range src {
		out[id] = struct{}{}
	}
	return out
}

// valueKey 返回统一属性键。
func valueKey(value graph.PropertyValue) string { key, _ := value.Key(); return key }
