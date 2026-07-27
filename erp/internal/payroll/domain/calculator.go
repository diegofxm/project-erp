package domain

// CalcParams agrupa los parámetros externos necesarios para liquidar un período.
type CalcParams struct {
	Contract   *Contract
	WorkedDays int
	SMMLVCents int64
	ARLRateBP  int
}

// CalcResult contiene las líneas calculadas y los totales del período.
type CalcResult struct {
	Lines              []*PayslipLine
	TotalEarnedCents   int64
	TotalDeductedCents int64
	NetPayCents        int64
}

const (
	CodeSalario          = "SALARIO"
	CodeAuxTransporte    = "AUX_TRANSPORTE"
	CodeSaludEmpleado    = "SALUD_EMP"
	CodePensionEmpleado  = "PENSION_EMP"
	CodeSaludEmpleador   = "SALUD_EMPL"
	CodePensionEmpleador = "PENSION_EMPL"
	CodeARL              = "ARL"
	CodeCaja             = "CAJA"
	CodeSENA             = "SENA"
	CodeICBF             = "ICBF"
)

const (
	rateSaludEmpleado    = 400
	rateSaludEmpleador   = 850
	ratePensionEmpleado  = 400
	ratePensionEmpleador = 1200
	rateCaja             = 400
	rateSENA             = 200
	rateICBF             = 300
)

// Calculate produce las líneas de nómina a partir del contrato y los parámetros del período.
// Función pura: sin acceso a BD, determinista.
func Calculate(p CalcParams) CalcResult {
	var lines []*PayslipLine

	ibc := p.Contract.SalaryCents
	if p.Contract.SalaryType == SalaryIntegral {
		ibc = ibc * 70 / 100
	}

	salarioCents := p.Contract.SalaryCents * int64(p.WorkedDays) / 30
	lines = append(lines, &PayslipLine{
		ConceptCode: CodeSalario,
		ConceptName: "Salario básico",
		ConceptType: ConceptEarned,
		Quantity:    float64(p.WorkedDays),
		AmountCents: salarioCents,
	})

	if p.Contract.SalaryType == SalaryOrdinary &&
		p.Contract.SalaryCents <= 2*p.SMMLVCents {
		auxCents := int64(20000000) * int64(p.WorkedDays) / 30
		lines = append(lines, &PayslipLine{
			ConceptCode: CodeAuxTransporte,
			ConceptName: "Auxilio de transporte",
			ConceptType: ConceptEarned,
			Quantity:    float64(p.WorkedDays),
			AmountCents: auxCents,
		})
	}

	lines = append(lines, &PayslipLine{
		ConceptCode: CodeSaludEmpleado,
		ConceptName: "Salud (empleado 4%)",
		ConceptType: ConceptDeduction,
		Quantity:    1,
		AmountCents: bp(ibc, rateSaludEmpleado),
	})

	lines = append(lines, &PayslipLine{
		ConceptCode: CodePensionEmpleado,
		ConceptName: "Pensión (empleado 4%)",
		ConceptType: ConceptDeduction,
		Quantity:    1,
		AmountCents: bp(ibc, ratePensionEmpleado),
	})

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
		ConceptName: "ARL (clase " + string(p.Contract.RiskClass) + ")",
		ConceptType: ConceptEmployerContribution,
		Quantity:    1,
		AmountCents: bp(ibc, p.ARLRateBP),
	})

	if p.Contract.SalaryType == SalaryOrdinary {
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

	var earned, deducted int64
	for _, l := range lines {
		switch l.ConceptType {
		case ConceptEarned:
			earned += l.AmountCents
		case ConceptDeduction:
			deducted += l.AmountCents
		}
	}

	return CalcResult{
		Lines:              lines,
		TotalEarnedCents:   earned,
		TotalDeductedCents: deducted,
		NetPayCents:        earned - deducted,
	}
}

func bp(baseCents int64, rateBP int) int64 {
	return baseCents * int64(rateBP) / 10000
}
