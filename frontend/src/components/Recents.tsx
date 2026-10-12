import { ArrowRight, Clock } from "lucide-react";
import { recentGroup, recentlyUsed, startedLabel, type Cue } from "../api";
import { t, tn } from "../i18n";
import { Toolbar } from "./Toolbar";
import { Button, InsetList, SectionTitle } from "./ui";

/** A list of Cues with when each was last started and a Start button. */
export function RecentList({
  cues,
  starting,
  shortcut,
  onOpen,
  onStart,
}: {
  cues: Cue[];
  starting: string | null;
  shortcut?: (c: Cue) => string | undefined;
  onOpen: (c: Cue) => void;
  onStart: (c: Cue) => void;
}) {
  return (
    <InsetList>
      {cues.map((c) => {
        const count = c.resources.filter((r) => !r.disabled).length;
        const key = shortcut?.(c);
        return (
          <li
            key={c.id}
            role="button"
            tabIndex={0}
            onClick={() => onOpen(c)}
            onKeyDown={(e) => e.key === "Enter" && e.target === e.currentTarget && onOpen(c)}
            className="group flex items-center gap-3 px-3.5 py-2.5 hover:bg-fill/50"
          >
            <span className="flex size-8 shrink-0 items-center justify-center rounded-[10px] bg-fill text-[16px]">{c.icon || "◌"}</span>
            <div className="min-w-0 flex-1">
              <div className="truncate text-[13.5px] font-semibold tracking-[-0.01em]">{c.name}</div>
              <div className="truncate text-[11.5px] text-graphite">
                {c.lastRunAt ? t("Started {time}", { time: startedLabel(c.lastRunAt) }) : t("Never started")} · {tn(count, "{n} item", "{n} items")}
              </div>
            </div>
            {key && starting !== c.id && <span className="text-[10.5px] font-medium text-faint">{key}</span>}
            <Button
              variant="secondary"
              size="sm"
              className="group-hover:bg-red group-hover:text-white"
              busy={starting === c.id}
              disabled={count === 0}
              title={key ? t("Start Cue ({shortcut})", { shortcut: key }) : t("Start Cue")}
              onClick={(e) => {
                e.stopPropagation();
                onStart(c);
              }}
            >
              {t("Start")}
              {starting !== c.id && <ArrowRight className="size-3.5" />}
            </Button>
          </li>
        );
      })}
    </InsetList>
  );
}

/** Every Cue you’ve started, newest first, grouped by day like Finder’s Recents. */
export function Recents({
  cues,
  starting,
  shortcut,
  onOpen,
  onStart,
  onLibrary,
}: {
  cues: Cue[];
  starting: string | null;
  shortcut: (c: Cue) => string | undefined;
  onOpen: (c: Cue) => void;
  onStart: (c: Cue) => void;
  onLibrary: () => void;
}) {
  const recent = recentlyUsed(cues);
  const groups: { title: string; cues: Cue[] }[] = [];
  for (const c of recent) {
    const title = recentGroup(c.lastRunAt!);
    if (groups.at(-1)?.title !== title) groups.push({ title, cues: [] });
    groups.at(-1)!.cues.push(c);
  }

  return (
    <div className="flex h-full flex-col">
      <Toolbar title={t("Recently Used")} />
      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-[720px] px-8 pb-16 pt-2">
          {recent.length === 0 ? (
            <div className="mx-auto max-w-sm py-16 text-center">
              <Clock className="mx-auto size-8 text-faint" strokeWidth={1.5} />
              <h2 className="mt-3 text-[17px] font-semibold">{t("Nothing started yet")}</h2>
              <p className="mt-1.5 text-[13px] text-graphite">{t("Cues you start show up here, newest first.")}</p>
              {cues.length > 0 && (
                <Button className="mt-5" onClick={onLibrary}>
                  {t("Show All Cues")}
                </Button>
              )}
            </div>
          ) : (
            <div className="space-y-7">
              {groups.map((g) => (
                <section key={g.title}>
                  <SectionTitle>{g.title}</SectionTitle>
                  <RecentList cues={g.cues} starting={starting} shortcut={shortcut} onOpen={onOpen} onStart={onStart} />
                </section>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
