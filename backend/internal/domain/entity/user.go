package entity

import (
	"time"

	"github.com/google/uuid"
)

// User representa un usuario del sistema (para autenticación y autorización)
type User struct {
	ID           uuid.UUID  `json:"id"`
	Email        string     `json:"email"`
	PasswordHash string     `json:"-"`
	FullName     string     `json:"full_name"`
	Role         UserRole   `json:"role"`
	CountryIDs   []uuid.UUID `json:"country_ids,omitempty"` // Países a los que tiene acceso
	IsActive     bool       `json:"is_active"`
	LastLoginAt  *time.Time `json:"last_login_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	UpdatedAt    time.Time  `json:"updated_at"`
}

// UserRole roles de usuario
type UserRole string

const (
	RoleAdmin     UserRole = "ADMIN"      // Acceso total
	RoleAnalyst   UserRole = "ANALYST"    // Puede revisar y aprobar/rechazar
	RoleOperator  UserRole = "OPERATOR"   // Solo lectura y creación
	RoleViewer    UserRole = "VIEWER"     // Solo lectura
)

// HasPermission verifica si el usuario tiene permiso para una acción
func (u *User) HasPermission(action string) bool {
	permissions := map[UserRole][]string{
		RoleAdmin:    {"create", "read", "update", "delete", "approve", "reject", "admin"},
		RoleAnalyst:  {"create", "read", "update", "approve", "reject"},
		RoleOperator: {"create", "read", "update"},
		RoleViewer:   {"read"},
	}
	
	allowed, exists := permissions[u.Role]
	if !exists {
		return false
	}
	
	for _, p := range allowed {
		if p == action {
			return true
		}
	}
	return false
}

// CanAccessCountry verifica si el usuario puede acceder a un país específico
func (u *User) CanAccessCountry(countryID uuid.UUID) bool {
	if u.Role == RoleAdmin {
		return true
	}
	
	for _, id := range u.CountryIDs {
		if id == countryID {
			return true
		}
	}
	return false
}

// Session representa una sesión de usuario
type Session struct {
	ID           uuid.UUID `json:"id"`
	UserID       uuid.UUID `json:"user_id"`
	Token        string    `json:"-"`
	RefreshToken string    `json:"-"`
	IPAddress    string    `json:"ip_address"`
	UserAgent    string    `json:"user_agent"`
	ExpiresAt    time.Time `json:"expires_at"`
	CreatedAt    time.Time `json:"created_at"`
}

// TokenClaims claims del JWT
type TokenClaims struct {
	UserID     uuid.UUID   `json:"user_id"`
	Email      string      `json:"email"`
	Role       UserRole    `json:"role"`
	CountryIDs []uuid.UUID `json:"country_ids,omitempty"`
}

