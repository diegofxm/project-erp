package events

import (
	"context"
	"sync"
)

// Event es la interfaz base de todos los eventos de dominio.
type Event interface {
	EventName() string
}

// Handler procesa un evento. Recibe el context.Context real de quien publicó (normalmente el de
// la request HTTP que disparó la operación) -- antes cada suscriptor se armaba su propio
// context.Background(), perdiendo cualquier deadline/cancelación/valor propagado desde la
// request original. El error de retorno tampoco se descarta -- Publish lo recolecta y lo
// devuelve a quien publicó, para que decida si es un problema real (registrarlo en
// audit.events, advertir al usuario) en vez de perderse en un log.Printf que nadie mira.
type Handler func(ctx context.Context, event Event) error

// Bus distribuye eventos a sus suscriptores en el mismo proceso (in-memory, síncrono).
// Si en el futuro se necesita garantía de entrega o comunicación con sistemas externos,
// esta implementación se reemplaza por una sobre RabbitMQ/NATS sin tocar los dominios
// — los suscriptores solo conocen la interfaz Event, no el transporte.
type Bus struct {
	mu       sync.RWMutex
	handlers map[string][]Handler
}

func NewBus() *Bus {
	return &Bus{handlers: make(map[string][]Handler)}
}

// Subscribe registra un handler para un tipo de evento.
func (b *Bus) Subscribe(eventName string, h Handler) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.handlers[eventName] = append(b.handlers[eventName], h)
}

// Publish entrega el evento a todos sus handlers en el orden en que se registraron y devuelve
// los errores de los que fallen (nil si todos los handlers tuvieron éxito). No detiene la
// entrega al primer error -- todos los suscriptores de un evento son independientes entre sí
// (ej. el asiento contable y el descuento de inventario de una misma venta no deberían
// bloquearse mutuamente).
func (b *Bus) Publish(ctx context.Context, e Event) []error {
	b.mu.RLock()
	handlers := b.handlers[e.EventName()]
	b.mu.RUnlock()
	var errs []error
	for _, h := range handlers {
		if err := h(ctx, e); err != nil {
			errs = append(errs, err)
		}
	}
	return errs
}
