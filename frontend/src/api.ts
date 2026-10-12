// One place to import the generated Go bindings from.
export {
  AIService,
  CueService,
  PrefsService,
  SystemService,
} from "../bindings/github.com/ShehabEddin64/Cue_Space/services";
export { HomePrefs, HomeSection, Prefs } from "../bindings/github.com/ShehabEddin64/Cue_Space/internal/settings";
export type { AISettings, Library } from "../bindings/github.com/ShehabEddin64/Cue_Space/services";
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
      return r.url || "Link";
    }
    case ResourceType.TypeApp:
      return r.app?.name || r.app?.bundleId || "App";
    case ResourceType.TypeFile:
    case ResourceType.TypeFolder: {
      const parts = (r.path || "").replace(/\/+$/, "").split("/");
      return parts[parts.length - 1] || (r.type === ResourceType.TypeFile ? "File" : "Folder");
    }
  }
  return "Item";
}

export function timeAgo(iso?: string): string {
  if (!iso) return "";
  const then = new Date(iso).getTime();
  if (!then) return "";
  const s = Math.max(0, (Date.now() - then) / 1000);
  if (s < 60) return "just now";
  if (s < 3600) return `${Math.floor(s / 60)}m ago`;
  if (s < 86400) return `${Math.floor(s / 3600)}h ago`;
  if (s < 86400 * 30) return `${Math.floor(s / 86400)}d ago`;
  return new Date(iso).toLocaleDateString();
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
  if (days <= 0) return "Today";
  if (days === 1) return "Yesterday";
  if (days < 7) return "Previous 7 Days";
  if (days < 30) return "Previous 30 Days";
  return "Earlier";
}

/** When a Cue was last started: “2h ago” today, a day and time before that. */
export function startedLabel(iso: string): string {
  const d = new Date(iso);
  const time = d.toLocaleTimeString(undefined, { hour: "numeric", minute: "2-digit" });
  const days = dayDiff(iso);
  if (days <= 0) return timeAgo(iso);
  if (days === 1) return `Yesterday at ${time}`;
  if (days < 7) return `${d.toLocaleDateString(undefined, { weekday: "long" })} at ${time}`;
  return d.toLocaleDateString(undefined, { month: "short", day: "numeric", year: days > 300 ? "numeric" : undefined });
}

/** Where AI requests go, in one honest sentence. */
export function disclosureSentence(d: { model: string; vendor: string; local: boolean; note?: string }): string {
  if (!d.model && !d.vendor) return "";
  if (d.local) return `Runs on this Mac with ${d.model}${d.vendor ? ` in ${d.vendor}` : ""}. Nothing leaves your Mac.`;
  const base = `Your description and the names of apps on this Mac are sent to ${d.vendor} (${d.model}). Cue Space has no servers.`;
  return d.note ? `${base} ${d.note}` : base;
}

/** Examples that show what describing a Cue can do. */
/** Ideas the composer offers: shown inside it, Tab to use one. */
export const composerIdeas = [
  "Work on Atlas: my GitHub repo, my IDE, Claude and localhost:3000",
  "Study linear algebra: my textbook PDF, my notes folder and the course website",
  "Design session: Figma, my moodboard folder and Dribbble",
  "Morning check-in: Mail, Calendar, Notion and the weather",
  "Write my essay: Google Docs, my research folder and Google Scholar",
  "Edit a video: CapCut, my footage folder and YouTube Studio",
  "Plan the week: Calendar, Reminders and my Notion to-do list",
  "Electronics project: KiCad, my datasheets folder and Digi-Key",
  "Team meeting: Zoom, the agenda doc and Slack",
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
