import { fr } from "./i18n.fr";

/*
 * Translation. English text is the key: t("Start Cue") returns the French
 * when French is on and a translation exists, and the English otherwise, so
 * nothing is ever blank. Placeholders are written {name}:
 * t("Started {time}", { time }).
 *
 * t() reads the current language when it's called, so call it while
 * rendering, not when a module loads.
 */
export type Lang = "en" | "fr";

let current: Lang = "en";

export function setLanguage(lang: Lang) {
  current = lang;
  document.documentElement.lang = lang;
}

export function language(): Lang {
  return current;
}

/** The locale for dates and times. */
export function locale(): string {
  return current === "fr" ? "fr-CA" : "en-US";
}

export function t(text: string, vars?: Record<string, string | number>): string {
  let out = (current === "fr" && fr[text]) || text;
  if (vars) for (const [k, v] of Object.entries(vars)) out = out.split(`{${k}}`).join(String(v));
  return out;
}

/**
 * Marks English text kept in data (menus, lists, labels) so the checker
 * finds it. It returns the text unchanged; translate it with t() when shown.
 */
export const msg = (text: string) => text;

/** t() with a singular and a plural form: tn(n, "{n} item", "{n} items"). */
export function tn(n: number, one: string, other: string, vars?: Record<string, string | number>): string {
  return t(n === 1 ? one : other, { n, ...vars });
}
