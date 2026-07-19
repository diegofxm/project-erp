import { useState } from "react";
import type { PDFFormat } from "./documents";

const KEY = "apidian.pdfFormat";

export function usePdfFormat(): [PDFFormat, (f: PDFFormat) => void] {
  const [format, setFormat] = useState<PDFFormat>(
    () => (localStorage.getItem(KEY) as PDFFormat) ?? "full_a4"
  );
  function save(f: PDFFormat) {
    setFormat(f);
    localStorage.setItem(KEY, f);
  }
  return [format, save];
}
