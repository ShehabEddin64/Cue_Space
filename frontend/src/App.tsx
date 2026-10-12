import { Events } from "@wailsio/runtime";
import { useCallback, useEffect, useRef, useState } from "react";
import {
  AIService,
  Cue,
  CueService,
  errorMessage,
  SystemService,
  type AISettings,
  type Library as LibraryData,
  type Proposal,
} from "./api";
import { CueDetail } from "./components/CueDetail";
import { Editor, type EditorSession } from "./components/Editor";
import { Home, type HomeHandle } from "./components/Home";
import { RunToast, type RunReport } from "./components/RunToast";
import { SettingsDialog } from "./components/SettingsDialog";
import { Sidebar, type SidebarHandle } from "./components/Sidebar";
import { LayoutContext } from "./components/Toolbar";

type View = { kind: "home" } | { kind: "cue"; id: string } | { kind: "edit"; session: EditorSession; back: View };

function loadSidebarPref(): boolean {
  try {
    return localStorage.getItem("sidebar") !== "hidden";
  } catch {
    return true;
  }
}

export default function App() {
  const [library, setLibrary] = useState<LibraryData | null>(null);
  const [loadError, setLoadError] = useState("");
  const [ai, setAI] = useState<AISettings | null>(null);
  const [view, setView] = useState<View>({ kind: "home" });
  const [query, setQuery] = useState("");
  const [sidebarOpen, setSidebarOpen] = useState(loadSidebarPref);
  const [settingsOpen, setSettingsOpen] = useState(false);
  const [starting, setStarting] = useState<string | null>(null);
  const [report, setReport] = useState<(RunReport & { cueId: string }) | null>(null);
  const pendingPrompt = useRef<string | null>(null);
  const home = useRef<HomeHandle>(null);
  const sidebar = useRef<SidebarHandle>(null);

  const cues = library?.cues ?? [];
  const current = view.kind === "cue" ? cues.find((c) => c.id === view.id) : undefined;

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
  const newBlank = () => setView((v) => ({ kind: "edit", session: { cue: new Cue({ schemaVersion: 1, resources: [] }), saved: null }, back: v }));
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
    setSettingsOpen(true);
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
    switch (action) {
      case "settings":
        return setSettingsOpen(true);
      case "new-cue":
        return goHome(true);
      case "new-blank-cue":
        return newBlank();
      case "home":
        return goHome();
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
      case "delete-cue":
        return current && editCue(current, true);
    }
  };
  useEffect(() => Events.On("menu", (e: { data: string }) => onMenu.current(e.data)), []);

  const onAISaved = (s: AISettings) => {
    setAI(s);
    const pending = pendingPrompt.current;
    if (s.ready && pending) {
      pendingPrompt.current = null;
      setSettingsOpen(false);
      setView({ kind: "home" });
      requestAnimationFrame(() => home.current?.compose(pending));
    }
  };

  const closeReport = useCallback(() => setReport(null), []);

  return (
    <LayoutContext.Provider value={{ sidebarOpen, toggleSidebar }}>
      <div className="flex h-full">
        {sidebarOpen && (
          <Sidebar
            ref={sidebar}
            cues={cues}
            selected={view.kind === "home" ? "home" : view.kind === "cue" ? view.id : view.session.saved?.id ?? null}
            query={query}
            ai={ai}
            starting={starting}
            onQuery={setQuery}
            onHome={() => goHome()}
            onSelect={(c) => setView({ kind: "cue", id: c.id })}
            onStart={start}
            onNewBlank={newBlank}
            onSettings={() => setSettingsOpen(true)}
            onHide={toggleSidebar}
          />
        )}

        <main className="relative min-w-0 flex-1 bg-canvas">
          {loadError && (
            <p role="alert" className="absolute inset-x-0 top-[52px] z-10 bg-red-tint px-6 py-2 text-[12px] text-red-on-tint">
              Couldn’t read your Cues: {loadError}
            </p>
          )}
          {library && library.problems.length > 0 && view.kind !== "edit" && (
            <button
              onClick={() => SystemService.RevealCuesFolder()}
              className="absolute bottom-4 left-1/2 z-10 -translate-x-1/2 rounded-full bg-red-tint px-3.5 py-1.5 text-[12px] text-red-on-tint hover:underline"
              title={library.problems.map((p) => `${p.file}: ${p.error}`).join("\n")}
            >
              {library.problems.length === 1 ? "1 file" : `${library.problems.length} files`} in your Cues folder couldn’t be read. Show in
              Finder
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
              onConnect={connect}
              onSaved={openSaved}
              onEditProposal={editProposal}
            />
          )}

          {view.kind === "cue" && current && (
            <CueDetail
              key={current.id}
              cue={current}
              ai={ai}
              starting={starting === current.id}
              onStart={() => start(current)}
              onEdit={() => editCue(current)}
              onConnect={() => connect()}
              onSaved={openSaved}
              onEditProposal={editProposal}
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

      {settingsOpen && (
        <SettingsDialog
          folder={library?.folder ?? ""}
          onClose={() => {
            pendingPrompt.current = null;
            setSettingsOpen(false);
          }}
          onSaved={onAISaved}
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
