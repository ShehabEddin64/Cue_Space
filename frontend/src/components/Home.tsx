import { ArrowRight, Pin, Plus, SlidersHorizontal } from "lucide-react";
import { forwardRef, useEffect, useImperativeHandle, useRef, useState, type ReactNode } from "react";
import { examplePrompts, shortcutOrder, timeAgo, type AISettings, type Cue, type Proposal } from "../api";
import { findCue, homeCommands, parseCommand } from "../commands";
import { usePrefs } from "../prefs";
import { useProposal } from "../useProposal";
import { Composer, type ComposerHandle } from "./Composer";
import { ProposalCard } from "./ProposalCard";
import { Toolbar } from "./Toolbar";
import { Button } from "./ui";

export type HomeHandle = { focusComposer: (text?: string) => void; compose: (text: string) => void };

export const DEFAULT_GREETING = "What do you want to do?";

/**
 * Home leads with the composer: say what you want to do and Cue Space
 * proposes the Cue. Everything below it can be shown, hidden and reordered.
 */
export const Home = forwardRef<
  HomeHandle,
  {
    ai: AISettings | null;
    cues: Cue[];
    starting: string | null;
    onOpen: (c: Cue) => void;
    onStart: (c: Cue) => void;
    onNewBlank: (name?: string) => void;
    onConnect: (pendingText?: string) => void;
    onSaved: (c: Cue) => void;
    onEditProposal: (p: Proposal) => void;
    onCustomize: () => void;
    onHelp: () => void;
  }
>(function Home({ ai, cues, starting, onOpen, onStart, onNewBlank, onConnect, onSaved, onEditProposal, onCustomize, onHelp }, ref) {
  const { prefs } = usePrefs();
  const composer = useRef<ComposerHandle>(null);
  const flow = useProposal(null, onSaved);
  const [status, setStatus] = useState("");

  useImperativeHandle(ref, () => ({
    focusComposer: (text?: string) => (text ? composer.current?.setText(text) : composer.current?.focus()),
    compose: (text: string) => {
      composer.current?.setText(text);
      flow.compose(text);
    },
  }));

  // Once a proposal arrives, the composer empties so the card's Save is the
  // one primary action on screen.
  const hasProposal = !!flow.proposal;
  useEffect(() => {
    if (hasProposal) composer.current?.clear();
  }, [hasProposal]);

  const say = (message: string) => {
    setStatus(message);
    setTimeout(() => setStatus((s) => (s === message ? "" : s)), 3500);
  };

  const run = (text: string) => {
    const cmd = parseCommand(text);
    if (!cmd) return flow.compose(text);
    const done = () => composer.current?.clear();
    switch (cmd.name) {
      case "create":
        if (!cmd.arg) return say("Say what the Cue is for, e.g. /create study math");
        if (!ai?.ready) return onConnect(cmd.arg);
        return flow.compose(cmd.arg);
      case "new":
        done();
        return onNewBlank(cmd.arg || undefined);
      case "start":
      case "open": {
        const c = findCue(cues, cmd.arg);
        if (!c) return say(cmd.arg ? `No Cue matches “${cmd.arg}”` : `Say which Cue, e.g. /${cmd.name} atlas`);
        done();
        return cmd.name === "start" ? onStart(c) : onOpen(c);
      }
      case "help":
        done();
        return onHelp();
    }
    say(`There’s no /${cmd.name} here. Type / to see commands.`);
  };

  const order = shortcutOrder(cues, prefs.pinned);
  const shortcut = (c: Cue) => {
    const i = order.findIndex((x) => x.id === c.id);
    return i >= 0 && i < 9 ? `⌘${i + 1}` : undefined;
  };
  const pinned = order.filter((c) => prefs.pinned.includes(c.id));
  const recent = [...cues]
    .filter((c) => !prefs.pinned.includes(c.id))
    .sort((a, b) => (b.lastRunAt || b.updatedAt || "").localeCompare(a.lastRunAt || a.updatedAt || ""))
    .slice(0, prefs.home.recentCount);

  const card = (c: Cue) => (
    <CueCard
      key={c.id}
      cue={c}
      shortcut={shortcut(c)}
      pinned={prefs.pinned.includes(c.id)}
      starting={starting === c.id}
      onOpen={() => onOpen(c)}
      onStart={() => onStart(c)}
    />
  );

  const sections: Record<string, ReactNode> = {
    suggestions: (
      <div className="flex flex-wrap items-center justify-center gap-2">
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
        <button onClick={() => onNewBlank()} className="rounded-full px-2.5 py-1.5 text-[12.5px] font-medium text-graphite hover:text-ink">
          Build one yourself
        </button>
      </div>
    ),
    pinned:
      pinned.length > 0 ? (
        <Section title="Pinned">
          <Grid>{pinned.map(card)}</Grid>
        </Section>
      ) : cues.length > 0 ? (
        <Section title="Pinned">
          <p className="rounded-2xl border border-dashed border-hairline px-4 py-5 text-center text-[12.5px] text-graphite">
            Pin the Cues you use most to keep them here and give them the first ⌘ numbers. Open a Cue and choose Pin, or press ⇧⌘P.
          </p>
        </Section>
      ) : null,
    recent:
      recent.length > 0 ? (
        <Section title="Jump back in">
          <Grid>{recent.map(card)}</Grid>
        </Section>
      ) : null,
  };

  return (
    <div className="flex h-full flex-col">
      <Toolbar
        trailing={
          <Button variant="ghost" size="sm" onClick={onCustomize} title="Customize Home">
            <SlidersHorizontal className="size-3.5" />
            Customize
          </Button>
        }
      />
      <div className="flex-1 overflow-y-auto">
        <div className={`mx-auto w-full max-w-[720px] px-8 pb-16 ${hasProposal ? "pt-4" : "pt-[7vh]"}`}>
          {!hasProposal && (
            <header className="mb-7 text-center">
              <h1 className="text-[34px] font-semibold leading-[1.1] tracking-[-0.03em]">{prefs.home.greeting || DEFAULT_GREETING}</h1>
              <p className="mt-2.5 text-[15px] text-graphite">Describe it, and Cue Space gathers everything you need to start.</p>
            </header>
          )}

          <Composer
            ref={composer}
            ai={ai}
            busy={flow.busy === "compose"}
            commands={homeCommands}
            status={status}
            placeholder={hasProposal ? "Describe a different Cue" : "Work on Atlas: my GitHub repo, my IDE, Claude and localhost"}
            onSubmit={run}
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
            <div className="mt-4 space-y-12">
              {prefs.home.sections
                .filter((s) => s.visible && sections[s.id])
                .map((s) => (
                  <div key={s.id}>{sections[s.id]}</div>
                ))}
              {cues.length === 0 && (
                <div className="flex justify-center">
                  <Button variant="ghost" onClick={() => onNewBlank()}>
                    <Plus className="size-3.5" />
                    New Blank Cue
                  </Button>
                </div>
              )}
            </div>
          )}
        </div>
      </div>
    </div>
  );
});

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section>
      <h2 className="mb-3 px-1 text-[13px] font-semibold">{title}</h2>
      {children}
    </section>
  );
}

