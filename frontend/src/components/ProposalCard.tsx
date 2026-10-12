import { CornerDownLeft, HelpCircle, Info, Minus, RotateCcw, Undo2 } from "lucide-react";
import { useEffect, useState } from "react";
import {
  Cue,
  CueService,
  disclosureSentence,
  displayName,
  normalizeURL,
  Resource,
  ResourceType,
  SystemService,
  type AISettings,
  type Proposal,
} from "../api";
import { AppPicker } from "./AppPicker";
import { Button, CueMark, Input, TypeIcon } from "./ui";

/**
 * An AI proposal, reviewed in place. Nothing is saved until the person saves
 * it, and every result can be fixed, refined, retried, undone or discarded
 * (HIG › Generative AI › Keep people in control, Outputs).
 */
export function ProposalCard({
  proposal,
  ai,
  mode,
  refining,
  canUndo,
  updated,
  onChange,
  onSave,
  onEdit,
  onRetry,
  onDiscard,
  onRefine,
  onUndo,
}: {
  proposal: Proposal;
  ai: AISettings | null;
  mode: "new" | "edit";
  refining: boolean;
  canUndo: boolean;
  updated: boolean;
  onChange: (p: Proposal) => void;
  onSave: () => void;
  onEdit: () => void;
  onRetry: () => void;
  onDiscard: () => void;
  onRefine: (text: string) => void;
  onUndo: () => void;
}) {
  const [problems, setProblems] = useState<string[]>([]);
  const [refineText, setRefineText] = useState("");
  const cue = proposal.cue;

  useEffect(() => {
    const t = setTimeout(() => CueService.Problems(cue).then(setProblems), 200);
    return () => clearTimeout(t);
  }, [cue]);

  const setCue = (patch: Partial<Cue>) => onChange({ ...proposal, cue: new Cue({ ...cue, ...patch }) });
  const setResource = (i: number, next: Resource) => {
    // A question goes away once the thing it asked for is filled in.
    const resolved =
      next.type === ResourceType.TypeURL ? !!next.url : next.type === ResourceType.TypeApp ? !!next.app : !!next.path;
    const questions = resolved ? { ...proposal.questions, [next.id]: undefined } : proposal.questions;
    onChange({ ...proposal, questions, cue: new Cue({ ...cue, resources: cue.resources.map((r, j) => (j === i ? next : r)) }) });
  };
  const removeResource = (i: number) => setCue({ resources: cue.resources.filter((_, j) => j !== i) });

  const open = cue.resources.filter((r) => proposal.questions[r.id]).length;
  const blocked = problems.length > 0;
  const disclosure = ai ? disclosureSentence(ai.disclosure) : "";

  return (
    <section
      aria-label="Proposed Cue"
      className="animate-pop-in overflow-hidden rounded-[22px] border border-hairline bg-surface shadow-composer"
    >
      {/* Who made this, and that it is a proposal. */}
      <header className="flex items-center justify-between gap-3 px-5 pt-4">
        <div className="flex items-center gap-2 text-[12px] font-medium text-graphite">
          <CueMark size={14} />
          {mode === "new" ? "Proposed Cue" : "Proposed changes"}
          {updated && <span className="animate-pop-in rounded-full bg-red-tint px-2 py-px text-[11px] text-red-on-tint">Updated</span>}
        </div>
        {ai?.disclosure.model && <span className="truncate text-[11px] text-faint">Made with {ai.disclosure.model}</span>}
      </header>

      {/* Identity, editable in place. */}
      <div className="flex items-center gap-3.5 px-5 pb-4 pt-3">
        <div className="flex size-12 shrink-0 items-center justify-center rounded-[14px] bg-fill text-[24px]">{cue.icon || "◌"}</div>
        <div className="min-w-0 flex-1">
          <input
            value={cue.name}
            onChange={(e) => setCue({ name: e.target.value })}
            aria-label="Name"
            spellCheck={false}
            className="w-full bg-transparent text-[19px] font-semibold tracking-[-0.015em] focus:outline-none"
          />
          <input
            value={cue.description || ""}
            onChange={(e) => setCue({ description: e.target.value })}
            aria-label="Description"
            placeholder="Add a description"
            className="w-full bg-transparent text-[13px] text-graphite placeholder:text-faint focus:outline-none"
          />
        </div>
      </div>

      {/* What it will open, with inline fixes for anything the AI couldn't know. */}
      <ol className="mx-3 mb-3 divide-y divide-hairline rounded-2xl bg-canvas/70">
        {cue.resources.map((r, i) => (
          <ProposalRow
            key={r.id}
            index={i}
            resource={r}
            question={proposal.questions[r.id]}
            onChange={(next) => setResource(i, next)}
            onRemove={() => removeResource(i)}
          />
        ))}
        {cue.resources.length === 0 && <li className="px-4 py-4 text-[13px] text-graphite">Nothing to open yet.</li>}
      </ol>

      {proposal.notes && (
        <p className="mx-5 mb-3 flex gap-2 text-[12px] leading-relaxed text-graphite">
          <Info className="mt-[2px] size-3.5 shrink-0" />
          {proposal.notes}
        </p>
      )}

      {/* Refine without starting over. */}
      <div className="mx-5 mb-4 flex items-center gap-2 rounded-xl border border-hairline bg-surface px-3 focus-within:border-red-text/45">
        <input
          value={refineText}
          onChange={(e) => setRefineText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && refineText.trim() && !refining) {
              onRefine(refineText.trim());
              setRefineText("");
            }
          }}
          disabled={refining}
          placeholder="Refine it, e.g. “add Figma and drop localhost”"
          aria-label="Refine the proposal"
          className="h-9 flex-1 bg-transparent text-[13px] placeholder:text-faint focus:outline-none"
        />
        {refining ? (
          <CueMark size={16} thinking />
        ) : (
          <CornerDownLeft className="size-3.5 text-faint" />
        )}
      </div>

      <footer className="flex flex-wrap items-center gap-2 border-t border-hairline bg-canvas/50 px-5 py-3.5">
        <Button variant="primary" onClick={onSave} disabled={blocked || refining} title={blocked ? problems.join("\n") : undefined}>
          {mode === "new" ? "Save Cue" : "Save Changes"}
        </Button>
        <Button onClick={onEdit} disabled={refining}>
          Edit Details
        </Button>
        <Button variant="ghost" onClick={onRetry} disabled={refining} title="Ask again with the same description">
          <RotateCcw className="size-3.5" />
          Try Again
        </Button>
        {canUndo && (
          <Button variant="ghost" onClick={onUndo} disabled={refining}>
            <Undo2 className="size-3.5" />
            Undo
          </Button>
        )}
        <div className="flex-1" />
        <Button variant="ghost" onClick={onDiscard}>
          Discard
        </Button>
        {(open > 0 || blocked) && (
          <p className="w-full pt-1 text-[12px] text-graphite">
            {open > 0 ? `${open} item${open === 1 ? " needs" : "s need"} your input before saving.` : problems[0]}
          </p>
        )}
      </footer>

      {disclosure && (
        <p className="border-t border-hairline px-5 py-2.5 text-[11px] leading-relaxed text-faint">
          {disclosure} AI can get details wrong, so check each item. Starting a Cue never uses AI.
        </p>
      )}
    </section>
  );
}

