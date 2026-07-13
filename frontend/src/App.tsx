import { BrowserRouter, Routes, Route, Navigate } from "react-router";
import { AuthProvider } from "./context/AuthContext";
import { ThemeProvider } from "./context/ThemeContext";
import { ToastProvider } from "./context/ToastContext";
import { ConfirmProvider } from "./context/ConfirmContext";
import { NotificationProvider } from "./context/NotificationContext";
import { ProtectedRoute } from "./components/ProtectedRoute";
import { DashboardLayout } from "./components/DashboardLayout";
import { ConnectionBanner } from "./components/ConnectionBanner";
import { LoginPage } from "./pages/LoginPage";
import { RegisterPage } from "./pages/RegisterPage";
import { DashboardPage } from "./pages/DashboardPage";
import { SettingsGeneralPage } from "./pages/SettingsGeneralPage";
import { SettingsAccountPage } from "./pages/SettingsAccountPage";
import { SettingsCompanyPage } from "./pages/SettingsCompanyPage";
import { IssuersPage } from "./pages/IssuersPage";
import { CustomersPage } from "./pages/CustomersPage";
import { ProductsPage } from "./pages/ProductsPage";
import { InvoicesPage } from "./pages/InvoicesPage";
import { InvoiceEditorPage } from "./pages/InvoiceEditorPage";
import { CreditNotesPage } from "./pages/CreditNotesPage";
import { CreditNoteEditorPage } from "./pages/CreditNoteEditorPage";
import { DebitNotesPage } from "./pages/DebitNotesPage";
import { DebitNoteEditorPage } from "./pages/DebitNoteEditorPage";
import { PublicCustomerRegisterPage } from "./pages/PublicCustomerRegisterPage";

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
        <NotificationProvider>
        <ToastProvider>
          <ConfirmProvider>
            <BrowserRouter>
              <ConnectionBanner />
              <Routes>
                <Route path="/login" element={<LoginPage />} />
                <Route path="/register" element={<RegisterPage />} />
                <Route path="/r/:issuerId" element={<PublicCustomerRegisterPage />} />
                <Route element={<ProtectedRoute />}>
                  <Route element={<DashboardLayout />}>
                    <Route path="/" element={<DashboardPage />} />
                    <Route path="/settings" element={<Navigate to="/settings/general" replace />} />
                    <Route path="/settings/general" element={<SettingsGeneralPage />} />
                    <Route path="/settings/account" element={<SettingsAccountPage />} />
                    <Route path="/settings/company" element={<SettingsCompanyPage />} />
                    <Route path="/issuers" element={<IssuersPage />} />
                    <Route path="/customers" element={<CustomersPage />} />
                    <Route path="/products" element={<ProductsPage />} />
                    <Route path="/documents/invoices" element={<InvoicesPage />} />
                    <Route path="/documents/invoices/:id" element={<InvoiceEditorPage />} />
                    <Route path="/documents/credit-notes" element={<CreditNotesPage />} />
                    <Route path="/documents/credit-notes/:id" element={<CreditNoteEditorPage />} />
                    <Route path="/documents/debit-notes" element={<DebitNotesPage />} />
                    <Route path="/documents/debit-notes/:id" element={<DebitNoteEditorPage />} />
                  </Route>
                </Route>
                <Route path="*" element={<Navigate to="/" replace />} />
              </Routes>
            </BrowserRouter>
          </ConfirmProvider>
        </ToastProvider>
        </NotificationProvider>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
