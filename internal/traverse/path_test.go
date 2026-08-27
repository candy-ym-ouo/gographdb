package traverse_test

import (
	"gographdb/internal/graph"
	"gographdb/internal/storage"
	"gographdb/internal/traverse"
	"testing"
)

func TestShortestPaths(t *testing.T) {
	g := graph.New(storage.NewMemStore())
	a, _ := g.CreateVertex(nil, nil)
	b, _ := g.CreateVertex(nil, nil)
	c, _ := g.CreateVertex(nil, nil)
	g.CreateEdge(a.ID, b.ID, "ROAD", map[string]graph.PropertyValue{"weight": {Type: graph.IntType, Value: int64(5)}})
	g.CreateEdge(a.ID, c.ID, "ROAD", map[string]graph.PropertyValue{"weight": {Type: graph.IntType, Value: int64(1)}})
	g.CreateEdge(c.ID, b.ID, "ROAD", map[string]graph.PropertyValue{"weight": {Type: graph.IntType, Value: int64(1)}})
	walker := traverse.New(g)
	weighted, err := walker.ShortestWeighted(a.ID, b.ID, traverse.Options{Direction: graph.Out})
	if err != nil {
		t.Fatal(err)
	}
	if weighted.Distance != 2 || len(weighted.Path) != 3 {
		t.Fatalf("unexpected path: %+v", weighted)
	}
}

func TestShortestPathsRejectMissingVertices(t *testing.T) {
	g := graph.New(storage.NewMemStore())
	vertex, err := g.CreateVertex(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	walker := traverse.New(g)
	for _, test := range []struct {
		name string
		run  func() error
	}{
		{
			name: "unweighted same missing vertex",
			run: func() error {
				_, err := walker.ShortestUnweighted(vertex.ID+1, vertex.ID+1, traverse.Options{Direction: graph.Out})
				return err
			},
		},
		{
			name: "weighted missing destination",
			run: func() error {
				_, err := walker.ShortestWeighted(vertex.ID, vertex.ID+1, traverse.Options{Direction: graph.Out})
				return err
			},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			if err := test.run(); err != graph.ErrVertexNotFound {
				t.Fatalf("error = %v, want %v", err, graph.ErrVertexNotFound)
			}
		})
	}
}
