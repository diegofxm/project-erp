import { useEffect, useState } from "react";
import { FileWarning } from "lucide-react";
import {
  fileICA, fileIncomeTax, fileIVA, generateICA, generateIncomeTax, generateIVA,
  listICA, listICATariffs, listIncomeTax, listIVA, setICATariff,
} from "../lib/accounting";
import { ApiError } from "../lib/apiClient";
import { formatCOP } from "../lib/currency";
import { useToast } from "../context/ToastContext";
import type { ICADeclaration, ICATariff, IncomeTaxDeclaration, IVADeclaration } from "../lib/types";
import { Banner } from "../components/ui/Banner";
import { Button } from "../components/ui/Button";
import { Card } from "../components/ui/Card";
import { Input } from "../components/ui/Input";
import { Select } from "../components/ui/Select";
import { Spinner } from "../components/ui/Spinner";
import { Breadcrumbs } from "../components/ui/Breadcrumbs";
import { StatusPill, type StatusTone } from "../components/ui/StatusPill";

type Tab = "iva" | "renta" | "ica";

function money(v: number): string {
  return formatCOP.format(v / 100);
}

function todayISO(): string {
  return new Date().toISOString().slice(0, 10);
}

function firstOfYearISO(): string {
  return `${new Date().getFullYear()}-01-01`;
}

const DECL_LABEL: Record<string, string> = { DRAFT: "Borrador", FILED: "Presentada", PAID: "Pagada", CORRECTED: "Corregida" };
const DECL_TONE: Record<string, StatusTone> = { DRAFT: "neutral", FILED: "success", PAID: "success", CORRECTED: "warning" };

