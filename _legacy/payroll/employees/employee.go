package employees

import (
	"time"

	"github.com/google/uuid"
)

// Employee representa un trabajador vinculado a una empresa.
type Employee struct {
	ID                       uuid.UUID
	CompanyID                uuid.UUID
	IdentificationTypeCode   string
	IdentificationNumber     string
	FirstName                string
	LastName                 string
	Email                    string
	Phone                    string
	DepartmentCode           string
	MunicipalityCode         string
	AddressLine              string
	IsActive                 bool
	CreatedAt                time.Time
	UpdatedAt                time.Time
}

func (e Employee) FullName() string {
	return e.FirstName + " " + e.LastName
}

// CreateInput contiene los datos requeridos para registrar un empleado.
type CreateInput struct {
	CompanyID                uuid.UUID
	IdentificationTypeCode   string
	IdentificationNumber     string
	FirstName                string
	LastName                 string
	Email                    string
	Phone                    string
	DepartmentCode           string
	MunicipalityCode         string
	AddressLine              string
}
