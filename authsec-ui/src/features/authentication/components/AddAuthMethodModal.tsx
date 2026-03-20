import { useNavigate } from 'react-router-dom';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Button } from '@/components/ui/button';

interface AddAuthMethodModalProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
}

export function AddAuthMethodModal({ open, onOpenChange }: AddAuthMethodModalProps) {
  const navigate = useNavigate();

  const handleOidcSelect = () => {
    onOpenChange(false);
    navigate('/authentication/create');
  };

  const handleSamlSelect = () => {
    onOpenChange(false);
    navigate('/authentication/saml/create');
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>Add Authentication Method</DialogTitle>
          <DialogDescription>
            Choose the type of authentication method
          </DialogDescription>
        </DialogHeader>

        <div className="grid gap-3 py-4">
          <button
            onClick={handleOidcSelect}
            className="flex items-center gap-3 rounded-md border border-border p-3 text-left transition-all hover:bg-accent"
          >
            <div className="flex-1">
              <h3 className="font-medium text-sm">OIDC / OAuth 2.0</h3>
              <p className="text-xs text-foreground mt-0.5">
                Google, GitHub, or custom providers
              </p>
            </div>
          </button>

          <button
            onClick={handleSamlSelect}
            className="flex items-center gap-3 rounded-md border border-border p-3 text-left transition-all hover:bg-accent"
          >
            <div className="flex-1">
              <h3 className="font-medium text-sm">SAML 2.0</h3>
              <p className="text-xs text-foreground mt-0.5">
                Okta, Azure AD, or enterprise SSO
              </p>
            </div>
          </button>
        </div>

        <div className="flex justify-end">
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            Cancel
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  );
}