export function AccountingTaxDeclarationsPage() {
  const toast = useToast();
  const [tab, setTab] = useState<Tab>("iva");
  const [error, setError] = useState<string | null>(null);

  const [ivaList, setIvaList] = useState<IVADeclaration[] | null>(null);
  const [ivaFrom, setIvaFrom] = useState(firstOfYearISO());
  const [ivaTo, setIvaTo] = useState(todayISO());
  const [ivaGenerating, setIvaGenerating] = useState(false);

  const [incomeTaxList, setIncomeTaxList] = useState<IncomeTaxDeclaration[] | null>(null);
  const [incomeTaxYear, setIncomeTaxYear] = useState(new Date().getFullYear());
  const [incomeTaxGenerating, setIncomeTaxGenerating] = useState(false);

  const [icaList, setIcaList] = useState<ICADeclaration[] | null>(null);
  const [tariffs, setTariffs] = useState<ICATariff[]>([]);
  const [icaMunicipality, setIcaMunicipality] = useState("");
  const [icaCiiu, setIcaCiiu] = useState("");
  const [icaFrom, setIcaFrom] = useState(firstOfYearISO());
  const [icaTo, setIcaTo] = useState(todayISO());
  const [icaGenerating, setIcaGenerating] = useState(false);
  const [showTariffForm, setShowTariffForm] = useState(false);
  const [tariffMunicipality, setTariffMunicipality] = useState("");
  const [tariffCiiu, setTariffCiiu] = useState("");
  const [tariffYear, setTariffYear] = useState(new Date().getFullYear());
  const [tariffRate, setTariffRate] = useState("");
  const [tariffSurcharge, setTariffSurcharge] = useState("0");
  const [savingTariff, setSavingTariff] = useState(false);

  function refreshIVA() { listIVA().then(setIvaList).catch(() => setIvaList([])); }
  function refreshIncomeTax() { listIncomeTax().then(setIncomeTaxList).catch(() => setIncomeTaxList([])); }
  function refreshICA() { listICA().then(setIcaList).catch(() => setIcaList([])); listICATariffs().then(setTariffs).catch(() => setTariffs([])); }

  useEffect(() => {
    if (tab === "iva" && ivaList === null) refreshIVA();
    if (tab === "renta" && incomeTaxList === null) refreshIncomeTax();
    if (tab === "ica" && icaList === null) refreshICA();
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [tab]);

  async function handleGenerateIVA() {
    setIvaGenerating(true);
    setError(null);
    try {
      await generateIVA(ivaFrom, ivaTo, "BIMESTRAL");
      toast.success("Declaración de IVA generada.");
      refreshIVA();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo generar la declaración de IVA");
    } finally {
      setIvaGenerating(false);
    }
  }

  async function handleGenerateIncomeTax() {
    setIncomeTaxGenerating(true);
    setError(null);
    try {
      await generateIncomeTax(incomeTaxYear);
      toast.success("Declaración de renta generada.");
      refreshIncomeTax();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo generar la declaración de renta");
    } finally {
      setIncomeTaxGenerating(false);
    }
  }

  async function handleSetTariff() {
    if (!tariffMunicipality || !tariffCiiu || !tariffRate) return;
    setSavingTariff(true);
    try {
      await setICATariff({ municipality_code: tariffMunicipality, ciiu_code: tariffCiiu, fiscal_year: tariffYear, rate_bp: Math.round(Number(tariffRate) * 100), surcharge_bp: Math.round(Number(tariffSurcharge || 0) * 100) });
      toast.success("Tarifa ICA registrada.");
      setShowTariffForm(false);
      setTariffMunicipality(""); setTariffCiiu(""); setTariffRate(""); setTariffSurcharge("0");
      refreshICA();
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "No se pudo registrar la tarifa");
    } finally {
      setSavingTariff(false);
    }
  }

  async function handleGenerateICA() {
    if (!icaMunicipality || !icaCiiu) return;
    setIcaGenerating(true);
    setError(null);
    try {
      await generateICA({ municipality_code: icaMunicipality, ciiu_code: icaCiiu, period_start: icaFrom, period_end: icaTo, period_type: "ANUAL" });
      toast.success("Declaración de ICA generada.");
      refreshICA();
    } catch (err) {
      setError(err instanceof ApiError ? err.message : "No se pudo generar la declaración de ICA");
    } finally {
      setIcaGenerating(false);
    }
  }

  return (
    <div className="p-4">
      <Breadcrumbs items={[{ label: "Contabilidad", to: "/accounting" }, { label: "Declaraciones" }]} />
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <FileWarning className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Declaraciones de impuestos
      </h1>

      <div className="mb-3 flex h-9 w-fit overflow-hidden rounded border border-(--border-color)">
        {([["iva", "IVA"], ["renta", "Renta"], ["ica", "ICA"]] as [Tab, string][]).map(([t, label]) => (
          <button key={t} type="button" onClick={() => setTab(t)} className={`px-3 text-xs font-medium transition-colors ${tab === t ? "bg-(--accent-primary) text-white" : "bg-(--bg-secondary) text-(--text-secondary) hover:bg-(--bg-hover)"}`}>
            {label}
          </button>
        ))}
      </div>

      {error && <Banner tone="danger">{error}</Banner>}

      {tab === "iva" && (
        <>
          <Card className="mb-3 p-4">
            <div className="flex flex-wrap items-end gap-2">
              <Input type="date" label="Desde" value={ivaFrom} onChange={(e) => setIvaFrom(e.target.value)} />
              <Input type="date" label="Hasta" value={ivaTo} onChange={(e) => setIvaTo(e.target.value)} />
              <Button type="button" loading={ivaGenerating} onClick={handleGenerateIVA}>Generar declaración</Button>
            </div>
          </Card>
          {ivaList === null ? (
            <div className="flex min-h-24 items-center justify-center"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>
          ) : ivaList.length === 0 ? (
            <p className="text-xs text-(--text-secondary)">Sin declaraciones de IVA todavía.</p>
          ) : (
            <div className="overflow-x-auto rounded border border-(--border-color)">
              <table className="w-full text-left text-xs">
                <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                  <tr>
                    <th className="px-3 py-2 font-medium">Periodo</th>
                    <th className="px-3 py-2 text-right font-medium">Generado</th>
                    <th className="px-3 py-2 text-right font-medium">Descontable</th>
                    <th className="px-3 py-2 text-right font-medium">A pagar</th>
                    <th className="px-3 py-2 text-right font-medium">Saldo a favor</th>
                    <th className="px-3 py-2 font-medium">Estado</th>
                    <th className="px-3 py-2" />
                  </tr>
                </thead>
                <tbody>
                  {ivaList.map((d, i) => (
                    <tr key={d.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                      <td className="px-3 py-2 text-(--text-primary)">{new Date(d.period_start).toLocaleDateString("es-CO")} – {new Date(d.period_end).toLocaleDateString("es-CO")}</td>
                      <td className="px-3 py-2 text-right font-mono text-(--text-secondary)">{money(d.generated_iva)}</td>
                      <td className="px-3 py-2 text-right font-mono text-(--text-secondary)">{money(d.deductible_iva)}</td>
                      <td className="px-3 py-2 text-right font-mono text-(--text-primary)">{money(d.amount_to_pay)}</td>
                      <td className="px-3 py-2 text-right font-mono text-(--text-primary)">{money(d.carry_forward)}</td>
                      <td className="px-3 py-2"><StatusPill tone={DECL_TONE[d.status]} label={DECL_LABEL[d.status]} /></td>
                      <td className="px-3 py-2">
                        {d.status === "DRAFT" && (
                          <button type="button" onClick={() => fileIVA(d.id).then(refreshIVA)} className="text-(--accent-primary) hover:underline">Marcar presentada</button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {tab === "renta" && (
        <>
          <Card className="mb-3 p-4">
            <div className="flex flex-wrap items-end gap-2">
              <Input type="number" label="Año fiscal" value={incomeTaxYear} onChange={(e) => setIncomeTaxYear(Number(e.target.value))} className="w-28" />
              <Button type="button" loading={incomeTaxGenerating} onClick={handleGenerateIncomeTax}>Generar declaración</Button>
            </div>
          </Card>
          {incomeTaxList === null ? (
            <div className="flex min-h-24 items-center justify-center"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>
          ) : incomeTaxList.length === 0 ? (
            <p className="text-xs text-(--text-secondary)">Sin declaraciones de renta todavía.</p>
          ) : (
            <div className="overflow-x-auto rounded border border-(--border-color)">
              <table className="w-full text-left text-xs">
                <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                  <tr>
                    <th className="px-3 py-2 font-medium">Año</th>
                    <th className="px-3 py-2 text-right font-medium">Renta líquida gravable</th>
                    <th className="px-3 py-2 font-medium">Tarifa</th>
                    <th className="px-3 py-2 text-right font-medium">Impuesto a pagar</th>
                    <th className="px-3 py-2 font-medium">Estado</th>
                    <th className="px-3 py-2" />
                  </tr>
                </thead>
                <tbody>
                  {incomeTaxList.map((d, i) => (
                    <tr key={d.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                      <td className="px-3 py-2 text-(--text-primary)">{d.fiscal_year}</td>
                      <td className="px-3 py-2 text-right font-mono text-(--text-secondary)">{money(d.taxable_income)}</td>
                      <td className="px-3 py-2 text-(--text-secondary)">{(d.tax_rate_bp / 100).toFixed(1)}%</td>
                      <td className="px-3 py-2 text-right font-mono text-(--text-primary)">{money(d.tax_to_pay)}</td>
                      <td className="px-3 py-2"><StatusPill tone={DECL_TONE[d.status]} label={DECL_LABEL[d.status]} /></td>
                      <td className="px-3 py-2">
                        {d.status === "DRAFT" && (
                          <button type="button" onClick={() => fileIncomeTax(d.id).then(refreshIncomeTax)} className="text-(--accent-primary) hover:underline">Marcar presentada</button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}

      {tab === "ica" && (
        <>
          <Card className="mb-3 p-4">
            <div className="mb-3 flex items-center justify-between">
              <h3 className="text-xs font-semibold text-(--text-primary)">Tarifas registradas</h3>
              <Button type="button" variant="secondary" onClick={() => setShowTariffForm((v) => !v)}>
                {showTariffForm ? "Cancelar" : "Registrar tarifa"}
              </Button>
            </div>
            {showTariffForm && (
              <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-5">
                <Input label="Cód. municipio" value={tariffMunicipality} onChange={(e) => setTariffMunicipality(e.target.value)} placeholder="Ej. 11001" />
                <Input label="CIIU" value={tariffCiiu} onChange={(e) => setTariffCiiu(e.target.value)} placeholder="Ej. 4711" />
                <Input label="Año" type="number" value={tariffYear} onChange={(e) => setTariffYear(Number(e.target.value))} />
                <Input label="Tarifa %" type="number" step="0.001" value={tariffRate} onChange={(e) => setTariffRate(e.target.value)} placeholder="Ej. 1.0" />
                <Input label="Sobretasa %" type="number" step="0.001" value={tariffSurcharge} onChange={(e) => setTariffSurcharge(e.target.value)} />
                <div className="col-span-2 sm:col-span-5">
                  <Button type="button" disabled={!tariffMunicipality || !tariffCiiu || !tariffRate} loading={savingTariff} onClick={handleSetTariff}>Guardar tarifa</Button>
                </div>
              </div>
            )}
            {tariffs.length === 0 ? (
              <p className="text-xs text-(--text-secondary)">No hay tarifas de ICA registradas — regístralas con el valor exacto de tu municipio antes de generar una declaración.</p>
            ) : (
              <div className="flex flex-wrap gap-1.5">
                {tariffs.map((t) => (
                  <span key={t.id} className="rounded bg-(--bg-tertiary) px-2 py-1 text-[11px] text-(--text-secondary)">
                    {t.municipality_code} / CIIU {t.ciiu_code} / {t.fiscal_year}: {(t.rate_bp / 100).toFixed(2)}%
                  </span>
                ))}
              </div>
            )}
          </Card>

          <Card className="mb-3 p-4">
            <div className="flex flex-wrap items-end gap-2">
              <Select label="Municipio" value={icaMunicipality} onChange={(e) => setIcaMunicipality(e.target.value)}>
                <option value="">Elegir…</option>
                {[...new Set(tariffs.map((t) => t.municipality_code))].map((m) => <option key={m} value={m}>{m}</option>)}
              </Select>
              <Select label="CIIU" value={icaCiiu} onChange={(e) => setIcaCiiu(e.target.value)}>
                <option value="">Elegir…</option>
                {[...new Set(tariffs.filter((t) => t.municipality_code === icaMunicipality).map((t) => t.ciiu_code))].map((c) => <option key={c} value={c}>{c}</option>)}
              </Select>
              <Input type="date" label="Desde" value={icaFrom} onChange={(e) => setIcaFrom(e.target.value)} />
              <Input type="date" label="Hasta" value={icaTo} onChange={(e) => setIcaTo(e.target.value)} />
              <Button type="button" disabled={!icaMunicipality || !icaCiiu} loading={icaGenerating} onClick={handleGenerateICA}>Generar declaración</Button>
            </div>
          </Card>

          {icaList === null ? (
            <div className="flex min-h-24 items-center justify-center"><Spinner className="h-5 w-5 text-(--text-muted)" /></div>
          ) : icaList.length === 0 ? (
            <p className="text-xs text-(--text-secondary)">Sin declaraciones de ICA todavía.</p>
          ) : (
            <div className="overflow-x-auto rounded border border-(--border-color)">
              <table className="w-full text-left text-xs">
                <thead className="bg-(--bg-tertiary) text-(--text-secondary)">
                  <tr>
                    <th className="px-3 py-2 font-medium">Periodo</th>
                    <th className="px-3 py-2 font-medium">Municipio</th>
                    <th className="px-3 py-2 text-right font-medium">Ingresos brutos</th>
                    <th className="px-3 py-2 text-right font-medium">A pagar</th>
                    <th className="px-3 py-2 font-medium">Estado</th>
                    <th className="px-3 py-2" />
                  </tr>
                </thead>
                <tbody>
                  {icaList.map((d, i) => (
                    <tr key={d.id} className={i % 2 === 1 ? "bg-(--bg-secondary)" : "bg-(--bg-primary)"}>
                      <td className="px-3 py-2 text-(--text-primary)">{new Date(d.period_start).toLocaleDateString("es-CO")} – {new Date(d.period_end).toLocaleDateString("es-CO")}</td>
                      <td className="px-3 py-2 text-(--text-secondary)">{d.municipality_code}</td>
                      <td className="px-3 py-2 text-right font-mono text-(--text-secondary)">{money(d.gross_revenue)}</td>
                      <td className="px-3 py-2 text-right font-mono text-(--text-primary)">{money(d.amount_due)}</td>
                      <td className="px-3 py-2"><StatusPill tone={DECL_TONE[d.status]} label={DECL_LABEL[d.status]} /></td>
                      <td className="px-3 py-2">
                        {d.status === "DRAFT" && (
                          <button type="button" onClick={() => fileICA(d.id).then(refreshICA)} className="text-(--accent-primary) hover:underline">Marcar presentada</button>
                        )}
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </div>
  );
}
