import { Maximize2, Plus, ShieldCheck, Trash2 } from "lucide-react";
import { useEffect, useState } from "react";
import {
  Arrangement,
  Cue,
  Desktop,
  displayName,
  Layout,
  newId,
  SystemService,
  type ArrangementInfo,
  type Resource,
} from "../api";
import { msg, t } from "../i18n";
import { Button, IconButton, TypeIcon } from "./ui";

/*
 * Desktop layouts: which windows go where when a Cue starts. Each desktop is
 * a macOS Space with one arrangement, modelled on the system's own
 * Move & Resize / Fill & Arrange choices, and numbered slots.
 */

let arrangementsCache: Promise<ArrangementInfo[]> | null = null;
export function useArrangements(): ArrangementInfo[] {
  const [list, setList] = useState<ArrangementInfo[]>([]);
  useEffect(() => {
    arrangementsCache ??= SystemService.Arrangements();
    arrangementsCache.then(setList);
  }, []);
  return list;
}

/** Polls whether Cue Space may arrange windows (Accessibility permission). */
export function useCanArrange(active: boolean): [boolean | null, () => void] {
  const [ok, setOk] = useState<boolean | null>(null);
  useEffect(() => {
    if (!active) return;
    const check = () => SystemService.CanArrangeWindows().then(setOk);
    check();
    const t = setInterval(check, 2000);
    return () => clearInterval(t);
  }, [active]);
  return [ok, () => SystemService.RequestArrangeWindows().then(setOk)];
}

/** A small icon of an arrangement, like the ones in macOS's Window menu. */
export function ArrangementGlyph({ info, size = 30 }: { info: ArrangementInfo; size?: number }) {
  if (info.id === Arrangement.ArrangeFullScreen) {
    return (
      <span className="relative inline-flex items-center justify-center" style={{ width: size, height: size * 0.66 }}>
        <span className="absolute inset-0 rounded-[4px] border-[1.5px] border-current" />
        <Maximize2 className="size-3" strokeWidth={2.2} />
      </span>
    );
  }
  const W = 30,
    H = 20,
    g = 1.6;
  return (
    <svg width={size} height={size * (H / W)} viewBox={`0 0 ${W} ${H}`} aria-hidden="true">
      <rect x="0.75" y="0.75" width={W - 1.5} height={H - 1.5} rx="3.5" fill="none" stroke="currentColor" strokeWidth="1.5" />
      {info.slots.map((r, i) => (
        <rect
          key={i}
          x={2.5 + r.x * (W - 5) + g / 2}
          y={2.5 + r.y * (H - 5) + g / 2}
          width={r.w * (W - 5) - g}
          height={r.h * (H - 5) - g}
          rx="1.2"
          fill="currentColor"
          opacity={i === 0 ? 1 : 0.55}
        />
      ))}
    </svg>
  );
}

function itemNumber(cue: Cue, id: string) {
  return cue.resources.findIndex((r) => r.id === id) + 1;
}

/** A desktop drawn like a Mission Control thumbnail, with numbered slots. */
export function DesktopThumb({
  cue,
  desktop,
  info,
  selected,
  onClick,
}: {
  cue: Cue;
  desktop: Desktop;
  info: ArrangementInfo;
  selected?: boolean;
  onClick?: () => void;
}) {
  const byId = new Map(cue.resources.map((r) => [r.id, r]));
  const full = desktop.arrangement === Arrangement.ArrangeFullScreen;
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={!onClick}
      style={{ background: "linear-gradient(135deg, var(--wall-a), var(--wall-b))" }}
      className={`group relative aspect-[16/10] w-full overflow-hidden rounded-xl border text-left transition ${
        selected ? "border-red-text ring-3 ring-red-tint" : "border-hairline"
      } ${onClick ? "hover:brightness-[1.02]" : ""}`}
    >
      {info.slots.map((r, i) => {
        const rsc = byId.get(desktop.slots[i] ?? "");
        return (
          <div
            key={i}
            className={`absolute flex flex-col overflow-hidden rounded-[6px] border p-1.5 ${
              rsc ? "border-hairline bg-surface shadow-card" : "border-dashed border-ink/15 bg-surface/35"
            }`}
            style={{
              left: `calc(${r.x * 100}% + 4px)`,
              top: `calc(${r.y * 100}% + 4px)`,
              width: `calc(${r.w * 100}% - 8px)`,
              height: `calc(${r.h * 100}% - 8px)`,
            }}
          >
            <span
              className={`flex size-4 shrink-0 items-center justify-center rounded-full text-[9.5px] font-semibold ${
                rsc ? "bg-ink text-surface" : "bg-ink/15 text-ink/60"
              }`}
            >
              {i + 1}
            </span>
            {rsc ? (
              <span className="mt-auto flex min-w-0 items-center gap-1 text-[10.5px] font-medium leading-tight">
                <TypeIcon type={rsc.type} className="size-3 shrink-0 text-graphite" />
                <span className="truncate">
                  {itemNumber(cue, rsc.id)} · {displayName(rsc)}
                </span>
              </span>
            ) : (
              <span className="mt-auto truncate text-[10px] text-ink/45">Empty</span>
            )}
          </div>
        );
      })}
      {full && (
        <span className="absolute right-1.5 top-1.5 flex items-center gap-1 rounded-full bg-ink/75 px-1.5 py-0.5 text-[9.5px] font-medium text-surface">
          <Maximize2 className="size-2.5" /> Full screen
        </span>
      )}
    </button>
  );
}

