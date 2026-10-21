import React from "react";

export interface InputProps
  extends React.InputHTMLAttributes<HTMLInputElement> {
  label?: string;
  error?: string;
  helperText?: string;
}

export const Input = React.forwardRef<HTMLInputElement, InputProps>(
  ({ label, error, helperText, id, className = "", ...props }, ref) => {
    const autoId = React.useId();
    const inputId =
      id || (label ? label.toLowerCase().replace(/[^a-z0-9]+/g, "-") : autoId);
    const errorId = error ? `${inputId}-error` : undefined;
    const helperId = helperText ? `${inputId}-helper` : undefined;
    const describedBy = [errorId || helperId, props["aria-describedby"]].filter(Boolean).join(" ") || undefined;

    return (
      <div className="w-full">
        {label && (
          <label
            htmlFor={inputId}
            className="block text-sm font-medium text-fg mb-1"
          >
            {label}
          </label>
        )}
        <input
          ref={ref}
          id={inputId}
          aria-invalid={Boolean(error)}
          aria-describedby={describedBy}
          className={`block w-full min-h-[44px] rounded-lg border px-3 py-2 text-sm transition-colors placeholder:text-fg-subtle bg-surface text-fg focus:outline-none focus:ring-2 focus:ring-offset-1 ${
            error
              ? "border-status-overdue focus:border-status-overdue focus:ring-status-overdue"
              : "border-border-strong focus:border-accent focus:ring-accent"
          } disabled:bg-surface-subtle disabled:text-fg-muted disabled:cursor-not-allowed ${className}`}
          {...props}
        />
        {error ? (
          <p id={errorId} className="mt-1 text-xs text-status-overdue">
            {error}
          </p>
        ) : helperText ? (
          <p id={helperId} className="mt-1 text-xs text-fg-muted">
            {helperText}
          </p>
        ) : null}
      </div>
    );
  },
);

Input.displayName = "Input";
