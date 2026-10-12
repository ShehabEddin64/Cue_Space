import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { HomePrefs, HomeSection, Prefs, PrefsService } from "./api";
import { setLanguage, type Lang } from "./i18n";

export const defaultPrefs = new Prefs({
  appearance: "system",
  language: "system",
  onboarded: true, // assume onboarded until the real prefs load, so the tour never flashes
  pinned: [],
  home: new HomePrefs({
    greeting: "",
    recentCount: 6,
    sections: [
      new HomeSection({ id: "pinned", visible: true, size: "medium" }),
      new HomeSection({ id: "recent", visible: true, size: "medium" }),
    ],
  }),
});

type PrefsContextValue = {
  prefs: Prefs;
  loaded: boolean;
  save: (change: (p: Prefs) => Prefs) => Promise<void>;
  togglePin: (id: string) => Promise<void>;
  lang: Lang;
};

const PrefsContext = createContext<PrefsContextValue>({
  prefs: defaultPrefs,
  loaded: false,
  save: async () => {},
  togglePin: async () => {},
  lang: "en",
});

export function PrefsProvider({ children }: { children: ReactNode }) {
  const [prefs, setPrefs] = useState<Prefs>(defaultPrefs);
  const [loaded, setLoaded] = useState(false);
  const [lang, setLang] = useState<Lang | null>(null);
  const current = useRef(prefs);
  current.current = prefs;

  useEffect(() => {
    PrefsService.GetPrefs()
      .then((p) => setPrefs(p))
      .finally(() => setLoaded(true));
  }, []);

  // The language follows the setting ("system" asks macOS). The module-level
  // language changes before the state does, so the re-render that follows
  // shows the new language everywhere without resetting the window.
  useEffect(() => {
    if (!loaded) return;
    let live = true;
    PrefsService.ResolveLanguage(prefs.language || "system")
      .catch(() => "en")
      .then((l) => {
        if (!live) return;
        const next: Lang = l === "fr" ? "fr" : "en";
        setLanguage(next);
        setLang(next);
      });
    return () => {
      live = false;
    };
  }, [loaded, prefs.language]);

  // Mirror the appearance on <html> too. macOS already switches the web
  // view's color scheme natively; this keeps CSS in step even before it does.
  useEffect(() => {
    const root = document.documentElement;
    if (prefs.appearance === "system") root.removeAttribute("data-theme");
    else root.setAttribute("data-theme", prefs.appearance);
  }, [prefs.appearance]);

  // Saves run one at a time, in order, so quick changes (like clicking
  // through sizes) can't land out of order and leave an older value stored.
  const queue = useRef<Promise<void>>(Promise.resolve());
  const latest = useRef(0);
  const save = useCallback((change: (p: Prefs) => Prefs) => {
    const next = change(current.current);
    current.current = next;
    setPrefs(next); // show it immediately
    const n = ++latest.current;
    queue.current = queue.current
      .then(() => PrefsService.SavePrefs(next))
      .then((stored) => {
        // Then show what was stored, unless a newer change is on its way.
        if (n === latest.current) {
          current.current = stored;
          setPrefs(stored);
        }
      })
      .catch(() => {});
    return queue.current;
  }, []);

  const togglePin = useCallback(
    (id: string) =>
      save((p) => new Prefs({ ...p, pinned: p.pinned.includes(id) ? p.pinned.filter((x) => x !== id) : [...p.pinned, id] })),
    [save],
  );

  // Wait for the language (a quick local call) so English never flashes first.
  if (!lang) return null;
  return <PrefsContext.Provider value={{ prefs, loaded, save, togglePin, lang }}>{children}</PrefsContext.Provider>;
}

export const usePrefs = () => useContext(PrefsContext);
