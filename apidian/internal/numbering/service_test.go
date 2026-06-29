package numbering_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/diegofxm/apidian/internal/numbering"
	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newService() *numbering.Service {
	return numbering.New(numbering.NewMemoryRepository())
}

func validRange() numbering.NumberingRange {
	rangeTo := int64(5000)
	return numbering.NumberingRange{
		IssuerID:             uuid.New(),
		DianDocumentTypeCode: "01",
		Prefix:               "SETP",
		ResolutionNumber:     "18760000001",
		ResolutionDate:       time.Now(),
		RangeFrom:            1,
		RangeTo:              &rangeTo,
		ValidFrom:            time.Now(),
		ValidTo:              time.Now().AddDate(2, 0, 0),
		Environment:          numbering.EnvironmentHabilitacion,
		TechnicalKey:         "clave-tecnica-de-prueba",
	}
}

func TestRegisterRange_OK(t *testing.T) {
	svc := newService()
	nr, err := svc.RegisterRange(context.Background(), validRange(), nil)
	require.NoError(t, err)
	assert.NotEqual(t, uuid.Nil, nr.ID)
	assert.True(t, nr.IsActive)
}

func TestRegisterRange_Validations(t *testing.T) {
	rangeTo := int64(10)
	tests := []struct {
		name    string
		mutate  func(*numbering.NumberingRange)
		wantErr error
	}{
		{"sin emisor", func(nr *numbering.NumberingRange) { nr.IssuerID = uuid.Nil }, numbering.ErrMissingIssuer},
		{"sin tipo de documento", func(nr *numbering.NumberingRange) { nr.DianDocumentTypeCode = "" }, numbering.ErrMissingDocumentType},
		{"sin prefijo", func(nr *numbering.NumberingRange) { nr.Prefix = "" }, numbering.ErrEmptyPrefix},
		{"range_from en cero", func(nr *numbering.NumberingRange) { nr.RangeFrom = 0 }, numbering.ErrInvalidRange},
		{"range_from mayor que range_to", func(nr *numbering.NumberingRange) { nr.RangeFrom = 20; nr.RangeTo = &rangeTo }, numbering.ErrInvalidRange},
		{"ambiente inválido", func(nr *numbering.NumberingRange) { nr.Environment = "9" }, numbering.ErrInvalidEnvironment},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			nr := validRange()
			tt.mutate(&nr)
			_, err := newService().RegisterRange(context.Background(), nr, nil)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

// TestRegisterRange_NextNumber_ResumesRealSequence cubre el escenario real que motivó este
// parámetro: una resolución que YA tiene números autorizados de verdad en la DIAN (ej.
// emitidos antes directamente con cofacture, o manualmente) — registrarla vía apidian debe
// poder continuar exactamente donde la DIAN se quedó, no reclamar desde range_from otra vez.
func TestRegisterRange_NextNumber_ResumesRealSequence(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	nr := validRange()
	nr.RangeFrom = 990068700
	nr.RangeTo = nil               // sin tope superior para este escenario
	nextNumber := int64(990068707) // la DIAN ya autorizó hasta ...706

	created, err := svc.RegisterRange(ctx, nr, &nextNumber)
	require.NoError(t, err)

	claimed, err := svc.ClaimNext(ctx, created.ID)
	require.NoError(t, err)
	assert.Equal(t, nextNumber, claimed, "el primer ClaimNext debe entregar exactamente next_number, no range_from")
}

func TestRegisterRange_NextNumber_OutOfRange(t *testing.T) {
	tests := []struct {
		name       string
		nextNumber int64
	}{
		{"menor que range_from", 0},
		{"mayor que range_to", 6000},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			svc := newService()
			nr := validRange() // range_from=1, range_to=5000
			next := tt.nextNumber
			_, err := svc.RegisterRange(context.Background(), nr, &next)
			assert.ErrorIs(t, err, numbering.ErrNextNumberOutOfRange)
		})
	}
}

func TestClaimNext_StartsAtRangeFrom(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	nr, err := svc.RegisterRange(ctx, validRange(), nil)
	require.NoError(t, err)

	first, err := svc.ClaimNext(ctx, nr.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), first) // RangeFrom de validRange()

	second, err := svc.ClaimNext(ctx, nr.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(2), second)
}

func TestClaimNext_RangeExhausted(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	rangeTo := int64(2)
	nr := validRange()
	nr.RangeFrom = 1
	nr.RangeTo = &rangeTo

	created, err := svc.RegisterRange(ctx, nr, nil)
	require.NoError(t, err)

	_, err = svc.ClaimNext(ctx, created.ID) // 1
	require.NoError(t, err)
	_, err = svc.ClaimNext(ctx, created.ID) // 2 (= rangeTo)
	require.NoError(t, err)

	_, err = svc.ClaimNext(ctx, created.ID) // ya no hay más
	assert.ErrorIs(t, err, numbering.ErrRangeExhausted)
}

