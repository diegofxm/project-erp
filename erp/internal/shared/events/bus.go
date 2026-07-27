package events

import "sync"

// Event es la interfaz base de todos los eventos de dominio.
type Event interface {
	EventName() string
}

// Handler procesa un evento.
type Handler func(event Event)

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

// Publish entrega el evento a todos sus handlers en el orden en que se registraron.
func (b *Bus) Publish(e Event) {
	b.mu.RLock()
	handlers := b.handlers[e.EventName()]
	b.mu.RUnlock()
	for _, h := range handlers {
		h(e)
	}
}
