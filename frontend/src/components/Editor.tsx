import { closestCenter, DndContext, KeyboardSensor, PointerSensor, useSensor, useSensors, type DragEndEvent } from "@dnd-kit/core";
import { arrayMove, SortableContext, sortableKeyboardCoordinates, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { ArrowLeft, ArrowRight, Sparkles, Trash2 } from "lucide-react";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { AIService, Cue, CueService, errorMessage, newResource, Resource, ResourceType, SystemService } from "../api";
import { ResourceRow } from "./ResourceRow";
import { Button, Input, Kbd, TypeIcon, typeNames } from "./ui";

export type EditorSession = {
  cue: Cue;
  /** The saved version, or null for a Cue that has never been saved. */
  saved: Cue | null;
  questions?: Record<string, string | undefined>;
  notes?: string;
  fromAI?: boolean;
};

const addable = [ResourceType.TypeURL, ResourceType.TypeApp, ResourceType.TypeFile, ResourceType.TypeFolder];

/** Keeps only the last character typed, so the icon stays one emoji. */
function lastGrapheme(s: string) {
  const parts = [...new Intl.Segmenter().segment(s)];
  return parts.length ? parts[parts.length - 1].segment : "";
}

function looksLikeURL(text: string) {
  return /^[a-z][a-z0-9+.-]*:\/\/\S+$/i.test(text.trim());
}

export function Editor({
  session,
  aiReady,
  onClose,
  onSaved,
  onDeleted,
  onStart,
}: {
  session: EditorSession;
  aiReady: boolean;
  onClose: () => void;
  onSaved: (c: Cue) => void;
  onDeleted: () => void;
  onStart: (c: Cue) => void;
}) {
  const [draft, setDraft] = useState<Cue>(session.cue);
  const [saved, setSaved] = useState<Cue | null>(session.saved);
  const [questions, setQuestions] = useState(session.questions ?? {});
  const [notes, setNotes] = useState(session.notes ?? "");
  const [fromAI, setFromAI] = useState(!!session.fromAI);
  const [problems, setProblems] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [confirm, setConfirm] = useState<"discard" | "delete" | null>(null);
  const [focusId, setFocusId] = useState<string | null>(null);
  const [aiText, setAIText] = useState("");
  const [aiBusy, setAIBusy] = useState(false);
  const nameRef = useRef<HTMLInputElement>(null);

  const dirty = useMemo(() => !saved || JSON.stringify(draft) !== JSON.stringify(saved), [draft, saved]);

  useEffect(() => {
    if (!session.saved && !session.fromAI) nameRef.current?.focus();
  }, [session]);

  // Live validation from the same Go code that guards saving.
  useEffect(() => {
    const t = setTimeout(() => CueService.Problems(draft).then(setProblems), 250);
    return () => clearTimeout(t);
  }, [draft]);

  const set = (patch: Partial<Cue>) => setDraft((d) => new Cue({ ...d, ...patch }));
  const setResources = (fn: (rs: Resource[]) => Resource[]) => setDraft((d) => new Cue({ ...d, resources: fn(d.resources) }));

  const add = useCallback(async (type: ResourceType, init: Partial<Resource> = {}) => {
    const r = new Resource({ ...newResource(type), ...init });
    if ((type === ResourceType.TypeFile || type === ResourceType.TypeFolder) && !init.path) {
      const p = type === ResourceType.TypeFolder ? await SystemService.ChooseFolder() : await SystemService.ChooseFile();
      if (!p) return;
      r.path = p;
    }
    setDraft((d) => new Cue({ ...d, resources: [...d.resources, r] }));
    setFocusId(r.id);
  }, []);

  const save = useCallback(async () => {
    setSaving(true);
    setError("");
    try {
      const stored = await CueService.Save(draft);
      setDraft(stored);
      setSaved(stored);
      setQuestions({});
      setNotes("");
      setFromAI(false);
      onSaved(stored);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setSaving(false);
    }
  }, [draft, onSaved]);

  const close = useCallback(() => (dirty && !confirm ? setConfirm("discard") : onClose()), [dirty, confirm, onClose]);

  // ⌘S saves, Esc goes back, pasting a URL anywhere adds a link.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "s") {
        e.preventDefault();
        if (problems.length === 0 && dirty) save();
      } else if (e.key === "Escape" && !document.querySelector("[role=dialog]")) {
        close();
      }
    };
    const onPaste = (e: ClipboardEvent) => {
      const el = e.target as HTMLElement;
      if (el.closest("input, textarea")) return;
      const text = e.clipboardData?.getData("text") ?? "";
      if (looksLikeURL(text)) {
        e.preventDefault();
        add(ResourceType.TypeURL, { url: text.trim() });
      }
    };
    window.addEventListener("keydown", onKey);
    window.addEventListener("paste", onPaste);
    return () => {
      window.removeEventListener("keydown", onKey);
      window.removeEventListener("paste", onPaste);
    };
  }, [problems, dirty, save, close, add]);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );
  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (!over || active.id === over.id) return;
    setResources((rs) => arrayMove(rs, rs.findIndex((r) => r.id === active.id), rs.findIndex((r) => r.id === over.id)));
  };

  const askAI = async () => {
    if (!saved || !aiText.trim()) return;
    setAIBusy(true);
    setError("");
    try {
      const p = await AIService.Propose(aiText, saved.id);
      setDraft(p.cue);
      setQuestions(p.questions);
      setNotes(p.notes);
      setFromAI(true);
      setAIText("");
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setAIBusy(false);
    }
  };

  const remove = async () => {
    if (!saved) return onClose();
    try {
      await CueService.Delete(saved.id);
      onDeleted();
    } catch (e) {
      setError(errorMessage(e));
      setConfirm(null);
    }
  };

  const openQuestions = Object.values(questions).filter(Boolean).length;

  return (
    <div className="flex h-full flex-col">
      {/* Top bar */}
      <div className="drag flex h-[52px] shrink-0 items-center gap-2 border-b border-border pl-[84px] pr-4">
        <Button variant="ghost" size="sm" onClick={close}>
          <ArrowLeft className="size-3.5" />
          Cues
        </Button>
        <div className="flex-1" />
        {confirm === "discard" ? (
          <ConfirmBar text="Discard unsaved changes?" action="Discard" onConfirm={onClose} onCancel={() => setConfirm(null)} />
        ) : confirm === "delete" ? (
          <ConfirmBar text="Move this Cue to the Trash?" action="Move to Trash" onConfirm={remove} onCancel={() => setConfirm(null)} />
        ) : (
          <>
            {saved && (
              <Button variant="danger" size="sm" onClick={() => setConfirm("delete")}>
                <Trash2 className="size-3.5" />
                Delete
              </Button>
            )}
            {saved && !dirty && (
              <Button size="sm" onClick={() => onStart(saved)}>
                Start Cue
                <ArrowRight className="size-3.5" />
              </Button>
            )}
            <Button
              variant="primary"
              size="sm"
              busy={saving}
              disabled={!dirty || problems.length > 0}
              onClick={save}
              title={problems.length ? "Fix the issues below first" : "Save (⌘S)"}
            >
              {saved ? "Save" : "Save Cue"}
            </Button>
          </>
        )}
      </div>

      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-2xl px-6 pb-16 pt-6">
          {fromAI && (
            <div className="mb-5 rounded-xl border border-accent/25 bg-accent-soft px-4 py-3 text-[13px]">
              <div className="flex items-center gap-2 font-medium text-accent">
                <Sparkles className="size-4" />
                Proposed by AI. Review it, then save.
              </div>
              {(notes || openQuestions > 0) && (
                <p className="mt-1 text-xs leading-relaxed text-muted">
                  {notes}
                  {openQuestions > 0 && ` ${openQuestions} item${openQuestions === 1 ? " needs" : "s need"} your input.`}
                </p>
              )}
            </div>
          )}

          {/* Identity */}
          <div className="flex items-start gap-3">
            <input
              value={draft.icon || ""}
              onChange={(e) => set({ icon: lastGrapheme(e.target.value) })}
              placeholder="✦"
              aria-label="Icon (emoji)"
              title="Type or paste an emoji (⌃⌘Space)"
              className="size-14 shrink-0 rounded-2xl border border-border bg-surface-2 text-center text-[26px] placeholder:text-faint focus:border-accent focus:outline-none"
            />
            <div className="min-w-0 flex-1">
              <input
                ref={nameRef}
                value={draft.name}
                onChange={(e) => set({ name: e.target.value })}
                placeholder="Name this Cue"
                aria-label="Name"
                spellCheck={false}
                className="w-full bg-transparent text-2xl font-semibold tracking-tight placeholder:text-faint focus:outline-none"
              />
              <input
                value={draft.description || ""}
                onChange={(e) => set({ description: e.target.value })}
                placeholder="What is it for? (optional)"
                aria-label="Description"
                className="mt-1 w-full bg-transparent text-[13px] text-muted placeholder:text-faint focus:outline-none"
              />
            </div>
          </div>

          {/* Change with AI */}
          {saved && aiReady && (
            <div className="mt-6 flex gap-2">
              <div className="relative flex-1">
                <Sparkles className="pointer-events-none absolute left-2.5 top-2.5 size-3.5 text-accent" />
                <Input
                  value={aiText}
                  onChange={(e) => setAIText(e.target.value)}
                  onKeyDown={(e) => e.key === "Enter" && !dirty && askAI()}
                  placeholder={dirty ? "Save your changes to edit with AI" : "Change with AI, e.g. “add the Figma app and remove localhost”"}
                  disabled={dirty || aiBusy}
                  className="pl-8"
                />
              </div>
              <Button busy={aiBusy} disabled={dirty || !aiText.trim()} onClick={askAI}>
                Propose
              </Button>
            </div>
          )}

          {/* Resources */}
          <div className="mt-7 mb-2 flex items-center justify-between">
            <h3 className="text-xs font-semibold uppercase tracking-wide text-faint">Opens in this order</h3>
            <span className="text-[11px] text-faint">
              Paste a link anywhere to add it · <Kbd>⌘S</Kbd> to save
            </span>
          </div>

          <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
            <SortableContext items={draft.resources.map((r) => r.id)} strategy={verticalListSortingStrategy}>
              <ul className="space-y-2">
                {draft.resources.map((r, i) => (
                  <ResourceRow
                    key={r.id}
                    resource={r}
                    question={questions[r.id]}
                    autoFocus={focusId === r.id}
                    onChange={(next) => setResources((rs) => rs.map((x, j) => (j === i ? next : x)))}
                    onRemove={() => setResources((rs) => rs.filter((_, j) => j !== i))}
                    onDismissQuestion={() => setQuestions((q) => ({ ...q, [r.id]: undefined }))}
                  />
                ))}
              </ul>
            </SortableContext>
          </DndContext>

          {draft.resources.length === 0 && (
            <div className="rounded-xl border border-dashed border-border-strong px-4 py-8 text-center text-[13px] text-muted">
              Add the links, apps, files and folders you use for this activity.
            </div>
          )}

          <div className="mt-3 flex flex-wrap gap-2">
            {addable.map((t) => (
              <Button key={t} size="sm" onClick={() => add(t)}>
                <TypeIcon type={t} className="size-3.5" />
                {typeNames[t]}
              </Button>
            ))}
          </div>

          {(problems.length > 0 || error) && (
            <div className="mt-6 space-y-1 rounded-xl bg-surface-2 px-4 py-3 text-xs">
              {error && <p className="font-medium text-danger">{error}</p>}
              {problems.map((p) => (
                <p key={p} className="text-muted">
                  • {p}
                </p>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

function ConfirmBar({ text, action, onConfirm, onCancel }: { text: string; action: string; onConfirm: () => void; onCancel: () => void }) {
  return (
    <div className="no-drag animate-pop-in flex items-center gap-2">
      <span className="text-xs text-muted">{text}</span>
      <Button size="sm" variant="ghost" onClick={onCancel}>
        Cancel
      </Button>
      <Button size="sm" className="!bg-danger !text-white !border-transparent" onClick={onConfirm}>
        {action}
      </Button>
    </div>
  );
}
