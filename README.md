# Go-Common Library

Library standar untuk pengembangan microservice berbasis **Go Fiber**. Menyediakan standarisasi Respon API, Messaging (RabbitMQ), Middleware, Health Check, Audit Trail, Logging, dan utilitas database.

Library ini **tidak terikat ke project tertentu**: setiap komponen punya default yang aman dan varian `...WithConfig` / struct `Config` sehingga bisa disesuaikan per project (nama service, timezone, CORS, header, timeout, dsb.).

---

## 📋 Daftar Isi
1. [Prasyarat](#-prasyarat)
2. [Instalasi](#-instalasi)
3. [Standarisasi Respon API](#1-standarisasi-respon-api-response)
4. [Messaging RabbitMQ](#2-messaging-rabbitmq-pkgrabbitmq)
5. [Database & Transaction Manager](#3-database--transaction-manager-pkgdatabase)
6. [Audit Trail](#4-audit-trail)
7. [Structured Logging](#5-structured-logging-pkglogger)
8. [Middleware](#6-middleware-pkgmiddleware)
9. [Error Handling & AppError](#7-error-handling--apperror-pkgapperror)
10. [Health Check, Shutdown & HTTP Client](#8-health-check-shutdown--http-client-pkgutils)
11. [Environment Variable](#9-environment-variable)
12. [Troubleshooting Intranet](#10-troubleshooting-intranet)

---

## 🏗 Prasyarat
*   **Go Version**: 1.25 atau lebih baru.
*   **Dependencies utama**: Fiber v2, GORM (Postgres), amqp091-go, go-redis v9, Logrus.

---

## 🛠 Instalasi

```bash
go get github.com/yuusufyan/go-common
```

---

## 1. Standarisasi Respon API (`/response`)

```go
import "github.com/yuusufyan/go-common/response"

// Respon sukses
return response.Success(c, 200, "Success", data)

// Respon error dari error object (AppError otomatis dipetakan)
return response.RespondWithError(c, err)

// Respon dengan pagination
return response.Paginate(c, "Success", users, total, page, limit)
```

Format envelope:
```json
{ "code": 200, "status": "success", "message": "Success", "data": {}, "meta": {} }
```

---

## 2. Messaging RabbitMQ (`/pkg/rabbitmq`)

Fitur:
*   **Auto-Reconnect** saat koneksi putus.
*   **Auto-DLQ**: setiap antrean `X` otomatis dibuatkan `X.dlx` dan `X.dlq`.
*   **Context Timeout** per pesan.

```go
// Default
mq, err := rabbitmq.NewRabbitMQClient(cfg.RabbitMQURL)

// Dengan konfigurasi (semua field opsional)
mq, err := rabbitmq.NewRabbitMQClientWithConfig(rabbitmq.Config{
    URL:            cfg.RabbitMQURL,
    Logger:         log,              // logger.Logger / *logrus.Logger
    ReconnectDelay: 5 * time.Second,  // default 5s
    HandlerTimeout: 5 * time.Minute,  // default 5m
    PrefetchCount:  10,               // default 1
    DLXSuffix:      ".dlx",           // default ".dlx"
    DLQSuffix:      ".dlq",           // default ".dlq"
})

mq.Publish(ctx, "order.created", payload)

mq.Consume("order.created", func(ctx context.Context, body []byte) error {
    return nil // return error => pesan dikirim ke DLQ
})
```

---

## 3. Database & Transaction Manager (`/pkg/database`)

### Koneksi PostgreSQL
```go
db, err := database.Connect(&database.DBConfig{
    Host:     cfg.DBHost,
    User:     cfg.DBUser,
    Password: cfg.DBPassword,
    DBName:   cfg.DBName,
    Port:     cfg.DBPort,

    // Opsional
    SSLMode:            "require",      // default "disable"
    TimeZone:           "Asia/Jakarta", // kosong = default server DB
    DSN:                "",             // jika diisi, mengabaikan field koneksi di atas
    MaxIdleConns:       10,             // default 10
    MaxOpenConns:       100,            // default 100
    ConnMaxLifetime:    60,             // menit, default 60
    SlowQueryThreshold: 500 * time.Millisecond, // default 200ms
    DisableAuditPlugin: false,
}, log, isProd)
```

### Redis
```go
rdb := database.InitRedis(&database.RedisConfig{Host: "localhost", Port: 6379}, log)
// rdb == nil jika Redis tidak tersedia (caching dinonaktifkan)
```

### Transaction Manager
```go
txManager := database.NewTxManager(db)

err := txManager.Do(ctx, func(ctx context.Context) error {
    if err := repo1.Create(ctx, data); err != nil { return err }
    return repo2.Update(ctx, data)
})

// Di repository: ambil tx dari context (fallback ke db global)
func (r *repo) Create(ctx context.Context, data *Entity) error {
    return database.GetDBFromContext(ctx, r.db).Create(data).Error
}
```

---

## 4. Audit Trail

Embed `AuditModel` ke entitas:
```go
type Product struct {
    ID uuid.UUID `gorm:"primaryKey"`
    database.AuditModel // CreatedAt, UpdatedAt, DeletedAt, CreatedBy, UpdatedBy
}
```

`AuditPlugin` otomatis didaftarkan oleh `database.Connect` (nonaktifkan dengan `DisableAuditPlugin: true`). Plugin mengisi `CreatedBy`/`UpdatedBy` dari user di context:
```go
ctx = context.WithValue(ctx, database.UserContextKey, user)
// user: struct dengan field ID/UserID (string/uuid) atau implementasi GetID() string
```

---

## 5. Structured Logging (`/pkg/logger`)

```go
// Sederhana
log := logger.New(isProd) // prod = JSON + level info, dev = text + level debug

// Dengan konfigurasi
log := logger.NewWithConfig(logger.Config{
    JSON:          true,
    Level:         "info",
    Fields:        logrus.Fields{"service": "order-service", "env": "staging"},
    SensitiveKeys: append(logger.DefaultSensitiveKeys, "pin", "otp"),
})

// Trace ID & Request ID otomatis muncul di log
log.WithCtx(ctx).Info("Processing data...")
```

---

## 6. Middleware (`/pkg/middleware`)

### Setup cepat
```go
import mw "github.com/yuusufyan/go-common/pkg/middleware/fiber"

app := fiber.New(fiber.Config{ErrorHandler: utils.NewErrorHandler(log)})

// Default: Telemetry, Helmet, CORS (AllowOrigins "*"), Recover, Logger
mw.InstallCommonMiddleware(app, log)

// Atau dengan konfigurasi
mw.InstallCommonMiddlewareWithConfig(app, log, mw.CommonConfig{
    Security: mw.SecurityConfig{
        AllowOrigins:     "https://app.example.com",
        AllowCredentials: true,
    },
    DisableSecurity: false, // true jika CORS/helmet ditangani API gateway
    DisableLogger:   false,
})
```

Middleware juga bisa dipasang satu per satu: `mw.Telemetry()`, `mw.UseSecurity(app, cfg)` / `mw.Helmet(cfg)` + `mw.CORS(cfg)`, `mw.Recover(log)`, `mw.Logger(log)`.

### Identity (header dari gateway/BFF)
```go
app.Use(mw.Identity()) // X-User-ID, X-User-Email, X-User-Role, X-User-Permissions

// Nama header kustom
app.Use(mw.IdentityWithConfig(mw.IdentityConfig{
    HeaderUserID:   "X-Auth-User",
    HeaderUserRole: "X-Auth-Roles",
    Separator:      ",",
}))

app.Get("/admin", mw.RequireRole("admin"), handler)
app.Post("/orders", mw.RequirePermission("order:create"), handler)

identity := mw.GetUserIdentity(c)
```

### Internal Secret (service-to-service)
```go
// Baca dari env INTERNAL_SERVICE_SECRET
app.Use(mw.InternalSecretFromEnv(""))

// Atau dengan konfigurasi
app.Use(mw.InternalSecretWithConfig(mw.InternalSecretConfig{
    EnvKey:  "ORDER_SERVICE_SECRET",          // atau isi Secret langsung
    Header:  "X-Internal-Secret",
    Message: "Forbidden",
    Next:    func(c *fiber.Ctx) bool { return c.Path() == "/health" },
}))
```
Jika secret kosong, semua request ditolak (fail-closed).

### Swagger
```go
swagger.SetupSwagger(app)                              // GET /swagger/*
swagger.SetupSwaggerWithConfig(app, "/docs/*")         // path kustom
swagger.SetupSwaggerWithConfig(app, "/docs/*", fiberSwagger.Config{Title: "Order API"})
```

---

## 7. Error Handling & AppError (`/pkg/apperror`)

```go
return apperror.NotFound("Product not found")
return apperror.New(422, "Unprocessable")

// Validasi struct
if err := apperror.Validate(validate, req); err != nil { return err }

// Kustomisasi / terjemahan pesan validasi (panggil saat startup)
apperror.SetValidationMessages(map[string]string{
    "required": "Field %s wajib diisi",
    "min":      "Field %s minimal %s karakter",
})
```

---

## 8. Health Check, Shutdown & HTTP Client (`/pkg/utils`)

### Health Check
```go
// DB (critical) + Redis (non-critical)
utils.RegisterHealthCheck(app, db, rdb)

// Check kustom
app.Get("/health", utils.NewHealthHandlerWithConfig(utils.HealthConfig{
    Timeout: 3 * time.Second,
    Checks: []utils.HealthCheck{
        utils.DBHealthCheck(db),
        utils.RedisHealthCheck(rdb),
        {Name: "payment-api", Critical: false, Check: func(ctx context.Context) error { return pingPayment(ctx) }},
    },
}))
```

### Graceful Shutdown
```go
sh := utils.NewShutdownHelper(log).WithTimeout(15 * time.Second)
sh.Wait() // default SIGINT & SIGTERM

sh.Graceful(map[string]func(ctx context.Context) error{
    "HTTP":     func(ctx context.Context) error { return app.ShutdownWithContext(ctx) },
    "RabbitMQ": func(ctx context.Context) error { return mq.Close() },
})
```

### HTTP Client dengan Trace Propagation
```go
client := utils.NewTracingClientWithConfig(utils.TracingClientConfig{
    Timeout:    10 * time.Second,
    MaxRetries: 3,                      // retry untuk network error / 5xx
    RetryDelay: 200 * time.Millisecond, // backoff linear
})
resp, err := client.Get(ctx, "http://inventory-service/items")
```

### JWT
```go
claims, err := utils.VerifyToken(tokenStr, secret, "access") // "" untuk skip cek claim "type"
```

---

## 9. Environment Variable

Library ini **tidak membaca konfigurasi secara otomatis**; semua nilai diteruskan dari aplikasi melalui struct config. Satu-satunya env yang dibaca langsung:

| Key | Digunakan oleh |
| :--- | :--- |
| `INTERNAL_SERVICE_SECRET` | `InternalSecretFromEnv` / `InternalSecretWithConfig` (default `EnvKey`) |

---

## 10. Troubleshooting Intranet
Jika bekerja di lingkungan intranet tanpa akses internet dan VS Code menampilkan error merah pada library ini, tambahkan di `.vscode/settings.json`:
```json
{
    "go.toolsEnvVars": {
        "GONOSUMDB": "*",
        "GOPROXY": "off"
    },
    "gopls": {
        "build.env": {
            "GONOSUMDB": "*",
            "GOPROXY": "off"
        }
    }
}
```

---

## 🛡 Best Practices
*   **Traceability**: `Telemetry` otomatis menyuntikkan `X-Request-ID` dan `X-Trace-ID`. Gunakan `TracingClient` agar ID ikut diteruskan antar service.
*   **Timeout**: Jangan gunakan `context.Background()` di service logic; selalu teruskan `ctx` dari handler.
*   **CORS**: Batasi `AllowOrigins` di production.

---

## 👨‍💻 Kontribusi
Pastikan kode lulus `go build ./... && go vet ./... && go test ./...` sebelum membuat Pull Request.
