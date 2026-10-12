import { ArrowRight, Plus } from "lucide-react";
import { forwardRef, useEffect, useImperativeHandle, useRef } from "react";
import { examplePrompts, timeAgo, type AISettings, type Cue, type Proposal } from "../api";
import { useProposal } from "../useProposal";
import { Composer, type ComposerHandle } from "./Composer";
import { ProposalCard } from "./ProposalCard";
import { Toolbar } from "./Toolbar";
import { Button } from "./ui";

export type HomeHandle = { focusComposer: (text?: string) => void; compose: (text: string) => void };

/**
 * Home leads with the composer: say what you want to do and Cue Space
 * proposes the Cue. Building by hand is always one click away.
 */
export const Home = forwardRef<
  HomeHandle,
  {
    ai: AISettings | null;
    cues: Cue[];
    starting: string | null;
    onOpen: (c: Cue) => void;
    onStart: (c: Cue) => void;
    onNewBlank: () => void;
    onConnect: (pendingText?: string) => void;
    onSaved: (c: Cue) => void;
    onEditProposal: (p: Proposal) => void;
  }
>(function Home({ ai, cues, starting, onOpen, onStart, onNewBlank, onConnect, onSaved, onEditProposal }, ref) {
  const composer = useRef<ComposerHandle>(null);
  const flow = useProposal(null, onSaved);

  useImperativeHandle(ref, () => ({
    focusComposer: (text?: string) => (text ? composer.current?.setText(text) : composer.current?.focus()),
    compose: (text: string) => {
      composer.current?.setText(text);
      flow.compose(text);
    },
  }));

  // Once a proposal arrives, the composer empties so the card's Save is the
  // one primary action on screen.
  const hasProposalNow = !!flow.proposal;
  useEffect(() => {
    if (hasProposalNow) composer.current?.clear();
  }, [hasProposalNow]);

  const recent = [...cues]
    .sort((a, b) => (b.lastRunAt || b.updatedAt || "").localeCompare(a.lastRunAt || a.updatedAt || ""))
    .slice(0, 6);
  const hasProposal = !!flow.proposal;

  return (
    <div className="flex h-full flex-col">
      <Toolbar />
      <div className="flex-1 overflow-y-auto">
        <div className={`mx-auto w-full max-w-[680px] px-8 pb-16 ${hasProposal ? "pt-6" : "pt-[9vh]"}`}>
          {!hasProposal && (
            <header className="mb-7 text-center">
              <h1 className="text-[34px] font-semibold leading-[1.1] tracking-[-0.03em]">What do you want to do?</h1>
              <p className="mt-2.5 text-[15px] text-graphite">Describe it, and Cue Space gathers everything you need to start.</p>
            </header>
          )}

          <Composer
            ref={composer}
            ai={ai}
            busy={flow.busy === "compose"}
            placeholder={hasProposal ? "Describe a different Cue" : "Work on Atlas: my GitHub repo, my IDE, Claude and localhost"}
            onSubmit={flow.compose}
            onCancel={flow.cancel}
            onConnect={() => onConnect()}
          />

          {flow.error && (
            <p role="alert" className="mt-3 px-2 text-[13px] text-red-text">
              {flow.error}
            </p>
          )}

          {hasProposal ? (
            <div className="mt-5">
              <ProposalCard
                proposal={flow.proposal!}
                ai={ai}
                mode="new"
                refining={flow.busy === "refine"}
                canUndo={flow.canUndo}
                updated={flow.updated}
                onChange={flow.setProposal}
                onSave={flow.save}
                onEdit={() => {
                  onEditProposal(flow.proposal!);
                  flow.discard();
                }}
                onRetry={flow.retry}
                onDiscard={flow.discard}
                onRefine={flow.refine}
                onUndo={flow.undo}
              />
            </div>
          ) : (
            <div className="mt-4 flex flex-wrap items-center justify-center gap-2">
              {examplePrompts.map((ex) => (
                <button
                  key={ex.label}
                  title={ex.text}
                  onClick={() => (ai?.ready ? composer.current?.setText(ex.text) : onConnect(ex.text))}
                  className="rounded-full border border-hairline bg-surface px-3.5 py-1.5 text-[12.5px] text-graphite transition-colors hover:border-red-text/30 hover:text-ink"
                >
                  {ex.label}
                </button>
              ))}
              <span className="px-1 text-faint" aria-hidden="true">
                ·
              </span>
              <button onClick={onNewBlank} className="rounded-full px-2.5 py-1.5 text-[12.5px] font-medium text-graphite hover:text-ink">
                Build one yourself
              </button>
            </div>
          )}

          {recent.length > 0 && !hasProposal && (
            <section className="mt-16">
              <h2 className="mb-3 px-1 text-[13px] font-semibold">Jump back in</h2>
              <div className="grid grid-cols-[repeat(auto-fill,minmax(196px,1fr))] gap-3">
                {recent.map((c) => (
                  <RecentCard key={c.id} cue={c} starting={starting === c.id} onOpen={() => onOpen(c)} onStart={() => onStart(c)} />
                ))}
              </div>
            </section>
          )}

          {cues.length === 0 && !hasProposal && (
            <div className="mt-16 flex justify-center">
              <Button variant="ghost" onClick={onNewBlank}>
                <Plus className="size-3.5" />
                New Blank Cue
              </Button>
            </div>
          )}
        </div>
      </div>
    </div>
  );
});

function RecentCard({ cue, starting, onOpen, onStart }: { cue: Cue; starting: boolean; onOpen: () => void; onStart: () => void }) {
  const count = cue.resources.filter((r) => !r.disabled).length;
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(e) => e.key === "Enter" && e.target === e.currentTarget && onOpen()}
      className="group flex flex-col rounded-2xl border border-hairline bg-surface p-3.5 shadow-card transition-transform duration-150 hover:-translate-y-px"
    >
      <div className="flex items-center gap-2.5">
        <span className="flex size-9 shrink-0 items-center justify-center rounded-[11px] bg-fill text-[18px]">{cue.icon || "◌"}</span>
        <div className="min-w-0">
          <div className="truncate text-[14px] font-semibold tracking-[-0.01em]">{cue.name}</div>
          <div className="truncate text-[11.5px] text-graphite">
            {count} item{count === 1 ? "" : "s"}
            {cue.lastRunAt && ` · ${timeAgo(cue.lastRunAt)}`}
          </div>
        </div>
      </div>
      <Button
        variant="secondary"
        size="sm"
        className="mt-3.5 w-full group-hover:bg-red group-hover:text-white"
        busy={starting}
        disabled={count === 0}
        onClick={(e) => {
          e.stopPropagation();
          onStart();
        }}
      >
        Start Cue
        {!starting && <ArrowRight className="size-3.5" />}
      </Button>
    </div>
  );
}
