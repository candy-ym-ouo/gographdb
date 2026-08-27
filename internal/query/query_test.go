package query_test

import (
	"gographdb/internal/graph"
	"gographdb/internal/index"
	"gographdb/internal/query"
	"gographdb/internal/storage"
	"testing"
)

func TestCreateAndMatch(t *testing.T) {
	g := graph.New(storage.NewMemStore())
	m := index.NewManager()
	g.AddListener(m)
	e := query.NewExecutor(g, m)
	if _, err := e.Execute(`CREATE (n:Person {name:"张三", age:30})`); err != nil {
		t.Fatal(err)
	}
	result, err := e.Execute(`MATCH (n:Person) WHERE n.age >= 18 RETURN n.id, n.name LIMIT 10`)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Rows) != 1 || result.Rows[0]["n.name"] != "张三" {
		t.Fatalf("unexpected result: %#v", result)
	}
}
