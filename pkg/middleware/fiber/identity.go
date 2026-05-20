package fiber

import (
	"strings"

	"github.com/gofiber/fiber/v2"
)

const (
	UserIdentityKey = "user_identity"

	HeaderUserID          = "X-User-ID"
	HeaderUserEmail       = "X-User-Email"
	HeaderUserRole        = "X-User-Role"
	HeaderUserPermissions = "X-User-Permissions"
)

// UserIdentity represents the information passed from BFF/Identity service
type UserIdentity struct {
	ID          string   `json:"id"`
	Email       string   `json:"email"`
	Roles       []string `json:"roles"`
	Permissions []string `json:"permissions"`
}

// Identity extracts user information from trusted headers injected by the BFF.
func Identity() fiber.Handler {
	return func(c *fiber.Ctx) error {
		userID := c.Get(HeaderUserID)
		userEmail := c.Get(HeaderUserEmail)
		userRolesRaw := c.Get(HeaderUserRole)
		userPermissionsRaw := c.Get(HeaderUserPermissions)

		// If no user ID is present, we assume it's an unauthenticated internal request
		if userID == "" {
			return c.Next()
		}

		// Parse roles from comma-separated string
		var roles []string
		if userRolesRaw != "" {
			roles = strings.Split(userRolesRaw, ",")
		}

		// Parse permissions from comma-separated string
		var permissions []string
		if userPermissionsRaw != "" {
			permissions = strings.Split(userPermissionsRaw, ",")
		}

		identity := &UserIdentity{
			ID:          userID,
			Email:       userEmail,
			Roles:       roles,
			Permissions: permissions,
		}

		// Store in Fiber Locals for easy access in handlers
		c.Locals(UserIdentityKey, identity)

		return c.Next()
	}
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
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "unauthorized",
			})
		}

		for _, requiredRole := range requiredRoles {
			for _, role := range identity.Roles {
				if role == requiredRole {
					return c.Next()
				}
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "forbidden: insufficient permissions (role)",
		})
	}
}

// RequirePermission is a helper middleware to check if the user has any of the specific permissions
func RequirePermission(requiredPermissions ...string) fiber.Handler {
	return func(c *fiber.Ctx) error {
		identity := GetUserIdentity(c)
		if identity == nil {
			return c.Status(fiber.StatusUnauthorized).JSON(fiber.Map{
				"message": "unauthorized",
			})
		}

		for _, requiredPermission := range requiredPermissions {
			for _, permission := range identity.Permissions {
				if permission == requiredPermission {
					return c.Next()
				}
			}
		}

		return c.Status(fiber.StatusForbidden).JSON(fiber.Map{
			"message": "forbidden: insufficient permissions (permission)",
		})
	}
}
