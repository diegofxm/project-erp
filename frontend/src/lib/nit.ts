const WEIGHTS = [3, 7, 13, 17, 19, 23, 29, 37, 41, 43, 47, 53, 59, 67, 71];

export function computeNITCheckDigit(nit: string): string | null {
  const digits = nit.replace(/[\s.\-]/g, "");
  if (!/^\d+$/.test(digits) || digits.length === 0 || digits.length > WEIGHTS.length) return null;
  let sum = 0;
  for (let i = 0; i < digits.length; i++) {
    sum += parseInt(digits[digits.length - 1 - i], 10) * WEIGHTS[i];
  }
  const remainder = sum % 11;
  return remainder === 0 || remainder === 1 ? String(remainder) : String(11 - remainder);
}
