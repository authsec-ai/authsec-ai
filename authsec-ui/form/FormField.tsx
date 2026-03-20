import * as React from "react";
import { Label } from "@/components/ui/label";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { RadioGroup, RadioGroupItem } from "@/components/ui/radio-group";
import { cn } from "@/lib/utils";
import { AlertCircle, Check } from "lucide-react";
import { motion, AnimatePresence } from "framer-motion";

// Base container for all form fields with consistent spacing
interface FormFieldContainerProps {
  children: React.ReactNode;
  className?: string;
}

export function FormFieldContainer({ children, className }: FormFieldContainerProps) {
  return <div className={cn("space-y-2 group", className)}>{children}</div>;
}

// Reusable Label with consistent styling and required indicator
interface FormFieldLabelProps {
  htmlFor?: string;
  required?: boolean;
  children: React.ReactNode;
  icon?: React.ReactNode;
  className?: string;
}

export function FormFieldLabel({
  htmlFor,
  required,
  children,
  icon,
  className,
}: FormFieldLabelProps) {
  return (
    <Label
      htmlFor={htmlFor}
      className={cn(
        "text-sm font-medium flex items-center gap-2 text-foreground/80 group-focus-within:text-primary transition-colors duration-200",
        className
      )}
    >
      {icon && <span className="text-muted-foreground group-focus-within:text-primary transition-colors">{icon}</span>}
      {children}
      {required && <span className="text-destructive/80">*</span>}
    </Label>
  );
}

// Helper text for form fields
interface FormFieldHelperTextProps {
  children: React.ReactNode;
  className?: string;
}

export function FormFieldHelperText({ children, className }: FormFieldHelperTextProps) {
  return (
    <p className={cn("text-[0.8rem] text-muted-foreground/80 leading-relaxed", className)}>
      {children}
    </p>
  );
}

// Error message component
interface FormFieldErrorProps {
  children: React.ReactNode;
  className?: string;
}

export function FormFieldError({ children, className }: FormFieldErrorProps) {
  if (!children) return null;

  return (
    <AnimatePresence mode="wait">
      <motion.div
        initial={{ opacity: 0, y: -5 }}
        animate={{ opacity: 1, y: 0 }}
        exit={{ opacity: 0, y: -5 }}
        className={cn("flex items-center gap-2 text-sm text-destructive font-medium mt-1.5", className)}
      >
        <AlertCircle className="h-4 w-4 shrink-0" />
        <span>{children}</span>
      </motion.div>
    </AnimatePresence>
  );
}

// Character counter
interface FormFieldCounterProps {
  current: number;
  max: number;
  className?: string;
}

export function FormFieldCounter({ current, max, className }: FormFieldCounterProps) {
  const isOverLimit = current > max;

  return (
    <span
      className={cn(
        "text-xs font-medium tabular-nums transition-colors",
        isOverLimit ? "text-destructive" : "text-muted-foreground/60",
        className
      )}
    >
      {current}/{max}
    </span>
  );
}

// Complete Input Field with all features
export interface FormInputProps extends Omit<React.ComponentProps<typeof Input>, "id"> {
  id: string;
  label: string;
  required?: boolean;
  helperText?: string;
  error?: string;
  maxLength?: number;
  showCounter?: boolean;
  icon?: React.ReactNode;
  containerClassName?: string;
}