export function Grid({ children }: { children: ReactNode }) {
  return <div className="grid grid-cols-[repeat(auto-fill,minmax(200px,1fr))] gap-3">{children}</div>;
}

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
  const count = cue.resources.filter((r) => !r.disabled).length;
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(e) => e.key === "Enter" && e.target === e.currentTarget && onOpen()}
      className="group relative flex flex-col rounded-2xl border border-hairline bg-surface p-3.5 shadow-card transition-transform duration-150 hover:-translate-y-px"
    >
      <div className="flex items-center gap-2.5">
        <span className="flex size-9 shrink-0 items-center justify-center rounded-[11px] bg-fill text-[18px]">{cue.icon || "◌"}</span>
        <div className="min-w-0 flex-1">
          <div className="flex items-center gap-1.5">
            <span className="truncate text-[14px] font-semibold tracking-[-0.01em]">{cue.name}</span>
            {pinned && <Pin className="size-3 shrink-0 fill-current text-faint" aria-label="Pinned" />}
          </div>
          <div className="truncate text-[11.5px] text-graphite">
            {count} item{count === 1 ? "" : "s"}
            {cue.lastRunAt && ` · ${timeAgo(cue.lastRunAt)}`}
          </div>
        </div>
        {trailing}
      </div>
      <Button
        variant="secondary"
        size="sm"
        className="mt-3.5 w-full group-hover:bg-red group-hover:text-white"
        busy={starting}
        disabled={count === 0}
        title={shortcut ? `Start Cue (${shortcut})` : "Start Cue"}
        onClick={(e) => {
          e.stopPropagation();
          onStart();
        }}
      >
        Start Cue
        {!starting && <ArrowRight className="size-3.5" />}
        {shortcut && !starting && <span className="pl-1 text-[10.5px] font-medium opacity-60">{shortcut}</span>}
      </Button>
    </div>
  );
}
