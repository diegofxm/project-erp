// Etiqueta humana para dian_status_code — los códigos reales y conocidos están confirmados en
// cofacture/dian/parser.go/parser_test.go: "00" válido/aceptado, "99" rechazado, "2" caso
// especial de set de pruebas cerrado (no es un rechazo de contenido, ver
// Result.IsTestSetClosed). dian_status_description/dian_status_message ya traen el texto libre
// real de la DIAN — esta etiqueta es solo para no mostrar el código crudo como título.
const LABELS: Record<string, string> = {
  "00": "Validado",
  "99": "Rechazado",
  "2": "Set de pruebas cerrado",
};

export function dianStatusLabel(code: string | undefined): string {
  if (!code) return "Sin información";
  return LABELS[code] ?? `Código ${code}`;
}
