import { BrowserRouter, Routes, Route, Navigate } from "react-router";
import { AuthProvider } from "./context/AuthContext";
import { ThemeProvider } from "./context/ThemeContext";
import { ToastProvider } from "./context/ToastContext";
import { ConfirmProvider } from "./context/ConfirmContext";
import { ProtectedRoute } from "./components/ProtectedRoute";
import { DashboardLayout } from "./components/DashboardLayout";
import { ConnectionBanner } from "./components/ConnectionBanner";
import { LoginPage } from "./pages/LoginPage";
import { RegisterPage } from "./pages/RegisterPage";
import { DashboardPage } from "./pages/DashboardPage";
import { SettingsPage } from "./pages/SettingsPage";
import { IssuersPage } from "./pages/IssuersPage";
import { CustomersPage } from "./pages/CustomersPage";
import { ProductsPage } from "./pages/ProductsPage";
import { InvoicesPage } from "./pages/InvoicesPage";
import { InvoiceEditorPage } from "./pages/InvoiceEditorPage";
import { PublicCustomerRegisterPage } from "./pages/PublicCustomerRegisterPage";

function App() {
  return (
    <ThemeProvider>
      <AuthProvider>
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
                    <Route path="/settings" element={<SettingsPage />} />
                    <Route path="/issuers" element={<IssuersPage />} />
                    <Route path="/customers" element={<CustomersPage />} />
                    <Route path="/products" element={<ProductsPage />} />
                    <Route path="/documents/invoices" element={<InvoicesPage />} />
                    <Route path="/documents/invoices/:id" element={<InvoiceEditorPage />} />
                  </Route>
                </Route>
                <Route path="*" element={<Navigate to="/" replace />} />
              </Routes>
            </BrowserRouter>
          </ConfirmProvider>
        </ToastProvider>
      </AuthProvider>
    </ThemeProvider>
  );
}

export default App;
