package payslips

import (
	"github.com/diegofxm/payroll/contracts"
)

// CalcParams agrupa los parámetros externos necesarios para liquidar un período.
// Son valores de catálogo que el servicio resuelve antes de llamar a Calculate.
type CalcParams struct {
	Contract    *contracts.Contract
	WorkedDays  int  // días trabajados en el período (máx 30 para nómina mensual)
	SMMLVCents  int64
	ARLRateBP   int  // tasa ARL en puntos base para la clase de riesgo del contrato
}

// Result contiene las líneas calculadas y los totales del período.
type Result struct {
	Lines              []*PayslipLine
	TotalEarnedCents   int64
	TotalDeductedCents int64
	NetPayCents        int64
}

// Códigos de concepto estándar usados en todas las liquidaciones.
const (
	CodeSalario           = "SALARIO"
	CodeAuxTransporte     = "AUX_TRANSPORTE"
	CodeSaludEmpleado     = "SALUD_EMP"
	CodePensionEmpleado   = "PENSION_EMP"
	CodeSaludEmpleador    = "SALUD_EMPL"
	CodePensionEmpleador  = "PENSION_EMPL"
	CodeARL               = "ARL"
	CodeCaja              = "CAJA"
	CodeSENA              = "SENA"
	CodeICBF              = "ICBF"
)

// Tasas fijas de ley (Colombia, constantes mientras no cambie el decreto).
const (
	rateSaludEmpleado   = 400  // 4% en bp
	rateSaludEmpleador  = 850  // 8.5% en bp
	ratePensionEmpleado = 400  // 4% en bp
	ratePensionEmpleador = 1200 // 12% en bp
	rateCaja            = 400  // 4% en bp
	rateSENA            = 200  // 2% en bp
	rateICBF            = 300  // 3% en bp

	// Auxilio de transporte 2025 (en centavos).
	// Actualizar vía seed cuando el gobierno expida el decreto anual.
	auxTransporte2025Cents int64 = 20000000 // $200,000
)

// Calculate produce las líneas de nómina a partir del contrato y los parámetros del período.
// Es una función pura: sin acceso a BD, determinista, fácil de testear unitariamente.
func Calculate(p CalcParams) Result {
	var lines []*PayslipLine

	// ── IBC (Ingreso Base de Cotización) ──────────────────────────────────────────
	// Para salario ordinario: el IBC es el salario. Para integral: 70% del salario.
	ibc := p.Contract.SalaryCents
	if p.Contract.SalaryType == contracts.SalaryTypeIntegral {
		ibc = ibc * 70 / 100
	}

	// ── Salario proporcional a días trabajados ────────────────────────────────────
	salarioCents := p.Contract.SalaryCents * int64(p.WorkedDays) / 30
	lines = append(lines, &PayslipLine{
		ConceptCode: CodeSalario,
		ConceptName: "Salario básico",
		ConceptType: ConceptEarned,
		Quantity:    float64(p.WorkedDays),
		AmountCents: salarioCents,
	})

	// ── Auxilio de transporte ─────────────────────────────────────────────────────
	// Aplica si salario ≤ 2 SMMLV y el tipo de salario es ordinario.
	// Para salario integral nunca aplica (ya está incluido).
	auxCents := int64(0)
	if p.Contract.SalaryType == contracts.SalaryTypeOrdinary &&
		p.Contract.SalaryCents <= 2*p.SMMLVCents {
		auxCents = auxTransporte2025Cents * int64(p.WorkedDays) / 30
		lines = append(lines, &PayslipLine{
			ConceptCode: CodeAuxTransporte,
			ConceptName: "Auxilio de transporte",
			ConceptType: ConceptEarned,
			Quantity:    float64(p.WorkedDays),
			AmountCents: auxCents,
		})
	}

	// ── Deducciones del empleado ──────────────────────────────────────────────────
	// Para salario integral solo se descuenta sobre el 70% del IBC.

	saludEmpCents := bp(ibc, rateSaludEmpleado)
	lines = append(lines, &PayslipLine{
		ConceptCode: CodeSaludEmpleado,
		ConceptName: "Salud (empleado 4%)",
		ConceptType: ConceptDeduction,
		Quantity:    1,
		AmountCents: saludEmpCents,
	})

	pensionEmpCents := bp(ibc, ratePensionEmpleado)
	lines = append(lines, &PayslipLine{
		ConceptCode: CodePensionEmpleado,
		ConceptName: "Pensión (empleado 4%)",
		ConceptType: ConceptDeduction,
		Quantity:    1,
		AmountCents: pensionEmpCents,
	})

	// ── Aportes del empleador (no afectan el neto del empleado) ──────────────────
	// Para salario integral no aplican Caja/SENA/ICBF.

	lines = append(lines, &PayslipLine{
		ConceptCode: CodeSaludEmpleador,
		ConceptName: "Salud (empleador 8.5%)",
		ConceptType: ConceptEmployerContribution,
		Quantity:    1,
		AmountCents: bp(ibc, rateSaludEmpleador),
	})

	lines = append(lines, &PayslipLine{
		ConceptCode: CodePensionEmpleador,
		ConceptName: "Pensión (empleador 12%)",
		ConceptType: ConceptEmployerContribution,
		Quantity:    1,
		AmountCents: bp(ibc, ratePensionEmpleador),
	})

	lines = append(lines, &PayslipLine{
		ConceptCode: CodeARL,
		ConceptName: "ARL (" + riskLabel(p.Contract.RiskClass) + ")",
		ConceptType: ConceptEmployerContribution,
		Quantity:    1,
		AmountCents: bp(ibc, p.ARLRateBP),
	})

	if p.Contract.SalaryType == contracts.SalaryTypeOrdinary {
		lines = append(lines, &PayslipLine{
			ConceptCode: CodeCaja,
			ConceptName: "Caja de Compensación (4%)",
			ConceptType: ConceptEmployerContribution,
			Quantity:    1,
			AmountCents: bp(ibc, rateCaja),
		})
		lines = append(lines, &PayslipLine{
			ConceptCode: CodeSENA,
			ConceptName: "SENA (2%)",
			ConceptType: ConceptEmployerContribution,
			Quantity:    1,
			AmountCents: bp(ibc, rateSENA),
		})
		lines = append(lines, &PayslipLine{
			ConceptCode: CodeICBF,
			ConceptName: "ICBF (3%)",
			ConceptType: ConceptEmployerContribution,
			Quantity:    1,
			AmountCents: bp(ibc, rateICBF),
		})
	}

	// ── Totales ───────────────────────────────────────────────────────────────────
	var earned, deducted int64
	for _, l := range lines {
		switch l.ConceptType {
		case ConceptEarned:
			earned += l.AmountCents
		case ConceptDeduction:
			deducted += l.AmountCents
		}
	}

	return Result{
		Lines:              lines,
		TotalEarnedCents:   earned,
		TotalDeductedCents: deducted,
		NetPayCents:        earned - deducted,
	}
}

// bp convierte una tasa en puntos base a su valor en centavos sobre una base dada.
// 1 bp = 0.01%, por tanto amount × bp / 10000.
func bp(baseCents int64, rateBP int) int64 {
	return baseCents * int64(rateBP) / 10000
}

func riskLabel(rc contracts.WorkRiskClass) string {
	return "clase " + string(rc)
}
