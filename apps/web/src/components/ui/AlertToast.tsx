import React from "react";

export interface AlertToastProps {
  type?: "success" | "error" | "info" | "warning";
  message: string;
  requestId?: string;
  onClose?: () => void;
  className?: string;
}

export function AlertToast({
  type = "info",
  message,
  requestId,
  onClose,
  className = "",
}: AlertToastProps) {
  const styles = {
    success: "bg-emerald-50 text-emerald-800 border-emerald-200",
    error: "bg-rose-50 text-rose-800 border-rose-200",
    warning: "bg-amber-50 text-amber-800 border-amber-200",
    info: "bg-sky-50 text-sky-800 border-sky-200",
  }[type];

  return (
    <div
      className={`flex items-center justify-between rounded-xl border p-4 text-sm shadow-xs ${styles} ${className}`}
      role="alert"
    >
      <div className="flex items-center gap-2">
        {type === "success" && (
          <svg className="h-4 w-4 shrink-0 text-emerald-600" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zm3.707-9.293a1 1 0 00-1.414-1.414L9 10.586 7.707 9.293a1 1 0 00-1.414 1.414l2 2a1 1 0 001.414 0l4-4z" clipRule="evenodd" />
          </svg>
        )}
        {type === "error" && (
          <svg className="h-4 w-4 shrink-0 text-rose-600" fill="currentColor" viewBox="0 0 20 20">
            <path fillRule="evenodd" d="M10 18a8 8 0 100-16 8 8 0 000 16zM8.707 7.293a1 1 0 00-1.414 1.414L8.586 10l-1.293 1.293a1 1 0 101.414 1.414L10 11.414l1.293 1.293a1 1 0 001.414-1.414L11.414 10l1.293-1.293a1 1 0 00-1.414-1.414L10 8.586 8.707 7.293z" clipRule="evenodd" />
          </svg>
        )}
        <div className="flex flex-col">
          <span>{message}</span>
          {requestId && (
            <span className="font-mono text-xs opacity-80 select-all">
              Request ID: {requestId}
            </span>
          )}
        </div>
      </div>
      {onClose && (
        <button
          type="button"
          onClick={onClose}
          className="ml-3 inline-flex text-slate-400 hover:text-slate-600 cursor-pointer"
          aria-label="Tutup"
        >
          <svg className="h-4 w-4" fill="none" viewBox="0 0 24 24" stroke="currentColor">
            <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      )}
    </div>
  );
}
