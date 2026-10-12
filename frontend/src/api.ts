// One place to import the generated Go bindings from.
export {
  AIService,
  ConnectService,
  CueService,
  PrefsService,
  SystemService,
} from "../bindings/github.com/ShehabEddin64/Cue_Space/services";
export { HomePrefs, HomeSection, Prefs } from "../bindings/github.com/ShehabEddin64/Cue_Space/internal/settings";
export type { AISettings, Connections, Library, Problem } from "../bindings/github.com/ShehabEddin64/Cue_Space/services";
export {
  AppRef,
  Arrangement,
  ArrangementInfo,
  Cue,
  Desktop,
  Layout,
  Rect,
  Resource,
  ResourceType,
} from "../bindings/github.com/ShehabEddin64/Cue_Space/internal/cue";
export type { App } from "../bindings/github.com/ShehabEddin64/Cue_Space/internal/platform";
export { Status } from "../bindings/github.com/ShehabEddin64/Cue_Space/internal/runner";
export type { Result, StepResult } from "../bindings/github.com/ShehabEddin64/Cue_Space/internal/runner";
export { Config } from "../bindings/github.com/ShehabEddin64/Cue_Space/internal/ai";
export type { Disclosure, Proposal } from "../bindings/github.com/ShehabEddin64/Cue_Space/internal/ai";

import { Resource, ResourceType } from "../bindings/github.com/ShehabEddin64/Cue_Space/internal/cue";
import { locale, msg, t } from "./i18n";

/** Turns a rejected binding call into a sentence for the UI. */
export function errorMessage(e: unknown): string {
  if (e instanceof Error) return e.message;
  if (typeof e === "object" && e && "message" in e) return String((e as { message: unknown }).message);
  return String(e);
}

