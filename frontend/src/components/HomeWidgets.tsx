import { ArrowRight, Pin, Play } from "lucide-react";
import type { ReactNode } from "react";
import { displayName, startedLabel, timeAgo, type Cue } from "../api";
import { msg, t, tn } from "../i18n";
import { RecentList } from "./Recents";
import { Button, CueMark, TypeIcon } from "./ui";

/*
 * Home's sections come in three sizes, like macOS widgets: Small packs many
 * Cues into compact tiles, Medium is the standard card or list, Large shows
 * what each Cue opens.
 */
export type SectionSize = "small" | "medium" | "large";

export const sizes: { id: SectionSize; label: string }[] = [
  { id: "small", label: msg("Small") },
  { id: "medium", label: msg("Medium") },
  { id: "large", label: msg("Large") },
];

export const asSize = (s: string): SectionSize => (s === "small" || s === "large" ? s : "medium");

/** A tiny drawing of each size: many tiles, two cards, one wide card. */
export function SizeGlyph({ size, className = "" }: { size: SectionSize; className?: string }) {
  return (
    <svg viewBox="0 0 16 12" className={`h-3 w-4 ${className}`} aria-hidden="true">
      {size === "small" &&
        [1, 6, 11].flatMap((x) => [1, 7].map((y) => <rect key={`${x}${y}`} x={x} y={y} width="4" height="4" rx="1" fill="currentColor" />))}
      {size === "medium" && [1, 9].map((x) => <rect key={x} x={x} y="1" width="6" height="10" rx="1.5" fill="currentColor" />)}
      {size === "large" && <rect x="1" y="1" width="14" height="10" rx="1.5" fill="currentColor" />}
    </svg>
  );
}

/** Small / Medium / Large, as a segmented control. */
export function SizePicker({
  value,
  onChange,
  compact,
  label,
}: {
  value: SectionSize;
  onChange: (s: SectionSize) => void;
  compact?: boolean;
  label: string;
}) {
  return (
    <div role="radiogroup" aria-label={label} className="inline-flex gap-0.5 rounded-full bg-fill p-0.5">
      {sizes.map((s) => (
        <button
          key={s.id}
          role="radio"
          aria-checked={value === s.id}
          aria-label={t(s.label)}
          title={t(s.label)}
          onClick={(e) => {
            e.stopPropagation();
            onChange(s.id);
          }}
          className={`flex h-6 items-center gap-1.5 rounded-full text-[11.5px] font-medium transition ${compact ? "px-2" : "px-2.5"} ${
            value === s.id ? "bg-surface text-ink shadow-card" : "text-graphite hover:text-ink"
          }`}
        >
          <SizeGlyph size={s.id} />
          {!compact && t(s.label)}
        </button>
      ))}
    </div>
  );
}

export function Grid({ children, min = 200 }: { children: ReactNode; min?: number }) {
  return (
    <div className="grid gap-3" style={{ gridTemplateColumns: `repeat(auto-fill, minmax(${min}px, 1fr))` }}>
      {children}
    </div>
  );
}

type CueHandlers = {
  starting: string | null;
  shortcut?: (c: Cue) => string | undefined;
  pinned?: (c: Cue) => boolean;
  onOpen: (c: Cue) => void;
  onStart: (c: Cue) => void;
};

/**
 * A set of Cues at one size. `recent` shows when each was started instead of
 * how many items it has, and uses a list at Medium.
 */
export function CueCollection({ cues, size, recent, ...h }: CueHandlers & { cues: Cue[]; size: SectionSize; recent?: boolean }) {
  if (size === "small") {
    return (
      <Grid min={168}>
        {cues.map((c) => (
          <CueTile key={c.id} cue={c} recent={recent} shortcut={h.shortcut?.(c)} starting={h.starting === c.id} onOpen={() => h.onOpen(c)} onStart={() => h.onStart(c)} />
        ))}
      </Grid>
    );
  }
  if (size === "large") {
    return (
      <Grid min={300}>
        {cues.map((c) => (
          <CueCardLarge
            key={c.id}
            cue={c}
            recent={recent}
            pinned={h.pinned?.(c)}
            shortcut={h.shortcut?.(c)}
            starting={h.starting === c.id}
            onOpen={() => h.onOpen(c)}
            onStart={() => h.onStart(c)}
          />
        ))}
      </Grid>
    );
  }
  if (recent) return <RecentList cues={cues} starting={h.starting} shortcut={h.shortcut} onOpen={h.onOpen} onStart={h.onStart} />;
  return (
    <Grid>
      {cues.map((c) => (
        <CueCard
          key={c.id}
          cue={c}
          pinned={h.pinned?.(c)}
          shortcut={h.shortcut?.(c)}
          starting={h.starting === c.id}
          onOpen={() => h.onOpen(c)}
          onStart={() => h.onStart(c)}
        />
      ))}
    </Grid>
  );
}

const enabled = (c: Cue) => c.resources.filter((r) => !r.disabled);

function openOnKey(onOpen: () => void) {
  return (e: React.KeyboardEvent) => e.key === "Enter" && e.target === e.currentTarget && onOpen();
}

