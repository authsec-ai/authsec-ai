import { Button } from "@/components/ui/button";
import { SkipForward } from "lucide-react";

interface WizardFooterProps {
  onSkip: () => void;
  showSkip?: boolean;
}

export function WizardFooter({ onSkip, showSkip = true }: WizardFooterProps) {
  if (!showSkip) return null;

  return (
    <div className="px-4 py-3 border-t border-border bg-background/95">
      <Button
        variant="ghost"
        size="sm"
        onClick={onSkip}
        className="w-full text-muted-foreground hover:text-foreground"
      >
        <SkipForward className="mr-2 h-4 w-4" />
        Skip Tour
      </Button>
    </div>
  );
}
