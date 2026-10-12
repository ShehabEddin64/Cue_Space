import { useSortable } from "@dnd-kit/sortable";
import { CSS } from "@dnd-kit/utilities";
import { AlertTriangle, Clock, GripVertical, HelpCircle, Trash2, X } from "lucide-react";
import { useEffect, useState } from "react";
import { displayName, normalizeURL, parseExtensions, Resource, ResourceType, SystemService } from "../api";
import { AppPicker } from "./AppPicker";
import { Button, IconButton, Input, TypeIcon, typeNames } from "./ui";

export function ResourceRow({
  index,
  resource: r,
  question,
  autoFocus,
  onChange,
  onRemove,
  onDismissQuestion,
}: {
  index: number;
  resource: Resource;
  question?: string;
  autoFocus?: boolean;
  onChange: (r: Resource) => void;
  onRemove: () => void;
  onDismissQuestion: () => void;
}) {
  const { attributes, listeners, setNodeRef, setActivatorNodeRef, transform, transition, isDragging } = useSortable({ id: r.id });
  const update = (patch: Partial<Resource>) => onChange(new Resource({ ...r, ...patch }));
  const missing = usePathMissing(r);
  const isPath = r.type === ResourceType.TypeFile || r.type === ResourceType.TypeFolder;

  const choosePath = async () => {
    const p = r.type === ResourceType.TypeFolder ? await SystemService.ChooseFolder() : await SystemService.ChooseFile();
    if (p) update({ path: p });
  };

  return (
    <li
      ref={setNodeRef}
      style={{ transform: CSS.Translate.toString(transform), transition }}
      className={`group relative rounded-xl border bg-surface transition-shadow ${
        isDragging ? "z-10 border-red-text shadow-pop" : "border-hairline"
      } ${r.disabled ? "opacity-55" : ""}`}
    >
      <div className="flex items-start gap-2 p-2.5">
        <button
          ref={setActivatorNodeRef}
          {...attributes}
          {...listeners}
          aria-label="Drag to reorder"
          className="no-drag mt-1.5 cursor-grab touch-none rounded p-0.5 text-faint hover:text-graphite active:cursor-grabbing"
        >
          <GripVertical className="size-4" />
        </button>

        <span className="mt-[7px] w-4 shrink-0 text-right font-mono text-[11px] text-faint" aria-label={`Item ${index + 1}`}>
          {index + 1}
        </span>
        <div
          title={typeNames[r.type]}
          className="mt-0.5 flex size-7 shrink-0 items-center justify-center rounded-lg bg-fill text-graphite"
        >
          <TypeIcon type={r.type} className="size-3.5" />
        </div>

        <div className="min-w-0 flex-1 space-y-1.5">
          <input
            value={r.label || ""}
            onChange={(e) => update({ label: e.target.value })}
            placeholder={displayName(new Resource({ ...r, label: "" }))}
            spellCheck={false}
            aria-label="Label"
            className="h-7 w-full rounded-md bg-transparent px-1 text-[13px] font-medium placeholder:font-normal placeholder:text-faint hover:bg-fill focus:bg-fill focus:outline-none"
          />

          {r.type === ResourceType.TypeURL && (
            <Input
              autoFocus={autoFocus}
              value={r.url || ""}
              onChange={(e) => update({ url: e.target.value })}
              onBlur={() => {
                const url = normalizeURL(r.url || "");
                if (url !== r.url) update({ url });
              }}
              placeholder="https://"
              aria-label="URL"
              className="font-mono text-xs"
            />
          )}

          {r.type === ResourceType.TypeApp && <AppPicker value={r.app} onChange={(app) => update({ app })} />}

          {isPath && (
            <>
              <div className="flex gap-1.5">
                <Input
                  value={r.path || ""}
                  onChange={(e) => update({ path: e.target.value })}
                  placeholder={r.type === ResourceType.TypeFolder ? "~/path/to/folder" : "~/path/to/file"}
                  aria-label="Path"
                  className={`font-mono text-xs ${missing ? "border-red-text/50" : ""}`}
                />
                <Button size="md" onClick={choosePath}>
                  Choose…
                </Button>
              </div>
              {missing && (
                <p className="flex items-center gap-1.5 px-1 text-[11px] text-red-on-tint">
                  <AlertTriangle className="size-3" />
                  Not found on this Mac
                </p>
              )}
              {r.type === ResourceType.TypeFolder && <NewestOption resource={r} onChange={update} />}
              <div className="flex items-center gap-2">
                <span className="shrink-0 pl-1 text-[11px] text-faint">Open with</span>
                <AppPicker
                  value={r.openWith}
                  onChange={(openWith) => update({ openWith })}
                  placeholder="Default app"
                  clearable
                />
              </div>
            </>
          )}

          {question && (
            <div className="flex items-start gap-2 rounded-lg bg-red-tint px-2.5 py-2 text-xs text-red-on-tint">
              <HelpCircle className="mt-px size-3.5 shrink-0" />
              <span className="flex-1">{question}</span>
              <button aria-label="Dismiss" onClick={onDismissQuestion} className="opacity-70 hover:opacity-100">
                <X className="size-3.5" />
              </button>
            </div>
          )}
        </div>

        <div className="flex shrink-0 items-center gap-0.5 pt-0.5">
          <Toggle checked={!r.disabled} onChange={(on) => update({ disabled: !on })} label={r.disabled ? "Turned off" : "Turned on"} />
          <IconButton label="Remove" onClick={onRemove} className="hover:!text-red-text">
            <Trash2 className="size-3.5" />
          </IconButton>
        </div>
      </div>
    </li>
  );
}

