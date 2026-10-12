import { AppWindow, File, Folder, Globe, Loader2, X } from "lucide-react";
import { forwardRef, useEffect, type ButtonHTMLAttributes, type InputHTMLAttributes, type ReactNode } from "react";
import { ResourceType } from "../api";

type Variant = "primary" | "secondary" | "ghost" | "danger";

const variants: Record<Variant, string> = {
  primary: "bg-accent text-accent-fg hover:bg-accent-hover shadow-sm",
  secondary: "bg-surface border border-border-strong text-text hover:bg-surface-2",
  ghost: "text-muted hover:text-text hover:bg-surface-2",
  danger: "text-danger hover:bg-danger-soft",
};

export function Button({
  variant = "secondary",
  size = "md",
  className = "",
  busy,
  children,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant; size?: "sm" | "md"; busy?: boolean }) {
  const sizing = size === "sm" ? "h-7 px-2.5 text-xs gap-1.5" : "h-8 px-3 text-[13px] gap-2";
  return (
    <button
      {...props}
      disabled={props.disabled || busy}
      className={`no-drag inline-flex shrink-0 items-center justify-center rounded-lg font-medium transition-colors disabled:opacity-50 disabled:pointer-events-none ${sizing} ${variants[variant]} ${className}`}
    >
      {busy && <Loader2 className="size-3.5 animate-spin" />}
      {children}
    </button>
  );
}

export function IconButton({
  label,
  className = "",
  children,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { label: string }) {
  return (
    <button
      {...props}
      aria-label={label}
      title={label}
      className={`no-drag inline-flex size-7 shrink-0 items-center justify-center rounded-md text-muted transition-colors hover:bg-surface-2 hover:text-text disabled:opacity-40 ${className}`}
    >
      {children}
    </button>
  );
}

export const Input = forwardRef<HTMLInputElement, InputHTMLAttributes<HTMLInputElement>>(function Input(
  { className = "", ...props },
  ref,
) {
  return (
    <input
      ref={ref}
      spellCheck={false}
      autoComplete="off"
      {...props}
      className={`h-8 w-full min-w-0 rounded-lg border border-border-strong bg-surface px-2.5 text-[13px] text-text placeholder:text-faint focus:border-accent focus:outline-none focus:ring-3 focus:ring-accent-soft ${className}`}
    />
  );
});

export function Label({ children, hint }: { children: ReactNode; hint?: ReactNode }) {
  return (
    <div className="mb-1.5 flex items-baseline justify-between gap-2">
      <span className="text-xs font-medium text-muted">{children}</span>
      {hint && <span className="text-[11px] text-faint">{hint}</span>}
    </div>
  );
}

export function TypeIcon({ type, className = "size-4" }: { type: ResourceType; className?: string }) {
  switch (type) {
    case ResourceType.TypeURL:
      return <Globe className={className} />;
    case ResourceType.TypeApp:
      return <AppWindow className={className} />;
    case ResourceType.TypeFile:
      return <File className={className} />;
    case ResourceType.TypeFolder:
      return <Folder className={className} />;
  }
  return null;
}

export const typeNames: Record<string, string> = {
  [ResourceType.TypeURL]: "Link",
  [ResourceType.TypeApp]: "App",
  [ResourceType.TypeFile]: "File",
  [ResourceType.TypeFolder]: "Folder",
};

export function Dialog({
  title,
  onClose,
  children,
  footer,
  width = "max-w-lg",
}: {
  title: ReactNode;
  onClose: () => void;
  children: ReactNode;
  footer?: ReactNode;
  width?: string;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);
  return (
    <div
      className="no-drag fixed inset-0 z-40 flex items-start justify-center bg-black/35 px-4 pt-[12vh]"
      onMouseDown={(e) => e.target === e.currentTarget && onClose()}
    >
      <div role="dialog" aria-modal="true" className={`animate-pop-in w-full ${width} overflow-hidden rounded-2xl border border-border bg-surface shadow-pop`}>
        <div className="flex items-center justify-between border-b border-border px-5 py-3.5">
          <h2 className="text-[15px] font-semibold">{title}</h2>
          <IconButton label="Close" onClick={onClose}>
            <X className="size-4" />
          </IconButton>
        </div>
        <div className="px-5 py-4">{children}</div>
        {footer && <div className="flex justify-end gap-2 border-t border-border bg-surface-2/50 px-5 py-3">{footer}</div>}
      </div>
    </div>
  );
}

export function Kbd({ children }: { children: ReactNode }) {
  return (
    <kbd className="rounded border border-border-strong bg-surface-2 px-1 py-px font-sans text-[10px] font-medium text-muted">
      {children}
    </kbd>
  );
}
