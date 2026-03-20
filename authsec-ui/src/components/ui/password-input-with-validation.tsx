import React, { useState } from "react";
import { Eye, EyeOff, Check, X } from "lucide-react";
import { cn } from "@/lib/utils";
import { Input } from "./input";
import { validatePassword, type PasswordRequirement } from "@/utils/passwordValidation";

export interface PasswordInputWithValidationProps
  extends Omit<React.InputHTMLAttributes<HTMLInputElement>, 'type' | 'onChange'> {
  value: string;
  onChange: (value: string) => void;
  showValidation?: boolean;
  onValidationChange?: (isValid: boolean) => void;
}

export const PasswordInputWithValidation = React.forwardRef<
  HTMLInputElement,
  PasswordInputWithValidationProps
>(({ 
  className, 
  value, 
  onChange, 
  showValidation = true,
  onValidationChange,
  ...props 
}, ref) => {
  const [show, setShow] = useState(false);
  const validation = validatePassword(value);

  React.useEffect(() => {
    onValidationChange?.(validation.isValid);
  }, [validation.isValid, onValidationChange]);

  const getStrengthColor = () => {
    const metCount = validation.requirements.filter(req => req.met).length;
    if (metCount <= 1) return "bg-red-500";
    if (metCount <= 2) return "bg-orange-500"; 
    if (metCount <= 3) return "bg-yellow-500";
    if (metCount <= 4) return "bg-blue-500";
    return "bg-green-500";
  };

  const getStrengthText = () => {
    const metCount = validation.requirements.filter(req => req.met).length;
    if (metCount <= 1) return "Very Weak";
    if (metCount <= 2) return "Weak";
    if (metCount <= 3) return "Fair";
    if (metCount <= 4) return "Good";
    return "Strong";
  };

  return (
    <div className="space-y-2">
      <div className="relative">
        <Input
          type={show ? "text" : "password"}
          className={cn("pr-10", className)}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          ref={ref}
          {...props}
        />
        <button
          type="button"
          className="absolute right-0 top-0 h-full px-3 py-2 hover:bg-transparent"
          onClick={() => setShow(!show)}
          tabIndex={-1}
        >
          {show ? (
            <EyeOff className="h-4 w-4 text-foreground" />
          ) : (
            <Eye className="h-4 w-4 text-foreground" />
          )}
        </button>
      </div>

      {showValidation && value && (
        <div className="space-y-2">
          {/* Strength Indicator */}
          <div className="w-full bg-muted rounded-full h-1.5">
            <div
              className={cn("h-1.5 rounded-full transition-all duration-300", getStrengthColor())}
              style={{
                width: `${(validation.requirements.filter(req => req.met).length / validation.requirements.length) * 100}%`
              }}
            />
          </div>

          {/* Only show missing requirements */}
          {!validation.isValid && (
            <div className="space-y-1">
              {validation.requirements
                .filter(req => !req.met)
                .map((requirement, index) => (
                  <div key={index} className="flex items-center gap-2 text-xs text-destructive">
                    <X className="h-3 w-3" />
                    <span>{requirement.text}</span>
                  </div>
                ))}
            </div>
          )}
        </div>
      )}
    </div>
  );
});

PasswordInputWithValidation.displayName = "PasswordInputWithValidation";