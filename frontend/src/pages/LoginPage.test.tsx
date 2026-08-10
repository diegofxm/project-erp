import { describe, it, expect, vi, beforeEach } from "vitest";
import { render, screen, waitFor } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { MemoryRouter, Routes, Route } from "react-router";
import { AuthProvider } from "../context/AuthContext";
import { LoginPage } from "./LoginPage";

// Smoke test del flujo crítico "iniciar sesión" (ver docs/auditorias/2026-08-09/plan-de-accion.md
// punto 25). Monta LoginPage con su árbol de contexto real (AuthProvider real, sin mockear) y solo
// intercepta fetch() -- así se prueba el camino completo LoginPage -> AuthContext.login ->
// apiClient.post -> fetch, no una versión simplificada del flujo.
function renderLoginPage() {
  return render(
    <MemoryRouter initialEntries={["/login"]}>
      <AuthProvider>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="/" element={<div>Panel principal</div>} />
        </Routes>
      </AuthProvider>
    </MemoryRouter>,
  );
}

function mockFetchOnce(status: number, body: unknown) {
  return vi.spyOn(globalThis, "fetch").mockResolvedValueOnce({
    ok: status >= 200 && status < 300,
    status,
    text: async () => JSON.stringify(body),
  } as Response);
}

describe("LoginPage", () => {
  beforeEach(() => {
    localStorage.clear();
    vi.restoreAllMocks();
  });

  it("inicia sesión con credenciales correctas y navega al panel principal", async () => {
    const user = userEvent.setup();
    mockFetchOnce(200, {
      token: "fake-jwt-token",
      user: { id: "u1", email: "diego@example.com", name: "Diego", role: "owner", is_superadmin: false },
      company_id: null,
    });

    renderLoginPage();

    await user.type(screen.getByLabelText(/^Correo/), "diego@example.com");
    await user.type(screen.getByLabelText(/^Contraseña/), "secret123");
    await user.click(screen.getByRole("button", { name: "Iniciar sesión" }));

    await waitFor(() => expect(screen.getByText("Panel principal")).toBeInTheDocument());

    // El token quedó guardado para futuras peticiones -- señal de que applyAuthResult() corrió
    // completo, no solo la navegación.
    expect(localStorage.getItem("apidian.session")).toContain("fake-jwt-token");
  });

  it("muestra el mensaje de error del servidor cuando las credenciales son incorrectas", async () => {
    const user = userEvent.setup();
    mockFetchOnce(401, { error: "correo o contraseña incorrectos" });

    renderLoginPage();

    await user.type(screen.getByLabelText(/^Correo/), "diego@example.com");
    await user.type(screen.getByLabelText(/^Contraseña/), "mala-contraseña");
    await user.click(screen.getByRole("button", { name: "Iniciar sesión" }));

    expect(await screen.findByText("correo o contraseña incorrectos")).toBeInTheDocument();
    // Sigue en /login -- no navegó por error.
    expect(screen.queryByText("Panel principal")).not.toBeInTheDocument();
  });
});
