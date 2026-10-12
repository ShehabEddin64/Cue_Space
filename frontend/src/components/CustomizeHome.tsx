import { ChevronDown, ChevronUp } from "lucide-react";
import { HomePrefs, HomeSection, Prefs } from "../api";
import { defaultPrefs, usePrefs } from "../prefs";
import { DEFAULT_GREETING } from "./Home";
import { asSize, SizeGlyph, SizePicker, type SectionSize } from "./HomeWidgets";
import { Button, IconButton, Input, Label, Sheet } from "./ui";

const sectionInfo: Record<string, { title: string; detail: string; sizes: Record<SectionSize, string> }> = {
  pinned: {
    title: "Pinned",
    detail: "The Cues you pin, with ⌘1, ⌘2…",
    sizes: { small: "Compact tiles, many per row", medium: "Cards with a Start button", large: "Cards that list what opens" },
  },
  recent: {
    title: "Recently Used",
    detail: "The Cues you started last, newest first",
    sizes: { small: "Compact tiles, many per row", medium: "A list with start times", large: "Cards that list what opens" },
  },
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
  const patch = (i: number, change: Partial<HomeSection>) =>
    setSections(home.sections.map((s, j) => (j === i ? new HomeSection({ ...s, ...change }) : s)));

  return (
    <Sheet
      title="Customize Home"
      width="max-w-xl"
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
        <p className="-mt-0.5 mb-2.5 text-[11.5px] text-graphite">
          Pick a size for each section, like a widget. You can also hover a section’s title on Home to change its size there.
        </p>
        <ul className="space-y-2.5">
          {home.sections.map((s, i) => {
            const info = sectionInfo[s.id];
            const size = asSize(s.size);
            return (
              <li key={s.id} className="rounded-2xl border border-hairline bg-surface p-3.5 shadow-card">
                <div className="flex items-center gap-3">
                  <span
                    className={`flex size-9 shrink-0 items-center justify-center rounded-[10px] ${s.visible ? "bg-red-tint text-red-text" : "bg-fill text-faint"}`}
                  >
                    <SizeGlyph size={size} className="h-3.5 w-5" />
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="text-[13px] font-semibold">{info?.title ?? s.id}</div>
                    <div className="text-[11.5px] text-graphite">{info?.detail}</div>
                  </div>
                  <IconButton label="Move up" disabled={i === 0} onClick={() => move(i, -1)}>
                    <ChevronUp className="size-4" />
                  </IconButton>
                  <IconButton label="Move down" disabled={i === home.sections.length - 1} onClick={() => move(i, 1)}>
                    <ChevronDown className="size-4" />
                  </IconButton>
                  <Switch checked={s.visible} onChange={() => patch(i, { visible: !s.visible })} label={`Show ${info?.title ?? s.id}`} />
                </div>
                {s.visible && (
                  <div className="mt-3 flex flex-wrap items-center gap-x-3 gap-y-2 border-t border-hairline pt-3">
                    <SizePicker value={size} onChange={(v) => patch(i, { size: v })} label={`${info?.title} size`} />
                    <span className="text-[11.5px] text-graphite">{info?.sizes[size]}</span>
                  </div>
                )}
              </li>
            );
          })}
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

/** A macOS-style switch. */
function Switch({ checked, onChange, label }: { checked: boolean; onChange: () => void; label: string }) {
  return (
    <button
      role="switch"
      aria-checked={checked}
      aria-label={label}
      title={label}
      onClick={onChange}
      className={`relative ml-1 h-[22px] w-[38px] shrink-0 rounded-full transition-colors ${checked ? "bg-red" : "bg-fill-strong"}`}
    >
      <span
        className={`absolute left-0 top-[2px] size-[18px] rounded-full bg-white shadow-[0_1px_2px_rgb(0_0_0/0.25)] transition-transform ${
          checked ? "translate-x-[18px]" : "translate-x-[2px]"
        }`}
      />
    </button>
  );
}
