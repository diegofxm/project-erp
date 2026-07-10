import { User } from "lucide-react";
import { Card } from "../components/ui/Card";
import { useAuth } from "../context/AuthContext";

export function SettingsAccountPage() {
  const { user } = useAuth();

  return (
    <div className="p-4">
      <h1 className="mb-3 flex items-center gap-2 text-sm font-semibold text-(--text-primary)">
        <User className="h-4 w-4 shrink-0 text-(--accent-primary)" />
        Mi cuenta
      </h1>
      <div className="flex flex-wrap gap-3">
        <Card className="p-3 text-xs">
          <p className="text-(--text-secondary)">Nombre</p>
          <p className="text-(--text-primary)">{user?.name}</p>
        </Card>
        <Card className="p-3 text-xs">
          <p className="text-(--text-secondary)">Correo</p>
          <p className="text-(--text-primary)">{user?.email}</p>
        </Card>
      </div>
    </div>
  );
}
