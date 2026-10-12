import { Events } from "@wailsio/runtime";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  AIService,
  Cue,
  CueService,
  errorMessage,
  Prefs,
  shortcutOrder,
  SystemService,
  type AISettings,
  type Library as LibraryData,
  type Proposal,
  type Result,
} from "./api";
import { CueDetail } from "./components/CueDetail";
import { CustomizeHome } from "./components/CustomizeHome";
import { Editor, type EditorSession } from "./components/Editor";
import { Home, type HomeHandle } from "./components/Home";
import { ConnectSheet } from "./components/ConnectSheet";
import { Library } from "./components/Library";
import { Recents } from "./components/Recents";
import { Tutorial, type TutorialAction } from "./components/Tutorial";
import { t, tn } from "./i18n";
import { RunToast, type RunReport } from "./components/RunToast";
import { SettingsDialog } from "./components/SettingsDialog";
import { ShortcutsSheet } from "./components/ShortcutsSheet";
import { Sidebar, type SidebarHandle } from "./components/Sidebar";
import { LayoutContext } from "./components/Toolbar";
import { WelcomeTour } from "./components/WelcomeTour";
import { PrefsProvider, usePrefs } from "./prefs";

type View =
  | { kind: "home" }
  | { kind: "library" }
  | { kind: "recents" }
  | { kind: "tutorial" }
  | { kind: "cue"; id: string }
  | { kind: "edit"; session: EditorSession; back: View };

type Sheet = "settings" | "customize" | "shortcuts" | "tour" | "connect" | null;

function loadSidebarPref(): boolean {
  try {
    return localStorage.getItem("sidebar") !== "hidden";
  } catch {
    return true;
  }
}

export default function App() {
  return (
    <PrefsProvider>
      <Shell />
    </PrefsProvider>
  );
}

