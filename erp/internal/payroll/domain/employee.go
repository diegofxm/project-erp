package domain

import (
	"time"

	"github.com/google/uuid"
)

type Employee struct {
	ID                     uuid.UUID
	CompanyID              uuid.UUID
	IdentificationTypeCode string
	IdentificationNumber   string
	FirstName              string
	LastName               string
	Email                  string
	Phone                  string
	DepartmentCode         string
	MunicipalityCode       string
	AddressLine            string
	IsActive               bool
	CreatedAt              time.Time
	UpdatedAt              time.Time
}

func (e Employee) FullName() string {
	return e.FirstName + " " + e.LastName
}

type CreateEmployeeInput struct {
	CompanyID              uuid.UUID
	IdentificationTypeCode string
	IdentificationNumber   string
	FirstName              string
	LastName               string
	Email                  string
	Phone                  string
	DepartmentCode         string
	MunicipalityCode       string
	AddressLine            string
}
