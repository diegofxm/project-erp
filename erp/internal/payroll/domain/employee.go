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

// UpdateEmployeeInput cubre los datos de contacto -- deliberadamente NO incluye
// IdentificationTypeCode/IdentificationNumber, que identifican legalmente al empleado y quedan
// ligados a sus contratos/liquidaciones ya generados; corregir un error de digitación ahí es un
// caso excepcional, no una edición rutinaria.
type UpdateEmployeeInput struct {
	FirstName        string
	LastName         string
	Email            string
	Phone            string
	DepartmentCode   string
	MunicipalityCode string
	AddressLine      string
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
