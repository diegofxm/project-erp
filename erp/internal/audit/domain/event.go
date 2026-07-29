package domain

import (
	"time"

	"github.com/google/uuid"
)

type Event struct {
	ID           uuid.UUID
	CompanyID    uuid.UUID
	UserID       *uuid.UUID
	UserEmail    string
	UserName     string
	Action       string
	ResourceType string
	ResourceID   *uuid.UUID
	Metadata     map[string]any
	CreatedAt    time.Time
}

type ListFilter struct {
	ResourceID *uuid.UUID
	Limit      int
	Offset     int
}
