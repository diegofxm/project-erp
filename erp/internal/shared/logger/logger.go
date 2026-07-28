package logger

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"
	"sync"
	"time"
)

const (
	colComponent = 10 // ancho fijo de la columna "componente"
)

// handler es el slog.Handler personalizado con formato de columnas | separadas.
type handler struct {
	mu        sync.Mutex
	out       io.Writer
	component string
	preAttrs  []slog.Attr
}

func (h *handler) Enabled(_ context.Context, _ slog.Level) bool { return true }

func (h *handler) Handle(_ context.Context, r slog.Record) error {
	ts := r.Time.Format("2006-01-02 15:04:05")
	lv := levelStr(r.Level)
	co := fmt.Sprintf("%-*s", colComponent, h.component)

	var parts []string
	for _, a := range h.preAttrs {
		parts = append(parts, fmtAttr(a))
	}
	r.Attrs(func(a slog.Attr) bool {
		parts = append(parts, fmtAttr(a))
		return true
	})

	line := fmt.Sprintf("%s | %s | %s | %s", ts, lv, co, r.Message)
	if len(parts) > 0 {
		line += " | " + strings.Join(parts, "  ")
	}

	h.mu.Lock()
	defer h.mu.Unlock()
	fmt.Fprintln(h.out, line)
	return nil
}

func (h *handler) WithAttrs(attrs []slog.Attr) slog.Handler {
	n := &handler{out: h.out, component: h.component}
	n.preAttrs = append(n.preAttrs, h.preAttrs...)
	n.preAttrs = append(n.preAttrs, attrs...)
	return n
}

func (h *handler) WithGroup(_ string) slog.Handler { return h }

// ── Constructor ───────────────────────────────────────────────────────────────

// New devuelve un *slog.Logger con formato columnar para el componente dado.
func New(component string) *slog.Logger {
	return slog.New(&handler{
		out:       os.Stdout,
		component: component,
	})
}

// NewWithWriter permite inyectar un io.Writer (útil en tests).
func NewWithWriter(component string, w io.Writer) *slog.Logger {
	return slog.New(&handler{out: w, component: component})
}

// ── helpers ───────────────────────────────────────────────────────────────────

func levelStr(l slog.Level) string {
	switch {
	case l >= slog.LevelError:
		return "ERROR"
	case l >= slog.LevelWarn:
		return "WARN "
	case l >= slog.LevelInfo:
		return "INFO "
	default:
		return "DEBUG"
	}
}

func fmtAttr(a slog.Attr) string {
	v := a.Value.Resolve()
	// duración: mostrar con unidad legible
	if d, ok := v.Any().(time.Duration); ok {
		return fmt.Sprintf("%s=%s", a.Key, fmtDuration(d))
	}
	return fmt.Sprintf("%s=%v", a.Key, v)
}

func fmtDuration(d time.Duration) string {
	switch {
	case d < time.Millisecond:
		return fmt.Sprintf("%dµs", d.Microseconds())
	case d < time.Second:
		return fmt.Sprintf("%dms", d.Milliseconds())
	default:
		return fmt.Sprintf("%.2fs", d.Seconds())
	}
}
