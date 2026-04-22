package entity

import (
	"time"

	"github.com/google/uuid"
)

type Role string
type Status string

const (
	RoleAdmin Role = "admin"
	RoleUser  Role = "user"

	StatusActive Status = "active"
	StatusBanned Status = "banned"
)

type User struct {
	ID           uuid.UUID
	EmailAddress string
	PhoneNumber  *string
	FullName     string
	Password     string
	Role         Role
	Status       Status
	CreatedAt    time.Time
	UpdatedAt    time.Time
	DeletedAt    *time.Time
}

type OTP struct {
	Code      string
	Email     string
	ExpiresAt time.Time
}
