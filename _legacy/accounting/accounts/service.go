package accounts

import "context"

// Service encapsula la lógica de acceso al PUC.
type Service struct {
	repo Repository
}

func NewService(repo Repository) *Service {
	return &Service{repo: repo}
}

// GetByCode devuelve una cuenta por su código PUC.
func (s *Service) GetByCode(ctx context.Context, code string) (*Account, error) {
	return s.repo.GetByCode(ctx, code)
}

// GetByID devuelve una cuenta por su UUID.
func (s *Service) GetByID(ctx context.Context, id string) (*Account, error) {
	return s.repo.GetByID(ctx, id)
}

// List devuelve todas las cuentas activas del PUC, ordenadas por código.
func (s *Service) List(ctx context.Context) ([]*Account, error) {
	return s.repo.List(ctx)
}

// GetPostable devuelve la cuenta validando que sea apta para recibir movimientos.
func (s *Service) GetPostable(ctx context.Context, code string) (*Account, error) {
	a, err := s.repo.GetByCode(ctx, code)
	if err != nil {
		return nil, err
	}
	if !a.IsActive {
		return nil, ErrAccountNotActive
	}
	if !a.IsPosting {
		return nil, ErrAccountNotPostable
	}
	return a, nil
}
