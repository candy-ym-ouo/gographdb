// Package traverse 实现图遍历与路径算法。
package traverse

import (
	"sort"

	"gographdb/internal/graph"
)

// Options 控制遍历方向、边标签和最大深度。
type Options struct {
	Direction  graph.Direction
	EdgeLabels []string
	MaxDepth   int
}

// Traverser 基于图引擎的稳定读取接口工作。
type Traverser struct{ Graph *graph.Graph }

// New 创建遍历器。
func New(g *graph.Graph) *Traverser { return &Traverser{Graph: g} }

// Neighbors 返回去重后的邻居及连接边。
func (t *Traverser) Neighbors(id uint64, options Options) (map[uint64][]*graph.Edge, error) {
	direction := options.Direction
	if direction == "" {
		direction = graph.Out
	}
	edges, err := t.Graph.EdgesOf(id, direction)
	if err != nil {
		return nil, err
	}
	filter := map[string]struct{}{}
	for _, label := range options.EdgeLabels {
		filter[label] = struct{}{}
	}
	out := map[uint64][]*graph.Edge{}
	for _, edge := range edges {
		if len(filter) > 0 {
			if _, ok := filter[edge.Label]; !ok {
				continue
			}
		}
		other, ok := edge.Other(id, direction)
		if ok {
			out[other] = append(out[other], edge)
		}
	}
	return out, nil
}

func sortedKeys(values map[uint64][]*graph.Edge) []uint64 {
	ids := make([]uint64, 0, len(values))
	for id := range values {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	return ids
}
