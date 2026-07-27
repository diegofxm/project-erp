// Package noop implementa el puerto Notifier sin enviar nada.
// Usado en desarrollo y tests para no enviar correos reales.
package noop

import (
	"context"
	"log"

	notificationdomain "github.com/diegofxm/erp/internal/shared/notification/domain"
)

type Sender struct{}

func New() *Sender { return &Sender{} }

var _ notificationdomain.Notifier = (*Sender)(nil)

func (s *Sender) Send(_ context.Context, msg notificationdomain.Message) error {
	log.Printf("[notification/noop] canal=%s to=%s template=%s subject=%q",
		msg.Channel, msg.To, msg.TemplateID, msg.Subject)
	return nil
}