// TestReleaseIfCurrent_RevertsLastClaim confirma el mecanismo de la sección 9.33: un número
// rechazado por la DIAN (o que nunca se logró transmitir) nunca quedó realmente gastado, así
// que el siguiente ClaimNext debe volver a entregarlo en vez de avanzar y dejar un hueco.
func TestReleaseIfCurrent_RevertsLastClaim(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	nr, err := svc.RegisterRange(ctx, validRange(), nil)
	require.NoError(t, err)

	claimed, err := svc.ClaimNext(ctx, nr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), claimed)

	require.NoError(t, svc.ReleaseIfCurrent(ctx, nr.ID, claimed))

	again, err := svc.ClaimNext(ctx, nr.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(1), again, "el mismo número debe volver a entregarse, no saltar al 2")
}

// TestClearTestSetID confirma el mecanismo de la sección 9.43: una vez la DIAN marca un set de
// pruebas como cerrado/Aceptado, el rango no debe seguir intentando SendTestSetAsync con él.
func TestClearTestSetID(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	r := validRange()
	r.TestSetID = "653bf9d9-b2b1-44ae-a66d-3b9cdc4271c3"
	nr, err := svc.RegisterRange(ctx, r, nil)
	require.NoError(t, err)
	require.NotEmpty(t, nr.TestSetID)

	require.NoError(t, svc.ClearTestSetID(ctx, nr.ID))

	got, err := svc.GetRange(ctx, nr.ID)
	require.NoError(t, err)
	assert.Empty(t, got.TestSetID)
}

// TestClearTestSetID_NoOpIfRangeMissing — mismo criterio que ReleaseIfCurrent: un rango
// inexistente no es un error, simplemente no hay nada que limpiar.
func TestClearTestSetID_NoOpIfRangeMissing(t *testing.T) {
	svc := newService()
	assert.NoError(t, svc.ClearTestSetID(context.Background(), uuid.New()))
}

// TestReleaseIfCurrent_NoOpIfNotCurrent es la mitad que hace seguro el mecanismo: si YA se
// reclamó otro número desde entonces, devolver el viejo sería arriesgar un duplicado real —
// debe ser un no-op silencioso, no un error (el llamador no tiene por qué saber si todavía
// tenía sentido intentarlo).
func TestReleaseIfCurrent_NoOpIfNotCurrent(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	nr, err := svc.RegisterRange(ctx, validRange(), nil)
	require.NoError(t, err)

	first, err := svc.ClaimNext(ctx, nr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(1), first)
	second, err := svc.ClaimNext(ctx, nr.ID)
	require.NoError(t, err)
	require.Equal(t, int64(2), second)

	// Intenta devolver el 1, pero el vigente ya es el 2 — no debe hacer nada.
	require.NoError(t, svc.ReleaseIfCurrent(ctx, nr.ID, first))

	third, err := svc.ClaimNext(ctx, nr.ID)
	require.NoError(t, err)
	assert.Equal(t, int64(3), third, "debe seguir avanzando normalmente, el intento de liberar un número viejo no debe afectar nada")
}

func TestListRanges_FiltersByIssuerAndDocumentType(t *testing.T) {
	svc := newService()
	ctx := context.Background()
	issuerA, issuerB := uuid.New(), uuid.New()

	invoiceA := validRange()
	invoiceA.IssuerID = issuerA
	invoiceA.DianDocumentTypeCode = "01"
	_, err := svc.RegisterRange(ctx, invoiceA, nil)
	require.NoError(t, err)

	creditNoteA := validRange()
	creditNoteA.IssuerID = issuerA
	creditNoteA.DianDocumentTypeCode = "91"
	creditNoteA.Prefix = "SETPNC"
	_, err = svc.RegisterRange(ctx, creditNoteA, nil)
	require.NoError(t, err)

	invoiceB := validRange()
	invoiceB.IssuerID = issuerB
	_, err = svc.RegisterRange(ctx, invoiceB, nil)
	require.NoError(t, err)

	all, err := svc.ListRanges(ctx, issuerA, "")
	require.NoError(t, err)
	assert.Len(t, all, 2, "solo los rangos del emisor A, sin importar tipo de documento")

	onlyInvoices, err := svc.ListRanges(ctx, issuerA, "01")
	require.NoError(t, err)
	require.Len(t, onlyInvoices, 1)
	assert.Equal(t, "01", onlyInvoices[0].DianDocumentTypeCode)

	none, err := svc.ListRanges(ctx, uuid.New(), "")
	require.NoError(t, err)
	assert.Empty(t, none)
}

