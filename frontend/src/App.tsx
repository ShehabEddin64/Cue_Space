import { Events } from "@wailsio/runtime";
import { Plus, Search, Settings2, Sparkles } from "lucide-react";
import { useCallback, useEffect, useRef, useState } from "react";
import { AIService, Cue, CueService, errorMessage, type AISettings, type Library as LibraryData, type Proposal } from "./api";
import { DescribeDialog } from "./components/DescribeDialog";
import { Editor, type EditorSession } from "./components/Editor";
import { Library } from "./components/Library";
import { RunToast, type RunReport } from "./components/RunToast";
import { SettingsDialog } from "./components/SettingsDialog";
import { Button, IconButton, Kbd } from "./components/ui";

function isAIReady(s: AISettings | null) {
  if (!s || !s.config.provider) return false;
  return s.config.provider === "openai" || s.hasKey;
}

export default function App() {
  const [library, setLibrary] = useState<LibraryData | null>(null);
  const [loadError, setLoadError] = useState("");
  const [query, setQuery] = useState("");
  const [editing, setEditing] = useState<EditorSession | null>(null);
  const [dialog, setDialog] = useState<"describe" | "settings" | null>(null);
  const [ai, setAI] = useState<AISettings | null>(null);
  const [starting, setStarting] = useState<string | null>(null);
  const [report, setReport] = useState<(RunReport & { cueId: string }) | null>(null);
  const searchRef = useRef<HTMLInputElement>(null);

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
    // Edits to .cue.json files, from this app or any other, refresh the library.
    return Events.On("cues:changed", reload);
  }, [reload]);

  // ⌘N new, ⌘K / ⌘F search, ⌘, settings (on the library screen).
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (editing || dialog || !(e.metaKey || e.ctrlKey)) return;
      if (e.key === "n") {
        e.preventDefault();
        newCue();
      } else if (e.key === "k" || e.key === "f") {
        e.preventDefault();
        searchRef.current?.focus();
      } else if (e.key === ",") {
        e.preventDefault();
        setDialog("settings");
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  });

  const newCue = () => setEditing({ cue: new Cue({ schemaVersion: 1, resources: [] }), saved: null });

  const openCue = (c: Cue) => setEditing({ cue: c, saved: c });

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

  const onProposal = (p: Proposal) => {
    setDialog(null);
    setEditing({ cue: p.cue, saved: null, questions: p.questions, notes: p.notes, fromAI: true });
  };

  const closeReport = useCallback(() => setReport(null), []);

  return (
    <div className="flex h-full flex-col">
      {editing ? (
        <Editor
          key={editing.cue.id || "new"}
          session={editing}
          aiReady={isAIReady(ai)}
          onClose={() => setEditing(null)}
          onSaved={(c) => {
            setEditing((s) => (s ? { ...s, saved: c } : s));
            reload();
          }}
          onDeleted={() => {
            setEditing(null);
            reload();
          }}
          onStart={start}
        />
      ) : (
        <>
          <header className="drag flex h-[52px] shrink-0 items-center gap-3 pl-[84px] pr-4">
            <h1 className="text-[13px] font-semibold">Cue Space</h1>
            <div className="no-drag relative ml-auto w-full max-w-64">
              <Search className="pointer-events-none absolute left-2.5 top-2 size-3.5 text-faint" />
              <input
                ref={searchRef}
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={(e) => e.key === "Escape" && setQuery("")}
                placeholder="Search Cues"
                className="h-7 w-full rounded-lg bg-surface-2 pl-8 pr-10 text-xs placeholder:text-faint focus:outline-none focus:ring-2 focus:ring-accent-soft"
              />
              <span className="pointer-events-none absolute right-2 top-1.5">
                <Kbd>⌘K</Kbd>
              </span>
            </div>
            <Button size="sm" onClick={() => setDialog("describe")}>
              <Sparkles className="size-3.5 text-accent" />
              Describe
            </Button>
            <Button size="sm" variant="primary" onClick={newCue}>
              <Plus className="size-3.5" />
              New Cue
            </Button>
            <IconButton label="Settings (⌘,)" onClick={() => setDialog("settings")}>
              <Settings2 className="size-4" />
            </IconButton>
          </header>
          <main className="flex-1 overflow-y-auto">
            {loadError && <p className="px-6 py-4 text-xs text-danger">Couldn’t read your Cues: {loadError}</p>}
            <Library
              library={library}
              query={query}
              starting={starting}
              onStart={start}
              onOpen={openCue}
              onNew={newCue}
              onDescribe={() => setDialog("describe")}
            />
          </main>
        </>
      )}

      {dialog === "describe" && (
        <DescribeDialog
          aiReady={isAIReady(ai)}
          onClose={() => setDialog(null)}
          onProposal={onProposal}
          onOpenSettings={() => setDialog("settings")}
        />
      )}
      {dialog === "settings" && <SettingsDialog folder={library?.folder ?? ""} onClose={() => setDialog(null)} onSaved={setAI} />}

      {report && (
        <RunToast
          report={report}
          onClose={closeReport}
          onFix={() => {
            const c = library?.cues.find((x) => x.id === report.cueId);
            setReport(null);
            if (c) openCue(c);
          }}
        />
      )}
    </div>
  );
}
