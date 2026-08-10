import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { apiClient, ApiError, setAuthToken, setSessionExpiredHandler } from "../lib/apiClient";
import type {
  AuthResult,
  Company,
  CreateCompanyPayload,
  LoginPayload,
  RegisterPayload,
  UpdateCompanyPayload,
  UpdateCompanyProfilePayload,
  User,
} from "../lib/types";

const STORAGE_KEY = "apidian.session";

interface StoredSession {
  token: string;
  user: User;
  company: Company | null;
}

interface AuthContextValue {
  user: User | null;
  activeCompany: Company | null;
  isAuthenticated: boolean;
  isReady: boolean;
  connectionError: boolean;
  retryConnection: () => Promise<void>;
  login: (payload: LoginPayload) => Promise<void>;
  acceptInvite: (token: string, password: string) => Promise<void>;
  register: (payload: RegisterPayload) => Promise<void>;
  logout: () => void;
  listCompanies: () => Promise<Company[]>;
  createCompany: (payload: CreateCompanyPayload) => Promise<void>;
  selectCompany: (id: string) => Promise<void>;
  updateProfile: (name: string, email: string) => Promise<User>;
  changePassword: (currentPassword: string, newPassword: string) => Promise<void>;
  updateCompany: (payload: UpdateCompanyPayload) => Promise<Company>;
  updateCompanyProfile: (payload: UpdateCompanyProfilePayload) => Promise<Company>;
  deleteCompanyLogo: () => Promise<Company>;
  deleteCompanySoftware: () => Promise<Company>;
  deleteCompanyNeSoftware: () => Promise<Company>;
  deleteCompanyCertificate: () => Promise<Company>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

function readStoredSession(): StoredSession | null {
  const raw = localStorage.getItem(STORAGE_KEY);
  if (!raw) return null;
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>;
    // Migración: sesiones antiguas usan la clave "issuer"
    if ("issuer" in parsed && !("company" in parsed)) {
      parsed.company = parsed.issuer;
    }
    return parsed as unknown as StoredSession;
  } catch {
    return null;
  }
}

function writeStoredSession(session: StoredSession) {
  localStorage.setItem(STORAGE_KEY, JSON.stringify(session));
}

