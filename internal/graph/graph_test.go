package graph_test

import (
	"gographdb/internal/graph"
	"gographdb/internal/storage"
	"testing"
)

func TestCRUDAndCascade(t *testing.T) {
	g := graph.New(storage.NewMemStore())
	a, err := g.CreateVertex([]string{"Person"}, map[string]graph.PropertyValue{"name": {Type: graph.StringType, Value: "A"}})
	if err != nil {
		t.Fatal(err)
	}
	b, _ := g.CreateVertex([]string{"Person"}, nil)
	edge, err := g.CreateEdge(a.ID, b.ID, "KNOWS", nil)
	if err != nil {
		t.Fatal(err)
	}
	if edge.From != a.ID {
		t.Fatal("wrong edge")
	}
	if err = g.DeleteVertex(a.ID); err != nil {
		t.Fatal(err)
	}
	if _, err = g.Edge(edge.ID); err != graph.ErrEdgeNotFound {
		t.Fatalf("edge was not cascaded: %v", err)
	}
}
