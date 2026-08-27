package traverse

// Visit 表示节点及其遍历深度。
type Visit struct {
	ID    uint64 `json:"id"`
	Depth int    `json:"depth"`
}

type queueItem struct {
	id    uint64
	depth int
}

// BFS 从起点执行层序遍历，结果包含起点。
func (t *Traverser) BFS(start uint64, options Options) ([]Visit, error) {
	if _, err := t.Graph.Vertex(start); err != nil {
		return nil, err
	}
	if options.MaxDepth < 0 {
		options.MaxDepth = 0
	}
	queue := []queueItem{{start, 0}}
	visited := map[uint64]struct{}{start: {}}
	result := []Visit{{ID: start, Depth: 0}}
	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]
		if item.depth >= options.MaxDepth {
			continue
		}
		neighbors, err := t.Neighbors(item.id, options)
		if err != nil {
			return nil, err
		}
		for _, id := range sortedKeys(neighbors) {
			if _, ok := visited[id]; ok {
				continue
			}
			visited[id] = struct{}{}
			next := queueItem{id, item.depth + 1}
			queue = append(queue, next)
			result = append(result, Visit{ID: id, Depth: next.depth})
		}
	}
	return result, nil
}

// Reachable 判断目标是否在给定深度内可达。
func (t *Traverser) Reachable(from, to uint64, options Options) (bool, error) {
	visits, err := t.BFS(from, options)
	if err != nil {
		return false, err
	}
	for _, visit := range visits {
		if visit.ID == to {
			return true, nil
		}
	}
	return false, nil
}
