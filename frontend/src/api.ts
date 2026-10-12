// One place to import the generated Go bindings from.
export {
  AIService,
  CueService,
  SystemService,
} from "../bindings/github.com/ShehabEddin64/Cue_Space/services";
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
export type { Proposal } from "../bindings/github.com/ShehabEddin64/Cue_Space/internal/ai";

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
