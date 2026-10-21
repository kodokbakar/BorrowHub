import React from "react";

export interface ButtonProps
  extends React.ButtonHTMLAttributes<HTMLButtonElement> {
  variant?: "primary" | "secondary" | "danger" | "outline" | "ghost";
  size?: "sm" | "md" | "lg";
  isLoading?: boolean;
}

export const Button = React.forwardRef<HTMLButtonElement, ButtonProps>(
  (
    {
      children,
      variant = "primary",
      size = "md",
      isLoading = false,
      disabled,
      className = "",
      ...props
    },
    ref,
  ) => {
    const base =
      "inline-flex items-center justify-center font-medium rounded-lg transition-colors focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-offset-2 disabled:opacity-50 disabled:cursor-not-allowed cursor-pointer";

    const variants = {
      primary:
        "bg-accent text-accent-fg hover:bg-accent-hover focus-visible:ring-accent shadow-xs",
      secondary:
        "bg-surface-hover text-fg hover:bg-border-subtle focus-visible:ring-border-strong border border-border",
      danger:
        "bg-status-overdue text-white hover:opacity-90 focus-visible:ring-status-overdue shadow-xs",
      outline:
        "border border-border-strong bg-surface text-fg hover:bg-surface-hover focus-visible:ring-accent shadow-xs",
      ghost:
        "text-fg-muted hover:text-fg hover:bg-surface-hover focus-visible:ring-accent",
    };

    const sizes = {
      sm: "min-h-[36px] sm:min-h-[40px] text-xs px-2.5 py-1.5 gap-1.5",
      md: "min-h-[44px] text-sm px-4 py-2 gap-2",
      lg: "min-h-[44px] text-base px-5 py-2.5 gap-2.5",
    };
    return (
      <button
        ref={ref}
        disabled={disabled || isLoading}
        className={`${base} ${variants[variant]} ${sizes[size]} ${className}`}
        {...props}
      >
        {isLoading && (
          <svg
            className="animate-spin -ml-1 h-4 w-4 text-current"
            xmlns="http://www.w3.org/2000/svg"
            fill="none"
            viewBox="0 0 24 24"
            aria-hidden="true"
          >
            <circle
              className="opacity-25"
              cx="12"
              cy="12"
              r="10"
              stroke="currentColor"
              strokeWidth="4"
            />
            <path
              className="opacity-75"
              fill="currentColor"
              d="M4 12a8 8 0 018-8v4a4 4 0 00-4 4H4z"
            />
          </svg>
        )}
        {children}
      </button>
    );
  },
);

Button.displayName = "Button";
