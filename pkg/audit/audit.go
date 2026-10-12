package audit

import (
	"context"
	"database/sql/driver"
	"encoding/json"
	"errors"
	"strings"
	"time"

	"github.com/yuusufyan/go-common/pkg/logger"
	"gorm.io/gorm"
)

// TableName is the table used to persist audit trails.
const TableName = "audit_trails"

// Trail types.
const (
	TypeData = "data" // data changes recorded manually by the project via Recorder.Record
	TypeHTTP = "http" // requests captured by the Fiber middleware
)

// Actions for TypeData trails.
const (
	ActionCreate = "CREATE"
	ActionUpdate = "UPDATE"
	ActionDelete = "DELETE"
)

// Trail is a single audit trail entry.
type Trail struct {
	ID         string    `gorm:"column:id;type:uuid;primaryKey" json:"id"`
	Type       string    `gorm:"column:type;size:16;index" json:"type"`
	Action     string    `gorm:"column:action;size:16" json:"action"`
	Entity     string    `gorm:"column:entity;size:255;index:idx_audit_trails_entity" json:"entity"`
	EntityID   string    `gorm:"column:entity_id;size:255;index:idx_audit_trails_entity" json:"entity_id,omitempty"`
	UserID     string    `gorm:"column:user_id;size:255;index" json:"user_id,omitempty"`
	UserEmail  string    `gorm:"column:user_email;size:255" json:"user_email,omitempty"`
	RequestID  string    `gorm:"column:request_id;size:64;index" json:"request_id,omitempty"`
	TraceID    string    `gorm:"column:trace_id;size:64" json:"trace_id,omitempty"`
	Method     string    `gorm:"column:method;size:16" json:"method,omitempty"`
	Path       string    `gorm:"column:path;size:1024" json:"path,omitempty"`
	StatusCode int       `gorm:"column:status_code" json:"status_code,omitempty"`
	LatencyMs  int64     `gorm:"column:latency_ms" json:"latency_ms,omitempty"`
	IP         string    `gorm:"column:ip;size:64" json:"ip,omitempty"`
	UserAgent  string    `gorm:"column:user_agent;size:512" json:"user_agent,omitempty"`
	OldValues  JSONMap   `gorm:"column:old_values;type:jsonb" json:"old_values,omitempty"`
	NewValues  JSONMap   `gorm:"column:new_values;type:jsonb" json:"new_values,omitempty"`
	Metadata   JSONMap   `gorm:"column:metadata;type:jsonb" json:"metadata,omitempty"`
	CreatedAt  time.Time `gorm:"column:created_at;index" json:"created_at"`
}

func (Trail) TableName() string {
	return TableName
}

// JSONMap is a map stored as a jsonb column.
type JSONMap map[string]any

func (m JSONMap) Value() (driver.Value, error) {
	if m == nil {
		return nil, nil
	}
	b, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return string(b), nil
}

func (m *JSONMap) Scan(src any) error {
	switch v := src.(type) {
	case nil:
		*m = nil
		return nil
	case []byte:
		return json.Unmarshal(v, m)
	case string:
		return json.Unmarshal([]byte(v), m)
	default:
		return errors.New("audit: unsupported JSONMap source type")
	}
}

// Migrate creates or updates the audit_trails table.
func Migrate(db *gorm.DB) error {
	return db.AutoMigrate(&Trail{})
}

// Store persists audit trails.
type Store interface {
	Save(ctx context.Context, trails []*Trail) error
}

// GormStore persists audit trails with GORM.
type GormStore struct {
	db *gorm.DB
}

// NewGormStore returns a Store writing to the audit_trails table.
func NewGormStore(db *gorm.DB) *GormStore {
	return &GormStore{db: db}
}

func (s *GormStore) Save(ctx context.Context, trails []*Trail) error {
	return s.db.Session(&gorm.Session{NewDB: true, SkipHooks: true}).
		WithContext(ctx).
		CreateInBatches(trails, 100).Error
}

const maskedValue = "********"

// mask replaces values whose key contains a sensitive key (case-insensitive) in m,
// so "password_hash" is masked by "password".
func mask(m JSONMap, sensitiveKeys []string) JSONMap {
	if len(sensitiveKeys) == 0 {
		sensitiveKeys = logger.DefaultSensitiveKeys
	}
	for k := range m {
		lk := strings.ToLower(k)
		for _, s := range sensitiveKeys {
			if strings.Contains(lk, strings.ToLower(s)) {
				m[k] = maskedValue
				break
			}
		}
	}
	return m
}