function Shell() {
  const { prefs, loaded, save, togglePin } = usePrefs();
  const [library, setLibrary] = useState<LibraryData | null>(null);
  const [loadError, setLoadError] = useState("");
  const [ai, setAI] = useState<AISettings | null>(null);
  const [view, setView] = useState<View>({ kind: "home" });
  const [query, setQuery] = useState("");
  const [sidebarOpen, setSidebarOpen] = useState(loadSidebarPref);
  const [sheet, setSheet] = useState<Sheet>(null);
  const [starting, setStarting] = useState<string | null>(null);
  const [report, setReport] = useState<(RunReport & { cueId: string }) | null>(null);
  const pendingPrompt = useRef<string | null>(null);
  const home = useRef<HomeHandle>(null);
  const sidebar = useRef<SidebarHandle>(null);

  const cues = library?.cues ?? [];
  const current = view.kind === "cue" ? cues.find((c) => c.id === view.id) : undefined;
  const order = shortcutOrder(cues, prefs.pinned);
  const shortcutFor = (c: Cue) => {
    const i = order.findIndex((x) => x.id === c.id);
    return i >= 0 && i < 9 ? `⌘${i + 1}` : undefined;
  };

  const reload = useCallback(() => {
    CueService.List()
      .then((lib) => {
        setLibrary(lib);
        setLoadError("");
      })
      .catch((e) => setLoadError(errorMessage(e)));
  }, []);

  useEffect(() => {
    reload();
    AIService.GetSettings().then(setAI).catch(() => setAI(null));
    // Edits to .cue.json files, from this app or any other, refresh everything.
    return Events.On("cues:changed", reload);
  }, [reload]);

  // First launch: show the welcome tour once.
  useEffect(() => {
    if (loaded && !prefs.onboarded) setSheet("tour");
  }, [loaded, prefs.onboarded]);

  // A Cue that disappears (deleted, or its file removed) sends you home.
  useEffect(() => {
    if (library && view.kind === "cue" && !current) setView({ kind: "home" });
  }, [library, view, current]);

  const toggleSidebar = useCallback(() => {
    setSidebarOpen((open) => {
      try {
        localStorage.setItem("sidebar", open ? "hidden" : "shown");
      } catch {
        /* preference only */
      }
      return !open;
    });
  }, []);

  const goHome = (focus = false) => {
    setView({ kind: "home" });
    if (focus) requestAnimationFrame(() => home.current?.focusComposer());
  };
  const newBlank = (name?: string) =>
    setView((v) => ({ kind: "edit", session: { cue: new Cue({ schemaVersion: 1, name: name ?? "", resources: [] }), saved: null }, back: v }));
  const editCue = (c: Cue, askDelete = false) => setView((v) => ({ kind: "edit", session: { cue: c, saved: c, askDelete }, back: v }));
  const editProposal = (p: Proposal) =>
    setView((v) => ({ kind: "edit", session: { cue: p.cue, saved: null, questions: p.questions, notes: p.notes, fromAI: true }, back: v }));
  const openSaved = (c: Cue) => {
    // Show the saved Cue right away; the reload (and the file watcher) follow.
    setLibrary((lib) => (lib ? { ...lib, cues: [...lib.cues.filter((x) => x.id !== c.id), c] } : lib));
    setView({ kind: "cue", id: c.id });
    reload();
  };
  const connect = (pending?: string) => {
    if (pending) pendingPrompt.current = pending;
    setSheet("settings");
  };
  const finishTour = () => {
    setSheet(null);
    if (!prefs.onboarded) save((p) => new Prefs({ ...p, onboarded: true }));
  };

  const start = async (c: Cue) => {
    setStarting(c.id);
    try {
      const result = await CueService.Start(c.id);
      setReport({ cueId: c.id, cueName: c.name, result });
    } catch (e) {
      setReport({ cueId: c.id, cueName: c.name, error: errorMessage(e) });
    } finally {
      setStarting(null);
      reload();
    }
  };

  // Menu bar commands (every command lives in the menu bar, per the HIG).
  const onMenu = useRef<(action: string) => void>(() => {});
  onMenu.current = (action: string) => {
    if (action.startsWith("start:")) {
      const c = cues.find((x) => x.id === action.slice(6));
      return c && void start(c);
    }
    if (action.startsWith("open:")) {
      return setView({ kind: "cue", id: action.slice(5) });
    }
    switch (action) {
      case "settings":
        return setSheet("settings");
      case "new-cue":
        return goHome(true);
      case "new-blank-cue":
        return newBlank();
      case "home":
        return goHome();
      case "library":
        return setView({ kind: "library" });
      case "recents":
        return setView({ kind: "recents" });
      case "tutorial":
        return setView({ kind: "tutorial" });
      case "customize-home":
        setView({ kind: "home" });
        return setSheet("customize");
      case "tour":
        return setSheet("tour");
      case "connect":
        return setSheet("connect");
      case "shortcuts":
        return setSheet("shortcuts");
      case "find":
        if (!sidebarOpen) toggleSidebar();
        return requestAnimationFrame(() => sidebar.current?.focusSearch());
      case "toggle-sidebar":
        return toggleSidebar();
      case "reveal-folder":
        return void SystemService.RevealCuesFolder();
      case "start-cue":
        return current && void start(current);
      case "edit-cue":
        return current && editCue(current);
      case "pin-cue":
        return current && void togglePin(current.id);
      case "delete-cue":
        return current && editCue(current, true);
    }
  };
  useEffect(() => Events.On("menu", (e: { data: string }) => onMenu.current(e.data)), []);

  // A Cue started from the widget or a cuespace:// link: show its report.
  useEffect(
    () =>
      Events.On("cue:ran", (e: { data: { cueId: string; cueName: string; result: Result; error?: string } }) => {
        const d = e.data;
        setReport(d.error ? { cueId: d.cueId, cueName: d.cueName, error: d.error } : { cueId: d.cueId, cueName: d.cueName, result: d.result });
        reload();
      }),
    [reload],
  );

  const onAISaved = (s: AISettings) => {
    setAI(s);
    const pending = pendingPrompt.current;
    if (s.ready && pending) {
      pendingPrompt.current = null;
      setSheet(null);
      setView({ kind: "home" });
      requestAnimationFrame(() => home.current?.compose(pending));
    }
  };

  const closeReport = useCallback(() => setReport(null), []);

  const selected =
    view.kind === "home" || view.kind === "library" || view.kind === "recents" || view.kind === "tutorial" ? view.kind : view.kind === "cue" ? view.id : (view.session.saved?.id ?? null);

  return (
    <LayoutContext.Provider value={{ sidebarOpen, toggleSidebar }}>
      <div className="flex h-full">
        {sidebarOpen && (
          <Sidebar
            ref={sidebar}
            cues={cues}
            selected={selected}
            query={query}
            ai={ai}
            starting={starting}
            onQuery={setQuery}
            onHome={() => goHome()}
            onLibrary={() => setView({ kind: "library" })}
            onRecents={() => setView({ kind: "recents" })}
            onTutorial={() => setView({ kind: "tutorial" })}
            onSelect={(c) => setView({ kind: "cue", id: c.id })}
            onStart={start}
            onNewBlank={() => newBlank()}
            onSettings={() => setSheet("settings")}
            onHide={toggleSidebar}
          />
        )}

        <main className="relative min-w-0 flex-1 bg-canvas">
          {loadError && (
            <p role="alert" className="absolute inset-x-0 top-[52px] z-10 bg-red-tint px-6 py-2 text-[12px] text-red-on-tint">
              {t("Couldn’t read your Cues: {error}", { error: loadError })}
            </p>
          )}
          {library && library.problems.length > 0 && view.kind !== "edit" && (
            <button
              onClick={() => SystemService.RevealCuesFolder()}
              className="absolute bottom-4 left-1/2 z-10 -translate-x-1/2 rounded-full bg-red-tint px-3.5 py-1.5 text-[12px] text-red-on-tint hover:underline"
              title={library.problems.map((p) => `${p.file}: ${p.error}`).join("\n")}
            >
              {tn(
                library.problems.length,
                "{n} file in your Cues folder couldn’t be read. Show in Finder",
                "{n} files in your Cues folder couldn’t be read. Show in Finder",
              )}
            </button>
          )}

          {view.kind === "home" && (
            <Home
              ref={home}
              ai={ai}
              cues={cues}
              starting={starting}
              onOpen={(c) => setView({ kind: "cue", id: c.id })}
              onStart={start}
              onNewBlank={newBlank}
              onRecents={() => setView({ kind: "recents" })}
              onConnect={connect}
              onSaved={openSaved}
              onEditProposal={editProposal}
              onCustomize={() => setSheet("customize")}
              onHelp={() => setSheet("shortcuts")}
            />
          )}

          {view.kind === "library" && (
            <Library
              cues={cues}
              starting={starting}
              onOpen={(c) => setView({ kind: "cue", id: c.id })}
              onStart={start}
              onNewBlank={() => newBlank()}
              onNew={() => goHome(true)}
            />
          )}

          {view.kind === "tutorial" && (
            <Tutorial
              onAction={(a: TutorialAction) =>
                a === "accessibility" ? void SystemService.RequestArrangeWindows() : onMenu.current(a)
              }
            />
          )}

          {view.kind === "recents" && (
            <Recents
              cues={cues}
              starting={starting}
              shortcut={shortcutFor}
              onOpen={(c) => setView({ kind: "cue", id: c.id })}
              onStart={start}
              onLibrary={() => setView({ kind: "library" })}
            />
          )}

          {view.kind === "cue" && current && (
            <CueDetail
              key={current.id}
              cue={current}
              ai={ai}
              starting={starting === current.id}
              shortcut={shortcutFor(current)}
              onStart={() => start(current)}
              onEdit={() => editCue(current)}
              onConnect={() => connect()}
              onSaved={openSaved}
              onEditProposal={editProposal}
              onHelp={() => setSheet("shortcuts")}
            />
          )}

          {view.kind === "edit" && (
            <Editor
              key={view.session.cue.id || "new"}
              session={view.session}
              onClose={() => setView(view.back.kind === "edit" ? { kind: "home" } : view.back)}
              onSaved={openSaved}
              onDeleted={() => {
                reload();
                setView({ kind: "home" });
              }}
            />
          )}
        </main>
      </div>

      {sheet === "settings" && (
        <SettingsDialog
          folder={library?.folder ?? ""}
          onClose={() => {
            pendingPrompt.current = null;
            setSheet(null);
          }}
          onSaved={onAISaved}
          onConnectApps={() => setSheet("connect")}
        />
      )}
      {sheet === "connect" && <ConnectSheet onClose={() => setSheet(null)} />}
      {sheet === "customize" && <CustomizeHome onClose={() => setSheet(null)} />}
      {sheet === "shortcuts" && <ShortcutsSheet onClose={() => setSheet(null)} />}
      {sheet === "tour" && (
        <WelcomeTour
          aiReady={!!ai?.ready}
          onDone={finishTour}
          onConnect={() => {
            finishTour();
            setSheet("settings");
          }}
        />
      )}

      {report && (
        <RunToast
          report={report}
          onClose={closeReport}
          onFix={() => {
            const c = cues.find((x) => x.id === report.cueId);
            setReport(null);
            if (c) editCue(c);
          }}
        />
      )}
    </LayoutContext.Provider>
  );
}