export function FormInput({
  id,
  label,
  required,
  helperText,
  error,
  maxLength,
  showCounter,
  icon,
  containerClassName,
  value,
  className,
  ...inputProps
}: FormInputProps) {
  const currentLength = typeof value === "string" ? value.length : 0;
  const showCharCounter = showCounter && maxLength;

  return (
    <FormFieldContainer className={containerClassName}>
      <FormFieldLabel htmlFor={id} required={required} icon={icon}>
        {label}
      </FormFieldLabel>
      <div className="relative">
        <Input
          id={id}
          value={value}
          maxLength={maxLength}
          className={cn(
            "transition-all duration-200 focus-visible:ring-2 focus-visible:ring-primary/20",
            error && "border-destructive focus-visible:ring-destructive/20",
            className
          )}
          aria-invalid={!!error}
          {...inputProps}
        />
      </div>
      <div className="flex items-start justify-between gap-2 min-h-[20px]">
        <div className="flex-1">
          {helperText && !error && <FormFieldHelperText>{helperText}</FormFieldHelperText>}
          {error && <FormFieldError>{error}</FormFieldError>}
        </div>
        {showCharCounter && <FormFieldCounter current={currentLength} max={maxLength!} />}
      </div>
    </FormFieldContainer>
  );
}

// Complete Textarea Field with all features
export interface FormTextareaProps extends Omit<React.ComponentProps<typeof Textarea>, "id"> {
  id: string;
  label: string;
  required?: boolean;
  helperText?: string;
  error?: string;
  maxLength?: number;
  showCounter?: boolean;
  icon?: React.ReactNode;
  containerClassName?: string;
}

export function FormTextarea({
  id,
  label,
  required,
  helperText,
  error,
  maxLength,
  showCounter,
  icon,
  containerClassName,
  value,
  className,
  ...textareaProps
}: FormTextareaProps) {
  const currentLength = typeof value === "string" ? value.length : 0;
  const showCharCounter = showCounter && maxLength;

  return (
    <FormFieldContainer className={containerClassName}>
      <FormFieldLabel htmlFor={id} required={required} icon={icon}>
        {label}
      </FormFieldLabel>
      <Textarea
        id={id}
        value={value}
        maxLength={maxLength}
        className={cn(
          "resize-y min-h-[100px] transition-all duration-200 focus-visible:ring-2 focus-visible:ring-primary/20",
          error && "border-destructive focus-visible:ring-destructive/20",
          className
        )}
        aria-invalid={!!error}
        {...textareaProps}
      />
      <div className="flex items-start justify-between gap-2 min-h-[20px]">
        <div className="flex-1">
          {helperText && !error && <FormFieldHelperText>{helperText}</FormFieldHelperText>}
          {error && <FormFieldError>{error}</FormFieldError>}
        </div>
        {showCharCounter && <FormFieldCounter current={currentLength} max={maxLength!} />}
      </div>
    </FormFieldContainer>
  );
}

// Complete Select Field with all features
export interface FormSelectOption {
  value: string;
  label: string;
  description?: string;
  icon?: React.ReactNode;
}

export interface FormSelectProps {
  id: string;
  label: string;
  required?: boolean;
  helperText?: string;
  error?: string;
  icon?: React.ReactNode;
  containerClassName?: string;
  placeholder?: string;
  value?: string;
  onValueChange?: (value: string) => void;
  disabled?: boolean;
  options: FormSelectOption[];
  emptyMessage?: string;
}

export function FormSelect({
  id,
  label,
  required,
  helperText,
  error,
  icon,
  containerClassName,
  placeholder = "Select an option",
  value,
  onValueChange,
  disabled,
  options,
  emptyMessage = "No options available",
}: FormSelectProps) {
  return (
    <FormFieldContainer className={containerClassName}>
      <FormFieldLabel htmlFor={id} required={required} icon={icon}>
        {label}
      </FormFieldLabel>
      <Select value={value} onValueChange={onValueChange} disabled={disabled}>
        <SelectTrigger
          id={id}
          className={cn(
            "transition-all duration-200 focus:ring-2 focus:ring-primary/20",
            error && "border-destructive focus:ring-destructive/20"
          )}
          aria-invalid={!!error}
        >
          <SelectValue placeholder={placeholder} />
        </SelectTrigger>
        <SelectContent>
          {options.length === 0 ? (
            <div className="p-4 text-center text-sm text-muted-foreground">
              {emptyMessage}
            </div>
          ) : (
            options.map((option) => (
              <SelectItem key={option.value} value={option.value} className="cursor-pointer">
                <div className="flex items-center gap-2">
                  {option.icon && <span className="text-muted-foreground">{option.icon}</span>}
                  <div className="flex flex-col text-left">
                    <span className="font-medium">{option.label}</span>
                    {option.description && (
                      <span className="text-xs text-muted-foreground">{option.description}</span>
                    )}
                  </div>
                </div>
              </SelectItem>
            ))
          )}
        </SelectContent>
      </Select>
      <div className="min-h-[20px]">
        {helperText && !error && <FormFieldHelperText>{helperText}</FormFieldHelperText>}
        {error && <FormFieldError>{error}</FormFieldError>}
      </div>
    </FormFieldContainer>
  );
}

