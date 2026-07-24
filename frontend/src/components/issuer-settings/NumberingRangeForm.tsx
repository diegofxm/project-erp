import { useState, type FormEvent } from "react";
import { useAuth } from "../../context/AuthContext";
import { listDianDocumentTypes } from "../../lib/catalogs";
import { useCatalog } from "../../lib/useCatalog";
import type { CreateNumberingRangePayload, IssuerEnvironment } from "../../lib/types";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import { Button } from "../ui/Button";

interface NumberingRangeFormProps {
  onSubmit: (payload: CreateNumberingRangePayload) => void;
  onCancel: () => void;
  loading: boolean;
}

const TODAY = new Date().toISOString().slice(0, 10);

// Registrar una resolución de numeración (POST /numbering-ranges) — technical_key solo aplica
// a Factura (CUFE); test_set_id solo aplica en habilitación, es el "set de pruebas" que la
// DIAN asigna para poder confirmar documentos durante el proceso de habilitación.
export function NumberingRangeForm({ onSubmit, onCancel, loading }: NumberingRangeFormProps) {
  const { activeIssuer } = useAuth();
  const { data: docTypes, loading: loadingDocTypes } = useCatalog(listDianDocumentTypes);
  const [docTypeCode, setDocTypeCode] = useState("01");
  const [prefix, setPrefix] = useState("");
  const [resolutionNumber, setResolutionNumber] = useState("");
  const [resolutionDate, setResolutionDate] = useState(TODAY);
  const [rangeFrom, setRangeFrom] = useState("");
  const [rangeTo, setRangeTo] = useState("");
  const [validFrom, setValidFrom] = useState(TODAY);
  const [validTo, setValidTo] = useState("");
  const [environment, setEnvironment] = useState<IssuerEnvironment>(activeIssuer?.environment ?? "2");
  const [technicalKey, setTechnicalKey] = useState("");
  const [testSetId, setTestSetId] = useState("");
  const [nextNumber, setNextNumber] = useState("");

  const isResolutionType = docTypeCode === "01" || docTypeCode === "05";

  function handleSubmit(e: FormEvent) {
    e.preventDefault();
    onSubmit({
      dian_document_type_code: docTypeCode,
      prefix,
      resolution_number: isResolutionType ? resolutionNumber : undefined,
      resolution_date: isResolutionType ? resolutionDate : undefined,
      range_from: Number(rangeFrom),
      range_to: rangeTo ? Number(rangeTo) : undefined,
      valid_from: isResolutionType ? validFrom : undefined,
      valid_to: isResolutionType ? validTo : undefined,
      environment,
      technical_key: docTypeCode === "01" ? technicalKey || undefined : undefined,
      test_set_id: environment === "2" ? testSetId || undefined : undefined,
      next_number: nextNumber ? Number(nextNumber) : undefined,
    });
  }

  return (
    <form className="flex flex-col gap-3 rounded border border-(--border-color) p-4" onSubmit={handleSubmit}>
      <div className="grid grid-cols-12 gap-3">
        <div className="col-span-12 sm:col-span-4">
          <Select
            label="Tipo de documento"
            required
            disabled={loadingDocTypes}
            value={docTypeCode}
            onChange={(e) => setDocTypeCode(e.target.value)}
          >
            {loadingDocTypes ? (
              <option>Cargando…</option>
            ) : (
              docTypes.map((t) => (
                <option key={t.code} value={t.code}>
                  {t.name}
                </option>
              ))
            )}
          </Select>
        </div>
        <div className="col-span-6 sm:col-span-3">
          <Select label="Ambiente" required value={environment} onChange={(e) => setEnvironment(e.target.value as IssuerEnvironment)}>
            <option value="2">Habilitación</option>
            <option value="1">Producción</option>
          </Select>
        </div>
        <div className="col-span-6 sm:col-span-2">
          <Input label="Prefijo" required value={prefix} onChange={(e) => setPrefix(e.target.value)} />
        </div>
        {isResolutionType && (
          <div className="col-span-12 sm:col-span-3">
            <Input label="Número de resolución" required value={resolutionNumber} onChange={(e) => setResolutionNumber(e.target.value)} />
          </div>
        )}

        <div className="col-span-6 sm:col-span-3">
          <Input label="Rango desde" type="number" required value={rangeFrom} onChange={(e) => setRangeFrom(e.target.value)} />
        </div>
        <div className="col-span-6 sm:col-span-3">
          <Input
            label="Rango hasta"
            type="number"
            required={isResolutionType}
            value={rangeTo}
            onChange={(e) => setRangeTo(e.target.value)}
          />
        </div>
        {isResolutionType && (
          <div className="col-span-6 sm:col-span-3">
            <Input
              label="Fecha de la resolución"
              type="date"
              required
              value={resolutionDate}
              onChange={(e) => setResolutionDate(e.target.value)}
            />
          </div>
        )}
        <div className="col-span-6 sm:col-span-3">
          <Input
            label="Próximo número (opcional)"
            type="number"
            value={nextNumber}
            onChange={(e) => setNextNumber(e.target.value)}
            placeholder="Vacío = inicio del rango"
          />
        </div>

        {isResolutionType && (
          <>
            <div className="col-span-12 sm:col-span-6">
              <Input label="Vigente desde" type="date" required value={validFrom} onChange={(e) => setValidFrom(e.target.value)} />
            </div>
            <div className="col-span-12 sm:col-span-6">
              <Input label="Vigente hasta" type="date" required value={validTo} onChange={(e) => setValidTo(e.target.value)} />
            </div>
          </>
        )}

        {docTypeCode === "01" && (
          <div className="col-span-12 sm:col-span-6">
            <Input label="Clave técnica (CUFE)" required value={technicalKey} onChange={(e) => setTechnicalKey(e.target.value)} />
          </div>
        )}
        {environment === "2" && (
          <div className="col-span-12 sm:col-span-6">
            <Input
              label="Set de pruebas (opcional)"
              value={testSetId}
              onChange={(e) => setTestSetId(e.target.value)}
              placeholder="ID del set de pruebas DIAN — vacío para habilitación libre"
            />
          </div>
        )}
      </div>
      <div className="flex gap-2">
        <Button type="button" variant="secondary" onClick={onCancel} className="flex-1">
          Cancelar
        </Button>
        <Button type="submit" loading={loading} className="flex-1">
          Registrar rango
        </Button>
      </div>
    </form>
  );
}
