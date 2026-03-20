import { useState } from "react";
import { Button } from "@/components/ui/button";
import { cn } from "@/lib/utils";
import { Check, ChevronRight, Shield, Users } from "lucide-react";
import { useRbacAudience } from "@/contexts/RbacAudienceContext";
import {
  Tooltip,
  TooltipTrigger,
  TooltipContent,
} from "@/components/ui/tooltip";

interface ContextSelectionStepProps {
  onComplete: (context: "admin" | "endUser") => void;
}

export function ContextSelectionStep({
  onComplete,
}: ContextSelectionStepProps) {
  const { audience, setAudience } = useRbacAudience();
  const [selectedContext, setSelectedContext] = useState<"admin" | "endUser">(
    audience,
  );

  const contexts = [
    {
      id: "admin" as const,
      title: "Admin Context",
      description: "Configure RBAC for privileged operators and administrators",
      icon: Shield,
      emoji: "🔐",
      tooltip: [
        "Manages system-wide access policies and tenant settings.",
        "Applied to internal operators, DevOps, and security teams.",
      ],
    },
    {
      id: "endUser" as const,
      title: "End User Context",
      description: "Configure RBAC for your application's end users",
      icon: Users,
      emoji: "👥",
      tooltip: [
        "Controls what your product's customers can see and do.",
        "Scoped to individual user permissions within your application.",
      ],
    },
  ];

  const handleContinue = () => {
    setAudience(selectedContext);
    onComplete(selectedContext);
  };

  return (
    <div className="space-y-4">
      <div className="grid grid-cols-1 gap-4">
        {contexts.map((context) => (
          <Tooltip key={context.id}>
            <TooltipTrigger asChild>
              <div
                className={cn(
                  "flex items-start gap-4 p-4 rounded-lg border cursor-pointer transition-all hover:border-primary/50",
                  selectedContext === context.id &&
                    "border-primary bg-primary/5 ring-2 ring-primary ring-offset-2",
                )}
                onClick={() => setSelectedContext(context.id)}
              >
                <div className="text-2xl">{context.emoji}</div>
                <div className="flex-1">
                  <h4 className="font-semibold text-sm">{context.title}</h4>
                  <p className="text-xs text-muted-foreground mt-1">
                    {context.description}
                  </p>
                </div>
                {selectedContext === context.id && (
                  <Check className="h-5 w-5 text-primary shrink-0" />
                )}
              </div>
            </TooltipTrigger>
            <TooltipContent
              side="left"
              className="max-w-[250px] text-[10px] leading-relaxed"
            >
              <p className="text-white">{context.tooltip[0]}</p>
              <p className="mt-1 opacity-80 text-white">{context.tooltip[1]}</p>
            </TooltipContent>
          </Tooltip>
        ))}
      </div>

      <div className="flex justify-center">
        <Button
          onClick={handleContinue}
          disabled={!selectedContext}
          variant="outline"
          className="bg-black text-white hover:bg-black/90 dark:bg-white dark:text-black dark:hover:bg-white/90 shadow-md hover:shadow-lg transition-all h-8 px-3 text-xs"
        >
          Continue with {selectedContext === "admin" ? "Admin" : "End User"}
          <ChevronRight className="ml-2 h-4 w-4" />
        </Button>
      </div>
    </div>
  );
}
