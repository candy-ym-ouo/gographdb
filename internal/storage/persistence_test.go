package storage_test

import (
	"gographdb/internal/graph"
	"gographdb/internal/storage"
	"path/filepath"
	"testing"
)

func TestSnapshotAndWAL(t *testing.T) {
	dir := t.TempDir()
	g := graph.New(storage.NewMemStore())
	v, _ := g.CreateVertex([]string{"Saved"}, nil)
	p := storage.NewPersistence(dir)
	if err := p.Save(g); err != nil {
		t.Fatal(err)
	}
	loaded := graph.New(storage.NewMemStore())
	if err := p.LoadInto(loaded); err != nil {
		t.Fatal(err)
	}
	if got, _ := loaded.Vertex(v.ID); !got.HasLabel("Saved") {
		t.Fatal("snapshot mismatch")
	}
	wal, err := storage.OpenWAL(filepath.Join(dir, "wal.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()
	next := graph.New(storage.NewMemStore())
	next.SetWAL(wal)
	created, _ := next.CreateVertex([]string{"WAL"}, nil)
	next.SetWAL(nil)
	recovered := graph.New(storage.NewMemStore())
	if err = wal.ReplayInto(recovered); err != nil {
		t.Fatal(err)
	}
	if got, _ := recovered.Vertex(created.ID); !got.HasLabel("WAL") {
		t.Fatal("WAL mismatch")
	}
}

func TestPersistenceStatus(t *testing.T) {
	dir := t.TempDir()
	p := storage.NewPersistence(dir)
	if status := p.Status(); status.Snapshot.Exists || status.Metadata.Exists || status.WAL.Exists {
		t.Fatalf("unexpected initial status: %+v", status)
	}
	g := graph.New(storage.NewMemStore())
	if err := p.Save(g); err != nil {
		t.Fatal(err)
	}
	wal, err := storage.OpenWAL(filepath.Join(dir, "wal.log"))
	if err != nil {
		t.Fatal(err)
	}
	defer wal.Close()
	status := p.Status()
	if !status.Snapshot.Exists || !status.Metadata.Exists || !status.WAL.Exists || status.Snapshot.Size == 0 {
		t.Fatalf("bad persistence status: %+v", status)
	}
}
