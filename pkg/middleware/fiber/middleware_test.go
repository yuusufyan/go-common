package fiber

import (
	"io"
	"net/http/httptest"
	"testing"

	gofiber "github.com/gofiber/fiber/v2"
	"github.com/yuusufyan/go-common/pkg/logger"
)

func TestCommonMiddlewareRunsHandlerOnce(t *testing.T) {
	app := gofiber.New()
	InstallCommonMiddleware(app, logger.NewWithConfig(logger.Config{Output: io.Discard}))

	calls := 0
	app.Get("/ping", func(c *gofiber.Ctx) error {
		calls++
		return c.SendString("pong")
	})

	resp, err := app.Test(httptest.NewRequest("GET", "/ping", nil))
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != 200 || calls != 1 {
		t.Fatalf("status=%d calls=%d, want 200 and 1", resp.StatusCode, calls)
	}
	if resp.Header.Get("X-Content-Type-Options") == "" || resp.Header.Get(HeaderTraceID) == "" {
		t.Fatal("expected helmet and trace headers")
	}
}

func TestCORSAllowOrigins(t *testing.T) {
	app := gofiber.New()
	UseSecurity(app, SecurityConfig{AllowOrigins: "https://app.example.com"})
	app.Get("/", func(c *gofiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set("Origin", "https://app.example.com")
	resp, _ := app.Test(req)
	if got := resp.Header.Get("Access-Control-Allow-Origin"); got != "https://app.example.com" {
		t.Fatalf("Access-Control-Allow-Origin=%q", got)
	}
}

func TestInternalSecretWithConfig(t *testing.T) {
	app := gofiber.New()
	app.Use(InternalSecretWithConfig(InternalSecretConfig{Secret: "s3cret", Header: "X-Key"}))
	app.Get("/", func(c *gofiber.Ctx) error { return c.SendStatus(200) })

	cases := map[string]int{"": 403, "wrong": 403, "s3cret": 200}
	for val, want := range cases {
		req := httptest.NewRequest("GET", "/", nil)
		if val != "" {
			req.Header.Set("X-Key", val)
		}
		resp, _ := app.Test(req)
		if resp.StatusCode != want {
			t.Errorf("secret %q: status=%d want %d", val, resp.StatusCode, want)
		}
	}
}

func TestInternalSecretEmptyRejectsAll(t *testing.T) {
	t.Setenv(DefaultInternalSecretEnv, "")
	app := gofiber.New()
	app.Use(InternalSecret(""))
	app.Get("/", func(c *gofiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/", nil)
	req.Header.Set(HeaderInternalSecret, "anything")
	resp, _ := app.Test(req)
	if resp.StatusCode != 403 {
		t.Fatalf("status=%d want 403", resp.StatusCode)
	}
}

func TestIdentityWithConfigAndRequireRole(t *testing.T) {
	app := gofiber.New()
	app.Use(IdentityWithConfig(IdentityConfig{HeaderUserID: "X-Uid", HeaderUserRole: "X-Roles"}))
	app.Get("/admin", RequireRole("admin"), func(c *gofiber.Ctx) error { return c.SendStatus(200) })

	req := httptest.NewRequest("GET", "/admin", nil)
	req.Header.Set("X-Uid", "u1")
	req.Header.Set("X-Roles", "user, admin")
	resp, _ := app.Test(req)
	if resp.StatusCode != 200 {
		t.Fatalf("status=%d want 200", resp.StatusCode)
	}

	resp, _ = app.Test(httptest.NewRequest("GET", "/admin", nil))
	if resp.StatusCode != 401 {
		t.Fatalf("anonymous status=%d want 401", resp.StatusCode)
	}
}
