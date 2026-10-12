import { closestCenter, DndContext, KeyboardSensor, PointerSensor, useSensor, useSensors, type DragEndEvent } from "@dnd-kit/core";
import { arrayMove, SortableContext, sortableKeyboardCoordinates, verticalListSortingStrategy } from "@dnd-kit/sortable";
import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import { Cue, CueService, errorMessage, newResource, Resource, ResourceType, SystemService } from "../api";
import { LayoutEditor } from "./Layout";
import { ResourceRow } from "./ResourceRow";
import { Toolbar } from "./Toolbar";
import { Button, CueMark, Kbd, SectionTitle, TypeIcon, typeNames } from "./ui";

export type EditorSession = {
  cue: Cue;
  /** The saved version, or null for a Cue that has never been saved. */
  saved: Cue | null;
  questions?: Record<string, string | undefined>;
  notes?: string;
  fromAI?: boolean;
  /** Open with the Move to Trash confirmation showing (from the menu bar). */
  askDelete?: boolean;
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

/** Building a Cue by hand: the path that never needs AI. */
export function Editor({
  session,
  onClose,
  onSaved,
  onDeleted,
}: {
  session: EditorSession;
  onClose: () => void;
  onSaved: (c: Cue) => void;
  onDeleted: () => void;
}) {
  const [draft, setDraft] = useState<Cue>(session.cue);
  const saved = session.saved;
  const [questions, setQuestions] = useState(session.questions ?? {});
  const [problems, setProblems] = useState<string[]>([]);
  const [error, setError] = useState("");
  const [saving, setSaving] = useState(false);
  const [confirm, setConfirm] = useState<"discard" | "delete" | null>(session.askDelete ? "delete" : null);
  const [focusId, setFocusId] = useState<string | null>(null);
  const nameRef = useRef<HTMLInputElement>(null);
  // Missing values (no name yet, an empty link) are only pointed out once
  // someone tries to save, so a new Cue or a just-added item doesn't greet
  // you with errors. A name typed and then cleared shows its hint right away.
  const [nameTouched, setNameTouched] = useState(false);
  const [triedSave, setTriedSave] = useState(false);

  const dirty = useMemo(() => !saved || JSON.stringify(draft) !== JSON.stringify(saved), [draft, saved]);

  useEffect(() => {
    if (!session.saved && !session.fromAI) nameRef.current?.focus();
  }, [session]);

  // Live validation from the same Go code that guards saving.
  useEffect(() => {
    const t = setTimeout(() => CueService.Problems(draft).then(setProblems), 200);
    return () => clearTimeout(t);
  }, [draft]);

  const nameMissing = problems.includes(NAME_MISSING);
  const showNameProblem = nameMissing && (nameTouched || triedSave);
  // The name has its own hint under the field; everything else is listed.
  const otherProblems = problems.filter((p) => p !== NAME_MISSING && (triedSave || !p.endsWith(MISSING)));
  const missingCount = problems.filter((p) => p.endsWith(MISSING)).length;

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
      onSaved(await CueService.Save(draft));
    } catch (e) {
      setError(errorMessage(e));
      setSaving(false);
    }
  }, [draft, onSaved]);

  // Save, or point at what's missing first.
  const trySave = useCallback(() => {
    if (missingCount > 0) {
      setTriedSave(true);
      if (nameMissing) nameRef.current?.focus();
      return;
    }
    if (problems.length === 0 && dirty) save();
  }, [missingCount, nameMissing, problems, dirty, save]);

  const close = useCallback(() => (dirty && !confirm ? setConfirm("discard") : onClose()), [dirty, confirm, onClose]);

  // ⌘S saves, Esc cancels, pasting a URL anywhere adds a link.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key === "s") {
        e.preventDefault();
        trySave();
      } else if (e.key === "Escape" && !document.querySelector("[role=dialog]")) {
        close();
      }
    };
    const onPaste = (e: ClipboardEvent) => {
      if ((e.target as HTMLElement).closest("input, textarea")) return;
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
  }, [trySave, close, add]);

  const sensors = useSensors(
    useSensor(PointerSensor, { activationConstraint: { distance: 4 } }),
    useSensor(KeyboardSensor, { coordinateGetter: sortableKeyboardCoordinates }),
  );
  const onDragEnd = ({ active, over }: DragEndEvent) => {
    if (!over || active.id === over.id) return;
    setResources((rs) => arrayMove(rs, rs.findIndex((r) => r.id === active.id), rs.findIndex((r) => r.id === over.id)));
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

  const openQuestions = draft.resources.filter((r) => questions[r.id]).length;

  return (
    <div className="flex h-full flex-col">
      <Toolbar
        leading={
          confirm ? null : (
            <Button variant="ghost" size="sm" onClick={close}>
              Cancel
            </Button>
          )
        }
        title={confirm ? null : saved ? "Edit Cue" : "New Cue"}
        trailing={
          confirm === "discard" ? (
            <ConfirmBar text="Discard your changes?" action="Discard" onConfirm={onClose} onCancel={() => setConfirm(null)} />
          ) : confirm === "delete" ? (
            <ConfirmBar text="Move this Cue to the Trash?" action="Move to Trash" onConfirm={remove} onCancel={() => setConfirm(null)} />
          ) : (
            <>
              {saved && (
                <Button variant="danger" size="sm" onClick={() => setConfirm("delete")}>
                  Move to Trash
                </Button>
              )}
              <Button
                variant="primary"
                size="sm"
                busy={saving}
                disabled={!dirty || otherProblems.length > 0 || showNameProblem}
                onClick={trySave}
                title={otherProblems.length ? "Fix the issues below first" : showNameProblem ? "Name this Cue first" : "Save (⌘S)"}
              >
                Save
              </Button>
            </>
          )
        }
      />

      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto max-w-[680px] px-8 pb-16 pt-4">
          {session.fromAI && (openQuestions > 0 || session.notes) && (
            <div className="mb-6 flex gap-2.5 rounded-2xl bg-red-tint px-4 py-3 text-[13px]">
              <CueMark size={16} className="mt-[2px]" />
              <p className="leading-relaxed text-red-on-tint">
                {openQuestions > 0 && `${openQuestions} item${openQuestions === 1 ? " needs" : "s need"} your input. `}
                {session.notes}
              </p>
            </div>
          )}

          {/* Identity */}
          <div className="flex items-center gap-4">
            <input
              value={draft.icon || ""}
              onChange={(e) => set({ icon: lastGrapheme(e.target.value) })}
              placeholder="◌"
              aria-label="Icon (emoji)"
              title="Type or paste an emoji (⌃⌘Space)"
              className="size-[72px] shrink-0 rounded-[20px] bg-surface text-center text-[34px] shadow-card placeholder:text-faint focus:outline-none focus:ring-4 focus:ring-red-tint"
            />
            <div className="min-w-0 flex-1">
              <input
                ref={nameRef}
                value={draft.name}
                onChange={(e) => {
                  set({ name: e.target.value });
                  if (e.target.value.trim()) setNameTouched(true);
                }}
                placeholder="Name this Cue"
                aria-label="Name"
                aria-invalid={showNameProblem}
                spellCheck={false}
                className="w-full bg-transparent text-[28px] font-semibold tracking-[-0.025em] placeholder:text-faint focus:outline-none"
              />
              <input
                value={draft.description || ""}
                onChange={(e) => set({ description: e.target.value })}
                placeholder="What is it for?"
                aria-label="Description"
                className="mt-0.5 w-full bg-transparent text-[15px] text-graphite placeholder:text-faint focus:outline-none"
              />
              {showNameProblem && <p className="mt-1 text-[12px] font-medium text-red-text">Give this Cue a name to save it.</p>}
            </div>
          </div>

          <div className="mt-9">
            <SectionTitle
              trailing={
                <span className="text-[11px] text-faint">
                  Drag to reorder · paste a link to add it · <Kbd>⌘S</Kbd>
                </span>
              }
            >
              Opens in this order
            </SectionTitle>

            <DndContext sensors={sensors} collisionDetection={closestCenter} onDragEnd={onDragEnd}>
              <SortableContext items={draft.resources.map((r) => r.id)} strategy={verticalListSortingStrategy}>
                <ul className="space-y-2">
                  {draft.resources.map((r, i) => (
                    <ResourceRow
                      key={r.id}
                      index={i}
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
              <div className="rounded-2xl border border-dashed border-hairline px-4 py-8 text-center text-[13px] text-graphite">
                Add the links, apps, files and folders you use for this activity.
              </div>
            )}

            <div className="mt-3 flex flex-wrap gap-2">
              {addable.map((t) => (
                <Button key={t} size="sm" onClick={() => add(t)}>
                  <TypeIcon type={t} className="size-3.5" />
                  Add {typeNames[t]}
                </Button>
              ))}
            </div>
          </div>

          <div className="mt-10">
            <SectionTitle trailing={<span className="text-[11px] text-faint">Needs Accessibility permission</span>}>Desktops</SectionTitle>
            <LayoutEditor cue={draft} onChange={(layout) => set({ layout })} />
          </div>

          {(otherProblems.length > 0 || error) && (
            <div className="mt-6 space-y-1 rounded-2xl bg-fill/70 px-4 py-3 text-[12px]">
              {error && <p className="font-medium text-red-text">{error}</p>}
              {otherProblems.map((p) => (
                <p key={p} className="text-graphite">
                  {p}
                </p>
              ))}
            </div>
          )}
        </div>
      </div>
    </div>
  );
}

// How Go's validation words values that aren't filled in yet
// (internal/cue/validate.go).
const MISSING = " is missing";
const NAME_MISSING = "Name" + MISSING;

function ConfirmBar({ text, action, onConfirm, onCancel }: { text: string; action: string; onConfirm: () => void; onCancel: () => void }) {
  return (
    <div className="animate-pop-in flex items-center gap-2">
      <span className="text-[12px] text-graphite">{text}</span>
      <Button size="sm" variant="ghost" onClick={onCancel}>
        Cancel
      </Button>
      <Button size="sm" variant="primary" onClick={onConfirm}>
        {action}
      </Button>
    </div>
  );
}
