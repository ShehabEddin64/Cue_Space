import { AppWindow, File, Folder, Globe, Loader2, X } from "lucide-react";
import { forwardRef, useEffect, type ButtonHTMLAttributes, type InputHTMLAttributes, type ReactNode } from "react";
import { ResourceType } from "../api";

/*
 * Controls follow the identity sheet: pill shapes, Cue Red for the one primary
 * action, a soft fill for secondary actions, a red outline for AI actions.
 */
type Variant = "primary" | "secondary" | "tinted" | "ghost" | "danger";

const variants: Record<Variant, string> = {
  primary: "bg-red text-white hover:bg-red-hover active:brightness-95 shadow-[0_1px_2px_rgb(217_48_37/0.3)]",
  secondary: "bg-fill text-ink hover:bg-fill-strong",
  tinted: "border border-red-text/40 text-red-text hover:bg-red-tint",
  ghost: "text-graphite hover:text-ink hover:bg-fill",
  danger: "text-red-text hover:bg-red-tint",
};

const sizes = {
  sm: "h-7 px-3 text-[12px] gap-1.5",
  md: "h-8 px-3.5 text-[13px] gap-1.5",
  lg: "h-10 px-5 text-[14px] gap-2",
};

export function Button({
  variant = "secondary",
  size = "md",
  className = "",
  busy,
  children,
  ...props
}: ButtonHTMLAttributes<HTMLButtonElement> & { variant?: Variant; size?: keyof typeof sizes; busy?: boolean }) {
  return (
    <button
      type="button"
      {...props}
      disabled={props.disabled || busy}
      className={`inline-flex shrink-0 select-none items-center justify-center whitespace-nowrap rounded-full font-medium transition-[background-color,color,opacity] duration-150 disabled:pointer-events-none disabled:opacity-45 ${sizes[size]} ${variants[variant]} ${className}`}
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
      type="button"
      {...props}
      aria-label={label}
      title={label}
      className={`inline-flex size-7 shrink-0 items-center justify-center rounded-md text-graphite transition-colors hover:bg-fill hover:text-ink disabled:opacity-40 ${className}`}
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
      className={`h-8 w-full min-w-0 rounded-lg border border-hairline bg-surface px-2.5 text-[13px] text-ink placeholder:text-faint focus:border-red-text/60 focus:outline-none focus:ring-3 focus:ring-red-tint ${className}`}
    />
  );
});

export function Label({ children, hint }: { children: ReactNode; hint?: ReactNode }) {
  return (
    <div className="mb-1.5 flex items-baseline justify-between gap-2">
      <span className="text-[12px] font-medium text-graphite">{children}</span>
      {hint && <span className="text-[11px] text-faint">{hint}</span>}
    </div>
  );
}

export function TypeIcon({ type, className = "size-4" }: { type: ResourceType; className?: string }) {
  switch (type) {
    case ResourceType.TypeURL:
      return <Globe className={className} strokeWidth={1.75} />;
    case ResourceType.TypeApp:
      return <AppWindow className={className} strokeWidth={1.75} />;
    case ResourceType.TypeFile:
      return <File className={className} strokeWidth={1.75} />;
    case ResourceType.TypeFolder:
      return <Folder className={className} strokeWidth={1.75} />;
  }
  return null;
}

export const typeNames: Record<string, string> = {
  [ResourceType.TypeURL]: "Link",
  [ResourceType.TypeApp]: "App",
  [ResourceType.TypeFile]: "File",
  [ResourceType.TypeFolder]: "Folder",
};

/** The Cue Space mark: a ring (the intention) and a dot (the cue that closes it). */
export function CueMark({
  size = 20,
  thinking = false,
  ring = "var(--red-text)",
  dot = "var(--ink)",
  className = "",
}: {
  size?: number;
  thinking?: boolean;
  ring?: string;
  dot?: string;
  className?: string;
}) {
  return (
    <svg width={size} height={size} viewBox="0 0 100 100" aria-hidden="true" className={`shrink-0 ${className}`}>
      <path d="M72.48 30.72 A30 30 0 1 0 72.48 69.28" fill="none" stroke={ring} strokeWidth="10" strokeLinecap="round" />
      <g className={thinking ? "cue-orbit" : undefined}>
        <circle cx="78.3" cy="50" r="7" fill={dot} />
      </g>
    </svg>
  );
}

/** A sheet-style modal: one short task, with an obvious way out. */
export function Sheet({
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
      className="fixed inset-0 z-40 flex items-start justify-center bg-black/25 px-4 pt-[10vh]"
      onMouseDown={(e) => e.target === e.currentTarget && onClose()}
    >
      <div
        role="dialog"
        aria-modal="true"
        className={`animate-pop-in w-full ${width} overflow-hidden rounded-[20px] border border-hairline bg-surface shadow-pop`}
      >
        <div className="flex items-center justify-between px-6 pb-2 pt-5">
          <h2 className="text-[17px] font-semibold tracking-[-0.01em]">{title}</h2>
          <IconButton label="Close" onClick={onClose}>
            <X className="size-4" />
          </IconButton>
        </div>
        <div className="px-6 pb-5 pt-2">{children}</div>
        {footer && <div className="flex items-center justify-end gap-2 border-t border-hairline px-6 py-3.5">{footer}</div>}
      </div>
    </div>
  );
}

export function Kbd({ children }: { children: ReactNode }) {
  return (
    <kbd className="rounded-[5px] border border-hairline bg-surface px-1 py-px font-sans text-[10.5px] font-medium text-graphite">
      {children}
    </kbd>
  );
}

/** A grouped, inset list in the macOS style. */
export function InsetList({ children, className = "" }: { children: ReactNode; className?: string }) {
  return (
    <ul className={`divide-y divide-hairline overflow-hidden rounded-2xl border border-hairline bg-surface shadow-card ${className}`}>
      {children}
    </ul>
  );
}

export function SectionTitle({ children, trailing }: { children: ReactNode; trailing?: ReactNode }) {
  return (
    <div className="mb-2.5 flex items-center justify-between px-1">
      <h3 className="text-[13px] font-semibold text-ink">{children}</h3>
      {trailing}
    </div>
  );
}

/** A failure from the AI, in plain words, with the two things people can do. */
export function AIErrorNotice({ message, onRetry, onSettings }: { message: string; onRetry?: () => void; onSettings?: () => void }) {
  return (
    <div role="alert" className="animate-pop-in mt-3 flex flex-wrap items-center gap-x-3 gap-y-2 rounded-2xl bg-red-tint px-4 py-3">
      <p className="min-w-[200px] flex-1 text-[13px] leading-relaxed text-red-on-tint">{message}</p>
      <div className="flex gap-1.5">
        {onRetry && (
          <Button size="sm" variant="tinted" onClick={onRetry}>
            Try Again
          </Button>
        )}
        {onSettings && (
          <Button size="sm" variant="ghost" onClick={onSettings}>
            Settings
          </Button>
        )}
      </div>
    </div>
  );
}
