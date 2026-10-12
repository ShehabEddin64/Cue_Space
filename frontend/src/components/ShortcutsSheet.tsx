import { cueCommands, homeCommands } from "../commands";
import { msg, t } from "../i18n";
import { Kbd, Sheet } from "./ui";

const keys: [string, string][] = [
  ["⌘1 … ⌘9", msg("Start your first nine Cues (pinned first)")],
  ["⌘N", msg("New Cue… (describe it)")],
  ["⌥⌘N", msg("New Cue manually")],
  ["Tab", msg("Use the idea shown in the composer")],
  ["↑ ↓", msg("See other ideas (composer empty)")],
  ["⌘Return", msg("Start the open Cue")],
  ["⌘E", msg("Edit the open Cue")],
  ["⇧⌘P", msg("Pin or unpin the open Cue")],
  ["⇧⌘H", msg("Home")],
  ["⇧⌘C", msg("All Cues")],
  ["⇧⌘R", msg("Recently used")],
  ["⌘F", msg("Find a Cue")],
  ["⌃⌘S", msg("Show or hide the sidebar")],
  ["⌘S", msg("Save while editing")],
  ["⌘,", msg("Settings")],
  ["⇧⌘T", msg("Tutorial")],
]; // descriptions shown with t()

/** Every shortcut and command on one page (Help › Keyboard Shortcuts, ⌘/). */
export function ShortcutsSheet({ onClose }: { onClose: () => void }) {
  const commands = [...homeCommands, ...cueCommands.filter((c) => !homeCommands.some((h) => h.name === c.name))];
  return (
    <Sheet title={t("Shortcuts and Commands")} onClose={onClose} width="max-w-2xl">
      <div className="grid gap-6 sm:grid-cols-2">
        <section>
          <h3 className="mb-2 text-[12px] font-semibold text-graphite">{t("Keyboard")}</h3>
          <ul className="space-y-1.5">
            {keys.map(([k, what]) => (
              <li key={k} className="flex items-center justify-between gap-3 text-[12.5px]">
                <span className="text-ink">{t(what)}</span>
                <Kbd>{k}</Kbd>
              </li>
            ))}
          </ul>
        </section>
        <section>
          <h3 className="mb-2 text-[12px] font-semibold text-graphite">{t("Commands (type / in a composer)")}</h3>
          <ul className="space-y-1.5">
            {commands.map((c) => (
              <li key={c.name} className="flex items-baseline gap-2 text-[12.5px]">
                <span className="w-[68px] shrink-0 font-mono font-medium">/{c.name}</span>
                <span className="min-w-0 flex-1 text-graphite">{t(c.description)}</span>
                <span className={`shrink-0 text-[10.5px] ${c.usesAI ? "text-red-text" : "text-faint"}`}>
                  {c.usesAI === "always" ? t("AI") : c.usesAI === "sometimes" ? t("AI if needed") : t("No AI")}
                </span>
              </li>
            ))}
          </ul>
          <p className="mt-3 text-[11.5px] leading-relaxed text-faint">
            {t("/start, /open and /new work on Home. /add, /remove, /off, /on, /rename, /icon and /pin work on a Cue’s page and while refining a proposal.")}
          </p>
        </section>
      </div>
    </Sheet>
  );
}