function desktopCaption(d: Desktop, info: ArrangementInfo | undefined) {
  if (d.arrangement === Arrangement.ArrangeFullScreen) return `${t("Own desktop")} · ${t("Full Screen")}`;
  return `${d.newDesktop ? t("New desktop") : t("This desktop")} · ${info ? t(info.title) : d.arrangement}`;
}

/** Read-only preview of a Cue's desktops, as they'll appear in Mission Control. */
export function DesktopStrip({ cue, onEdit }: { cue: Cue; onEdit?: () => void }) {
  const arrangements = useArrangements();
  const desktops = cue.layout?.desktops ?? [];
  const [canArrange, request] = useCanArrange(desktops.length > 0);
  if (!desktops.length || !arrangements.length) return null;
  const placed = new Set(desktops.flatMap((d) => d.slots).filter(Boolean));
  const loose = cue.resources.filter((r) => !placed.has(r.id) && !r.disabled);

  return (
    <div>
      {canArrange === false && <PermissionBanner onAllow={request} />}
      <div className="flex gap-3 overflow-x-auto pb-1">
        {loose.length > 0 && (
          <figure className="w-[190px] shrink-0">
            <div className="flex aspect-[16/10] w-full flex-col justify-end gap-0.5 rounded-xl border border-dashed border-hairline bg-surface/60 p-2.5 text-[10.5px] text-graphite">
              {loose.slice(0, 4).map((r) => (
                <span key={r.id} className="truncate">
                  {itemNumber(cue, r.id)} · {displayName(r)}
                </span>
              ))}
              {loose.length > 4 && <span className="text-faint">+{loose.length - 4} more</span>}
            </div>
            <figcaption className="mt-1.5 text-center text-[11.5px] text-graphite">Opens first, as usual</figcaption>
          </figure>
        )}
        {desktops.map((d, i) => {
          const info = arrangements.find((a) => a.id === d.arrangement);
          if (!info) return null;
          return (
            <figure key={d.id} className="w-[190px] shrink-0">
              <DesktopThumb cue={cue} desktop={d} info={info} onClick={onEdit} />
              <figcaption className="mt-1.5 text-center text-[11.5px]">
                <span className="font-medium">{t("Desktop {n}", { n: i + 1 })}</span>
                <span className="text-graphite"> · {desktopCaption(d, info).split(" · ").pop()}</span>
              </figcaption>
            </figure>
          );
        })}
      </div>
    </div>
  );
}

export function PermissionBanner({ onAllow }: { onAllow: () => void }) {
  return (
    <div className="mb-3 flex items-center gap-3 rounded-xl bg-red-tint px-3.5 py-2.5 text-[12.5px] text-red-on-tint">
      <ShieldCheck className="size-4 shrink-0" />
      <span className="flex-1">
        {t("To arrange windows and create desktops, allow Cue Space in System Settings › Privacy & Security › Accessibility.")}
      </span>
      <Button size="sm" variant="tinted" onClick={onAllow}>
        {t("Allow…")}
      </Button>
    </div>
  );
}

// Arrangement titles come from Go (cue.Arrangements); listed here so the
// translation checker knows them.
export const arrangementTitles = [
  msg("Full Screen"),
  msg("Fill"),
  msg("Left & Right"),
  msg("Top & Bottom"),
  msg("Left & Quarters"),
  msg("Quarters"),
  msg("Thirds"),
];

