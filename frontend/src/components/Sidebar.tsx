import { House, LayoutGrid, PanelLeft, Play, Plus, Search, Settings2 } from "lucide-react";
import { forwardRef, useImperativeHandle, useRef, type ReactNode } from "react";
import { displayName, shortcutOrder, type AISettings, type Cue } from "../api";
import { usePrefs } from "../prefs";
import { CueMark, IconButton } from "./ui";

export type SidebarHandle = { focusSearch: () => void };

/**
 * A standard Mac sidebar over the window's real material: search, Home, and
 * your Cues. Two levels at most, nothing critical pinned to the bottom.
 */
export const Sidebar = forwardRef<
  SidebarHandle,
  {
    cues: Cue[];
    selected: "home" | "library" | string | null;
    query: string;
    ai: AISettings | null;
    starting: string | null;
    onQuery: (q: string) => void;
    onHome: () => void;
    onLibrary: () => void;
    onSelect: (c: Cue) => void;
    onStart: (c: Cue) => void;
    onNewBlank: () => void;
    onSettings: () => void;
    onHide: () => void;
  }
>(function Sidebar({ cues, selected, query, ai, starting, onQuery, onHome, onLibrary, onSelect, onStart, onNewBlank, onSettings, onHide }, ref) {
  const { prefs } = usePrefs();
  const search = useRef<HTMLInputElement>(null);
  useImperativeHandle(ref, () => ({ focusSearch: () => search.current?.focus() }));

  const q = query.trim().toLowerCase();
  const ordered = shortcutOrder(cues, prefs.pinned);
  const shortcutOf = new Map(ordered.slice(0, 9).map((c, i) => [c.id, `⌘${i + 1}`]));
  const shown = q
    ? ordered.filter(
        (c) =>
          c.name.toLowerCase().includes(q) ||
          (c.description || "").toLowerCase().includes(q) ||
          c.resources.some((r) => displayName(r).toLowerCase().includes(q)),
      )
    : ordered;
  const pinnedShown = shown.filter((c) => prefs.pinned.includes(c.id));
  const restShown = shown.filter((c) => !prefs.pinned.includes(c.id));

  const row = (c: Cue) => (
    <Row key={c.id} active={selected === c.id} onClick={() => onSelect(c)}>
      <span className="flex w-4 justify-center text-[14px] leading-none">{c.icon || "◌"}</span>
      <span className="flex-1 truncate">{c.name}</span>
      {shortcutOf.has(c.id) && starting !== c.id && (
        <span className="text-[10.5px] text-faint group-hover:hidden">{shortcutOf.get(c.id)}</span>
      )}
      <button
        aria-label={`Start ${c.name}`}
        title={`Start Cue${shortcutOf.has(c.id) ? ` (${shortcutOf.get(c.id)})` : ""}`}
        disabled={starting === c.id}
        onClick={(e) => {
          e.stopPropagation();
          onStart(c);
        }}
        className={`size-5 items-center justify-center rounded-full text-red-text transition hover:bg-red-tint group-hover:flex ${
          starting === c.id ? "flex" : "hidden"
        }`}
      >
        {starting === c.id ? <CueMark size={12} thinking /> : <Play className="size-3 fill-current" />}
      </button>
    </Row>
  );

  return (
    <aside className="flex h-full w-[248px] shrink-0 flex-col border-r border-hairline bg-[var(--sidebar)]" aria-label="Sidebar">
      <div className="drag flex h-[52px] shrink-0 items-center justify-end px-3">
        <IconButton label="Hide Sidebar (⌃⌘S)" onClick={onHide}>
          <PanelLeft className="size-4" strokeWidth={1.75} />
        </IconButton>
      </div>

      <div className="px-3 pb-2">
        <label className="flex h-7 items-center gap-1.5 rounded-lg bg-fill/80 px-2 focus-within:ring-2 focus-within:ring-red-tint">
          <Search className="size-3.5 shrink-0 text-faint" strokeWidth={2} />
          <input
            ref={search}
            value={query}
            onChange={(e) => onQuery(e.target.value)}
            onKeyDown={(e) => e.key === "Escape" && onQuery("")}
            placeholder="Search"
            aria-label="Search Cues"
            className="h-full min-w-0 flex-1 bg-transparent text-[13px] placeholder:text-faint focus:outline-none"
          />
        </label>
      </div>

      <nav className="flex-1 overflow-y-auto px-2.5 pb-3">
        <Row active={selected === "home"} onClick={onHome}>
          <House className="size-4 text-red-text" strokeWidth={1.9} />
          <span className="flex-1 truncate">Home</span>
        </Row>
        <Row active={selected === "library"} onClick={onLibrary}>
          <LayoutGrid className="size-4 text-red-text" strokeWidth={1.9} />
          <span className="flex-1 truncate">Cues</span>
          <span className="text-[11px] text-faint">{cues.length}</span>
        </Row>

        {pinnedShown.length > 0 && (
          <>
            <div className="mb-1 mt-5 pl-2.5 text-[11px] font-semibold text-faint">Pinned</div>
            {pinnedShown.map(row)}
          </>
        )}

        <div className="mb-1 mt-5 flex items-center justify-between pl-2.5 pr-1">
          <span className="text-[11px] font-semibold text-faint">{pinnedShown.length > 0 ? "Other Cues" : "Your Cues"}</span>
          <IconButton label="New Blank Cue (⌥⌘N)" onClick={onNewBlank} className="size-6">
            <Plus className="size-3.5" />
          </IconButton>
        </div>
        {restShown.map(row)}
        {cues.length === 0 && <p className="px-2.5 py-1 text-[12px] text-faint">No Cues yet</p>}
        {cues.length > 0 && shown.length === 0 && <p className="px-2.5 py-1 text-[12px] text-faint">No matches</p>}
      </nav>

      <div className="border-t border-hairline p-2.5">
        <button
          onClick={onSettings}
          title="AI settings (⌘,)"
          className="flex w-full items-center gap-2 rounded-lg px-2 py-1.5 text-left text-[12px] text-graphite hover:bg-selected"
        >
          <CueMark size={14} />
          <span className="min-w-0 flex-1 truncate">
            {ai?.ready ? (
              <>
                {ai.disclosure.model}
                <span className="text-faint"> · {ai.disclosure.local ? "On this Mac" : ai.disclosure.vendor}</span>
              </>
            ) : (
              <span className="text-red-text">Connect an AI</span>
            )}
          </span>
          <Settings2 className="size-3.5 shrink-0 text-faint" />
        </button>
      </div>
    </aside>
  );
});

function Row({ active, onClick, children }: { active: boolean; onClick: () => void; children: ReactNode }) {
  return (
    <div
      role="button"
      tabIndex={0}
      aria-current={active ? "page" : undefined}
      onClick={onClick}
      onKeyDown={(e) => e.key === "Enter" && e.target === e.currentTarget && onClick()}
      className={`group flex h-7 items-center gap-2.5 rounded-md px-2.5 text-[13px] ${
        active ? "bg-selected font-medium text-ink" : "text-ink/85 hover:bg-selected/60"
      }`}
    >
      {children}
    </div>
  );
}