async function fetchCompany(): Promise<Company | null> {
  try {
    return await apiClient.get<Company>("/companies/active");
  } catch {
    return null;
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [activeCompany, setActiveCompany] = useState<Company | null>(null);
  const [isReady, setIsReady] = useState(false);
  const [connectionError, setConnectionError] = useState(false);
  // Evita que varios 401 casi simultáneos (ej. varias llamadas en paralelo al expirar la sesión)
  // disparen logout() más de una vez -- se reinicia en cada login/verificación exitosa.
  const sessionExpiredHandledRef = useRef(false);

  const applyAuthResult = useCallback(async (result: AuthResult) => {
    setAuthToken(result.token);
    setUser(result.user);
    sessionExpiredHandledRef.current = false;

    let company: Company | null = null;
    if (result.company_id) {
      company = await fetchCompany();
    }

    const session: StoredSession = { token: result.token, user: result.user, company };
    writeStoredSession(session);
    setActiveCompany(company);
  }, []);

  const login = useCallback(
    async (payload: LoginPayload) => {
      const result = await apiClient.post<AuthResult>("/auth/login", payload);
      await applyAuthResult(result);
    },
    [applyAuthResult],
  );

  const acceptInvite = useCallback(
    async (token: string, password: string) => {
      const result = await apiClient.post<AuthResult>("/auth/accept-invite", { token, password });
      await applyAuthResult(result);
    },
    [applyAuthResult],
  );

  const register = useCallback(
    async (payload: RegisterPayload) => {
      const result = await apiClient.post<AuthResult>("/auth/register", payload);
      await applyAuthResult(result);
    },
    [applyAuthResult],
  );

  const logout = useCallback(() => {
    // Revoca la sesión del lado del servidor (ver LogoutUseCase) -- best-effort: no bloquea el
    // logout local si falla o si no hay red, el usuario siempre debe poder salir localmente.
    apiClient.post("/auth/logout").catch(() => {});
    localStorage.removeItem(STORAGE_KEY);
    setAuthToken(null);
    setUser(null);
    setActiveCompany(null);
    setConnectionError(false);
  }, []);

  // Interceptor global de 401 (ver docs/auditorias/2026-08-09/05-frontend.md punto 21): cualquier
  // llamada autenticada que reciba 401 (fuera de las rutas exentas en apiClient.ts, como
  // /auth/password) dispara logout() automáticamente en vez de dejar que cada página muestre un
  // error genérico. ProtectedRoute ya redirige a /login reactivamente en cuanto isAuthenticated
  // pasa a false -- no hace falta navegar explícitamente desde acá (AuthProvider está fuera del
  // Router, no tiene acceso a useNavigate).
  useEffect(() => {
    setSessionExpiredHandler(() => {
      if (sessionExpiredHandledRef.current) return;
      sessionExpiredHandledRef.current = true;
      logout();
    });
    return () => setSessionExpiredHandler(null);
  }, [logout]);

  const verifySession = useCallback(async () => {
    const stored = readStoredSession();
    try {
      // Refresca el usuario contra /auth/me (no solo lo usa como ping) — así campos que cambian
      // fuera de un login normal (ej. is_superadmin, otorgado desde /admin/users) se reflejan al
      // recargar la página, sin necesitar cerrar e iniciar sesión de nuevo.
      const freshUser = await apiClient.get<User>("/auth/me");
      setUser(freshUser);
      if (stored) writeStoredSession({ ...stored, user: freshUser });

      if (stored?.company?.id) {
        const company = await fetchCompany();
        if (company && stored) {
          writeStoredSession({ ...stored, company });
          setActiveCompany(company);
        }
      }
      setConnectionError(false);
      sessionExpiredHandledRef.current = false;
    } catch (err) {
      if (err instanceof ApiError) {
        // 401 ya lo maneja el interceptor global de apiClient.ts (dispara logout() solo). 404
        // significa que el usuario ya no existe (ej. cuenta eliminada) -- eso el interceptor no lo
        // cubre porque no es un status de sesión inválida en el resto del API, así que sigue
        // manejándose acá explícitamente.
        if (err.status === 404) logout();
        setConnectionError(false);
      } else {
        setConnectionError(true);
      }
    }
  }, [logout]);

  useEffect(() => {
    const stored = readStoredSession();
    if (!stored) {
      setIsReady(true);
      return;
    }
    setAuthToken(stored.token);
    setUser(stored.user);
    setActiveCompany(stored.company);
    verifySession().finally(() => setIsReady(true));
  }, []);

  const retryConnection = useCallback(async () => {
    await verifySession();
  }, [verifySession]);

  const listCompanies = useCallback(async () => {
    try {
      return await apiClient.get<Company[]>("/companies");
    } catch {
      return [];
    }
  }, []);

  const createCompany = useCallback(
    async (payload: CreateCompanyPayload) => {
      const company = await apiClient.post<Company>("/companies", payload);
      const result = await apiClient.post<AuthResult>("/auth/select-company", { company_id: company.id });
      await applyAuthResult(result);
    },
    [applyAuthResult],
  );

  const selectCompany = useCallback(
    async (id: string) => {
      const result = await apiClient.post<AuthResult>("/auth/select-company", { company_id: id });
      await applyAuthResult(result);
    },
    [applyAuthResult],
  );

  const updateProfile = useCallback(async (name: string, email: string) => {
    const updated = await apiClient.put<User>("/auth/profile", { name, email });
    setUser(updated);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, user: updated });
    return updated;
  }, []);

  const changePassword = useCallback(async (currentPassword: string, newPassword: string) => {
    // El backend revoca todas las sesiones activas al cambiar la contraseña (ver
    // ChangePasswordUseCase) y devuelve un token fresco ya válido para no dejar a este mismo
    // dispositivo deslogueado -- hay que guardarlo, si no la sesión actual empieza a fallar en
    // el siguiente request.
    const { token } = await apiClient.put<{ token: string }>("/auth/password", {
      current_password: currentPassword,
      new_password: newPassword,
    });
    setAuthToken(token);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, token });
  }, []);

  const updateCompany = useCallback(async (payload: UpdateCompanyPayload) => {
    let updated: Company;
    if (payload.logo_base64) {
      updated = await apiClient.put<Company>("/companies/active/logo", {
        logo_base64: payload.logo_base64,
        logo_content_type: payload.logo_content_type,
      });
    } else {
      updated = await apiClient.put<Company>("/companies/active/credentials", payload);
    }
    setActiveCompany(updated);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, company: updated });
    return updated;
  }, []);

  const updateCompanyProfile = useCallback(async (payload: UpdateCompanyProfilePayload) => {
    const updated = await apiClient.put<Company>("/companies/active", payload);
    setActiveCompany(updated);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, company: updated });
    return updated;
  }, []);

  const deleteCompanyLogo = useCallback(async () => {
    const updated = await apiClient.del<Company>("/companies/active/logo");
    setActiveCompany(updated);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, company: updated });
    return updated;
  }, []);

  const deleteCompanySoftware = useCallback(async () => {
    const company = await apiClient.del<Company>("/companies/active/credentials/software");
    setActiveCompany(company);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, company });
    return company;
  }, []);

  const deleteCompanyNeSoftware = useCallback(async () => {
    const company = await apiClient.del<Company>("/companies/active/credentials/ne-software");
    setActiveCompany(company);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, company });
    return company;
  }, []);

  const deleteCompanyCertificate = useCallback(async () => {
    const company = await apiClient.del<Company>("/companies/active/credentials/certificate");
    setActiveCompany(company);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, company });
    return company;
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      activeCompany,
      isAuthenticated: user !== null,
      isReady,
      connectionError,
      retryConnection,
      login,
      acceptInvite,
      register,
      logout,
      listCompanies,
      createCompany,
      selectCompany,
      updateProfile,
      changePassword,
      updateCompany,
      updateCompanyProfile,
      deleteCompanyLogo,
      deleteCompanySoftware,
      deleteCompanyNeSoftware,
      deleteCompanyCertificate,
    }),
    [
      user,
      activeCompany,
      isReady,
      connectionError,
      retryConnection,
      login,
      acceptInvite,
      register,
      logout,
      listCompanies,
      createCompany,
      selectCompany,
      updateProfile,
      changePassword,
      updateCompany,
      updateCompanyProfile,
      deleteCompanyLogo,
      deleteCompanySoftware,
      deleteCompanyNeSoftware,
      deleteCompanyCertificate,
    ],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth debe usarse dentro de <AuthProvider>");
  return ctx;
}
