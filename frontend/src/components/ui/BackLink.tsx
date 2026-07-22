import { ChevronLeft } from "lucide-react";
import { Link } from "react-router";

interface BackLinkProps {
  to: string;
  label: string;
}

export function BackLink({ to, label }: BackLinkProps) {
  return (
    <Link
      to={to}
      className="mb-2 inline-flex items-center gap-0.5 text-xs font-medium text-(--accent-primary) hover:underline"
    >
      <ChevronLeft className="h-3.5 w-3.5" />
      {label}
    </Link>
  );
}
