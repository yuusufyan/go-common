package audit

import (
	"bytes"
	"context"
	"encoding/json"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/yuusufyan/go-common/pkg/database"
	"github.com/yuusufyan/go-common/pkg/logger"
)

type memStore struct {
	mu      sync.Mutex
	batches [][]*Trail
}

func (s *memStore) Save(_ context.Context, trails []*Trail) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.batches = append(s.batches, append([]*Trail(nil), trails...))
	return nil
}

func (s *memStore) all() []*Trail {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*Trail
	for _, b := range s.batches {
		out = append(out, b...)
	}
	return out
}

type testUser struct{ ID, Email string }

func (u testUser) GetEmail() string { return u.Email }

func TestRecorderFillsContextMasksAndFlushesOnClose(t *testing.T) {
	store := &memStore{}
	var logs bytes.Buffer
	rec := NewRecorder(Config{
		Store:         store,
		Logger:        logger.NewWithConfig(logger.Config{JSON: true, Output: &logs}),
		FlushInterval: time.Hour, // only Close flushes
	})

	ctx := context.WithValue(context.Background(), logger.RequestIDKey, "req-1")
	ctx = context.WithValue(ctx, logger.TraceIDKey, "trace-1")
	ctx = context.WithValue(ctx, database.UserContextKey, testUser{ID: "u1", Email: "u1@example.com"})

	rec.Record(ctx, &Trail{
		Type:      TypeData,
		Action:    ActionUpdate,
		Entity:    "users",
		NewValues: JSONMap{"name": "budi", "Password_Hash": "x", "access_token": "y"},
	})

	if err := rec.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	got := store.all()
	if len(got) != 1 {
		t.Fatalf("saved %d trails, want 1", len(got))
	}
	tr := got[0]
	if tr.ID == "" || tr.CreatedAt.IsZero() {
		t.Fatal("expected ID and CreatedAt to be set")
	}
	if tr.RequestID != "req-1" || tr.TraceID != "trace-1" || tr.UserID != "u1" || tr.UserEmail != "u1@example.com" {
		t.Fatalf("context not applied: %+v", tr)
	}
	if tr.NewValues["Password_Hash"] != maskedValue || tr.NewValues["access_token"] != maskedValue || tr.NewValues["name"] != "budi" {
		t.Fatalf("masking wrong: %v", tr.NewValues)
	}

	var entry map[string]any
	if err := json.Unmarshal(logs.Bytes(), &entry); err != nil {
		t.Fatalf("log is not JSON: %v (%s)", err, logs.String())
	}
	if entry["audit_type"] != TypeData || entry["request_id"] != "req-1" || entry["user_id"] != "u1" {
		t.Fatalf("unexpected log entry: %v", entry)
	}
}

func TestRecorderBatchesBySize(t *testing.T) {
	store := &memStore{}
	rec := NewRecorder(Config{Store: store, BatchSize: 2, FlushInterval: time.Hour})
	for i := 0; i < 5; i++ {
		rec.Record(context.Background(), &Trail{Type: TypeHTTP})
	}
	if err := rec.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	store.mu.Lock()
	defer store.mu.Unlock()
	if len(store.batches) != 3 || len(store.batches[0]) != 2 || len(store.batches[2]) != 1 {
		t.Fatalf("unexpected batches: %d", len(store.batches))
	}
}

func TestRecorderRecordAfterCloseIsNoop(t *testing.T) {
	store := &memStore{}
	rec := NewRecorder(Config{Store: store})
	_ = rec.Close(context.Background())
	_ = rec.Close(context.Background()) // idempotent

	rec.Record(context.Background(), &Trail{Type: TypeHTTP}) // must not panic
	if n := len(store.all()); n != 0 {
		t.Fatalf("saved %d trails after close", n)
	}
}

func TestRecorderDropsWhenQueueFull(t *testing.T) {
	block := make(chan struct{})
	store := &blockingStore{block: block}
	var logs bytes.Buffer
	rec := NewRecorder(Config{
		Store:     store,
		Logger:    logger.NewWithConfig(logger.Config{JSON: true, Output: &logs}),
		BatchSize: 1, BufferSize: 1,
	})

	// 1st is picked up by the worker (blocked in Save), 2nd fills the queue, 3rd is dropped.
	rec.Record(context.Background(), &Trail{})
	time.Sleep(50 * time.Millisecond)
	rec.Record(context.Background(), &Trail{})
	rec.Record(context.Background(), &Trail{})
	close(block)
	_ = rec.Close(context.Background())

	if !strings.Contains(logs.String(), "queue full") {
		t.Fatal("expected a queue full warning")
	}
}

type blockingStore struct{ block chan struct{} }

func (s *blockingStore) Save(context.Context, []*Trail) error {
	<-s.block
	return nil
}
