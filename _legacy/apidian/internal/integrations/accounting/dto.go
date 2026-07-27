package accounting

// Este paquete es el adaptador entre apidian y el módulo accounting.
// Llama las funciones de accounting.Core directamente (mismo proceso, via go.work).
// En Fase 3 (NATS), solo cambia client.go — el resto del dominio no se toca.
