# Go-Common Library

Library standar untuk pengembangan Microservices di project Go Fiber. Library ini menyediakan standarisasi untuk Respon API, Messaging (RabbitMQ), Middleware, Health Check, Audit Trail, dan utilitas database lainnya yang mendukung framework **Fiber**.

---

## 📋 Daftar Isi
1. [Prasyarat](#-prasyarat)
2. [Instalasi](#-instalasi)
3. [Standarisasi Respon API](#1-standarisasi-respon-api)
4. [Advanced Messaging (RabbitMQ HA)](#2-advanced-messaging-rabbitmq-ha)
5. [Database & Transaction Manager](#3-database--transaction-manager)
6. [Audit Trail & Entity Model](#4-audit-trail--entity-standardization)
7. [Structured Logging](#5-structured-logging-with-context)
8. [Middleware & Bootstrap](#6-middleware--bootstrap)
9. [Error Handling & AppError](#7-error-handling--apperror)
10. [Health Check & Shutdown Helper](#8-health-check--shutdown-helper)
11. [Konfigurasi Environment](#9-konfigurasi-environment)
12. [Troubleshooting Intranet](#10-troubleshooting-intranet)

---

## 🏗 Prasyarat
*   **Go Version**: 1.24.0 atau lebih baru.
*   **Dependencies**: GORM (v1.25+), amqp091-go, Viper, Logrus.

---

## 🛠 Instalasi

```bash
# Inisialisasi modul jika belum ada
go mod init <nama_modul>

# Tambahkan shared library
go get github.com/yuusufyan/go-common
```

---

## 1. Standarisasi Respon API (`/response`)

### Implementasi Fiber
```go
import "github.com/yuusufyan/go-common/response"

// Respon Sukses
return response.Success(c, 200, "Success", data)

// Respon dengan Pagination
return response.RespondWithPagination(c, "Success", &response.Pagination{
    Data: users,
    Total: 100,
    Page: 1,
    Limit: 10,
})
```

---

## 2. Advanced Messaging (`/pkg/rabbitmq`)

Client RabbitMQ HA yang mendukung ketahanan tinggi.

### Fitur Detail:
*   **Auto-Reconnect**: Mencoba menyambung kembali setiap 5 detik jika koneksi putus.
*   **Auto-DLQ**: Setiap antrean `X` akan otomatis dibuatkan `X.dlx` dan `X.dlq`.
*   **Context Timeout**: Batas waktu proses per-pesan (default 5 menit).

### Contoh Penggunaan:
```go
mq, _ := rabbitmq.NewRabbitMQClient(cfg.RabbitMQURL)

// Mengirim Pesan
mq.Publish(ctx, constant.MyQueue, payload)

// Menerima Pesan (Subscriber)
mq.Consume(constant.MyQueue, func(ctx context.Context, body []byte) error {
    // Gunakan ctx untuk operasi DB/Repo agar timeout terpropagasi
    return nil 
})
```

---

## 3. Database & Transaction Manager (`/pkg/database`)

### Inisialisasi Koneksi
```go
dbConfig := &database.DBConfig{
    Host: cfg.DBHost,
    User: cfg.DBUser,
    Password: cfg.DBPassword,
    DBName: cfg.DBName,
    Port: cfg.DBPort,
}
db, err := database.Connect(dbConfig, logger, isProd)
```

### Transaction Manager
```go
txManager := database.NewTxManager(db)

err := txManager.WithTransaction(ctx, func(tx *gorm.DB) error {
    if err := repo1.Create(tx, data); err != nil { return err }
    return repo2.Update(tx, data)
})
```

---

## 4. Audit Trail & Entity Standardization

Embed `AuditModel` ke struct entitas Anda:
```go
type Loan struct {
    ID uint `gorm:"primaryKey"`
    database.AuditModel // Menambahkan CreatedAt, UpdatedAt, CreatedBy, UpdatedBy
}

// Aktifkan plugin saat startup GORM
db.Use(database.NewAuditPlugin())
```

---

## 5. Structured Logging with Context (`/pkg/logger`)

MencatLog yang membawa Trace ID dari request API sangat krusial untuk debugging.

```go
// Inisialisasi
log := logger.New(isProd)

// Penggunaan dengan Context (Trace ID akan otomatis muncul di log)
logger.WithCtx(ctx, log).Info("Processing data...")
```

---

## 6. Middleware & Bootstrap (`/pkg/middleware`)

### Fiber Setup (main.go)
```go
import commonMiddleware "github.com/yuusufyan/go-common/pkg/middleware/fiber"

app := fiber.New()
commonMiddleware.InstallCommonMiddleware(app, log)
```

---

## 7. Error Handling & AppError (`/pkg/apperror`)

### Membuat Error Kustom
```go
import "github.com/yuusufyan/go-common/pkg/apperror"

return apperror.New(400, "Bad Request")
return apperror.New(404, "Not Found")
```

---

## 8. Health Check & Shutdown Helper (`/pkg/utils`)

### Graceful Shutdown
```go
sh := utils.NewShutdownHelper(log)
sh.Wait() 

sh.Graceful(map[string]func(ctx context.Context) error{
    "Database": func(ctx context.Context) error { return sqlDB.Close() },
    "RabbitMQ": func(ctx context.Context) error { return mq.Close() },
})
```

---

## 9. Konfigurasi Environment
Library ini secara otomatis membaca kunci-kunci berikut melalui Viper:

| Key | Description |
| :--- | :--- |
| `APP_ENV` | Mode aplikasi (`prod` atau `dev`) |
| `DB_HOST` | Host database PostgreSQL/MySQL |
| `DB_USER` | Username database |
| `RABBITMQ_URL` | URL koneksi RabbitMQ (`amqp://...`) |
| `JWT_SECRET` | Secret key untuk verifikasi token |

---

## 10. Troubleshooting Intranet
Jika Anda bekerja di lingkungan intranet tanpa akses internet dan VS Code menampilkan error merah pada library ini:
1.  Buka `.vscode/settings.json`.
2.  Tambahkan konfigurasi berikut:
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
*   **Traceability**: Library otomatis menyuntikkan `X-Trace-Id`. Gunakan ID ini saat melacak log antar service.
*   **Timeout**: Jangan gunakan `context.Background()` di dalam service logic; selalu teruskan `ctx` dari handler.

---

## 👨‍💻 Kontribusi
Pastikan kode lulus build: `go build ./...` sebelum melakukan Pull Request.
