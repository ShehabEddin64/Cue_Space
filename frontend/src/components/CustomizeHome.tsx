import { ChevronDown, ChevronUp } from "lucide-react";
import { HomePrefs, HomeSection, Prefs } from "../api";
import { defaultPrefs, usePrefs } from "../prefs";
import { DEFAULT_GREETING } from "./Home";
import { Button, IconButton, Input, Label, Sheet } from "./ui";

const sectionInfo: Record<string, { title: string; detail: string }> = {
  suggestions: { title: "Suggestions", detail: "Example ideas under the composer" },
  pinned: { title: "Pinned", detail: "The Cues you pin, with ⌘1, ⌘2…" },
  recent: { title: "Recently Used", detail: "The Cues you started last, newest first" },
};

/** Choose what Home shows and in which order. Changes apply as you make them. */
export function CustomizeHome({ onClose }: { onClose: () => void }) {
  const { prefs, save } = usePrefs();
  const home = prefs.home;

  const setHome = (patch: Partial<HomePrefs>) => save((p) => new Prefs({ ...p, home: new HomePrefs({ ...p.home, ...patch }) }));
  const setSections = (sections: HomeSection[]) => setHome({ sections });

  const move = (i: number, d: -1 | 1) => {
    const next = [...home.sections];
    [next[i], next[i + d]] = [next[i + d], next[i]];
    setSections(next);
  };
  const toggle = (i: number) =>
    setSections(home.sections.map((s, j) => (j === i ? new HomeSection({ ...s, visible: !s.visible }) : s)));

  return (
    <Sheet
      title="Customize Home"
      onClose={onClose}
      footer={
        <>
          <Button variant="ghost" className="mr-auto" onClick={() => setHome(defaultPrefs.home)}>
            Restore Defaults
          </Button>
          <Button variant="primary" onClick={onClose}>
            Done
          </Button>
        </>
      }
    >
      <section>
        <Label hint="Leave empty for the default">Greeting</Label>
        <Input value={home.greeting} onChange={(e) => setHome({ greeting: e.target.value })} placeholder={DEFAULT_GREETING} maxLength={60} />
      </section>

      <section className="mt-5">
        <Label hint="The composer always comes first">Sections</Label>
        <ul className="divide-y divide-hairline overflow-hidden rounded-2xl border border-hairline">
          {home.sections.map((s, i) => (
            <li key={s.id} className="flex items-center gap-3 px-3.5 py-2.5">
              <input
                type="checkbox"
                checked={s.visible}
                onChange={() => toggle(i)}
                aria-label={`Show ${sectionInfo[s.id]?.title}`}
                className="size-4 accent-[var(--red)]"
              />
              <div className={`min-w-0 flex-1 ${s.visible ? "" : "opacity-50"}`}>
                <div className="text-[13px] font-medium">{sectionInfo[s.id]?.title ?? s.id}</div>
                <div className="text-[11.5px] text-graphite">{sectionInfo[s.id]?.detail}</div>
              </div>
              <IconButton label="Move up" disabled={i === 0} onClick={() => move(i, -1)}>
                <ChevronUp className="size-4" />
              </IconButton>
              <IconButton label="Move down" disabled={i === home.sections.length - 1} onClick={() => move(i, 1)}>
                <ChevronDown className="size-4" />
              </IconButton>
            </li>
          ))}
        </ul>
      </section>

      <section className="mt-5">
        <Label>Cues in “Recently Used”</Label>
        <div role="radiogroup" className="inline-flex gap-1 rounded-full bg-fill p-1">
          {[3, 6, 9, 12].map((n) => (
            <button
              key={n}
              role="radio"
              aria-checked={home.recentCount === n}
              onClick={() => setHome({ recentCount: n })}
              className={`h-7 w-11 rounded-full text-[12px] font-medium transition ${
                home.recentCount === n ? "bg-surface text-ink shadow-card" : "text-graphite hover:text-ink"
              }`}
            >
              {n}
            </button>
          ))}
        </div>
      </section>
    </Sheet>
  );
}
