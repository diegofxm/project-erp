import { useEffect, useState } from "react";
import { FileBadge } from "lucide-react";
import { issueCertificates, listCertificates } from "../lib/accounting";
import { listSuppliers } from "../lib/suppliers";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { useToast } from "../context/ToastContext";
import type { Supplier, WithholdingCertificate } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Combobox } from "../components/ui/Combobox";
import { Input } from "../components/ui/Input";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { InfoTip } from "../components/ui/InfoTip";

// A diferencia del resto de accounting (centavos), estos montos vienen en pesos directo —
// son un espejo de purchase.PurchaseWithholding (Base/Amount float64), no journal_lines.
function money(v: number): string {
  return formatCOP.format(v);
}

export function AccountingCertificatesPage() {
  const toast = useToast();
  const [suppliers, setSuppliers] = useState<Supplier[]>([]);
  const [supplierId, setSupplierId] = useState("");
  const [year, setYear] = useState(new Date().getFullYear());
  const [certificates, setCertificates] = useState<WithholdingCertificate[] | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [issuing, setIssuing] = useState(false);

  useEffect(() => {
    listSuppliers().then(setSuppliers).catch(() => setSuppliers([]));
  }, []);

  function refresh(y: number) {
    listCertificates(y).then(setCertificates).catch((err) => setError(err instanceof ApiError ? err.message : "No se pudieron cargar los certificados"));
  }

  useEffect(() => { refresh(year); }, [year]);

  async function handleIssue() {
    const supplier = suppliers.find((s) => s.id === supplierId);
    if (!supplier) return;
    setIssuing(true);
    setError(null);
    try {
      await issueCertificates({ supplier_id: supplierId, third_party_nit: supplier.identification_number, fiscal_year: year });
      toast.success("Certificado emitido.");
      setSupplierId("");
      refresh(year);
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo emitir el certificado — verifica que el proveedor tenga retenciones aplicadas en ese año");
    } finally {
      setIssuing(false);
    }
  }

  const supplierOptions = suppliers.map((s) => ({ value: s.id, label: s.name }));

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Certificados de retención" }]} />
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <FileBadge className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Certificados de retención
        <InfoTip>
          Suma todas las retenciones que le aplicaste a un proveedor durante el año fiscal (capturadas al recibir
          compras) y emite un certificado por concepto — lo que el proveedor usa para soportar su propia declaración.
        </InfoTip>
      </h1>

      <Card className="mb-3 p-4">
        <div className="flex flex-wrap items-end gap-2">
          <div className="w-64">
            <Combobox label="Proveedor" value={supplierId} onChange={setSupplierId} options={supplierOptions} placeholder="Buscar proveedor…" />
          </div>
          <Input type="number" label="Año fiscal" value={year} onChange={(e) => setYear(Number(e.target.value))} className="w-28" />
          <Button type="button" disabled={!supplierId} loading={issuing} onClick={handleIssue}>Emitir certificados</Button>
        </div>
      </Card>

      {error && <Banner tone="danger">{error}</Banner>}

      {certificates === null ? (
        <div className="flex min-h-24 items-center justify-center"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>
      ) : certificates.length === 0 ? (
        <p className="text-xs text-(--text-secondary)">Sin certificados emitidos para {year}.</p>
      ) : (
        <div className="overflow-x-auto rounded border border-(--border-color)">
          <table className="w-full text-left text-xs">
            <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
              <tr>
                <th className="px-3 py-2 font-medium">NIT tercero</th>
                <th className="px-3 py-2 font-medium">Concepto</th>
                <th className="px-3 py-2 text-right font-medium">Base</th>
                <th className="px-3 py-2 text-right font-medium">Retenido</th>
              </tr>
            </thead>
            <tbody>
              {certificates.map((c, i) => (
                <tr key={c.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                  <td className="px-3 py-2 font-mono text-(--text-primary)">{c.third_party_nit}</td>
                  <td className="px-3 py-2 text-(--text-primary)">{c.concept_name}</td>
                  <td className="px-3 py-2 text-right font-mono text-(--text-secondary)">{money(c.gross_amount)}</td>
                  <td className="px-3 py-2 text-right font-mono text-(--text-primary)">{money(c.tax_withheld)}</td>
                </tr>
              ))}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
