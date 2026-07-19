const LABELS: Record<string, string> = {
  "11": "R.C.",
  "12": "T.I.",
  "13": "CC",
  "21": "T.E.",
  "22": "C.E.",
  "31": "NIT",
  "41": "Pasaporte",
  "42": "D.I.E.",
  "47": "NIT ext.",
  "50": "NIT DIAN",
  "91": "NUIP",
};

export function idTypeLabel(typeCode: string): string {
  return LABELS[typeCode] ?? typeCode;
}