// Titles are English; shown with t().
const groups: { title: string; ids: Arrangement[] }[] = [
  { title: msg("Own desktop"), ids: [Arrangement.ArrangeFullScreen] },
  {
    title: msg("Fill & Arrange"),
    ids: [Arrangement.ArrangeFill, Arrangement.ArrangeLeftRight, Arrangement.ArrangeLeftQuarters, Arrangement.ArrangeQuarters],
  },
  { title: msg("More"), ids: [Arrangement.ArrangeTopBottom, Arrangement.ArrangeThirds] },
];

/** The layout section of the editor. */
export function LayoutEditor({ cue, onChange }: { cue: Cue; onChange: (layout: Layout | null) => void }) {
  const arrangements = useArrangements();
  const desktops = cue.layout?.desktops ?? [];
  const [selected, setSelected] = useState(0);
  const [canArrange, request] = useCanArrange(true);
  const current = desktops[Math.min(selected, desktops.length - 1)];
  const info = current && arrangements.find((a) => a.id === current.arrangement);

  const setDesktops = (next: Desktop[]) => onChange(next.length ? new Layout({ desktops: next }) : null);
  const update = (i: number, patch: Partial<Desktop>) =>
    setDesktops(desktops.map((d, j) => (j === i ? new Desktop({ ...d, ...patch }) : d)));

  const add = () => {
    setDesktops([...desktops, new Desktop({ id: newId(), arrangement: Arrangement.ArrangeLeftRight, slots: ["", ""], newDesktop: true })]);
    setSelected(desktops.length);
  };
  const remove = (i: number) => {
    setDesktops(desktops.filter((_, j) => j !== i));
    setSelected(Math.max(0, i - 1));
  };
  const setArrangement = (a: ArrangementInfo) => {
    const slots = a.slots.map((_, k) => current.slots[k] ?? "");
    const wasFull = current.arrangement === Arrangement.ArrangeFullScreen;
    const newDesktop = a.id === Arrangement.ArrangeFullScreen ? false : wasFull ? true : !!current.newDesktop;
    update(selected, { arrangement: a.id, slots, newDesktop });
  };
  // A resource can sit in one slot only; choosing it here frees it elsewhere.
  const assign = (slot: number, id: string) =>
    setDesktops(
      desktops.map((d, j) =>
        new Desktop({
          ...d,
          slots: d.slots.map((s, k) => (j === selected && k === slot ? id : id && s === id ? "" : s)),
        }),
      ),
    );

  const placedElsewhere = (id: string) => desktops.some((d, j) => j !== selected && d.slots.includes(id));

  return (
    <div>
      {canArrange === false && desktops.length > 0 && <PermissionBanner onAllow={request} />}

      {desktops.length === 0 ? (
        <div className="rounded-2xl border border-dashed border-hairline px-5 py-6 text-center">
          <p className="text-[13px] text-graphite">
            {t("Choose where windows go: a full-screen app on its own desktop, two apps side by side on a new desktop, and more.")}
          </p>
          <Button className="mt-3" size="sm" onClick={add}>
            <Plus className="size-3.5" />
            {t("Add Desktop")}
          </Button>
        </div>
      ) : (
        <>
          {/* The strip: like Mission Control's row of desktops. */}
          <div className="flex gap-3 overflow-x-auto pb-1">
            {desktops.map((d, i) => {
              const di = arrangements.find((a) => a.id === d.arrangement);
              if (!di) return null;
              return (
                <figure key={d.id} className="w-[168px] shrink-0">
                  <DesktopThumb cue={cue} desktop={d} info={di} selected={i === selected} onClick={() => setSelected(i)} />
                  <figcaption className={`mt-1.5 text-center text-[11.5px] ${i === selected ? "font-semibold" : "text-graphite"}`}>
                    {t("Desktop {n}", { n: i + 1 })}
                  </figcaption>
                </figure>
              );
            })}
            <button
              type="button"
              onClick={add}
              aria-label={t("Add Desktop")}
              title={t("Add Desktop")}
              className="flex aspect-[16/10] w-[96px] shrink-0 items-center justify-center rounded-xl border border-dashed border-hairline text-graphite hover:bg-surface hover:text-ink"
            >
              <Plus className="size-5" />
            </button>
          </div>

          {current && info && (
            <div className="mt-4 rounded-2xl border border-hairline bg-surface p-4 shadow-card">
              <div className="flex items-center justify-between">
                <h4 className="text-[13px] font-semibold">
                  {t("Desktop {n}", { n: selected + 1 })} <span className="font-normal text-graphite">· {desktopCaption(current, info)}</span>
                </h4>
                <IconButton label={t("Remove Desktop {n}", { n: selected + 1 })} onClick={() => remove(selected)} className="hover:!text-red-text">
                  <Trash2 className="size-3.5" />
                </IconButton>
              </div>

              {/* Arrangement picker, grouped like the system's tile menu. */}
              <div className="mt-3 flex flex-wrap gap-x-6 gap-y-3">
                {groups.map((g) => (
                  <div key={g.title}>
                    <div className="mb-1.5 text-[11px] font-semibold text-faint">{t(g.title)}</div>
                    <div className="flex gap-1.5">
                      {g.ids.map((id) => {
                        const a = arrangements.find((x) => x.id === id);
                        if (!a) return null;
                        const on = current.arrangement === id;
                        return (
                          <button
                            key={id}
                            type="button"
                            title={t(a.title)}
                            aria-label={t(a.title)}
                            aria-pressed={on}
                            onClick={() => setArrangement(a)}
                            className={`flex h-10 w-12 items-center justify-center rounded-lg border transition ${
                              on ? "border-red-text bg-red-tint text-red-on-tint" : "border-hairline text-graphite hover:bg-fill hover:text-ink"
                            }`}
                          >
                            <ArrangementGlyph info={a} size={28} />
                          </button>
                        );
                      })}
                    </div>
                  </div>
                ))}
              </div>

              {current.arrangement !== Arrangement.ArrangeFullScreen && (
                <div role="radiogroup" aria-label={t("Which desktop")} className="mt-4 inline-flex gap-1 rounded-full bg-fill p-1">
                  {[
                    [true, msg("On a new desktop")],
                    [false, msg("On the current desktop")],
                  ].map(([value, label]) => (
                    <button
                      key={String(value)}
                      role="radio"
                      aria-checked={!!current.newDesktop === value}
                      onClick={() => update(selected, { newDesktop: value as boolean })}
                      className={`h-7 rounded-full px-3 text-[12px] font-medium transition ${
                        !!current.newDesktop === value ? "bg-surface text-ink shadow-card" : "text-graphite hover:text-ink"
                      }`}
                    >
                      {t(label as string)}
                    </button>
                  ))}
                </div>
              )}

              {/* Slots: pick a numbered item for each. */}
              <ul className="mt-4 space-y-2">
                {info.slots.map((_, k) => (
                  <li key={k} className="flex items-center gap-3">
                    <span className="flex size-6 shrink-0 items-center justify-center rounded-full bg-ink text-[11px] font-semibold text-surface">
                      {k + 1}
                    </span>
                    <span className="w-24 shrink-0 text-[12px] text-graphite">{slotName(info, k)}</span>
                    <select
                      value={current.slots[k] ?? ""}
                      onChange={(e) => assign(k, e.target.value)}
                      aria-label={t("Slot {n}", { n: k + 1 })}
                      className="h-8 min-w-0 flex-1 rounded-lg border border-hairline bg-surface px-2 text-[13px] focus:border-red-text/60 focus:outline-none"
                    >
                      <option value="">{t("Empty")}</option>
                      {cue.resources.map((r: Resource, n) => (
                        <option key={r.id} value={r.id}>
                          {n + 1} · {displayName(r)}
                          {placedElsewhere(r.id) ? ` (${t("moves from another desktop")})` : ""}
                          {r.disabled ? ` (${t("off")})` : ""}
                        </option>
                      ))}
                    </select>
                  </li>
                ))}
              </ul>
            </div>
          )}
        </>
      )}
    </div>
  );
}

function slotName(info: ArrangementInfo, k: number): string {
  const names: Record<string, string[]> = {
    fullscreen: [msg("Full screen")],
    fill: [msg("Whole desktop")],
    "left-right": [msg("Left half"), msg("Right half")],
    "top-bottom": [msg("Top half"), msg("Bottom half")],
    "left-quarters": [msg("Left half"), msg("Top right"), msg("Bottom right")],
    quarters: [msg("Top left"), msg("Top right"), msg("Bottom left"), msg("Bottom right")],
    thirds: [msg("Left third"), msg("Middle third"), msg("Right third")],
  };
  const name = names[info.id]?.[k];
  return name ? t(name) : t("Slot {n}", { n: k + 1 });
}
