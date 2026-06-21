package numbering_test

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/diegofxm/api-dian/internal/numbering"
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
	nr, err := svc.RegisterRange(context.Background(), validRange())
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
			_, err := newService().RegisterRange(context.Background(), nr)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestClaimNext_StartsAtRangeFrom(t *testing.T) {
	svc := newService()
	ctx := context.Background()

	nr, err := svc.RegisterRange(ctx, validRange())
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

	created, err := svc.RegisterRange(ctx, nr)
	require.NoError(t, err)

	_, err = svc.ClaimNext(ctx, created.ID) // 1
	require.NoError(t, err)
	_, err = svc.ClaimNext(ctx, created.ID) // 2 (= rangeTo)
	require.NoError(t, err)

	_, err = svc.ClaimNext(ctx, created.ID) // ya no hay más
	assert.ErrorIs(t, err, numbering.ErrRangeExhausted)
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

	created, err := svc.RegisterRange(ctx, nr)
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
