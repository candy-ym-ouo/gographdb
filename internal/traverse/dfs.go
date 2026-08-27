package traverse

// DFS 以确定性顺序深度优先遍历。
func (t *Traverser) DFS(start uint64, options Options) ([]Visit, error) {
	if _, err := t.Graph.Vertex(start); err != nil {
		return nil, err
	}
	visited := map[uint64]struct{}{}
	result := []Visit{}
	var walk func(uint64, int) error
	walk = func(id uint64, depth int) error {
		visited[id] = struct{}{}
		result = append(result, Visit{ID: id, Depth: depth})
		if depth >= options.MaxDepth {
			return nil
		}
		neighbors, err := t.Neighbors(id, options)
		if err != nil {
			return err
		}
		for _, next := range sortedKeys(neighbors) {
			if _, ok := visited[next]; !ok {
				if err = walk(next, depth+1); err != nil {
					return err
				}
			}
		}
		return nil
	}
	return result, walk(start, 0)
}

// HasCycle 使用三色标记检测有向环。
func (t *Traverser) HasCycle() bool {
	colors := map[uint64]uint8{}
	var visit func(uint64) bool
	visit = func(id uint64) bool {
		colors[id] = 1
		neighbors, err := t.Neighbors(id, Options{Direction: "OUT", MaxDepth: 1})
		if err != nil {
			return false
		}
		for _, next := range sortedKeys(neighbors) {
			if colors[next] == 1 {
				return true
			}
			if colors[next] == 0 && visit(next) {
				return true
			}
		}
		colors[id] = 2
		return false
	}
	for _, vertex := range t.Graph.Vertices() {
		if colors[vertex.ID] == 0 && visit(vertex.ID) {
			return true
		}
	}
	return false
}
