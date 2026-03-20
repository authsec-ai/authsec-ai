// @ts-nocheck
import * as React from "react";
import { useCallback, useRef, useState } from "react";
import { cn } from "../../lib/utils";

export interface DropzoneProps {
  accept?: Record<string, string[]>;
  disabled?: boolean;
  onFiles?: (files: File[]) => void;
  className?: string;
  children?: React.ReactNode;
}

export interface DropzoneInputProps extends React.InputHTMLAttributes<HTMLInputElement> {
  multiple?: boolean;
}

export const Dropzone = React.forwardRef<HTMLDivElement, DropzoneProps>(
  ({ accept, disabled, onFiles, className, children }, ref) => {
    const inputRef = useRef<HTMLInputElement>(null);
    const [isDragActive, setIsDragActive] = useState(false);

    const isAccepted = useCallback(
      (file: File) => {
        if (!accept) return true;
        const mime = file.type;
        const ext = file.name.split(".").pop()?.toLowerCase();
        return Object.entries(accept).some(([type, exts]) => {
          const typeMatch = type === "*/*" || mime.startsWith(type.replace("/*", ""));
          const extMatch = exts?.length ? exts.some((e) => e.replace(".", "") === ext) : true;
          return typeMatch && extMatch;
        });
      },
      [accept]
    );

    const handleDrop = (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      setIsDragActive(false);
      if (disabled) return;
      const files = Array.from(e.dataTransfer.files).filter(isAccepted);
      if (files.length && onFiles) onFiles(files);
    };
    const handleDragOver = (e: React.DragEvent<HTMLDivElement>) => {
      e.preventDefault();
      if (!disabled) setIsDragActive(true);
    };
    const handleDragLeave = () => setIsDragActive(false);
    const handleClick = () => {
      if (!disabled) inputRef.current?.click();
    };
    const handleInput = (e: React.ChangeEvent<HTMLInputElement>) => {
      if (disabled) return;
      const files = e.target.files ? Array.from(e.target.files).filter(isAccepted) : [];
      if (files.length && onFiles) onFiles(files);
      e.target.value = "";
    };

    return (
      <div
        ref={ref}
        tabIndex={disabled ? -1 : 0}
        aria-disabled={disabled}
        className={cn(
          "relative border border-dashed rounded-md transition-colors cursor-pointer focus:outline-none focus:ring-2 focus:ring-primary/30",
          disabled && "opacity-60 pointer-events-none bg-muted",
          isDragActive && "border-primary bg-primary/10",
          className
        )}
        onClick={handleClick}
        onDrop={handleDrop}
        onDragOver={handleDragOver}
        onDragLeave={handleDragLeave}
        onDragEnd={handleDragLeave}
        role="button"
      >
        <input
          ref={inputRef}
          type="file"
          style={{ display: "none" }}
          accept={accept ? Object.keys(accept).join(",") : undefined}
          multiple={false}
          disabled={disabled}
          tabIndex={-1}
          onChange={handleInput}
          data-testid="dropzone-input"
        />
        {children}
      </div>
    );
  }
);
Dropzone.displayName = "Dropzone";

export const DropzoneInput = React.forwardRef<HTMLInputElement, DropzoneInputProps>(
  ({ className, ...props }, ref) => (
    <input ref={ref} type="file" className={cn("sr-only", className)} tabIndex={-1} {...props} />
  )
);
DropzoneInput.displayName = "DropzoneInput";
