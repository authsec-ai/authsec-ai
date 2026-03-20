import { Button } from "@/components/ui/button";
import { Globe, Plus } from "lucide-react";

interface DomainEmptyStateProps {
  onAddDomain: () => void;
}

export function DomainEmptyState({ onAddDomain }: DomainEmptyStateProps) {
  return (
    <div className="flex flex-col items-center justify-center py-16 px-4">
      <div className="rounded-full bg-muted p-4 mb-4">
        <Globe className="h-10 w-10 text-muted-foreground" />
      </div>

      <h3 className="text-xl font-semibold mb-2">No Custom Domains</h3>

      <p className="text-muted-foreground text-center max-w-md mb-6">
        Add your first custom domain to provide branded authentication
        experiences for your users. Custom domains allow you to use your own
        domain (e.g., auth.yourcompany.com) instead of the default platform
        subdomain.
      </p>

      <Button onClick={onAddDomain}>
        <Plus className="h-4 w-4 mr-2" />
        Add Your First Domain
      </Button>
    </div>
  );
}