/** Small: icon, name and a round Start button. */
function CueTile({
  cue,
  recent,
  shortcut,
  starting,
  onOpen,
  onStart,
}: {
  cue: Cue;
  recent?: boolean;
  shortcut?: string;
  starting: boolean;
  onOpen: () => void;
  onStart: () => void;
}) {
  const count = enabled(cue).length;
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={openOnKey(onOpen)}
      className="group flex items-center gap-2.5 rounded-2xl border border-hairline bg-surface py-2 pl-2 pr-2.5 shadow-card transition-transform duration-150 hover:-translate-y-px"
    >
      <span className="flex size-8 shrink-0 items-center justify-center rounded-[10px] bg-fill text-[16px]">{cue.icon || "◌"}</span>
      <div className="min-w-0 flex-1">
        <div className="truncate text-[13px] font-semibold tracking-[-0.01em]">{cue.name}</div>
        <div className="truncate text-[11px] text-graphite">
          {recent && cue.lastRunAt ? timeAgo(cue.lastRunAt) : shortcut || tn(count, "{n} item", "{n} items")}
        </div>
      </div>
      <button
        aria-label={t("Start {name}", { name: cue.name })}
        title={shortcut ? t("Start Cue ({shortcut})", { shortcut }) : t("Start Cue")}
        disabled={starting || count === 0}
        onClick={(e) => {
          e.stopPropagation();
          onStart();
        }}
        className="flex size-7 shrink-0 items-center justify-center rounded-full bg-fill text-red-text transition group-hover:bg-red group-hover:text-white disabled:opacity-45"
      >
        {starting ? <CueMark size={14} thinking ring="currentColor" dot="currentColor" /> : <Play className="size-3 translate-x-px fill-current" />}
      </button>
    </div>
  );
}

/** Medium: the standard card. */
export function CueCard({
  cue,
  shortcut,
  pinned,
  starting,
  onOpen,
  onStart,
  trailing,
}: {
  cue: Cue;
  shortcut?: string;
  pinned?: boolean;
  starting: boolean;
  onOpen: () => void;
  onStart: () => void;
  trailing?: ReactNode;
}) {
  const count = enabled(cue).length;
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={openOnKey(onOpen)}
      className="group relative flex flex-col rounded-2xl border border-hairline bg-surface p-3.5 shadow-card transition-transform duration-150 hover:-translate-y-px"
    >
      <div className="flex items-center gap-2.5">
        <span className="flex size-9 shrink-0 items-center justify-center rounded-[11px] bg-fill text-[18px]">{cue.icon || "◌"}</span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <span className="truncate text-[14px] font-semibold tracking-[-0.01em]">{cue.name}</span>
            {pinned && <Pin className="size-3 shrink-0 fill-current text-faint" aria-label={t("Pinned")} />}
          </div>
          <div className="truncate text-[11.5px] text-graphite">
            {tn(count, "{n} item", "{n} items")}
            {cue.lastRunAt && ` · ${timeAgo(cue.lastRunAt)}`}
          </div>
        </div>
        {trailing}
      </div>
      <StartButton className="mt-3.5 w-full" count={count} shortcut={shortcut} starting={starting} onStart={onStart} />
    </div>
  );
}

/** Large: the card plus what the Cue opens. */
function CueCardLarge({
  cue,
  recent,
  pinned,
  shortcut,
  starting,
  onOpen,
  onStart,
}: {
  cue: Cue;
  recent?: boolean;
  pinned?: boolean;
  shortcut?: string;
  starting: boolean;
  onOpen: () => void;
  onStart: () => void;
}) {
  const items = enabled(cue);
  const shown = items.slice(0, 4);
  const subtitle =
    recent && cue.lastRunAt
      ? t("Started {time}", { time: startedLabel(cue.lastRunAt) })
      : cue.description || tn(items.length, "{n} item", "{n} items");
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={openOnKey(onOpen)}
      className="group flex flex-col rounded-[20px] border border-hairline bg-surface p-4 shadow-card transition-transform duration-150 hover:-translate-y-px"
    >
      <div className="flex items-center gap-3">
        <span className="flex size-11 shrink-0 items-center justify-center rounded-[13px] bg-fill text-[22px]">{cue.icon || "◌"}</span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <span className="truncate text-[15px] font-semibold tracking-[-0.015em]">{cue.name}</span>
            {pinned && <Pin className="size-3 shrink-0 fill-current text-faint" aria-label={t("Pinned")} />}
          </div>
          <div className="truncate text-[12px] text-graphite">{subtitle}</div>
        </div>
      </div>
      <ul className="mt-3.5 flex-1 space-y-1.5 border-t border-hairline pt-3">
        {shown.map((r) => (
          <li key={r.id} className="flex items-center gap-2 text-[12.5px] text-ink/85">
            <TypeIcon type={r.type} className="size-3.5 shrink-0 text-graphite" />
            <span className="truncate">{displayName(r)}</span>
          </li>
        ))}
        {items.length > shown.length && <li className="pl-5.5 text-[11.5px] text-faint">{t("and {n} more", { n: items.length - shown.length })}</li>}
        {items.length === 0 && <li className="text-[12px] text-faint">{t("Nothing to open yet")}</li>}
      </ul>
      <StartButton className="mt-4 w-full" size="md" count={items.length} shortcut={shortcut} starting={starting} onStart={onStart} />
    </div>
  );
}

function StartButton({
  count,
  shortcut,
  starting,
  onStart,
  className,
  size = "sm",
}: {
  count: number;
  shortcut?: string;
  starting: boolean;
  onStart: () => void;
  className: string;
  size?: "sm" | "md";
}) {
  return (
    <Button
      variant="secondary"
      size={size}
      className={`${className} group-hover:bg-red group-hover:text-white`}
      busy={starting}
      disabled={count === 0}
      title={shortcut ? t("Start Cue ({shortcut})", { shortcut }) : t("Start Cue")}
      onClick={(e) => {
        e.stopPropagation();
        onStart();
      }}
    >
      {t("Start Cue")}
      {!starting && <ArrowRight className="size-3.5" />}
      {shortcut && !starting && <span className="pl-1 text-[10.5px] font-medium opacity-60">{shortcut}</span>}
    </Button>
  );
}