export function newId(): string {
  const bytes = new Uint8Array(8);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

export function newResource(type: ResourceType): Resource {
  return new Resource({ id: newId(), type });
}

/** What to show for a resource when it has no label. */
export function displayName(r: Resource): string {
  if (r.label) return r.label;
  switch (r.type) {
    case ResourceType.TypeURL: {
      // "moodle.concordia.ca" reads better than a long URL.
      try {
        const u = new URL(r.url || "");
        if ((u.protocol === "http:" || u.protocol === "https:") && u.hostname) return u.hostname.replace(/^www\./, "");
      } catch {
        /* not a full URL yet */
      }
      return r.url || t("Link");
    }
    case ResourceType.TypeApp:
      return r.app?.name || r.app?.bundleId || t("App");
    case ResourceType.TypeFile:
    case ResourceType.TypeFolder: {
      const parts = (r.path || "").replace(/\/+$/, "").split("/");
      const name = parts[parts.length - 1] || (r.type === ResourceType.TypeFile ? t("File") : t("Folder"));
      return r.type === ResourceType.TypeFolder && r.newest ? t("{kind} in {folder}", { kind: newestKind(r.extensions), folder: name }) : name;
    }
  }
  return t("Item");
}

/** "Newest file", "Newest PDF" or "Newest PDF or DOCX" (mirrors Go's NewestKind). */
export function newestKind(exts?: string[]): string {
  const e = (exts || []).map((x) => x.toUpperCase());
  if (!e.length) return t("Newest file");
  if (e.length === 1) return t("Newest {type}", { type: e[0] });
  return t("Newest {types} or {last}", { types: e.slice(0, -1).join(", "), last: e[e.length - 1] });
}

/** "pdf, .DOCX" → ["pdf", "docx"] */
export function parseExtensions(text: string): string[] {
  const out: string[] = [];
  for (const raw of text.split(/[\s,]+/)) {
    const e = raw.replace(/^[*.]+/, "").toLowerCase();
    if (e && !out.includes(e)) out.push(e);
  }
  return out;
}

export function timeAgo(iso?: string): string {
  if (!iso) return "";
  const then = new Date(iso).getTime();
  if (!then) return "";
  const s = Math.max(0, (Date.now() - then) / 1000);
  if (s < 60) return t("just now");
  if (s < 3600) return t("{n}m ago", { n: Math.floor(s / 60) });
  if (s < 86400) return t("{n}h ago", { n: Math.floor(s / 3600) });
  if (s < 86400 * 30) return t("{n}d ago", { n: Math.floor(s / 86400) });
  return new Date(iso).toLocaleDateString(locale());
}

/** Cues that have been started, most recent first. */
export function recentlyUsed<T extends { lastRunAt?: string }>(cues: T[]): T[] {
  return cues.filter((c) => c.lastRunAt).sort((a, b) => b.lastRunAt!.localeCompare(a.lastRunAt!));
}

function dayDiff(iso: string): number {
  const day = (d: Date) => new Date(d.getFullYear(), d.getMonth(), d.getDate()).getTime();
  return Math.round((day(new Date()) - day(new Date(iso))) / 86400000);
}

/** The heading a start time falls under, like Finder’s Recents. */
export function recentGroup(iso: string): string {
  const days = dayDiff(iso);
  if (days <= 0) return t("Today");
  if (days === 1) return t("Yesterday");
  if (days < 7) return t("Previous 7 Days");
  if (days < 30) return t("Previous 30 Days");
  return t("Earlier");
}

/** When a Cue was last started: “2h ago” today, a day and time before that. */
export function startedLabel(iso: string): string {
  const d = new Date(iso);
  const time = d.toLocaleTimeString(locale(), { hour: "numeric", minute: "2-digit" });
  const days = dayDiff(iso);
  if (days <= 0) return timeAgo(iso);
  if (days === 1) return t("Yesterday at {time}", { time });
  if (days < 7) return t("{day} at {time}", { day: d.toLocaleDateString(locale(), { weekday: "long" }), time });
  return d.toLocaleDateString(locale(), { month: "short", day: "numeric", year: days > 300 ? "numeric" : undefined });
}

/** Where AI requests go, in one honest sentence. */
export function disclosureSentence(d: { model: string; vendor: string; local: boolean; note?: string }): string {
  if (!d.model && !d.vendor) return "";
  if (d.local)
    return d.vendor
      ? t("Runs on this Mac with {model} in {vendor}. Nothing leaves your Mac.", { model: d.model, vendor: d.vendor })
      : t("Runs on this Mac with {model}. Nothing leaves your Mac.", { model: d.model });
  const base = t("Your description and the names of apps on this Mac are sent to {vendor} ({model}). Cue Space has no servers.", {
    vendor: d.vendor,
    model: d.model,
  });
  return d.note ? `${base} ${d.note}` : base;
}

/** Ideas the composer offers (shown inside it, Tab uses one), in the current language. */
export const composerIdeas = () => ideas.map((i) => t(i));

const ideas = [
  msg("Work on Atlas: my GitHub repo, my IDE, Claude and localhost:3000"),
  msg("Study linear algebra: my textbook PDF, my notes folder and the course website"),
  msg("Design session: Figma, my moodboard folder and Dribbble"),
  msg("Morning check-in: Mail, Calendar, Notion and the weather"),
  msg("Write my essay: Google Docs, my research folder and Google Scholar"),
  msg("Edit a video: CapCut, my footage folder and YouTube Studio"),
  msg("Plan the week: Calendar, Reminders and my Notion to-do list"),
  msg("Electronics project: KiCad, my datasheets folder and Digi-Key"),
  msg("Team meeting: Zoom, the agenda doc and Slack"),
];

/** "github.com/me/atlas" → "https://github.com/me/atlas"; leaves full URLs alone. */
export function normalizeURL(raw: string): string {
  const url = raw.trim();
  if (/^[\w-]+(\.[\w-]+)+(:\d+)?(\/\S*)?$/.test(url)) return "https://" + url;
  if (/^localhost(:\d+)?(\/\S*)?$/.test(url)) return "http://" + url;
  return url;
}

/** Pinned Cues first (in pinned order), then the rest by name. Matches the ⌘1–⌘9 order in the Cues menu. */
export function shortcutOrder<T extends { id: string }>(cues: T[], pinned: string[]): T[] {
  const byId = new Map(cues.map((c) => [c.id, c]));
  const first = pinned.map((id) => byId.get(id)).filter((c): c is T => !!c);
  const rest = cues.filter((c) => !pinned.includes(c.id));
  return [...first, ...rest];
}