// New Component: Radio Card Selection
// Useful for making choices more visual and prominent
export interface FormRadioCardOption {
  value: string;
  label: string;
  description?: string;
  icon?: React.ReactNode;
  disabled?: boolean;
}

export interface FormRadioCardProps {
  id: string;
  label: string;
  required?: boolean;
  helperText?: string;
  error?: string;
  icon?: React.ReactNode;
  containerClassName?: string;
  value?: string;
  onValueChange?: (value: string) => void;
  options: FormRadioCardOption[];
  columns?: 1 | 2 | 3;
}

export function FormRadioCard({
  id,
  label,
  required,
  helperText,
  error,
  icon,
  containerClassName,
  value,
  onValueChange,
  options,
  columns = 2,
}: FormRadioCardProps) {
  return (
    <FormFieldContainer className={containerClassName}>
      <FormFieldLabel htmlFor={id} required={required} icon={icon}>
        {label}
      </FormFieldLabel>
      <RadioGroup
        value={value}
        onValueChange={onValueChange}
        className={cn(
          "grid gap-4 pt-2",
          columns === 1 && "grid-cols-1",
          columns === 2 && "grid-cols-1 sm:grid-cols-2",
          columns === 3 && "grid-cols-1 sm:grid-cols-3"
        )}
      >
        {options.map((option) => {
          const isSelected = value === option.value;
          return (
            <div key={option.value} className="relative">
              <RadioGroupItem
                value={option.value}
                id={`${id}-${option.value}`}
                className="peer sr-only"
                disabled={option.disabled}
              />
              <Label
                htmlFor={`${id}-${option.value}`}
                className={cn(
                  "flex flex-col h-full p-4 border-2 rounded-xl cursor-pointer transition-all duration-200 hover:bg-accent/50",
                  "peer-focus-visible:ring-2 peer-focus-visible:ring-primary peer-focus-visible:ring-offset-2",
                  isSelected
                    ? "border-primary bg-primary/5 ring-1 ring-primary/20"
                    : "border-muted bg-card hover:border-primary/50",
                  option.disabled && "opacity-50 cursor-not-allowed"
                )}
              >
                <div className="flex items-start justify-between w-full mb-2">
                  {option.icon && (
                    <div
                      className={cn(
                        "p-2 rounded-lg transition-colors",
                        isSelected ? "bg-primary text-primary-foreground" : "bg-muted text-muted-foreground"
                      )}
                    >
                      {option.icon}
                    </div>
                  )}
                  {isSelected && (
                    <div className="text-primary">
                      <Check className="h-5 w-5" />
                    </div>
                  )}
                </div>
                <div className="space-y-1">
                  <span className={cn("font-semibold block", isSelected ? "text-primary" : "text-foreground")}>
                    {option.label}
                  </span>
                  {option.description && (
                    <span className="text-xs text-muted-foreground block leading-normal">
                      {option.description}
                    </span>
                  )}
                </div>
              </Label>
            </div>
          );
        })}
      </RadioGroup>
      <div className="min-h-[20px]">
        {helperText && !error && <FormFieldHelperText>{helperText}</FormFieldHelperText>}
        {error && <FormFieldError>{error}</FormFieldError>}
      </div>
    </FormFieldContainer>
  );
}
