package application

import (
	"context"
	"fmt"

	"github.com/google/uuid"

	"github.com/diegofxm/erp/internal/saas/domain"
	notificationdomain "github.com/diegofxm/erp/internal/shared/notification/domain"
)

type ProspectUseCase struct {
	repo      domain.ProspectRepository
	users     domain.UserPort
	companies domain.CompanyProvisioningPort
	notifier  notificationdomain.Notifier
	appURL    string
}

func NewProspectUseCase(
	repo domain.ProspectRepository,
	users domain.UserPort,
	companies domain.CompanyProvisioningPort,
	notifier notificationdomain.Notifier,
	appURL string,
) *ProspectUseCase {
	return &ProspectUseCase{repo: repo, users: users, companies: companies, notifier: notifier, appURL: appURL}
}

type SubmitProspectRequest struct {
	Name              string
	Email             string
	NIT               string
	CedulaFile        []byte
	CedulaContentType string
	RUTFile           []byte
	RUTContentType    string
}

// Submit registra una solicitud de acceso pública (sin cuenta) — queda "pending" hasta que un
// superadmin la revise.
func (uc *ProspectUseCase) Submit(ctx context.Context, req SubmitProspectRequest) (*domain.Prospect, error) {
	return uc.repo.Create(ctx, domain.Prospect{
		Name: req.Name, Email: req.Email, NIT: req.NIT,
		CedulaFile: req.CedulaFile, CedulaContentType: req.CedulaContentType,
		RUTFile: req.RUTFile, RUTContentType: req.RUTContentType,
		Status: domain.ProspectPending,
	})
}

func (uc *ProspectUseCase) List(ctx context.Context) ([]domain.Prospect, error) {
	return uc.repo.List(ctx)
}

func (uc *ProspectUseCase) GetByID(ctx context.Context, id uuid.UUID) (*domain.Prospect, error) {
	return uc.repo.GetByID(ctx, id)
}

// Approve aprovisiona la cuenta real: crea el usuario invitado (sin password todavía), crea su
// primera empresa con el NIT que declaró, y le envía el correo para configurar la contraseña —
// reutilizando el mismo flujo de invitación que ya existía para invitar compañeros de equipo. El
// prospecto solo queda "approved" al final: si algo falla a mitad de camino, se queda "pending" y
// se puede reintentar sin duplicar nada (InviteOwner es idempotente por correo).
func (uc *ProspectUseCase) Approve(ctx context.Context, id uuid.UUID) (*domain.Prospect, error) {
	p, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Status != domain.ProspectPending {
		return nil, domain.ErrProspectNotPending
	}

	userID, inviteToken, err := uc.users.InviteOwner(ctx, p.Email, p.Name)
	if err != nil {
		return nil, fmt.Errorf("crear usuario del nuevo dueño: %w", err)
	}
	if _, err := uc.companies.CreateCompanyForOwner(ctx, userID, p.Name, p.NIT, p.Email); err != nil {
		return nil, fmt.Errorf("crear empresa del nuevo dueño: %w", err)
	}

	if uc.notifier != nil {
		msg := notificationdomain.Message{
			To:         p.Email,
			Channel:    notificationdomain.ChannelEmail,
			Subject:    "Tu solicitud de acceso fue aprobada",
			TemplateID: "invite_owner",
			Data: map[string]any{
				"Name":            p.Name,
				"CompanyName":     p.Name,
				"AcceptInviteURL": uc.appURL + "/accept-invite?token=" + inviteToken,
			},
		}
		if err := uc.notifier.Send(ctx, msg); err != nil {
			return nil, fmt.Errorf("enviar invitación: %w", err)
		}
	}

	return uc.repo.UpdateStatus(ctx, id, domain.ProspectApproved, p.Notes)
}

func (uc *ProspectUseCase) Reject(ctx context.Context, id uuid.UUID, notes string) (*domain.Prospect, error) {
	p, err := uc.repo.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if p.Status != domain.ProspectPending {
		return nil, domain.ErrProspectNotPending
	}
	return uc.repo.UpdateStatus(ctx, id, domain.ProspectRejected, notes)
}
