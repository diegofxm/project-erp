// Catálogos DIAN — api-dian no expone un endpoint para leerlos (viven solo como FK en
// Postgres, sembrados desde internal/database/seed/*.csv), así que se replican aquí a mano.
// Si se siembran más departamentos/municipios en el backend, hay que actualizar esta lista a
// mano también — no hay forma de mantenerlos sincronizados automáticamente todavía.

export const IDENTIFICATION_TYPES = [
  { code: "13", name: "Cédula de Ciudadanía" },
  { code: "31", name: "NIT" },
  { code: "22", name: "Cédula de Extranjería" },
  { code: "41", name: "Pasaporte" },
  { code: "12", name: "Tarjeta de Identidad" },
  { code: "21", name: "Tarjeta de Extranjería" },
  { code: "91", name: "NUIP" },
];

export const DEPARTMENTS = [
  { code: "05", name: "Antioquia" },
  { code: "08", name: "Atlántico" },
  { code: "11", name: "Bogotá D.C." },
  { code: "13", name: "Bolívar" },
  { code: "15", name: "Boyacá" },
  { code: "17", name: "Caldas" },
  { code: "18", name: "Caquetá" },
  { code: "19", name: "Cauca" },
  { code: "20", name: "Cesar" },
  { code: "23", name: "Córdoba" },
  { code: "25", name: "Cundinamarca" },
  { code: "27", name: "Chocó" },
  { code: "41", name: "Huila" },
  { code: "44", name: "La Guajira" },
  { code: "47", name: "Magdalena" },
  { code: "50", name: "Meta" },
  { code: "52", name: "Nariño" },
  { code: "54", name: "Norte de Santander" },
  { code: "63", name: "Quindío" },
  { code: "66", name: "Risaralda" },
  { code: "68", name: "Santander" },
  { code: "70", name: "Sucre" },
  { code: "73", name: "Tolima" },
  { code: "76", name: "Valle del Cauca" },
];

// Solo una capital por departamento sembrada hoy en el backend — si el municipio que
// necesitas no está, créalo en internal/database/seed/municipalities.csv primero.
export const MUNICIPALITIES = [
  { code: "11001", name: "Bogotá D.C.", departmentCode: "11" },
  { code: "05001", name: "Medellín", departmentCode: "05" },
  { code: "76001", name: "Cali", departmentCode: "76" },
  { code: "08001", name: "Barranquilla", departmentCode: "08" },
  { code: "13001", name: "Cartagena", departmentCode: "13" },
  { code: "66001", name: "Pereira", departmentCode: "66" },
  { code: "17001", name: "Manizales", departmentCode: "17" },
  { code: "23001", name: "Montería", departmentCode: "23" },
  { code: "41001", name: "Neiva", departmentCode: "41" },
  { code: "52001", name: "Pasto", departmentCode: "52" },
];

export const TAX_TYPES = [
  { code: "01", name: "IVA" },
  { code: "02", name: "IC (Impuesto al Consumo)" },
  { code: "03", name: "ICA" },
  { code: "04", name: "INC" },
  { code: "ZZ", name: "No aplica" },
];

// "94" no está en el seed de unit_measures pero SÍ es el código que cofacture validó contra
// la DIAN real (ver docs/api-dian-architecture.md) — products.unit_code ya no tiene FK por
// esto mismo. Se deja como sugerencia (datalist), no como <select> cerrado.
export const UNIT_MEASURES = [
  { code: "94", name: "Unidad de servicio (usado en pruebas reales DIAN)" },
  { code: "NIU", name: "Unidad" },
  { code: "KGM", name: "Kilogramo" },
  { code: "LTR", name: "Litro" },
  { code: "MTR", name: "Metro" },
  { code: "HUR", name: "Hora" },
  { code: "DAY", name: "Día" },
  { code: "MON", name: "Mes" },
];

export const PAYMENT_METHODS = [
  { code: "10", name: "Efectivo" },
  { code: "20", name: "Tarjeta débito" },
  { code: "30", name: "Tarjeta crédito" },
  { code: "42", name: "Transferencia" },
  { code: "47", name: "Cheque" },
  { code: "48", name: "Otros" },
];

export function municipalitiesByDepartment(departmentCode) {
  return MUNICIPALITIES.filter((m) => m.departmentCode === departmentCode);
}
