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
import { SetupPasswordPage } from "./pages/SetupPasswordPage";
import { DashboardPage } from "./pages/DashboardPage";
import { SettingsGeneralPage } from "./pages/SettingsGeneralPage";
import { SettingsAccountPage } from "./pages/SettingsAccountPage";
import { SettingsCompanyPage } from "./pages/SettingsCompanyPage";
import { SettingsActivityPage } from "./pages/SettingsActivityPage";
import { IssuersPage } from "./pages/IssuersPage";
import { CustomersPage } from "./pages/CustomersPage";
import { VendorsPage } from "./pages/VendorsPage";
import { ProductsPage } from "./pages/ProductsPage";
import { InvoicesPage } from "./pages/InvoicesPage";
import { InvoiceEditorPage } from "./pages/InvoiceEditorPage";
import { CreditNotesPage } from "./pages/CreditNotesPage";
import { CreditNoteEditorPage } from "./pages/CreditNoteEditorPage";
import { DebitNotesPage } from "./pages/DebitNotesPage";
import { DebitNoteEditorPage } from "./pages/DebitNoteEditorPage";
import { SupportDocumentsPage } from "./pages/SupportDocumentsPage";
import { SupportDocumentEditorPage } from "./pages/SupportDocumentEditorPage";
import { AdjustmentNotesPage } from "./pages/AdjustmentNotesPage";
import { AdjustmentNoteEditorPage } from "./pages/AdjustmentNoteEditorPage";
import { PublicCustomerRegisterPage } from "./pages/PublicCustomerRegisterPage";
import { AdminBillingPage, AdminRenewalsPage, AdminIssuerPage, AdminPlansPage, AdminUsersPage, AdminProspectsPage } from "./pages/AdminPage";

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
                <Route path="/setup-password" element={<SetupPasswordPage />} />
                <Route path="/r/:issuerId" element={<PublicCustomerRegisterPage />} />
                <Route element={<ProtectedRoute />}>
                  <Route element={<DashboardLayout />}>
                    <Route path="/" element={<DashboardPage />} />
                    <Route path="/settings" element={<Navigate to="/settings/general" replace />} />
                    <Route path="/settings/general" element={<SettingsGeneralPage />} />
                    <Route path="/settings/account" element={<SettingsAccountPage />} />
                    <Route path="/settings/company" element={<SettingsCompanyPage />} />
                    <Route path="/settings/activity" element={<SettingsActivityPage />} />
                    <Route path="/issuers" element={<IssuersPage />} />
                    <Route path="/customers" element={<CustomersPage />} />
                    <Route path="/vendors" element={<VendorsPage />} />
                    <Route path="/products" element={<ProductsPage />} />
                    <Route path="/documents/invoices" element={<InvoicesPage />} />
                    <Route path="/documents/invoices/:id" element={<InvoiceEditorPage />} />
                    <Route path="/documents/credit-notes" element={<CreditNotesPage />} />
                    <Route path="/documents/credit-notes/:id" element={<CreditNoteEditorPage />} />
                    <Route path="/documents/debit-notes" element={<DebitNotesPage />} />
                    <Route path="/documents/debit-notes/:id" element={<DebitNoteEditorPage />} />
                    <Route path="/documents/support-documents" element={<SupportDocumentsPage />} />
                    <Route path="/documents/support-documents/:id" element={<SupportDocumentEditorPage />} />
                    <Route path="/documents/adjustment-notes" element={<AdjustmentNotesPage />} />
                    <Route path="/documents/adjustment-notes/:id" element={<AdjustmentNoteEditorPage />} />
                    <Route path="/admin" element={<Navigate to="/admin/billing" replace />} />
                    <Route path="/admin/billing" element={<AdminBillingPage />} />
                    <Route path="/admin/renewals" element={<AdminRenewalsPage />} />
                    <Route path="/admin/issuer" element={<AdminIssuerPage />} />
                    <Route path="/admin/plans" element={<AdminPlansPage />} />
                    <Route path="/admin/users" element={<AdminUsersPage />} />
                    <Route path="/admin/prospects" element={<AdminProspectsPage />} />
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
