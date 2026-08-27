package index_test

import (
	"gographdb/internal/graph"
	"gographdb/internal/index"
	"gographdb/internal/storage"
	"testing"
)

func TestIndexes(t *testing.T) {
	g := graph.New(storage.NewMemStore())
	m := index.NewManager()
	g.AddListener(m)
	age := graph.PropertyValue{Type: graph.IntType, Value: int64(30)}
	v, err := g.CreateVertex([]string{"Person"}, map[string]graph.PropertyValue{"age": age})
	if err != nil {
		t.Fatal(err)
	}
	if got := m.Labels.Lookup("Person"); len(got) != 1 || got[0] != v.ID {
		t.Fatalf("bad label index: %v", got)
	}
	if got := m.Ranges.Range("age", &age, nil, true, true); len(got) != 1 {
		t.Fatalf("bad range: %v", got)
	}
	if report := m.Audit(g); !report.Healthy {
		t.Fatalf("expected healthy indexes: %+v", report)
	}
	m.Labels.Clear()
	if report := m.Audit(g); report.Healthy || len(report.Problems) == 0 {
		t.Fatalf("expected audit to find a missing entry: %+v", report)
	}
}
