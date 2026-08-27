// Package storage 提供图数据的内存存储、快照和预写日志。
package storage

import "gographdb/internal/graph"

// Snapshot 是稳定的磁盘交换格式。
type Snapshot struct {
	Version      uint16          `json:"version"`
	NextVertexID uint64          `json:"nextVertexId"`
	NextEdgeID   uint64          `json:"nextEdgeId"`
	Vertices     []*graph.Vertex `json:"vertices"`
	Edges        []*graph.Edge   `json:"edges"`
}

// Store 编译期确认 MemStore 满足图引擎接口。
var _ graph.Store = (*MemStore)(nil)
