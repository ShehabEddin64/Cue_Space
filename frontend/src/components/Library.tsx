import { Pin, PinOff, Plus, Search } from "lucide-react";
import { useState } from "react";
import { displayName, shortcutOrder, type Cue } from "../api";
import { usePrefs } from "../prefs";
import { CueCard, Grid } from "./Home";
import { Toolbar } from "./Toolbar";
import { Button, IconButton } from "./ui";

type Sort = "name" | "recent" | "items";

const sorts: [Sort, string][] = [
  ["name", "Name"],
  ["recent", "Recently Started"],
  ["items", "Most Items"],
];

/** Every Cue in one place: search, sort, pin, start. */
export function Library({
  cues,
  starting,
  onOpen,
  onStart,
  onNewBlank,
  onNew,
}: {
  cues: Cue[];
  starting: string | null;
  onOpen: (c: Cue) => void;
  onStart: (c: Cue) => void;
  onNewBlank: () => void;
  onNew: () => void;
}) {
  const { prefs, togglePin } = usePrefs();
  const [query, setQuery] = useState("");
  const [sort, setSort] = useState<Sort>("name");

  const order = shortcutOrder(cues, prefs.pinned);
  const shortcut = (c: Cue) => {
    const i = order.findIndex((x) => x.id === c.id);
    return i >= 0 && i < 9 ? `⌘${i + 1}` : undefined;
  };

  const q = query.trim().toLowerCase();
  const shown = cues
    .filter(
      (c) =>
        !q ||
        c.name.toLowerCase().includes(q) ||
        (c.description || "").toLowerCase().includes(q) ||
        c.resources.some((r) => displayName(r).toLowerCase().includes(q)),
    )
    .sort((a, b) => {
      if (sort === "recent") return (b.lastRunAt || "").localeCompare(a.lastRunAt || "");
      if (sort === "items") return b.resources.length - a.resources.length;
      return a.name.localeCompare(b.name);
    });

  return (
    <div className="flex h-full flex-col">
      <Toolbar
        title="Cues"
        trailing={
          <>
            <Button variant="ghost" size="sm" onClick={onNewBlank} title="Create a Cue Manually (⌥⌘N)">
              <Plus className="size-3.5" />
              Create Manually
            </Button>
            <Button variant="primary" size="sm" onClick={onNew} title="New Cue (⌘N)">
              New Cue…
            </Button>
          </>
        }
      />
      <div className="flex-1 overflow-y-auto">
        <div className="mx-auto w-full max-w-[920px] px-8 pb-16 pt-2">
          <div className="mb-5 flex flex-wrap items-center gap-3">
            <label className="flex h-8 min-w-[220px] flex-1 items-center gap-2 rounded-full border border-hairline bg-surface px-3 focus-within:ring-3 focus-within:ring-red-tint">
              <Search className="size-3.5 text-faint" />
              <input
                value={query}
                onChange={(e) => setQuery(e.target.value)}
                onKeyDown={(e) => e.key === "Escape" && setQuery("")}
                placeholder={`Search ${cues.length} Cue${cues.length === 1 ? "" : "s"}`}
                aria-label="Search Cues"
                className="h-full flex-1 bg-transparent text-[13px] placeholder:text-faint focus:outline-none"
              />
            </label>
            <div role="radiogroup" aria-label="Sort by" className="flex gap-1 rounded-full bg-fill p-1">
              {sorts.map(([value, label]) => (
                <button
                  key={value}
                  role="radio"
                  aria-checked={sort === value}
                  onClick={() => setSort(value)}
                  className={`h-6 rounded-full px-3 text-[12px] font-medium transition ${
                    sort === value ? "bg-surface text-ink shadow-card" : "text-graphite hover:text-ink"
                  }`}
                >
                  {label}
                </button>
              ))}
            </div>
          </div>

          {cues.length === 0 ? (
            <div className="mx-auto max-w-sm py-16 text-center">
              <h2 className="text-[17px] font-semibold">No Cues yet</h2>
              <p className="mt-1.5 text-[13px] text-graphite">Describe one on Home, or build one yourself.</p>
              <div className="mt-5 flex justify-center gap-2">
                <Button variant="primary" onClick={onNew}>
                  New Cue…
                </Button>
                <Button onClick={onNewBlank}>Create Manually</Button>
              </div>
            </div>
          ) : shown.length === 0 ? (
            <p className="py-16 text-center text-[13px] text-graphite">No Cues match “{query}”.</p>
          ) : (
            <Grid>
              {shown.map((c) => {
                const pinned = prefs.pinned.includes(c.id);
                return (
                  <CueCard
                    key={c.id}
                    cue={c}
                    pinned={pinned}
                    shortcut={shortcut(c)}
                    starting={starting === c.id}
                    onOpen={() => onOpen(c)}
                    onStart={() => onStart(c)}
                    trailing={
                      <IconButton
                        label={pinned ? `Unpin ${c.name}` : `Pin ${c.name}`}
                        onClick={(e) => {
                          e.stopPropagation();
                          togglePin(c.id);
                        }}
                        className="opacity-0 focus:opacity-100 group-hover:opacity-100"
                      >
                        {pinned ? <PinOff className="size-3.5" /> : <Pin className="size-3.5" />}
                      </IconButton>
                    }
                  />
                );
              })}
            </Grid>
          )}
        </div>
      </div>
    </div>
  );
}
