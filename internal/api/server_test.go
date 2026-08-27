package api_test

import (
	"encoding/json"
	"gographdb/internal/api"
	"gographdb/internal/graph"
	"gographdb/internal/index"
	"gographdb/internal/storage"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestAPI(t *testing.T) {
	g := graph.New(storage.NewMemStore())
	m := index.NewManager()
	g.AddListener(m)
	server := httptest.NewServer(api.NewServer(g, m, storage.NewPersistence(t.TempDir()), nil).Handler())
	defer server.Close()
	response, err := http.Post(server.URL+"/api/graph/vertices", "application/json", strings.NewReader(`{"labels":["Person"],"properties":{"name":"A"}}`))
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusCreated {
		t.Fatalf("status %d", response.StatusCode)
	}
	health, err := http.Get(server.URL + "/api/health")
	if err != nil {
		t.Fatal(err)
	}
	health.Body.Close()
	if health.StatusCode != http.StatusOK {
		t.Fatal(health.Status)
	}
	for _, path := range []string{"/api/graph/analytics?limit=5", "/api/graph/index/audit", "/api/graph/schema", "/api/graph/quality", "/api/persistence/status"} {
		response, err := http.Get(server.URL + path)
		if err != nil {
			t.Fatal(err)
		}
		response.Body.Close()
		if response.StatusCode != http.StatusOK {
			t.Fatalf("%s: %s", path, response.Status)
		}
	}
}

func TestExport(t *testing.T) {
	g := graph.New(storage.NewMemStore())
	vertex, err := g.CreateVertex([]string{"Person"}, nil)
	if err != nil {
		t.Fatal(err)
	}
	m := index.NewManager()
	g.AddListener(m)
	server := httptest.NewServer(api.NewServer(g, m, storage.NewPersistence(t.TempDir()), nil).Handler())
	defer server.Close()
	response, err := http.Get(server.URL + "/api/persistence/export")
	if err != nil {
		t.Fatal(err)
	}
	defer response.Body.Close()
	if response.StatusCode != http.StatusOK || response.Header.Get("Content-Disposition") == "" {
		t.Fatalf("bad export response: %s", response.Status)
	}
	var snapshot storage.Snapshot
	if err := json.NewDecoder(response.Body).Decode(&snapshot); err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Vertices) != 1 || snapshot.Vertices[0].ID != vertex.ID {
		t.Fatalf("bad export: %+v", snapshot)
	}
}
