// Package hr gestiona recursos humanos: ausencias, vacaciones e incapacidades.
package domain

import (
	"time"

	"github.com/google/uuid"
)

type AbsenceType string

const (
	AbsenceVacation   AbsenceType = "vacation"
	AbsenceSickLeave  AbsenceType = "sick_leave"
	AbsenceMaternity  AbsenceType = "maternity"
	AbsencePaternity  AbsenceType = "paternity"
	AbsencePersonal   AbsenceType = "personal"
	AbsenceUnpaid     AbsenceType = "unpaid"
)

type AbsenceStatus string

const (
	AbsencePending  AbsenceStatus = "pending"
	AbsenceApproved AbsenceStatus = "approved"
	AbsenceRejected AbsenceStatus = "rejected"
)

type Absence struct {
	ID         uuid.UUID
	CompanyID  uuid.UUID
	EmployeeID uuid.UUID
	Type       AbsenceType
	Status     AbsenceStatus
	StartDate  time.Time
	EndDate    time.Time
	Days       int
	Reason     string
	Notes      string
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

type CreateAbsenceInput struct {
	CompanyID  uuid.UUID
	EmployeeID uuid.UUID
	Type       AbsenceType
	StartDate  time.Time
	EndDate    time.Time
	Reason     string
}
