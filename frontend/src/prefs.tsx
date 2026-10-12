import { createContext, useCallback, useContext, useEffect, useRef, useState, type ReactNode } from "react";
import { HomePrefs, HomeSection, Prefs, PrefsService } from "./api";

export const defaultPrefs = new Prefs({
  appearance: "system",
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
};

const PrefsContext = createContext<PrefsContextValue>({
  prefs: defaultPrefs,
  loaded: false,
  save: async () => {},
  togglePin: async () => {},
});

export function PrefsProvider({ children }: { children: ReactNode }) {
  const [prefs, setPrefs] = useState<Prefs>(defaultPrefs);
  const [loaded, setLoaded] = useState(false);
  const current = useRef(prefs);
  current.current = prefs;

  useEffect(() => {
    PrefsService.GetPrefs()
      .then((p) => setPrefs(p))
      .finally(() => setLoaded(true));
  }, []);

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

  return <PrefsContext.Provider value={{ prefs, loaded, save, togglePin }}>{children}</PrefsContext.Provider>;
}

export const usePrefs = () => useContext(PrefsContext);
