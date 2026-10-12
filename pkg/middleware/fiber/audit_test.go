package fiber

import (
	"context"
	"net/http/httptest"
	"sync"
	"testing"

	gofiber "github.com/gofiber/fiber/v3"
	"github.com/yuusufyan/go-common/pkg/apperror"
	"github.com/yuusufyan/go-common/pkg/audit"
	"github.com/yuusufyan/go-common/pkg/database"
)

type memStore struct {
	mu     sync.Mutex
	trails []*audit.Trail
}

func (s *memStore) Save(_ context.Context, trails []*audit.Trail) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trails = append(s.trails, trails...)
	return nil
}

func TestAuditTrailRecordsMutatingRequests(t *testing.T) {
	store := &memStore{}
	rec := audit.NewRecorder(audit.Config{Store: store})

	app := gofiber.New()
	app.Use(Telemetry(), Identity(), AuditTrail(rec))
	app.Get("/orders", func(c gofiber.Ctx) error { return c.SendStatus(200) })
	app.Post("/orders/:id", func(c gofiber.Ctx) error { return apperror.New(422, "invalid") })

	if _, err := app.Test(httptest.NewRequest("GET", "/orders", nil)); err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest("POST", "/orders/42?x=1", nil)
	req.Header.Set(HeaderUserID, "u1")
	req.Header.Set(HeaderUserEmail, "u1@example.com")
	req.Header.Set(gofiber.HeaderXRequestID, "req-1")
	if _, err := app.Test(req); err != nil {
		t.Fatal(err)
	}
	if err := rec.Close(context.Background()); err != nil {
		t.Fatal(err)
	}

	if len(store.trails) != 1 {
		t.Fatalf("got %d trails, want 1 (GET must be skipped)", len(store.trails))
	}
	tr := store.trails[0]
	if tr.Type != audit.TypeHTTP || tr.Method != "POST" || tr.Entity != "/orders/:id" || tr.Path != "/orders/42?x=1" {
		t.Fatalf("unexpected trail: %+v", tr)
	}
	if tr.StatusCode != 422 || tr.UserID != "u1" || tr.UserEmail != "u1@example.com" || tr.RequestID != "req-1" {
		t.Fatalf("unexpected trail: %+v", tr)
	}
}

func TestIdentityStoresUserInContext(t *testing.T) {
	app := gofiber.New()
	app.Use(Identity())

	var got string
	app.Get("/", func(c gofiber.Ctx) error {
		got = database.UserIDFromContext(c.Context())
		return c.SendStatus(200)
	})

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(HeaderUserID, "u1")
	if _, err := app.Test(req); err != nil {
		t.Fatal(err)
	}
	if got != "u1" {
		t.Fatalf("UserIDFromContext=%q want u1", got)
	}
}
