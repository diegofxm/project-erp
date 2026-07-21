package prospects

import (
	"context"
	"encoding/base64"
	"errors"
	"strings"

	"github.com/google/uuid"
)

type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Submit(ctx context.Context, name, email, nit, cedulaBase64, rutBase64 string) (*Prospect, error) {
	name = strings.TrimSpace(name)
	email = strings.TrimSpace(email)
	if name == "" {
		return nil, errors.New("el nombre es requerido")
	}
	if email == "" {
		return nil, errors.New("el correo es requerido")
	}

	var cedulaBytes, rutBytes []byte
	if cedulaBase64 != "" {
		b, err := base64.StdEncoding.DecodeString(cedulaBase64)
		if err != nil {
			return nil, errors.New("cedula_pdf_base64 no es base64 válido")
		}
		cedulaBytes = b
	}
	if rutBase64 != "" {
		b, err := base64.StdEncoding.DecodeString(rutBase64)
		if err != nil {
			return nil, errors.New("rut_pdf_base64 no es base64 válido")
		}
		rutBytes = b
	}

	return s.repo.Create(ctx, Prospect{Name: name, Email: email, NIT: nit}, cedulaBytes, rutBytes)
}

func (s *Service) List(ctx context.Context) ([]Prospect, error) {
	return s.repo.List(ctx)
}

func (s *Service) GetByID(ctx context.Context, id uuid.UUID) (*Prospect, error) {
	return s.repo.GetByID(ctx, id)
}

func (s *Service) Approve(ctx context.Context, id uuid.UUID) (*Prospect, error) {
	return s.repo.Approve(ctx, id)
}

func (s *Service) Reject(ctx context.Context, id uuid.UUID, notes string) (*Prospect, error) {
	return s.repo.Reject(ctx, id, notes)
}

func (s *Service) GetCedulaPDF(ctx context.Context, id uuid.UUID) ([]byte, error) {
	return s.repo.GetCedulaPDF(ctx, id)
}

func (s *Service) GetRutPDF(ctx context.Context, id uuid.UUID) ([]byte, error) {
	return s.repo.GetRutPDF(ctx, id)
}
