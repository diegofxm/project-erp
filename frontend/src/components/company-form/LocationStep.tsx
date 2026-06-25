import { useEffect, useState } from "react";
import { listDepartments, listMunicipalities } from "../../lib/catalogs";
import type { CatalogEntry, Municipality } from "../../lib/types";
import { Input } from "../ui/Input";
import { Select } from "../ui/Select";
import type { StepProps } from "./IdentificationStep";

export function LocationStep({ form, setField }: StepProps) {
  const [departments, setDepartments] = useState<CatalogEntry[]>([]);
  const [municipalities, setMunicipalities] = useState<Municipality[]>([]);

  useEffect(() => {
    listDepartments().then(setDepartments);
  }, []);

  useEffect(() => {
    if (!form.department_code) {
      setMunicipalities([]);
      return;
    }
    listMunicipalities(form.department_code).then(setMunicipalities);
  }, [form.department_code]);

  function handleDepartmentChange(code: string) {
    setField("department_code", code);
    setField("municipality_code", "");
  }

  return (
    <div className="flex flex-col gap-3 p-4">
      <div className="grid grid-cols-2 gap-2">
        <Select label="Departamento" required value={form.department_code} onChange={(e) => handleDepartmentChange(e.target.value)}>
          <option value="">Selecciona…</option>
          {departments.map((d) => (
            <option key={d.code} value={d.code}>
              {d.name}
            </option>
          ))}
        </Select>
        <Select
          label="Municipio"
          required
          disabled={!form.department_code}
          value={form.municipality_code}
          onChange={(e) => setField("municipality_code", e.target.value)}
        >
          <option value="">{form.department_code ? "Selecciona…" : "Elige un departamento primero"}</option>
          {municipalities.map((m) => (
            <option key={m.code} value={m.code}>
              {m.name}
            </option>
          ))}
        </Select>
      </div>
      <Input label="Dirección" required value={form.address_line} onChange={(e) => setField("address_line", e.target.value)} />
      <div className="grid grid-cols-2 gap-2">
        <Input
          label="Correo de la empresa"
          type="email"
          required
          value={form.email}
          onChange={(e) => setField("email", e.target.value)}
        />
        <Input label="Teléfono" value={form.phone ?? ""} onChange={(e) => setField("phone", e.target.value)} />
      </div>
    </div>
  );
}
