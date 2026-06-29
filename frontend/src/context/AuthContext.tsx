import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { apiClient, ApiError, setAuthToken } from "../lib/apiClient";
import type {
  AuthResult,
  CreateIssuerPayload,
  Issuer,
  ListIssuersResult,
  LoginPayload,
  RegisterPayload,
  UpdateIssuerPayload,
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
  isReady: boolean; // ya se intentó rehidratar la sesión desde localStorage
  // connectionError: el backend no respondió al validar la sesión guardada (servidor apagado,
  // sin red) — distinto de un token inválido (401 real, ese sí cierra sesión). No se castiga
  // al usuario por algo que no es culpa de sus credenciales.
  connectionError: boolean;
  retryConnection: () => Promise<void>;
  login: (payload: LoginPayload) => Promise<void>;
  register: (payload: RegisterPayload) => Promise<void>;
  logout: () => void;
  listIssuers: () => Promise<Issuer[]>;
  createIssuer: (payload: CreateIssuerPayload) => Promise<void>;
  selectIssuer: (id: string) => Promise<void>;
  updateIssuer: (payload: UpdateIssuerPayload) => Promise<Issuer>;
  deleteIssuerLogo: () => Promise<Issuer>;
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

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<User | null>(null);
  const [activeIssuer, setActiveIssuer] = useState<Issuer | null>(null);
  const [isReady, setIsReady] = useState(false);
  const [connectionError, setConnectionError] = useState(false);

  const applyAuthResult = useCallback((result: AuthResult) => {
    const session: StoredSession = { token: result.token, user: result.user, issuer: result.issuer ?? null };
    setAuthToken(session.token);
    writeStoredSession(session);
    setUser(session.user);
    setActiveIssuer(session.issuer);
  }, []);

  const login = useCallback(
    async (payload: LoginPayload) => {
      const result = await apiClient.post<AuthResult>("/auth/login", payload);
      applyAuthResult(result);
    },
    [applyAuthResult],
  );

  const register = useCallback(
    async (payload: RegisterPayload) => {
      const result = await apiClient.post<AuthResult>("/auth/register", payload);
      applyAuthResult(result);
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

  // verifySession confirma que el token guardado todavía sirve Y que el backend responde —
  // GET /issuers (handleNoTenant en apidian) exige solo estar autenticado, nunca una empresa
  // activa, así que funciona sin importar si activeIssuer es null. Un 401 real (token
  // inválido/expirado) sí cierra sesión; cualquier otra falla (fetch nunca llegó a responder
  // porque el backend está apagado) NO toca la sesión, solo prende connectionError — no es
  // culpa de las credenciales que el servidor esté caído.
  const verifySession = useCallback(async () => {
    try {
      await apiClient.get("/issuers");
      setConnectionError(false);
    } catch (err) {
      if (err instanceof ApiError) {
        if (err.status === 401) logout();
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
    // Solo al montar a propósito — verifySession/logout no cambian de identidad entre renders
    // (useCallback sin dependencias variables), así que omitirlas del arreglo no esconde nada.
    verifySession().finally(() => setIsReady(true));
  }, []);

  const retryConnection = useCallback(async () => {
    await verifySession();
  }, [verifySession]);

  const listIssuers = useCallback(async () => {
    const result = await apiClient.get<ListIssuersResult>("/issuers");
    return result.issuers;
  }, []);

  const createIssuer = useCallback(
    async (payload: CreateIssuerPayload) => {
      const result = await apiClient.post<AuthResult>("/issuers", payload);
      applyAuthResult(result);
    },
    [applyAuthResult],
  );

  const selectIssuer = useCallback(
    async (id: string) => {
      const result = await apiClient.post<AuthResult>(`/issuers/${id}/select`, undefined);
      applyAuthResult(result);
    },
    [applyAuthResult],
  );

  // A diferencia de login/register/createIssuer/selectIssuer, PUT /issuers/me NO reemite el
  // token (sigue siendo la misma empresa activa, solo cambian sus datos) — se actualiza
  // activeIssuer y la sesión persistida directamente, sin pasar por applyAuthResult.
  const updateIssuer = useCallback(async (payload: UpdateIssuerPayload) => {
    const updated = await apiClient.put<Issuer>("/issuers/me", payload);
    setActiveIssuer(updated);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, issuer: updated });
    return updated;
  }, []);

  // Misma forma que updateIssuer — DELETE /issuers/me/logo tampoco reemite el token, solo
  // limpia has_logo en la empresa activa.
  const deleteIssuerLogo = useCallback(async () => {
    const updated = await apiClient.del<Issuer>("/issuers/me/logo");
    setActiveIssuer(updated);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, issuer: updated });
    return updated;
  }, []);

  const value = useMemo<AuthContextValue>(
    () => ({
      user,
      activeIssuer,
      isAuthenticated: user !== null,
      isReady,
      connectionError,
      retryConnection,
      login,
      register,
      logout,
      listIssuers,
      createIssuer,
      selectIssuer,
      updateIssuer,
      deleteIssuerLogo,
    }),
    [
      user,
      activeIssuer,
      isReady,
      connectionError,
      retryConnection,
      login,
      register,
      logout,
      listIssuers,
      createIssuer,
      selectIssuer,
      updateIssuer,
      deleteIssuerLogo,
    ],
  );

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>;
}

export function useAuth(): AuthContextValue {
  const ctx = useContext(AuthContext);
  if (!ctx) throw new Error("useAuth debe usarse dentro de <AuthProvider>");
  return ctx;
}
