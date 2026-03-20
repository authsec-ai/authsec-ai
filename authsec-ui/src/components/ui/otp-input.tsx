import React, { useState, useRef, useEffect } from "react";
import { cn } from "@/lib/utils";

export interface OTPInputProps {
  length?: number;
  value: string;
  onChange: (value: string) => void;
  onComplete?: (value: string) => void;
  disabled?: boolean;
  className?: string;
}

export const OTPInput = React.forwardRef<HTMLDivElement, OTPInputProps>(
  ({ 
    length = 6, 
    value, 
    onChange, 
    onComplete, 
    disabled = false,
    className,
    ...props 
  }, ref) => {
    const [activeIndex, setActiveIndex] = useState(0);
    const inputRefs = useRef<(HTMLInputElement | null)[]>([]);

    const handleChange = (index: number, inputValue: string) => {
      if (disabled) return;
      
      // Only allow digits
      const digit = inputValue.replace(/\D/g, '');
      if (digit.length > 1) return;

      const newValue = value.split('');
      newValue[index] = digit;
      const updatedValue = newValue.join('');
      
      onChange(updatedValue);

      // Move to next input if digit entered
      if (digit && index < length - 1) {
        setActiveIndex(index + 1);
        inputRefs.current[index + 1]?.focus();
      }

      // Call onComplete if all digits filled
      if (updatedValue.length === length && onComplete) {
        onComplete(updatedValue);
      }
    };

    const handleKeyDown = (index: number, e: React.KeyboardEvent<HTMLInputElement>) => {
      if (disabled) return;
      
      // Handle backspace
      if (e.key === 'Backspace') {
        e.preventDefault();
        
        if (value[index]) {
          // Clear current digit
          const newValue = value.split('');
          newValue[index] = '';
          onChange(newValue.join(''));
        } else if (index > 0) {
          // Move to previous input and clear it
          const newValue = value.split('');
          newValue[index - 1] = '';
          onChange(newValue.join(''));
          setActiveIndex(index - 1);
          inputRefs.current[index - 1]?.focus();
        }
      }
      
      // Handle left/right arrow keys
      if (e.key === 'ArrowLeft' && index > 0) {
        setActiveIndex(index - 1);
        inputRefs.current[index - 1]?.focus();
      }
      
      if (e.key === 'ArrowRight' && index < length - 1) {
        setActiveIndex(index + 1);
        inputRefs.current[index + 1]?.focus();
      }
    };

    const handlePaste = (e: React.ClipboardEvent) => {
      if (disabled) return;
      
      e.preventDefault();
      const pastedValue = e.clipboardData.getData('text').replace(/\D/g, '').slice(0, length);
      onChange(pastedValue);
      
      if (pastedValue.length === length && onComplete) {
        onComplete(pastedValue);
      }
    };

    const handleFocus = (index: number) => {
      setActiveIndex(index);
    };

    // Auto-focus first input on mount
    useEffect(() => {
      if (!disabled) {
        inputRefs.current[0]?.focus();
      }
    }, [disabled]);

    return (
      <div 
        ref={ref} 
        className={cn("flex gap-2 justify-center", className)}
        {...props}
      >
        {Array.from({ length }, (_, index) => (
          <input
            key={index}
            ref={(el) => {
              inputRefs.current[index] = el;
            }}
            type="text"
            inputMode="numeric"
            pattern="[0-9]*"
            maxLength={1}
            value={value[index] || ''}
            onChange={(e) => handleChange(index, e.target.value)}
            onKeyDown={(e) => handleKeyDown(index, e)}
            onPaste={handlePaste}
            onFocus={() => handleFocus(index)}
            disabled={disabled}
            className={cn(
              "w-12 h-12 text-center text-lg font-medium border border-input rounded-md",
              "focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2",
              "disabled:cursor-not-allowed disabled:opacity-50",
              "bg-background text-foreground",
              activeIndex === index && "ring-2 ring-ring ring-offset-2",
              value[index] && "bg-accent/20 border-accent"
            )}
          />
        ))}
      </div>
    );
  }
);

OTPInput.displayName = "OTPInput";