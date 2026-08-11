import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor, within } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router";
import { AuthProvider } from "../context/AuthContext";
import { ConfirmProvider } from "../context/ConfirmContext";
import { ToastProvider } from "../context/ToastContext";
import { SaleEditorPage } from "./SaleEditorPage";
import type { Sale } from "../lib/types";

// Smoke test del flujo crítico "confirmación de venta" (ver
// docs/auditorias/2026-08-09/plan-de-accion.md punto 25). Monta la página real con su árbol de
// contexto real (Auth/Confirm/Toast, sin mockear ninguno) y solo intercepta fetch() por ruta --
// prueba el camino completo: cargar la venta, hacer clic en "Confirmar", confirmar el diálogo
// modal real (no window.confirm), y ver el estado actualizado en pantalla.

const SALE_ID = "sale-1";

function draftSale(): Sale {
  return {
    id: SALE_ID,
    company_id: "company-1",
    customer_id: "cust-1",
    number: "V-001",
    status: "draft",
    issue_date: "2026-08-01",
    notes: "",
    lines: [],
    created_at: "2026-08-01T00:00:00Z",
    updated_at: "2026-08-01T00:00:00Z",
  };
}

// Router de fetch mínimo: cada test declara qué responder según método+ruta. Evita depender del
// orden de llamadas (la página dispara varios fetch en paralelo al montar: /customers, /sales/:id).
function mockFetchRouter(routes: { method: string; path: string; status?: number; body: unknown }[]) {
  vi.spyOn(globalThis, "fetch").mockImplementation(async (input, init) => {
    const url = typeof input === "string" ? input : input.toString();
    const method = (init?.method ?? "GET").toUpperCase();
    const route = routes.find((r) => r.method === method && url.includes(r.path));
    if (!route) {
      throw new Error(`fetch no interceptado en el test: ${method} ${url}`);
    }
    const status = route.status ?? 200;
    return {
      ok: status >= 200 && status < 300,
      status,
      text: async () => JSON.stringify(route.body),
    } as Response;
  });
}

function seedSession() {
  localStorage.setItem(
    "apidian.session",
    JSON.stringify({
      token: "fake-jwt-token",
      user: { id: "u1", email: "diego@example.com", name: "Diego", role: "owner", is_superadmin: false },
      company: null,
    }),
  );
}

function renderSaleEditorPage() {
  return render(
    <MemoryRouter initialEntries={[`/sales/${SALE_ID}`]}>
      <AuthProvider>
        <ToastProvider>
          <ConfirmProvider>
            <Routes>
              <Route path="/sales/:id" element={<SaleEditorPage />} />
            </Routes>
          </ConfirmProvider>
        </ToastProvider>
      </AuthProvider>
    </MemoryRouter>,
  );
}

describe("SaleEditorPage — confirmación de venta", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("confirma una venta en borrador: pide confirmación, llama al backend y actualiza el estado en pantalla", async () => {
    const user = userEvent.setup();
    seedSession();

    const sale = draftSale();
    mockFetchRouter([
      { method: "GET", path: "/auth/me", body: JSON.parse(localStorage.getItem("apidian.session")!).user },
      { method: "GET", path: "/customers", body: { customers: [], count: 0 } },
      { method: "GET", path: `/sales/${SALE_ID}`, body: sale },
      { method: "POST", path: `/sales/${SALE_ID}/confirm`, body: { ...sale, status: "confirmed" } },
      { method: "GET", path: "/electronic/numbering-ranges", body: [] },
      { method: "GET", path: "/catalogs/payment-terms", body: { payment_terms: [] } },
      { method: "GET", path: "/catalogs/payment-methods", body: { payment_methods: [] } },
    ]);

    renderSaleEditorPage();

    // Espera a que cargue la venta real (no el spinner).
    const confirmButton = await screen.findByRole("button", { name: "Confirmar" });

    await user.click(confirmButton);

    // El diálogo modal real (ConfirmDialog, no window.confirm) debe aparecer con el mensaje real.
    const dialog = await screen.findByRole("dialog");
    expect(within(dialog).getByText(/¿Confirmar esta venta\?/)).toBeInTheDocument();
    await user.click(within(dialog).getByRole("button", { name: "Confirmar" }));

    // Tras confirmar: el backend recibió POST /sales/:id/confirm (implícito, si no el mock
    // hubiera lanzado "fetch no interceptado") y la UI refleja el nuevo estado.
    await waitFor(() => expect(screen.getByText("Confirmada")).toBeInTheDocument());
    // El botón "Confirmar" ya no debe estar disponible sobre una venta ya confirmada.
    expect(screen.queryByRole("button", { name: "Confirmar" })).not.toBeInTheDocument();
  });

  it("si el usuario cancela el diálogo, no llama a confirmar y la venta sigue en borrador", async () => {
    const user = userEvent.setup();
    seedSession();

    const sale = draftSale();
    mockFetchRouter([
      { method: "GET", path: "/auth/me", body: JSON.parse(localStorage.getItem("apidian.session")!).user },
      { method: "GET", path: "/customers", body: { customers: [], count: 0 } },
      { method: "GET", path: `/sales/${SALE_ID}`, body: sale },
      { method: "GET", path: "/catalogs/payment-terms", body: { payment_terms: [] } },
      { method: "GET", path: "/catalogs/payment-methods", body: { payment_methods: [] } },
      // Sin ruta para POST /confirm -- si el código llegara a llamarla, el mock lanza y el test falla.
    ]);

    renderSaleEditorPage();

    const confirmButton = await screen.findByRole("button", { name: "Confirmar" });
    await user.click(confirmButton);

    const dialog = await screen.findByRole("dialog");
    await user.click(within(dialog).getByRole("button", { name: "Cancelar" }));

    expect(screen.queryByText(/¿Confirmar esta venta\?/)).not.toBeInTheDocument();
    expect(screen.getByText("Borrador")).toBeInTheDocument();
  });
});
