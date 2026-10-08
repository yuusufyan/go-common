package fiber

import (
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/yuusufyan/go-common/response"
)

const (
	UserIdentityKey = "user_identity"

	HeaderUserID          = "X-User-ID"
	HeaderUserEmail       = "X-User-Email"
	HeaderUserRole        = "X-User-Role"
	HeaderUserPermissions = "X-User-Permissions"
)

// UserIdentity represents the user information forwarded by an upstream gateway/identity service
type UserIdentity struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// GetID implements the interface used by the database audit plugin.
func (u *UserIdentity) GetID() string {
	return u.ID
}

// IdentityConfig configures the header names used by the Identity middleware.
// Empty fields fall back to the defaults (X-User-ID, X-User-Email, X-User-Role, X-User-Permissions, ",").
type IdentityConfig struct {
	HeaderUserID          string
	HeaderUserEmail       string
	HeaderUserRole        string
	HeaderUserPermissions string
	// Separator splits multi-value headers (roles & permissions).
	Separator string
}

// Identity extracts user information from trusted headers injected by the upstream gateway.
func Identity() fiber.Handler {
	return IdentityWithConfig(IdentityConfig{})
}

// IdentityWithConfig is the configurable version of Identity.
func IdentityWithConfig(cfg IdentityConfig) fiber.Handler {
	if cfg.HeaderUserID == "" {
		cfg.HeaderUserID = HeaderUserID
	}
	if cfg.HeaderUserEmail == "" {
		cfg.HeaderUserEmail = HeaderUserEmail
	}
	if cfg.HeaderUserRole == "" {
		cfg.HeaderUserRole = HeaderUserRole
	}
	if cfg.HeaderUserPermissions == "" {
		cfg.HeaderUserPermissions = HeaderUserPermissions
	}
	if cfg.Separator == "" {
		cfg.Separator = ","
	}

	return func(c *fiber.Ctx) error {
		userID := c.Get(cfg.HeaderUserID)

		// If no user ID is present, we assume it's an unauthenticated internal request
		if userID == "" {
			return c.Next()
		}

		identity := &UserIdentity{
			ID:          userID,
			Email:       c.Get(cfg.HeaderUserEmail),
			Roles:       splitHeader(c.Get(cfg.HeaderUserRole), cfg.Separator),
			Permissions: splitHeader(c.Get(cfg.HeaderUserPermissions), cfg.Separator),
		}

		// Store in Fiber Locals for easy access in handlers
		c.Locals(UserIdentityKey, identity)

		return c.Next()
	}
}

func splitHeader(raw, sep string) []string {
	if raw == "" {
		return nil
	}
	parts := strings.Split(raw, sep)
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		if p = strings.TrimSpace(p); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// GetUserIdentity retrieves the user identity from the fiber context
func GetUserIdentity(c *fiber.Ctx) *UserIdentity {
	identity, ok := c.Locals(UserIdentityKey).(*UserIdentity)
	if !ok {
		return nil
	}
	return identity
}

// RequireRole is a helper middleware to check if the user has any of the specific roles
func RequireRole(requiredRoles ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		identity := GetUserIdentity(c)
		if identity == nil {
			return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
		}

		if containsAny(identity.Roles, requiredRoles) {
			return c.Next()
		}

		return response.Error(c, fiber.StatusForbidden, "forbidden: insufficient permissions (role)", nil)
	}
}

// RequirePermission is a helper middleware to check if the user has any of the specific permissions
func RequirePermission(requiredPermissions ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		identity := GetUserIdentity(c)
		if identity == nil {
			return response.Error(c, fiber.StatusUnauthorized, "unauthorized", nil)
		}

		if containsAny(identity.Permissions, requiredPermissions) {
			return c.Next()
		}

		return response.Error(c, fiber.StatusForbidden, "forbidden: insufficient permissions (permission)", nil)
	}
}

func containsAny(have, want []string) bool {
	for _, w := range want {
		for _, h := range have {
			if h == w {
				return true
			}
		}
	}
	return false
}
