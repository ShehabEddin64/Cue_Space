import { ArrowRight, Pin, PinOff, Undo2 } from "lucide-react";
import { useRef, useState } from "react";
import { CueService, displayName, errorMessage, ResourceType, timeAgo, type AISettings, type Cue, type Proposal } from "../api";
import { applyEdit, cueCommands, parseCommand } from "../commands";
import { usePrefs } from "../prefs";
import { useProposal } from "../useProposal";
import { Composer, type ComposerHandle } from "./Composer";
import { ProposalCard } from "./ProposalCard";
import { Toolbar } from "./Toolbar";
import { Button, InsetList, SectionTitle, TypeIcon } from "./ui";

/** One Cue: start it, read what it opens, or change it with commands, AI, or by hand. */
export function CueDetail({
  cue,
  ai,
  starting,
  shortcut,
  onStart,
  onEdit,
  onConnect,
  onSaved,
  onEditProposal,
  onHelp,
}: {
  cue: Cue;
  ai: AISettings | null;
  starting: boolean;
  shortcut?: string;
  onStart: () => void;
  onEdit: () => void;
  onConnect: () => void;
  onSaved: (c: Cue) => void;
  onEditProposal: (p: Proposal) => void;
  onHelp: () => void;
}) {
  const { prefs, togglePin } = usePrefs();
  const flow = useProposal(cue, onSaved);
  const composer = useRef<ComposerHandle>(null);
  const [status, setStatus] = useState("");
  const [busy, setBusy] = useState(false);
  const [undo, setUndo] = useState<{ before: Cue; message: string } | null>(null);
  const enabled = cue.resources.filter((r) => !r.disabled).length;
  const pinned = prefs.pinned.includes(cue.id);

  const say = (message: string) => {
    setStatus(message);
    setTimeout(() => setStatus((s) => (s === message ? "" : s)), 3500);
  };

  // Commands run on this Mac and apply right away, with Undo. Sentences go to
  // the AI as a proposal to review.
  const run = async (text: string) => {
    const cmd = parseCommand(text);
    if (!cmd) {
      if (!ai?.ready) return onConnect();
      return flow.compose(text);
    }
    switch (cmd.name) {
      case "start":
        composer.current?.clear();
        return onStart();
      case "pin":
        composer.current?.clear();
        await togglePin(cue.id);
        return say(pinned ? "Unpinned" : "Pinned to Home and the top of the sidebar");
      case "help":
        composer.current?.clear();
        return onHelp();
    }
    setBusy(true);
    try {
      const res = await applyEdit(cue, text, ai);
      if (!res) return say(`There’s no /${cmd.name} here. Type / to see commands.`);
      if (!res.changed) return say(res.message);
      composer.current?.clear();
      if (Object.values(res.questions).some(Boolean)) {
        // Something new needs an answer (a path, a private URL): review it first.
        flow.setProposal({ cue: res.cue, questions: res.questions, notes: res.message });
        return;
      }
      const saved = await CueService.Save(res.cue);
      setUndo({ before: cue, message: res.message });
      onSaved(saved);
    } catch (e) {
      say(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  const revert = async () => {
    if (!undo) return;
    const saved = await CueService.Save(undo.before);
    setUndo(null);
    onSaved(saved);
  };

  return (
    <div className="flex h-full flex-col">
      <Toolbar
        trailing={
          <Button variant="ghost" size="sm" onClick={() => togglePin(cue.id)} title={pinned ? "Unpin (⇧⌘P)" : "Pin to Home (⇧⌘P)"}>
            {pinned ? <PinOff className="size-3.5" /> : <Pin className="size-3.5" />}
            {pinned ? "Unpin" : "Pin"}
          </Button>
        }
      />
      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-[680px] px-8 pb-16 pt-4">
          <header className="flex items-start gap-5">
            <div className="flex size-[72px] shrink-0 items-center justify-center rounded-[20px] bg-surface text-[36px] shadow-card">
              {cue.icon || "◌"}
            </div>
            <div className="min-w-0 flex-1 pt-1">
              <h1 className="flex items-center gap-2 truncate text-[28px] font-semibold leading-tight tracking-[-0.025em]">
                {cue.name}
                {pinned && <Pin className="size-4 shrink-0 fill-current text-faint" aria-label="Pinned" />}
              </h1>
              {cue.description && <p className="mt-1 text-[15px] leading-normal text-graphite">{cue.description}</p>}
              <p className="mt-1.5 text-[12px] text-faint">
                {enabled} item{enabled === 1 ? "" : "s"} · {cue.lastRunAt ? `Started ${timeAgo(cue.lastRunAt)}` : "Never started"}
                {shortcut && ` · ${shortcut} to start`}
              </p>
            </div>
          </header>

          <div className="mt-6 flex gap-2.5">
            <Button variant="primary" size="lg" busy={starting} disabled={enabled === 0} onClick={onStart} title={`Start Cue (${shortcut ?? "⌘Return"})`}>
              Start Cue
              {!starting && <ArrowRight className="size-4" />}
            </Button>
            <Button size="lg" onClick={onEdit} title="Edit Cue (⌘E)">
              Edit Cue
            </Button>
          </div>

          <div className="mt-8">
            {flow.proposal ? (
              <ProposalCard
                proposal={flow.proposal}
                ai={ai}
                mode="edit"
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
            ) : (
              <Composer
                ref={composer}
                compact
                ai={ai}
                busy={flow.busy === "compose" || busy}
                busyLabel={busy ? "Updating this Cue" : undefined}
                commands={cueCommands}
                status={status}
                placeholder="Change this Cue: /add figma, /off localhost, or describe it"
                onSubmit={run}
                onCancel={flow.cancel}
                onConnect={onConnect}
              />
            )}
            {flow.error && (
              <p role="alert" className="mt-2.5 px-2 text-[13px] text-red-text">
                {flow.error}
              </p>
            )}
            {undo && !flow.proposal && (
              <div className="animate-pop-in mt-2.5 flex items-center gap-2 px-2 text-[12.5px] text-graphite" role="status">
                <span className="min-w-0 flex-1 truncate">{undo.message}</span>
                <Button variant="ghost" size="sm" onClick={revert}>
                  <Undo2 className="size-3.5" />
                  Undo
                </Button>
              </div>
            )}
          </div>

          {!flow.proposal && (
            <section className="mt-9">
              <SectionTitle>Opens in this order</SectionTitle>
              {cue.resources.length === 0 ? (
                <p className="rounded-2xl border border-dashed border-hairline px-4 py-6 text-center text-[13px] text-graphite">
                  This Cue is empty. Try <span className="font-mono text-ink">/add</span> with a link, an app or a folder.
                </p>
              ) : (
                <InsetList>
                  {cue.resources.map((r, i) => (
                    <li key={r.id} className={`flex items-center gap-3 px-4 py-3 ${r.disabled ? "opacity-50" : ""}`}>
                      <span className="w-4 text-right font-mono text-[11px] text-faint">{i + 1}</span>
                      <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-fill text-graphite">
                        <TypeIcon type={r.type} className="size-3.5" />
                      </span>
                      <div className="min-w-0 flex-1">
                        <div className="truncate text-[13px] font-medium">{displayName(r)}</div>
                        <div className="truncate font-mono text-[11.5px] text-graphite">{target(r)}</div>
                      </div>
                      {r.disabled && <span className="rounded-full bg-fill px-2 py-0.5 text-[11px] text-graphite">Off</span>}
                    </li>
                  ))}
                </InsetList>
              )}
            </section>
          )}
        </div>
      </div>
    </div>
  );
}

function target(r: Cue["resources"][number]): string {
  switch (r.type) {
    case ResourceType.TypeURL:
      return r.url || "";
    case ResourceType.TypeApp:
      return r.app?.name || r.app?.bundleId || "";
    default:
      return [r.path, r.openWith?.name && `in ${r.openWith.name}`].filter(Boolean).join(" ");
  }
}
