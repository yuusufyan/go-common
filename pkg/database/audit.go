package database

import (
	"context"
	"reflect"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// AuditModel can be embedded in entities to provide automatic auditing
type AuditModel struct {
	CreatedAt time.Time      `gorm:"column:created_at;autoCreateTime" json:"created_at"`
	UpdatedAt time.Time      `gorm:"column:updated_at;autoUpdateTime" json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"column:deleted_at;index" json:"-"`
	CreatedBy string         `gorm:"column:created_by;type:uuid" json:"created_by"`
	UpdatedBy string         `gorm:"column:updated_by;type:uuid" json:"updated_by"`
}

// UserFromContext is an interface to extract user info from context
// Services should implement this or we use a standard key
type UserFromContext interface {
	GetID() string
}

// AuditPlugin is a GORM plugin that handles automatic auditing
type AuditPlugin struct{}

func NewAuditPlugin() *AuditPlugin {
	return &AuditPlugin{}
}

func (p *AuditPlugin) Name() string {
	return "audit_plugin"
}

func (p *AuditPlugin) Initialize(db *gorm.DB) error {
	// Register callbacks
	db.Callback().Create().Before("gorm:create").Register("audit:before_create", p.beforeCreate)
	db.Callback().Update().Before("gorm:update").Register("audit:before_update", p.beforeUpdate)
	return nil
}

func (p *AuditPlugin) beforeCreate(db *gorm.DB) {
	if db.Statement.Schema != nil {
		userID := getUserID(db.Statement.Context)
		if userID != "" {
			// Check if field exists in schema
			if f := db.Statement.Schema.LookUpField("CreatedBy"); f != nil {
				db.Statement.SetColumn("CreatedBy", userID)
			}
			if f := db.Statement.Schema.LookUpField("UpdatedBy"); f != nil {
				db.Statement.SetColumn("UpdatedBy", userID)
			}
		}
	}
}

func (p *AuditPlugin) beforeUpdate(db *gorm.DB) {
	if db.Statement.Schema != nil {
		userID := getUserID(db.Statement.Context)
		if userID != "" {
			if f := db.Statement.Schema.LookUpField("UpdatedBy"); f != nil {
				db.Statement.SetColumn("UpdatedBy", userID)
			}
		}
	}
}

func getUserID(ctx context.Context) string {
	if ctx == nil {
		return ""
	}
	userVal := ctx.Value(UserContextKey)
	if userVal == nil {
		return ""
	}

	// Try to get ID field via reflection (works with any struct having ID or UserID)
	v := reflect.Indirect(reflect.ValueOf(userVal))
	if v.Kind() == reflect.Struct {
		// Try "ID" field
		f := v.FieldByName("ID")
		if f.IsValid() {
			if id, ok := f.Interface().(uuid.UUID); ok {
				return id.String()
			}
			if id, ok := f.Interface().(string); ok {
				return id
			}
		}
		// Try "UserID" field
		f = v.FieldByName("UserID")
		if f.IsValid() {
			if id, ok := f.Interface().(uuid.UUID); ok {
				return id.String()
			}
			if id, ok := f.Interface().(string); ok {
				return id
			}
		}
	}

	// Fallback to GetID interface
	if u, ok := userVal.(interface{ GetID() string }); ok {
		return u.GetID()
	}
	
	return ""
}
