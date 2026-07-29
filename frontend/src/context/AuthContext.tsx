import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import { apiClient, ApiError, setAuthToken } from "../lib/apiClient";
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

  const applyAuthResult = useCallback(async (result: AuthResult) => {
    setAuthToken(result.token);
    setUser(result.user);

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
    localStorage.removeItem(STORAGE_KEY);
    setAuthToken(null);
    setUser(null);
    setActiveCompany(null);
    setConnectionError(false);
  }, []);

  const verifySession = useCallback(async () => {
    const stored = readStoredSession();
    try {
      await apiClient.get<User>("/auth/me");

      if (stored?.company?.id) {
        const company = await fetchCompany();
        if (company && stored) {
          writeStoredSession({ ...stored, company });
          setActiveCompany(company);
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

  const refreshCompanyState = useCallback(async (): Promise<Company> => {
    const company = await apiClient.get<Company>("/companies/active");
    setActiveCompany(company);
    const stored = readStoredSession();
    if (stored) writeStoredSession({ ...stored, company });
    return company;
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

  const deleteCompanySoftware = useCallback(() => refreshCompanyState(), [refreshCompanyState]);
  const deleteCompanyNeSoftware = useCallback(() => refreshCompanyState(), [refreshCompanyState]);
  const deleteCompanyCertificate = useCallback(() => refreshCompanyState(), [refreshCompanyState]);

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
