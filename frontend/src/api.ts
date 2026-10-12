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
  Cue,
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
    case ResourceType.TypeURL:
      return r.url || "Link";
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

/** Where AI requests go, in one honest sentence. */
export function disclosureSentence(d: { model: string; vendor: string; local: boolean }): string {
  if (!d.model && !d.vendor) return "";
  if (d.local) return `Runs on this Mac with ${d.model}${d.vendor ? ` in ${d.vendor}` : ""}. Nothing leaves your Mac.`;
  return `Your description and the names of apps on this Mac are sent to ${d.vendor} (${d.model}). Cue Space has no servers.`;
}

/** Examples that show what describing a Cue can do. */
export const examplePrompts = [
  { label: "Work on a project", text: "Work on Atlas: my GitHub repo, my IDE, Claude, localhost:3000 and the docs" },
  { label: "Study a subject", text: "Study linear algebra: my textbook PDF, notes folder, Calculator and the course website" },
  { label: "Design session", text: "Design session: Figma, my moodboard folder and Dribbble" },
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
