package analytics_test

import (
	"math"
	"testing"

	"gographdb/internal/analytics"
	"gographdb/internal/graph"
	"gographdb/internal/storage"
)

func TestBuild(t *testing.T) {
	g := graph.New(storage.NewMemStore())
	a, _ := g.CreateVertex(nil, nil)
	b, _ := g.CreateVertex(nil, nil)
	c, _ := g.CreateVertex(nil, nil)
	d, _ := g.CreateVertex(nil, nil)
	if _, err := g.CreateEdge(a.ID, b.ID, "LINK", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := g.CreateEdge(b.ID, a.ID, "LINK", nil); err != nil {
		t.Fatal(err)
	}
	if _, err := g.CreateEdge(c.ID, c.ID, "LINK", nil); err != nil {
		t.Fatal(err)
	}
	report := analytics.Build(g, 2)
	if report.Components != 3 || report.Isolated != 1 || report.SelfLoops != 1 {
		t.Fatalf("bad report: %+v", report)
	}
	if len(report.TopDegrees) != 2 || report.TopDegrees[0].VertexID != a.ID {
		t.Fatalf("bad degree ranking: %+v", report.TopDegrees)
	}
	if len(report.PageRank) != 2 || report.PageRank[0].Score < report.PageRank[1].Score {
		t.Fatalf("bad rank: %+v", report.PageRank)
	}
	if math.IsNaN(report.PageRank[0].Score) {
		t.Fatal("rank must be finite")
	}
	_ = d
}

func TestSchema(t *testing.T) {
	g := graph.New(storage.NewMemStore())
	a, err := g.CreateVertex([]string{"Person", "Employee"}, map[string]graph.PropertyValue{"name": {Type: graph.StringType, Value: "A"}, "age": {Type: graph.IntType, Value: int64(20)}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.CreateVertex(nil, map[string]graph.PropertyValue{"active": {Type: graph.BoolType, Value: true}}); err != nil {
		t.Fatal(err)
	}
	if _, err = g.CreateEdge(a.ID, a.ID, "KNOWS", map[string]graph.PropertyValue{"weight": {Type: graph.FloatType, Value: 1.5}}); err != nil {
		t.Fatal(err)
	}
	report := analytics.Schema(g)
	if report.Vertices != 2 || report.Edges != 1 || len(report.VertexLabels) != 3 || report.VertexLabels[0].Label != "Employee" {
		t.Fatalf("bad schema: %+v", report)
	}
	person := report.VertexLabels[1]
	if person.Label != "Person" || person.Count != 1 || len(person.Properties["age"]) != 1 || person.Properties["age"][0] != graph.IntType {
		t.Fatalf("bad person schema: %+v", person)
	}
	if report.EdgeLabels[0].Label != "KNOWS" || report.EdgeLabels[0].Properties["weight"][0] != graph.FloatType {
		t.Fatalf("bad edge schema: %+v", report.EdgeLabels)
	}
}

func TestQuality(t *testing.T) {
	g := graph.New(storage.NewMemStore())
	a, err := g.CreateVertex(nil, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := g.CreateVertex([]string{"Person"}, map[string]graph.PropertyValue{"name": {Type: graph.StringType, Value: "B"}})
	if err != nil {
		t.Fatal(err)
	}
	if _, err = g.CreateEdge(b.ID, b.ID, "KNOWS", nil); err != nil {
		t.Fatal(err)
	}
	report := analytics.Quality(g)
	if report.Healthy || report.Vertices != 2 || report.Edges != 1 || len(report.Issues) != 5 {
		t.Fatalf("bad quality report: %+v", report)
	}
	if report.Issues[0].Code != "unlabeled_vertices" || report.Issues[3].Count != 1 || a.ID == 0 {
		t.Fatalf("bad quality issues: %+v", report.Issues)
	}
}
