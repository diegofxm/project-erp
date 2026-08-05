// Package seed siembra el catálogo fijo de módulos y el punto de partida de planes comerciales —
// todo editable después desde /admin/plans sin tocar código. Cifras de precio son un punto de
// partida razonable, no un estudio de mercado (ver docs/Diseno_ERP_Go_Arquitectura_Hexagonal.md).
package seed

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
)

type moduleSeed struct {
	code, name, description string
}

var modules = []moduleSeed{
	{"electronic_invoicing", "Documentos Electrónicos", "Facturación electrónica, notas crédito/débito y documento soporte ante la DIAN"},
	{"erp_core", "ERP completo", "Ventas, compras, inventario, contabilidad, terceros y productos"},
	{"payroll_hr", "Nómina y RRHH", "Liquidación de nómina y gestión de empleados"},
}

type planSeed struct {
	code, name, description string
	billingCycle            string
	priceCents              int64
	includedDocuments       *int
	pricePerExtraDocCents   int64
	requiresCertificate     bool
	certificatePriceCents   int64
	annualIncrementPct      float64
	isInternal              bool
	moduleCodes             []string
}

func intp(v int) *int { return &v }

var plans = []planSeed{
	{
		code: "gratis", name: "Gratis", description: "Para probar la plataforma sin costo",
		billingCycle: "monthly", priceCents: 0,
		includedDocuments: intp(10), pricePerExtraDocCents: 60000,
		requiresCertificate: true, certificatePriceCents: 25_000_00 * 10, // 25.000.000 = $250.000/año
		annualIncrementPct: 0, isInternal: false,
		moduleCodes: []string{"electronic_invoicing"},
	},
	{
		code: "emprendedor", name: "Emprendedor", description: "Para negocios que están arrancando a facturar electrónicamente",
		billingCycle: "monthly", priceCents: 49_900_00,
		includedDocuments: intp(100), pricePerExtraDocCents: 60000,
		requiresCertificate: true, certificatePriceCents: 25_000_00 * 10,
		annualIncrementPct: 5, isInternal: false,
		moduleCodes: []string{"electronic_invoicing"},
	},
	{
		code: "ilimitado", name: "Ilimitado", description: "Documentos electrónicos sin límite mensual",
		billingCycle: "monthly", priceCents: 499_900_00,
		includedDocuments: nil, pricePerExtraDocCents: 0,
		requiresCertificate: true, certificatePriceCents: 25_000_00 * 10,
		annualIncrementPct: 5, isInternal: false,
		moduleCodes: []string{"electronic_invoicing"},
	},
	{
		code: "estrella", name: "Estrella", description: "Documentos electrónicos ilimitados + ERP completo, facturación anual",
		billingCycle: "annual", priceCents: 1_990_000_00,
		includedDocuments: nil, pricePerExtraDocCents: 0,
		requiresCertificate: true, certificatePriceCents: 400_000_00,
		annualIncrementPct: 5, isInternal: false,
		moduleCodes: []string{"electronic_invoicing", "erp_core"},
	},
	{
		code: "interno", name: "Interno", description: "Uso interno de la plataforma — no aparece en el catálogo público",
		billingCycle: "none", priceCents: 0,
		includedDocuments: nil, pricePerExtraDocCents: 0,
		requiresCertificate: false, certificatePriceCents: 0,
		annualIncrementPct: 0, isInternal: true,
		moduleCodes: []string{"electronic_invoicing", "erp_core", "payroll_hr"},
	},
}

func All(ctx context.Context, pool *pgxpool.Pool) error {
	moduleIDs := map[string]string{}
	for _, m := range modules {
		var id string
		err := pool.QueryRow(ctx, `
			INSERT INTO saas.modules (code, name, description)
			VALUES ($1, $2, $3)
			ON CONFLICT (code) DO UPDATE SET name = EXCLUDED.name, description = EXCLUDED.description
			RETURNING id`,
			m.code, m.name, m.description,
		).Scan(&id)
		if err != nil {
			return fmt.Errorf("seed saas modules %q: %w", m.code, err)
		}
		moduleIDs[m.code] = id
	}

	for _, p := range plans {
		var planID string
		err := pool.QueryRow(ctx, `
			INSERT INTO saas.plans
				(code, name, description, billing_cycle, price_cents, included_documents,
				 price_per_extra_document_cents, requires_certificate, certificate_price_cents,
				 annual_increment_pct, is_internal, is_active)
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,TRUE)
			ON CONFLICT (code) DO NOTHING
			RETURNING id`,
			p.code, p.name, p.description, p.billingCycle, p.priceCents, p.includedDocuments,
			p.pricePerExtraDocCents, p.requiresCertificate, p.certificatePriceCents,
			p.annualIncrementPct, p.isInternal,
		).Scan(&planID)
		if err != nil {
			// ON CONFLICT DO NOTHING sin RETURNING no encuentra fila — el plan ya existía y no se
			// vuelve a tocar (el superadmin pudo haberlo editado desde entonces).
			continue
		}
		for _, code := range p.moduleCodes {
			modID, ok := moduleIDs[code]
			if !ok {
				continue
			}
			if _, err := pool.Exec(ctx, `
				INSERT INTO saas.plan_modules (plan_id, module_id) VALUES ($1, $2)
				ON CONFLICT DO NOTHING`, planID, modID,
			); err != nil {
				return fmt.Errorf("seed saas plan_modules %q/%q: %w", p.code, code, err)
			}
		}
	}

	if _, err := pool.Exec(ctx, `
		INSERT INTO saas.settings (id, iva_rate_bp) VALUES (1, 1900)
		ON CONFLICT (id) DO NOTHING`,
	); err != nil {
		return fmt.Errorf("seed saas settings: %w", err)
	}

	return nil
}
