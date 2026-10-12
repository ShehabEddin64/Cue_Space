import { ArrowRight } from "lucide-react";
import { displayName, ResourceType, timeAgo, type AISettings, type Cue, type Proposal } from "../api";
import { useProposal } from "../useProposal";
import { Composer } from "./Composer";
import { ProposalCard } from "./ProposalCard";
import { Toolbar } from "./Toolbar";
import { Button, InsetList, SectionTitle, TypeIcon } from "./ui";

/** One Cue: start it, read what it opens, or change it (by hand or with AI). */
export function CueDetail({
  cue,
  ai,
  starting,
  onStart,
  onEdit,
  onConnect,
  onSaved,
  onEditProposal,
}: {
  cue: Cue;
  ai: AISettings | null;
  starting: boolean;
  onStart: () => void;
  onEdit: () => void;
  onConnect: () => void;
  onSaved: (c: Cue) => void;
  onEditProposal: (p: Proposal) => void;
}) {
  const flow = useProposal(cue, onSaved);
  const enabled = cue.resources.filter((r) => !r.disabled).length;

  return (
    <div className="flex h-full flex-col">
      <Toolbar />
      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-[680px] px-8 pb-16 pt-4">
          <header className="flex items-start gap-5">
            <div className="flex size-[72px] shrink-0 items-center justify-center rounded-[20px] bg-surface text-[36px] shadow-card">
              {cue.icon || "◌"}
            </div>
            <div className="min-w-0 flex-1 pt-1">
              <h1 className="truncate text-[28px] font-semibold leading-tight tracking-[-0.025em]">{cue.name}</h1>
              {cue.description && <p className="mt-1 text-[15px] leading-normal text-graphite">{cue.description}</p>}
              <p className="mt-1.5 text-[12px] text-faint">
                {enabled} item{enabled === 1 ? "" : "s"} · {cue.lastRunAt ? `Started ${timeAgo(cue.lastRunAt)}` : "Never started"}
              </p>
            </div>
          </header>

          <div className="mt-6 flex gap-2.5">
            <Button variant="primary" size="lg" busy={starting} disabled={enabled === 0} onClick={onStart} title="Start Cue (⌘Return)">
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
                compact
                ai={ai}
                busy={flow.busy === "compose"}
                placeholder="Change this Cue with AI, e.g. “add Figma and turn off localhost”"
                onSubmit={flow.compose}
                onCancel={flow.cancel}
                onConnect={onConnect}
              />
            )}
            {flow.error && (
              <p role="alert" className="mt-2.5 px-2 text-[13px] text-red-text">
                {flow.error}
              </p>
            )}
          </div>

          {!flow.proposal && (
            <section className="mt-9">
              <SectionTitle>Opens in this order</SectionTitle>
              {cue.resources.length === 0 ? (
                <p className="rounded-2xl border border-dashed border-hairline px-4 py-6 text-center text-[13px] text-graphite">
                  This Cue is empty. Edit it to add links, apps, files and folders.
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
