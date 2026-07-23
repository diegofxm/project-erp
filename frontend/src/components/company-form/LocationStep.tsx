import { useEffect, useState } from "react";
import { listDepartments, listMunicipalities } from "../../lib/catalogs";
import { useCatalog } from "../../lib/useCatalog";
import type { Municipality } from "../../lib/types";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import type { StepProps } from "./IdentificationStep";

export function LocationStep({ form, setField }: StepProps) {
  const { data: departments, loading: loadingDepartments } = useCatalog(listDepartments);
  const [municipalities, setMunicipalities] = useState<Municipality[]>([]);
  const [loadingMunicipalities, setLoadingMunicipalities] = useState(false);

  useEffect(() => {
    if (!form.department_code) {
      setMunicipalities([]);
      return;
    }
    setLoadingMunicipalities(true);
    listMunicipalities(form.department_code)
      .then(setMunicipalities)
      .finally(() => setLoadingMunicipalities(false));
  }, [form.department_code]);

  function handleDepartmentChange(code: string) {
    setField("department_code", code);
    setField("municipality_code", "");
  }

  // Grilla de 12 columnas fijas, ver IdentificationStep.tsx para la explicación de por qué
  // (no auto-fit: no debe reordenar campos al colapsar/expandir el Sidebar).
  return (
    <div className="grid grid-cols-12 gap-3 p-4">
      <div className="col-span-6 sm:col-span-3">
        <Select
          label="Departamento"
          required
          disabled={loadingDepartments}
          value={form.department_code}
          onChange={(e) => handleDepartmentChange(e.target.value)}
        >
          {loadingDepartments ? (
            <option>Cargando…</option>
          ) : (
            <>
              <option value="">Selecciona…</option>
              {departments.map((d) => (
                <option key={d.code} value={d.code}>
                  {d.name}
                </option>
              ))}
            </>
          )}
        </Select>
      </div>
      <div className="col-span-6 sm:col-span-5">
        <Select
          label="Municipio"
          required
          disabled={!form.department_code || loadingMunicipalities}
          value={form.municipality_code}
          onChange={(e) => setField("municipality_code", e.target.value)}
        >
          {loadingMunicipalities ? (
            <option>Cargando…</option>
          ) : (
            <>
              <option value="">{form.department_code ? "Selecciona…" : "Elige un departamento primero"}</option>
              {municipalities.map((m) => (
                <option key={m.code} value={m.code}>
                  {m.name}
                </option>
              ))}
            </>
          )}
        </Select>
      </div>
      <div className="col-span-12 sm:col-span-4">
        <Input label="Dirección" required value={form.address_line} onChange={(e) => setField("address_line", e.target.value)} />
      </div>
      <div className="col-span-12 sm:col-span-8">
        <Input
          label="Correo de la empresa"
          type="email"
          required
          value={form.email}
          onChange={(e) => setField("email", e.target.value)}
        />
      </div>
      <div className="col-span-12 sm:col-span-4">
        <Input label="Teléfono" value={form.phone ?? ""} onChange={(e) => setField("phone", e.target.value)} />
      </div>
    </div>
  );
}