/**
 * For folders: open the most recently added or changed file inside instead
 * of the folder, optionally only certain file types.
 */
function NewestOption({ resource: r, onChange }: { resource: Resource; onChange: (patch: Partial<Resource>) => void }) {
  const [types, setTypes] = useState((r.extensions || []).join(", "));
  useEffect(() => setTypes((r.extensions || []).join(", ")), [r.extensions]);
  const commit = () => {
    const extensions = parseExtensions(types);
    setTypes(extensions.join(", "));
    if (extensions.join() !== (r.extensions || []).join()) onChange({ extensions });
  };
  return (
    <div className={`rounded-lg px-1 ${r.newest ? "bg-fill/60 py-1.5 pr-1.5" : ""}`}>
      <label className="flex cursor-pointer items-center gap-2 text-[12px] text-graphite">
        <input
          type="checkbox"
          checked={!!r.newest}
          onChange={(e) => onChange({ newest: e.target.checked, extensions: e.target.checked ? r.extensions : [] })}
          className="size-3.5 accent-[var(--red)]"
        />
        <Clock className="size-3 text-faint" />
        <span className={r.newest ? "font-medium text-ink" : ""}>Open the newest file inside, not the folder</span>
      </label>
      {r.newest && (
        <div className="mt-1.5 flex items-center gap-2 pl-[22px]">
          <span className="shrink-0 text-[11px] text-faint">Only</span>
          <Input
            value={types}
            onChange={(e) => setTypes(e.target.value)}
            onBlur={commit}
            onKeyDown={(e) => e.key === "Enter" && commit()}
            placeholder="Any file, or types like pdf, docx"
            aria-label="Only these file types"
            className="h-7 font-mono text-xs"
          />
        </div>
      )}
    </div>
  );
}

function Toggle({ checked, onChange, label }: { checked: boolean; onChange: (v: boolean) => void; label: string }) {
  return (
    <button
      role="switch"
      aria-checked={checked}
      aria-label={label}
      title={checked ? "On: opens when the Cue starts" : "Off: skipped when the Cue starts"}
      onClick={() => onChange(!checked)}
      className={`no-drag relative mx-1 h-4 w-7 rounded-full transition-colors ${checked ? "bg-ink" : "bg-fill-strong"}`}
    >
      <span className={`absolute top-0.5 size-3 rounded-full bg-surface shadow transition-[left] ${checked ? "left-3.5" : "left-0.5"}`} />
    </button>
  );
}

/** Checks (debounced) whether a file/folder path exists, for an inline hint. */
function usePathMissing(r: Resource): boolean {
  const [missing, setMissing] = useState(false);
  const isPath = r.type === ResourceType.TypeFile || r.type === ResourceType.TypeFolder;
  const path = (r.path || "").trim();
  useEffect(() => {
    if (!isPath || !path || !(path.startsWith("/") || path.startsWith("~"))) {
      setMissing(false);
      return;
    }
    const t = setTimeout(() => SystemService.PathExists(path).then((ok) => setMissing(!ok)), 300);
    return () => clearTimeout(t);
  }, [isPath, path]);
  return missing;
}
