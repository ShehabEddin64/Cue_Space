import { Plus, SlidersHorizontal } from "lucide-react";
import { forwardRef, useEffect, useImperativeHandle, useRef, useState, type ReactNode } from "react";
import { composerIdeas, HomePrefs, HomeSection, Prefs, recentlyUsed, shortcutOrder, type AISettings, type Cue, type Proposal } from "../api";
import { findCue, homeCommands, parseCommand } from "../commands";
import { usePrefs } from "../prefs";
import { useProposal } from "../useProposal";
import { Composer, type ComposerHandle } from "./Composer";
import { ProposalCard } from "./ProposalCard";
import { asSize, CueCollection, SizePicker, type SectionSize } from "./HomeWidgets";
import { Toolbar } from "./Toolbar";
import { AIErrorNotice, Button } from "./ui";

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
    onRecents: () => void;
    onConnect: (pendingText?: string) => void;
    onSaved: (c: Cue) => void;
    onEditProposal: (p: Proposal) => void;
    onCustomize: () => void;
    onHelp: () => void;
  }
>(function Home({ ai, cues, starting, onOpen, onStart, onNewBlank, onRecents, onConnect, onSaved, onEditProposal, onCustomize, onHelp }, ref) {
  const { prefs, save } = usePrefs();
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
  const used = recentlyUsed(cues);
  const recent = used.slice(0, prefs.home.recentCount);

  const sizeOf = (id: string) => asSize(prefs.home.sections.find((s) => s.id === id)?.size ?? "");
  const setSize = (id: string) => (size: SectionSize) =>
    save(
      (p) =>
        new Prefs({
          ...p,
          home: new HomePrefs({ ...p.home, sections: p.home.sections.map((s) => (s.id === id ? new HomeSection({ ...s, size }) : s)) }),
        }),
    );
  const collection = { starting, shortcut, pinned: (c: Cue) => prefs.pinned.includes(c.id), onOpen, onStart };

  const empty = (text: string) => (
    <p className="rounded-2xl border border-dashed border-hairline px-4 py-5 text-center text-[12.5px] text-graphite">{text}</p>
  );

  const sections: Record<string, ReactNode> = {
    pinned:
      cues.length > 0 ? (
        <Section title="Pinned" size={sizeOf("pinned")} onSize={setSize("pinned")}>
          {pinned.length > 0
            ? <CueCollection cues={pinned} size={sizeOf("pinned")} {...collection} />
            : empty("Pin the Cues you use most to keep them here and give them the first ⌘ numbers. Open a Cue and choose Pin, or press ⇧⌘P.")}
        </Section>
      ) : null,
    recent:
      cues.length > 0 ? (
        <Section
          title="Recently Used"
          size={sizeOf("recent")}
          onSize={setSize("recent")}
          trailing={
            used.length > recent.length && (
              <button onClick={onRecents} className="text-[12px] font-medium text-red-text hover:underline">
                See All
              </button>
            )
          }
        >
          {recent.length > 0 ? <CueCollection cues={recent} size={sizeOf("recent")} recent {...collection} /> : empty("Cues you start show up here, newest first.")}
        </Section>
      ) : null,
  };

  return (
    <div className="flex h-full flex-col">
      <Toolbar
        trailing={
          <>
          <Button variant="secondary" size="sm" onClick={() => onNewBlank()} title="Create a Cue Manually (⌥⌘N)">
            <Plus className="size-3.5" />
            Create Manually
          </Button>
          <Button variant="ghost" size="sm" onClick={onCustomize} title="Customize Home">
            <SlidersHorizontal className="size-3.5" />
            Customize
          </Button>
          </>
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
            busyLabel={flow.status || undefined}
            commands={homeCommands}
            ideas={hasProposal ? [] : composerIdeas}
            status={status}
            placeholder={hasProposal ? "Describe a different Cue" : "Work on Atlas: my GitHub repo, my IDE, Claude and localhost"}
            onSubmit={run}
            onCancel={flow.cancel}
            onConnect={() => onConnect()}
          />

          {!hasProposal && !flow.busy && (
            <div className="mt-4 flex flex-wrap items-center justify-center gap-x-3 gap-y-2 text-[13.5px] text-graphite">
              <span>Prefer to pick everything yourself?</span>
              <button
                onClick={() => onNewBlank()}
                title="Create a Cue Manually (⌥⌘N)"
                className="inline-flex h-10 items-center gap-2 rounded-full border border-hairline bg-surface px-5 text-[14px] font-semibold text-ink shadow-card transition-[border-color,transform] duration-150 hover:-translate-y-px hover:border-red-text/40"
              >
                <Plus className="size-4 text-red-text" strokeWidth={2.25} />
                Create Manually
              </button>
            </div>
          )}

          {flow.error && <AIErrorNotice message={flow.error} onRetry={flow.lastPrompt ? flow.retry : undefined} onSettings={() => onConnect()} />}

          {hasProposal ? (
            <div className="mt-5">
              <ProposalCard
                proposal={flow.proposal!}
                ai={ai}
                mode="new"
                refining={flow.busy === "refine"}
                status={flow.status}
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
            </div>
          )}
        </div>
      </div>
    </div>
  );
});

/** A titled Home section. Hovering its title shows the size control. */
function Section({
  title,
  size,
  onSize,
  trailing,
  children,
}: {
  title: string;
  size: SectionSize;
  onSize: (s: SectionSize) => void;
  trailing?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className="group/section">
      <div className="mb-3 flex h-7 items-center gap-3 px-1">
        <h2 className="text-[13px] font-semibold">{title}</h2>
        <div className="opacity-0 transition-opacity group-hover/section:opacity-100 has-[:focus-visible]:opacity-100">
          <SizePicker value={size} onChange={onSize} compact label={`${title} size`} />
        </div>
        <div className="flex-1" />
        {trailing}
      </div>
      {children}
    </section>
  );
}

