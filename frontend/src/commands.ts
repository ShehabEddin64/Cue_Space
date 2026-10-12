import { AIService, Cue, displayName, Resource, SystemService, type AISettings } from "./api";
import { msg, t } from "./i18n";

/*
 * Slash commands. Most run entirely on this Mac and cost no tokens; only
 * /create, plain sentences, and /add items that can't be found locally reach
 * the AI, and /add uses a much smaller request than a full proposal.
 */

// args and description are English; show them with t().
export type CommandDef = { name: string; args?: string; description: string; usesAI?: "always" | "sometimes" };

export const homeCommands: CommandDef[] = [
  { name: "create", args: msg("what it’s for"), description: msg("Propose a new Cue"), usesAI: "always" },
  { name: "new", args: msg("name"), description: msg("Create a Cue manually") },
  { name: "start", args: msg("Cue name"), description: msg("Start a Cue") },
  { name: "open", args: msg("Cue name"), description: msg("Open a Cue") },
  { name: "help", description: msg("Show every command") },
];

export const cueCommands: CommandDef[] = [
  { name: "add", args: msg("figma, ~/Notes, github.com/…"), description: msg("Add items; found on your Mac first"), usesAI: "sometimes" },
  { name: "remove", args: msg("item"), description: msg("Remove items") },
  { name: "off", args: msg("item"), description: msg("Skip items when the Cue starts") },
  { name: "on", args: msg("item"), description: msg("Turn items back on") },
  { name: "rename", args: msg("new name"), description: msg("Rename this Cue") },
  { name: "icon", args: msg("emoji"), description: msg("Change the icon") },
  { name: "describe", args: msg("text"), description: msg("Change the description") },
  { name: "start", description: msg("Start this Cue") },
  { name: "pin", description: msg("Pin or unpin this Cue") },
  { name: "help", description: msg("Show every command") },
];

/** The subset that works on an unsaved AI proposal. */
export const refineCommands: CommandDef[] = cueCommands.filter((c) =>
  ["add", "remove", "off", "on", "rename", "icon", "describe"].includes(c.name),
);

export function parseCommand(text: string): { name: string; arg: string } | null {
  const m = /^\/([a-z]+)\b\s*([\s\S]*)$/i.exec(text.trim());
  return m ? { name: m[1].toLowerCase(), arg: m[2].trim() } : null;
}

/** Commands whose name starts with what was typed after "/". */
export function suggestCommands(text: string, defs: CommandDef[]): CommandDef[] {
  const m = /^\/([a-z]*)$/i.exec(text);
  if (!m) return [];
  return defs.filter((d) => d.name.startsWith(m[1].toLowerCase()));
}

export function findCue(cues: Cue[], query: string): Cue | undefined {
  const q = query.trim().toLowerCase();
  if (!q) return undefined;
  return (
    cues.find((c) => c.name.toLowerCase() === q) ??
    cues.find((c) => c.name.toLowerCase().startsWith(q)) ??
    cues.find((c) => c.name.toLowerCase().includes(q))
  );
}

function matchItems(resources: Resource[], query: string): Resource[] {
  const parts = query
    .split(/\s*(?:,|\band\b)\s*/i)
    .map((p) => p.trim().toLowerCase())
    .filter(Boolean);
  return resources.filter((r) => {
    const name = displayName(r).toLowerCase();
    const target = (r.url || r.path || r.app?.name || "").toLowerCase();
    return parts.some((p) => name === p || name.includes(p) || target.includes(p));
  });
}

export type EditResult = {
  cue: Cue;
  /** What happened, in a short sentence for the UI. */
  message: string;
  /** Questions for new items the AI couldn't complete, keyed by resource id. */
  questions: Record<string, string | undefined>;
  changed: boolean;
};

/**
 * Applies an editing command to a Cue (saved or proposed) and returns the
 * new Cue. Returns null when the text isn't an editing command.
 */
export async function applyEdit(cue: Cue, text: string, ai: AISettings | null): Promise<EditResult | null> {
  const cmd = parseCommand(text);
  if (!cmd) return null;
  const done = (next: Partial<Cue>, message: string, questions = {}): EditResult => ({
    cue: new Cue({ ...cue, ...next }),
    message,
    questions,
    changed: true,
  });
  const nothing = (message: string): EditResult => ({ cue, message, questions: {}, changed: false });

  switch (cmd.name) {
    case "rename":
      return cmd.arg ? done({ name: cmd.arg }, t("Renamed to “{name}”", { name: cmd.arg })) : nothing(t("Type a name after /rename"));
    case "icon": {
      const icon = [...new Intl.Segmenter().segment(cmd.arg)][0]?.segment ?? "";
      return icon ? done({ icon }, t("Icon changed to {icon}", { icon })) : nothing(t("Type or paste an emoji after /icon"));
    }
    case "describe":
      return done({ description: cmd.arg }, cmd.arg ? t("Description updated") : t("Description cleared"));
    case "remove":
    case "off":
    case "on": {
      if (!cmd.arg) return nothing(t("Say which item, e.g. /{command} localhost", { command: cmd.name }));
      const hits = matchItems(cue.resources, cmd.arg);
      if (!hits.length) return nothing(t("No item matches “{query}”", { query: cmd.arg }));
      const ids = new Set(hits.map((r) => r.id));
      const names = hits.map(displayName).join(", ");
      if (cmd.name === "remove") return done({ resources: cue.resources.filter((r) => !ids.has(r.id)) }, t("Removed {names}", { names }));
      const disabled = cmd.name === "off";
      return done(
        { resources: cue.resources.map((r) => (ids.has(r.id) ? new Resource({ ...r, disabled }) : r)) },
        disabled ? t("Turned off {names}", { names }) : t("Turned on {names}", { names }),
      );
    }
    case "add": {
      if (!cmd.arg) return nothing(t("Say what to add, e.g. /add figma, ~/Notes, github.com/me/repo"));
      // First on this Mac: links, existing paths, installed apps. No tokens.
      const local = await SystemService.ResolveItems(cmd.arg);
      let added = [...local.resources];
      let questions: Record<string, string | undefined> = {};
      let note = "";
      if (local.unresolved.length) {
        if (ai?.ready) {
          // Only the leftovers go to the AI, with a short prompt.
          const res = await AIService.Resolve(local.unresolved);
          added = [...added, ...res.resources];
          questions = res.questions;
          note = t(" ({n} with AI)", { n: local.unresolved.length });
        } else {
          note = t(". Couldn’t find {items} on this Mac", { items: local.unresolved.map((u) => `“${u}”`).join(", ") });
        }
      }
      if (!added.length)
        return nothing(
          t("Couldn’t find {items} on this Mac. Connect an AI to resolve names like that.", {
            items: local.unresolved.map((u) => `“${u}”`).join(", "),
          }),
        );
      return done({ resources: [...cue.resources, ...added] }, t("Added {names}", { names: added.map(displayName).join(", ") }) + note, questions);
    }
  }
  return null;
}