// TestStatus_ActiveExpiredExhaustedInactive cubre las 4 ramas de NumberingRange.Status() —
// usada tanto por el panel de administración (insignia) como por el selector de la factura
// (qué rangos ofrecer), ver punto #2 de la ronda de UX/datos.
func TestStatus_ActiveExpiredExhaustedInactive(t *testing.T) {
	rangeTo := int64(10)

	t.Run("activo", func(t *testing.T) {
		nr := validRange()
		nr.IsActive = true
		nr.CurrentNumber = 0
		nr.ValidTo = time.Now().AddDate(1, 0, 0)
		assert.Equal(t, "active", nr.Status())
	})

	t.Run("vencido", func(t *testing.T) {
		nr := validRange()
		nr.IsActive = true
		nr.ValidTo = time.Now().AddDate(0, 0, -1)
		assert.Equal(t, "expired", nr.Status())
	})

	t.Run("agotado", func(t *testing.T) {
		nr := validRange()
		nr.IsActive = true
		nr.RangeTo = &rangeTo
		nr.CurrentNumber = rangeTo
		nr.ValidTo = time.Now().AddDate(1, 0, 0)
		assert.Equal(t, "exhausted", nr.Status())
	})

	t.Run("inactivo pesa más que vencido o agotado", func(t *testing.T) {
		nr := validRange()
		nr.IsActive = false
		nr.RangeTo = &rangeTo
		nr.CurrentNumber = rangeTo
		nr.ValidTo = time.Now().AddDate(0, 0, -1)
		assert.Equal(t, "inactive", nr.Status())
	})
}

// TestDeactivateRange_OK confirma el mecanismo de "borrado" del punto #2: desactivar marca
// is_active=false (Status() pasa a "inactive") sin eliminar la fila — un rango ya usado tiene
// FK real desde documents.Document, no se puede borrar de verdad.
func TestDeactivateRange_OK(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	nr, err := svc.RegisterRange(ctx, validRange(), nil)
	require.NoError(t, err)
	require.NoError(t, svc.DeactivateRange(ctx, nr.ID))

	got, err := svc.GetRange(ctx, nr.ID)
	require.NoError(t, err)
	assert.False(t, got.IsActive)
	assert.Equal(t, "inactive", got.Status())
}

func TestDeactivateRange_NotFound(t *testing.T) {
	svc := newService()
	err := svc.DeactivateRange(context.Background(), uuid.New())
	assert.ErrorIs(t, err, numbering.ErrRangeNotFound)
}

func TestClaimNext_NotFound(t *testing.T) {
	svc := newService()
	_, err := svc.ClaimNext(context.Background(), uuid.New())
	assert.ErrorIs(t, err, numbering.ErrRangeNotFound)
}

// TestClaimNext_ConcurrentNoDuplicatesNoGaps es la prueba que de verdad importa: el
// invariante que la DIAN exige es que el consecutivo nunca se repita ni se salte, incluso
// bajo carga concurrente. 200 goroutines reclaman al mismo tiempo sobre el mismo rango; al
// final, el conjunto de números reclamados debe ser exactamente {1..200}, sin huecos ni
// repetidos.
func TestClaimNext_ConcurrentNoDuplicatesNoGaps(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	const n = 200
	rangeTo := int64(n)
	nr := validRange()
	nr.RangeFrom = 1
	nr.RangeTo = &rangeTo

	created, err := svc.RegisterRange(ctx, nr, nil)
	require.NoError(t, err)

	var (
		wg      sync.WaitGroup
		mu      sync.Mutex
		claimed = make(map[int64]int) // número → cuántas veces salió
	)

	for i := 0; i < n; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			num, err := svc.ClaimNext(ctx, created.ID)
			require.NoError(t, err)

			mu.Lock()
			claimed[num]++
			mu.Unlock()
		}()
	}
	wg.Wait()

	assert.Len(t, claimed, n, "deberían haber exactamente %d números distintos reclamados", n)
	for num := int64(1); num <= int64(n); num++ {
		assert.Equalf(t, 1, claimed[num], "el número %d debería haberse reclamado exactamente una vez", num)
	}

	// El rango debe quedar exactamente agotado, no más, no menos.
	_, err = svc.ClaimNext(ctx, created.ID)
	assert.ErrorIs(t, err, numbering.ErrRangeExhausted)
}
