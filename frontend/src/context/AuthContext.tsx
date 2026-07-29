import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { apiClient, ApiError, setAuthToken } from "../lib/apiClient";
import type {
  AuthResult,
  Company,
  CreateIssuerPayload,
  Issuer,
  LoginPayload,
  RegisterPayload,
  UpdateIssuerPayload,
  UpdateIssuerProfilePayload,
  User,
} from "../lib/types";

const STORAGE_KEY = "apidian.session";

interface StoredSession {
  token: string;
  user: User;
  issuer: Issuer | null;
}

interface AuthContextValue {
  user: User | null;
  activeIssuer: Issuer | null;
  isAuthenticated: boolean;
  isReady: boolean;
  connectionError: boolean;
  retryConnection: () => Promise<void>;
  login: (payload: LoginPayload) => Promise<void>;
  acceptInvite: (token: string, password: string) => Promise<void>;
  register: (payload: RegisterPayload) => Promise<void>;
  logout: () => void;
  listIssuers: () => Promise<Issuer[]>;
  createIssuer: (payload: CreateIssuerPayload) => Promise<void>;
  selectIssuer: (id: string) => Promise<void>;
  updateProfile: (name: string, email: string) => Promise<User>;
  updateIssuer: (payload: UpdateIssuerPayload) => Promise<Issuer>;
  updateIssuerProfile: (payload: UpdateIssuerProfilePayload) => Promise<Issuer>;
  deleteIssuerLogo: () => Promise<Issuer>;
  deleteIssuerSoftware: () => Promise<Issuer>;
  deleteIssuerNeSoftware: () => Promise<Issuer>;
  deleteIssuerCertificate: () => Promise<Issuer>;
}

const AuthContext = createContext<AuthContextValue | null>(null);

function readStoredSession(): StoredSession | null {
  const raw = localStorage.getItem(STORAGE_KEY);
  if (!raw) return null;
  try {
    return JSON.parse(raw) as StoredSession;
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
  const [activeIssuer, setActiveIssuer] = useState<Issuer | null>(null);
  const [isReady, setIsReady] = useState(false);
  const [connectionError, setConnectionError] = useState(false);

  // applyAuthResult: recibe {token, company_id, user} del ERP.
  // Si hay company_id, hace GET /companies/active para hidratar la empresa completa.
  const applyAuthResult = useCallback(async (result: AuthResult) => {
    setAuthToken(result.token);
    setUser(result.user);

    let company: Company | null = null;
    if (result.company_id) {
      company = await fetchCompany();
    }

    const session: StoredSession = { token: result.token, user: result.user, issuer: company };
    writeStoredSession(session);
    setActiveIssuer(company);
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
    localStorage.removeItem(STORAGE_KEY);
    setAuthToken(null);
    setUser(null);
    setActiveIssuer(null);
    setConnectionError(false);
  }, []);

  // verifySession: valida el token con GET /auth/me y refresca la empresa desde
  // GET /companies/active. Un 401 real cierra sesión; cualquier error de red solo activa
  // connectionError sin borrar las credenciales (el servidor puede estar apagado).
  const verifySession = useCallback(async () => {
    const stored = readStoredSession();
    try {
      await apiClient.get<User>("/auth/me");

      if (stored?.issuer?.id) {
        const company = await fetchCompany();
        if (company && stored) {
          writeStoredSession({ ...stored, issuer: company });
          setActiveIssuer(company);
        }
      }
      setConnectionError(false);
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 401 || err.status === 404) logout();
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
    setActiveIssuer(stored.issuer);
    verifySession().finally(() => setIsReady(true));
  }, []);

  const retryConnection = useCallback(async () => {
    await verifySession();
  }, [verifySession]);

  // listIssuers: devuelve todas las empresas del usuario autenticado.
  const listIssuers = useCallback(async () => {
    try {
      return await apiClient.get<Company[]>("/companies");
    } catch {
      return [];
    }
  }, []);

  const createIssuer = useCallback(
    async (payload: CreateIssuerPayload) => {
      // 1. Crea la empresa en el ERP
      const company = await apiClient.post<Company>("/companies", payload);
      // 2. Selecciona la empresa para emitir un token con company_id embebido
      const result = await apiClient.post<AuthResult>("/auth/select-company", { company_id: company.id });
      await applyAuthResult(result);
    },
    [applyAuthResult],
  );

  const selectIssuer = useCallback(
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

  // updateIssuer: enruta a /companies/active/logo si hay logo_base64, o a
  // /companies/active/credentials para software/certificado.
  const updateIssuer = useCallback(async (payload: UpdateIssuerPayload) => {
    let updated: Company;
    if (payload.logo_base64) {
      updated = await apiClient.put<Company>("/companies/active/logo", {
        logo_base64: payload.logo_base64,
        logo_content_type: payload.logo_content_type,
      });
    } else {
      updated = await apiClient.put<Company>("/companies/active/credentials", payload);
    }
    setActiveIssuer(updated);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, issuer: updated });
    return updated;
  }, []);

  const updateIssuerProfile = useCallback(async (payload: UpdateIssuerProfilePayload) => {
    const updated = await apiClient.put<Company>("/companies/active", payload);
    setActiveIssuer(updated);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, issuer: updated });
    return updated;
  }, []);

  const deleteIssuerLogo = useCallback(async () => {
    const updated = await apiClient.del<Company>("/companies/active/logo");
    setActiveIssuer(updated);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, issuer: updated });
    return updated;
  }, []);

  // El ERP no tiene endpoints de borrado de software/certificado. Se refrescan los datos
  // actuales desde el servidor para que el UI muestre el estado real.
  const refreshCompany = useCallback(async (): Promise<Company> => {
    const company = await apiClient.get<Company>("/companies/active");
    setActiveIssuer(company);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, issuer: company });
    return company;
  }, []);

  const deleteIssuerSoftware = useCallback(() => refreshCompany(), [refreshCompany]);
  const deleteIssuerNeSoftware = useCallback(() => refreshCompany(), [refreshCompany]);
  const deleteIssuerCertificate = useCallback(() => refreshCompany(), [refreshCompany]);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      activeIssuer,
      isAuthenticated: user !== null,
      isReady,
      connectionError,
      retryConnection,
      login,
      acceptInvite,
      register,
      logout,
      listIssuers,
      createIssuer,
      selectIssuer,
      updateProfile,
      updateIssuer,
      updateIssuerProfile,
      deleteIssuerLogo,
      deleteIssuerSoftware,
      deleteIssuerNeSoftware,
      deleteIssuerCertificate,
    }),
    [
      user,
      activeIssuer,
      isReady,
      connectionError,
      retryConnection,
      login,
      acceptInvite,
      register,
      logout,
      listIssuers,
      createIssuer,
      selectIssuer,
      updateProfile,
      updateIssuer,
      updateIssuerProfile,
      deleteIssuerLogo,
      deleteIssuerSoftware,
      deleteIssuerNeSoftware,
      deleteIssuerCertificate,
    ],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth debe usarse dentro de <AuthProvider>");
  return ctx;
}
