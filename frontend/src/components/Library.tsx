import { AlertTriangle, ArrowRight, FolderOpen, Plus, Sparkles } from "lucide-react";
import { displayName, SystemService, timeAgo, type Cue, type Library as LibraryData } from "../api";
import { Button, TypeIcon } from "./ui";

export function Library({
  library,
  query,
  starting,
  onStart,
  onOpen,
  onNew,
  onDescribe,
}: {
  library: LibraryData | null;
  query: string;
  starting: string | null;
  onStart: (c: Cue) => void;
  onOpen: (c: Cue) => void;
  onNew: () => void;
  onDescribe: () => void;
}) {
  if (!library) return null;
  const q = query.trim().toLowerCase();
  const cues = q
    ? library.cues.filter(
        (c) =>
          c.name.toLowerCase().includes(q) ||
          (c.description || "").toLowerCase().includes(q) ||
          c.resources.some((r) => displayName(r).toLowerCase().includes(q)),
      )
    : library.cues;

  return (
    <div className="mx-auto w-full max-w-5xl px-6 pb-10 pt-2">
      {library.problems.length > 0 && (
        <div className="mb-5 rounded-xl border border-warn/30 bg-warn-soft px-4 py-3 text-xs text-warn">
          <div className="flex items-center gap-2 font-medium">
            <AlertTriangle className="size-3.5" />
            {library.problems.length === 1 ? "One file in your Cues folder" : `${library.problems.length} files in your Cues folder`} could
            not be read
            <button className="ml-auto underline-offset-2 hover:underline" onClick={() => SystemService.RevealCuesFolder()}>
              Show in Finder
            </button>
          </div>
          <ul className="mt-1.5 space-y-0.5 pl-5.5 opacity-90">
            {library.problems.map((p) => (
              <li key={p.file} className="truncate">
                <span className="font-mono">{p.file}</span> — {p.error}
              </li>
            ))}
          </ul>
        </div>
      )}

      {library.cues.length === 0 ? (
        <EmptyState onNew={onNew} onDescribe={onDescribe} />
      ) : cues.length === 0 ? (
        <p className="py-20 text-center text-muted">No Cues match “{query}”.</p>
      ) : (
        <div className="grid grid-cols-[repeat(auto-fill,minmax(260px,1fr))] gap-4">
          {cues.map((c) => (
            <CueCard key={c.id} cue={c} starting={starting === c.id} onStart={() => onStart(c)} onOpen={() => onOpen(c)} />
          ))}
        </div>
      )}
    </div>
  );
}

function CueCard({ cue, starting, onStart, onOpen }: { cue: Cue; starting: boolean; onStart: () => void; onOpen: () => void }) {
  const enabled = cue.resources.filter((r) => !r.disabled);
  const shown = enabled.slice(0, 5);
  const more = enabled.length - shown.length;
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(e) => e.key === "Enter" && e.target === e.currentTarget && onOpen()}
      className="group flex min-h-48 flex-col rounded-2xl border border-border bg-surface p-4 shadow-card transition hover:border-border-strong"
    >
      <div className="flex items-start gap-3">
        <div className="flex size-10 shrink-0 items-center justify-center rounded-xl bg-surface-2 text-xl">{cue.icon || "✦"}</div>
        <div className="min-w-0 flex-1 pt-0.5">
          <h3 className="truncate text-[15px] font-semibold leading-tight">{cue.name}</h3>
          <p className="mt-0.5 truncate text-xs text-muted">{cue.description || `${enabled.length} item${enabled.length === 1 ? "" : "s"}`}</p>
        </div>
      </div>

      <ul className="mt-4 flex-1 space-y-1.5">
        {shown.map((r) => (
          <li key={r.id} className="flex items-center gap-2 text-[13px] text-muted">
            <TypeIcon type={r.type} className="size-3.5 shrink-0 text-faint" />
            <span className="truncate">{displayName(r)}</span>
          </li>
        ))}
        {more > 0 && <li className="pl-5.5 text-xs text-faint">+{more} more</li>}
        {enabled.length === 0 && <li className="text-xs text-faint">Nothing to open yet</li>}
      </ul>

      <div className="mt-4 flex items-center justify-between">
        <span className="text-[11px] text-faint">{cue.lastRunAt ? `Started ${timeAgo(cue.lastRunAt)}` : "Never started"}</span>
        <Button
          variant="primary"
          size="sm"
          busy={starting}
          disabled={enabled.length === 0}
          onClick={(e) => {
            e.stopPropagation();
            onStart();
          }}
        >
          Start Cue
          {!starting && <ArrowRight className="size-3.5" />}
        </Button>
      </div>
    </div>
  );
}

function EmptyState({ onNew, onDescribe }: { onNew: () => void; onDescribe: () => void }) {
  return (
    <div className="mx-auto flex max-w-md flex-col items-center py-20 text-center">
      <div className="mb-5 flex size-14 items-center justify-center rounded-2xl bg-accent-soft text-2xl">✦</div>
      <h2 className="text-lg font-semibold">What do you want to do?</h2>
      <p className="mt-2 text-[13px] leading-relaxed text-muted">
        A Cue holds everything you need for one activity: links, apps, files and folders. Start it and they all open, in order.
      </p>
      <div className="mt-6 flex gap-2">
        <Button variant="primary" onClick={onDescribe}>
          <Sparkles className="size-4" />
          Describe a Cue
        </Button>
        <Button onClick={onNew}>
          <Plus className="size-4" />
          Build one yourself
        </Button>
      </div>
      <button
        className="mt-8 inline-flex items-center gap-1.5 text-xs text-faint hover:text-muted"
        onClick={() => SystemService.RevealCuesFolder()}
      >
        <FolderOpen className="size-3.5" />
        Cues are plain .cue.json files. Show the folder
      </button>
    </div>
  );
}
