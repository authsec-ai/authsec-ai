import React, { useState } from "react";
import { Button } from "./button";
import { Copy, Check } from "lucide-react";
import { toast } from "react-hot-toast";
import { cn } from "../../lib/utils";

interface CopyButtonProps {
  text: string;
  label?: string;
  size?: "sm" | "md" | "lg";
  variant?: "outline" | "ghost" | "secondary";
  className?: string;
  showLabel?: boolean;
}

export function CopyButton({
  text,
  label = "Copy",
  size = "sm",
  variant = "ghost",
  className = "",
  showLabel = false
}: CopyButtonProps) {
  const [copied, setCopied] = useState(false);
  const iconOnly = !showLabel;

  const handleCopy = async () => {
    try {
      await navigator.clipboard.writeText(text);
      setCopied(true);
      toast.success(`${label} copied to clipboard`);

      // Reset after 2 seconds
      setTimeout(() => {
        setCopied(false);
      }, 2000);
    } catch (err) {
      toast.error("Failed to copy to clipboard");
    }
  };

  return (
    <Button
      size={iconOnly ? "icon" : size}
      variant={variant}
      onClick={handleCopy}
      className={cn(
        "transition-all duration-200",
        iconOnly ? "h-8 w-8 p-0" : "h-8 px-3",
        className
      )}
      title={`Copy ${label}`}
    >
      {copied ? (
        <>
          <Check className="h-3 w-3 text-green-600" />
          {showLabel && <span className="ml-1.5 text-xs font-medium">Copied!</span>}
        </>
      ) : (
        <>
          <Copy className="h-3 w-3" />
          {showLabel && <span className="ml-1.5 text-xs font-medium">{label}</span>}
        </>
      )}
    </Button>
  );
}
