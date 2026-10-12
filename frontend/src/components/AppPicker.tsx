import { AppWindow, ChevronDown, RefreshCw, X } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { AppRef, SystemService, type App } from "../api";

let cache: Promise<App[]> | null = null;
function installedApps(refresh = false): Promise<App[]> {
  if (!cache || refresh) cache = refresh ? SystemService.RefreshApps() : SystemService.InstalledApps();
  return cache;
}

/** A searchable dropdown of the apps installed on this Mac. */
export function AppPicker({
  value,
  onChange,
  placeholder = "Choose an app…",
  clearable,
}: {
  value?: AppRef | null;
  onChange: (app: AppRef | null) => void;
  placeholder?: string;
  clearable?: boolean;
}) {
  const [open, setOpen] = useState(false);
  const [apps, setApps] = useState<App[]>([]);
  const [query, setQuery] = useState("");
  const [active, setActive] = useState(0);
  const root = useRef<HTMLDivElement>(null);
  const list = useRef<HTMLDivElement>(null);

  useEffect(() => {
    if (open) installedApps().then(setApps);
  }, [open]);

  useEffect(() => {
    if (!open) return;
    const onDown = (e: MouseEvent) => !root.current?.contains(e.target as Node) && setOpen(false);
    window.addEventListener("mousedown", onDown);
    return () => window.removeEventListener("mousedown", onDown);
  }, [open]);

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    if (!q) return apps;
    return apps
      .filter((a) => a.name.toLowerCase().includes(q) || a.bundleId.toLowerCase().includes(q))
      .sort((a, b) => Number(!a.name.toLowerCase().startsWith(q)) - Number(!b.name.toLowerCase().startsWith(q)));
  }, [apps, query]);

  useEffect(() => setActive(0), [query]);
  useEffect(() => {
    list.current?.querySelector(`[data-index="${active}"]`)?.scrollIntoView({ block: "nearest" });
  }, [active]);

  const pick = (a: App) => {
    onChange(new AppRef({ bundleId: a.bundleId, name: a.name }));
    setOpen(false);
    setQuery("");
  };

  const installed = !value || apps.length === 0 || apps.some((a) => a.bundleId === value.bundleId);

  return (
    <div ref={root} className="relative min-w-0 flex-1">
      <button
        type="button"
        onClick={() => setOpen((o) => !o)}
        className="no-drag flex h-8 w-full min-w-0 items-center gap-2 rounded-lg border border-border-strong bg-surface px-2.5 text-left text-[13px] hover:bg-surface-2"
      >
        <AppWindow className="size-3.5 shrink-0 text-muted" />
        <span className={`min-w-0 flex-1 truncate ${value?.name || value?.bundleId ? "" : "text-faint"}`}>
          {value?.name || value?.bundleId || placeholder}
          {!installed && <span className="ml-1.5 text-warn">· not installed</span>}
        </span>
        {clearable && value ? (
          <X
            className="size-3.5 shrink-0 text-muted hover:text-text"
            onClick={(e) => {
              e.stopPropagation();
              onChange(null);
            }}
          />
        ) : (
          <ChevronDown className="size-3.5 shrink-0 text-muted" />
        )}
      </button>
      {open && (
        <div className="animate-pop-in absolute left-0 right-0 top-9 z-30 overflow-hidden rounded-xl border border-border bg-surface shadow-pop">
          <div className="flex items-center gap-1 border-b border-border p-1.5">
            <input
              autoFocus
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onKeyDown={(e) => {
                if (e.key === "ArrowDown") {
                  e.preventDefault();
                  setActive((i) => Math.min(i + 1, filtered.length - 1));
                } else if (e.key === "ArrowUp") {
                  e.preventDefault();
                  setActive((i) => Math.max(i - 1, 0));
                } else if (e.key === "Enter" && filtered[active]) {
                  e.preventDefault();
                  pick(filtered[active]);
                } else if (e.key === "Escape") {
                  e.stopPropagation();
                  setOpen(false);
                }
              }}
              placeholder="Search apps"
              className="h-7 w-full bg-transparent px-1.5 text-[13px] focus:outline-none"
            />
            <button
              type="button"
              title="Rescan applications"
              onClick={() => installedApps(true).then(setApps)}
              className="rounded p-1 text-muted hover:bg-surface-2 hover:text-text"
            >
              <RefreshCw className="size-3.5" />
            </button>
          </div>
          <div ref={list} className="max-h-64 overflow-y-auto p-1">
            {filtered.length === 0 && <div className="px-2 py-3 text-center text-xs text-muted">No apps found</div>}
            {filtered.map((a, i) => (
              <button
                key={a.bundleId}
                type="button"
                data-index={i}
                onMouseEnter={() => setActive(i)}
                onClick={() => pick(a)}
                className={`flex w-full items-center justify-between gap-3 rounded-md px-2 py-1.5 text-left ${i === active ? "bg-accent-soft" : ""}`}
              >
                <span className="truncate text-[13px]">{a.name}</span>
                <span className="truncate font-mono text-[10px] text-faint">{a.bundleId}</span>
              </button>
            ))}
          </div>
        </div>
      )}
    </div>
  );
}
