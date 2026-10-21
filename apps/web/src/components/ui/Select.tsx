import React from "react";

export interface SelectOption {
  value: string;
  label: string;
}

export interface SelectProps
  extends React.SelectHTMLAttributes<HTMLSelectElement> {
  label?: string;
  error?: string;
  helperText?: string;
  options: SelectOption[];
  placeholder?: string;
}

export const Select = React.forwardRef<HTMLSelectElement, SelectProps>(
  (
    {
      label,
      error,
      helperText,
      options,
      placeholder,
      id,
      className = "",
      ...props
    },
    ref,
  ) => {
    const autoId = React.useId();
    const selectId =
      id || (label ? label.toLowerCase().replace(/[^a-z0-9]+/g, "-") : autoId);
    const errorId = error ? `${selectId}-error` : undefined;
    const helperId = helperText ? `${selectId}-helper` : undefined;
    const describedBy = [errorId || helperId, props["aria-describedby"]].filter(Boolean).join(" ") || undefined;

    return (
      <div className="w-full">
        {label && (
          <label
            htmlFor={selectId}
            className="block text-sm font-medium text-fg mb-1"
          >
            {label}
          </label>
        )}
        <select
          ref={ref}
          id={selectId}
          aria-invalid={Boolean(error)}
          aria-describedby={describedBy}
          className={`block w-full min-h-[44px] rounded-lg border px-3 py-2 text-sm transition-colors bg-surface text-fg focus:outline-none focus:ring-2 focus:ring-offset-1 ${
            error
              ? "border-status-overdue focus:border-status-overdue focus:ring-status-overdue"
              : "border-border-strong focus:border-accent focus:ring-accent"
          } disabled:bg-surface-subtle disabled:text-fg-muted disabled:cursor-not-allowed ${className}`}
          {...props}
        >
          {placeholder && <option value="">{placeholder}</option>}
          {options.map((opt) => (
            <option key={opt.value} value={opt.value}>
              {opt.label}
            </option>
          ))}
        </select>
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

Select.displayName = "Select";