function ProposalRow({
  index,
  resource: r,
  question,
  onChange,
  onRemove,
}: {
  index: number;
  resource: Resource;
  question?: string;
  onChange: (r: Resource) => void;
  onRemove: () => void;
}) {
  const update = (patch: Partial<Resource>) => onChange(new Resource({ ...r, ...patch }));
  const isPath = r.type === ResourceType.TypeFile || r.type === ResourceType.TypeFolder;
  const target =
    r.type === ResourceType.TypeURL
      ? r.url
      : r.type === ResourceType.TypeApp
        ? r.app?.name
        : [r.path, r.openWith?.name && `in ${r.openWith.name}`].filter(Boolean).join(" ");

  const choosePath = async () => {
    const p = r.type === ResourceType.TypeFolder ? await SystemService.ChooseFolder() : await SystemService.ChooseFile();
    if (p) update({ path: p });
  };

  return (
    <li className="group px-3.5 py-2.5">
      <div className="flex items-center gap-3">
        <span className="w-4 text-right font-mono text-[11px] text-faint">{index + 1}</span>
        <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-surface text-graphite shadow-card">
          <TypeIcon type={r.type} className="size-3.5" />
        </span>
        <div className="min-w-0 flex-1">
          <div className="truncate text-[13px] font-medium">{displayName(r)}</div>
          {target && <div className="truncate font-mono text-[11.5px] text-graphite">{target}</div>}
        </div>
        <button
          onClick={onRemove}
          aria-label={`Remove ${displayName(r)}`}
          title="Remove"
          className="flex size-6 items-center justify-center rounded-full text-faint opacity-0 transition hover:bg-fill hover:text-ink focus:opacity-100 group-hover:opacity-100"
        >
          <Minus className="size-3.5" />
        </button>
      </div>

      {question && (
        <div className="ml-[52px] mt-2 rounded-xl bg-red-tint px-3 py-2.5">
          <p className="flex items-start gap-1.5 text-[12px] font-medium text-red-on-tint">
            <HelpCircle className="mt-[1px] size-3.5 shrink-0" />
            {question}
          </p>
          <div className="mt-2 flex gap-1.5">
            {r.type === ResourceType.TypeURL && (
              <Input
                autoFocus={index === 0}
                defaultValue={r.url}
                placeholder="https://"
                onBlur={(e) => e.target.value.trim() && update({ url: normalizeURL(e.target.value) })}
                onKeyDown={(e) => e.key === "Enter" && (e.target as HTMLInputElement).blur()}
                className="font-mono text-xs"
              />
            )}
            {isPath && (
              <>
                <Button variant="tinted" onClick={choosePath}>
                  Choose {r.type === ResourceType.TypeFolder ? "Folder" : "File"}…
                </Button>
                <AppPicker value={r.openWith} onChange={(openWith) => update({ openWith })} placeholder="Open with default app" clearable />
              </>
            )}
            {r.type === ResourceType.TypeApp && <AppPicker value={r.app} onChange={(app) => app && update({ app })} />}
          </div>
        </div>
      )}
    </li>
  );
}
