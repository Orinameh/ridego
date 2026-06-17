package models

import (
	"fmt"
	"time"

	"github.com/google/uuid"
)

type Role string

const (
	RoleRider  Role = "rider"
	RoleDriver Role = "driver"
	RoleAdmin  Role = "admin"
)

// DTO hence the json tags, used for API responses and requests. The PasswordHash field is never serialized to JSON for security reasons.
type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"` // never serialized
	Role         Role      `json:"role"`
	FullName     string    `json:"full_name"`
	Phone        string    `json:"phone"`
	AvatarURL    string    `json:"avatar_url,omitempty"`
	Rating       float64   `json:"rating"`
	IsActive     bool      `json:"is_active"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// RefreshToken stored in DB(Entity model), invalidated on use (rotation).
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string // SHA-256 of the raw token
	ExpiresAt time.Time
	CreatedAt time.Time
}

// Request / response DTOs
type RegisterRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required,min=8"`
	Role     Role   `json:"role" validate:"required,oneof=rider driver admin"`
	FullName string `json:"full_name" validate:"required"`
	Phone    string `json:"phone" validate:"required,e164"`
}

type LoginRequest struct {
	Email    string `json:"email" validate:"required,email"`
	Password string `json:"password" validate:"required"`
}

type AuthResponse struct {
	User         *User  `json:"user"`
	AccessToken  string `json:"access_token"`
	RefreshToken string `json:"refresh_token"`
	ExpiresIn    int    `json:"expires_in"` // seconds
}

type UpdateUserRequest struct {
	FullName  *string `json:"full_name,omitempty"`
	Phone     *string `json:"phone,omitempty" validate:"omitempty,e164"`
	AvatarURL *string `json:"avatar_url,omitempty" validate:"omitempty,url"`
}

func (r RegisterRequest) Validate() error {
	switch {
	case r.Email == "":
		return fmt.Errorf("email is required")
	case len(r.Password) < 8:
		return fmt.Errorf("password must be at least 8 characters")
	case r.Role != RoleRider && r.Role != RoleDriver && r.Role != RoleAdmin:
		return fmt.Errorf("invalid role: %s", r.Role)
	case r.FullName == "":
		return fmt.Errorf("full name is required")
	case r.Phone == "":
		return fmt.Errorf("phone number is required")
	}
	return nil
}
