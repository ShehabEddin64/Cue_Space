import { AppWindow, ArrowRight, ArrowUp, Folder, Globe } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";
import { t } from "../i18n";
import { CueMark } from "./ui";

function BigKey({ children }: { children: ReactNode }) {
  return (
    <kbd className="flex size-11 items-center justify-center rounded-xl border border-hairline bg-surface font-sans text-[18px] font-medium text-ink shadow-card">
      {children}
    </kbd>
  );
}

type Step = { title: string; body: string; art: ReactNode };

// Built when shown, so it follows the current language.
const steps = (): Step[] => [
  {
    title: t("Welcome to Cue Space"),
    body: t("Your computer is organized around apps and files. You think in activities. Cue Space closes that gap."),
    art: (
      <div className="flex size-[92px] items-center justify-center rounded-[24px] bg-gradient-to-b from-[#e5392d] to-[#c9281d] shadow-[0_14px_30px_rgb(217_48_37/0.35)]">
        <CueMark size={60} ring="#ffffff" dot="#ffffff" />
      </div>
    ),
  },
  {
    title: t("A Cue holds one activity"),
    body: t("Links, apps, files and folders: everything you need to work on a project or study a subject, opened together and in order."),
    art: (
      <div className="w-[260px] rounded-2xl border border-hairline bg-surface p-3.5 text-left shadow-card">
        <div className="flex items-center gap-2.5">
          <span className="flex size-8 items-center justify-center rounded-[10px] bg-fill text-[16px]">🛰️</span>
          <span className="text-[14px] font-semibold">Atlas</span>
        </div>
        <ul className="mt-3 space-y-1.5 text-[12px] text-graphite">
          <li className="flex items-center gap-2"><Globe className="size-3.5" /> {t("GitHub repo")}</li>
          <li className="flex items-center gap-2"><Folder className="size-3.5" /> {t("Code in Visual Studio Code")}</li>
          <li className="flex items-center gap-2"><AppWindow className="size-3.5" /> Claude</li>
          <li className="flex items-center gap-2"><Globe className="size-3.5" /> localhost:3000</li>
        </ul>
        <div className="mt-3 flex h-7 items-center justify-center gap-1.5 rounded-full bg-red text-[12px] font-medium text-white">
          {t("Start Cue")} <ArrowRight className="size-3.5" />
        </div>
      </div>
    ),
  },
  {
    title: t("Describe it, or build it"),
    body: t("Tell Cue Space what you want to do and your own AI proposes the Cue for you to review. Prefer doing it yourself? Build one by hand: AI is never required."),
    art: (
      <div className="w-[300px] rounded-[20px] border border-hairline bg-surface p-3.5 text-left shadow-composer">
        <div className="flex items-start gap-2.5">
          <CueMark size={20} thinking />
          <span className="text-[13px] text-ink">{t("Study math: my textbook, notes folder and Calculator")}</span>
        </div>
        <div className="mt-3 flex items-center justify-between text-[11px] text-graphite">
          <span>{t("Your AI · Gemini, Anthropic, OpenAI or local")}</span>
          <span className="flex size-6 items-center justify-center rounded-full bg-red text-white">
            <ArrowUp className="size-3.5" strokeWidth={2.5} />
          </span>
        </div>
      </div>
    ),
  },
  {
    title: t("Commands are faster, and free"),
    body: t("Type / for commands. Most run on your Mac without any AI, and /add only asks the AI about what it can’t find itself."),
    art: (
      <ul className="w-[300px] space-y-1.5 rounded-2xl border border-hairline bg-surface p-2 text-left shadow-card">
        {[
          ["/add", "figma, ~/Notes, github.com/…"],
          ["/off", "localhost"],
          ["/start", "atlas"],
          ["/new", "Research"],
        ].map(([c, a]) => (
          <li key={c} className="flex items-center gap-2 rounded-lg px-2 py-1.5">
            <span className="w-14 font-mono text-[12px] font-medium">{c}</span>
            <span className="min-w-0 flex-1 truncate text-[12px] text-graphite">{a}</span>
            <span className="rounded-full bg-fill px-2 py-0.5 text-[10px] font-medium text-graphite">{t("No AI")}</span>
          </li>
        ))}
      </ul>
    ),
  },
  {
    title: t("Start in one keystroke"),
    body: t("Your first nine Cues get ⌘1 to ⌘9. Pin favorites to move them to the front. Your Cues are plain files on your Mac: they belong to you."),
    art: (
      <div className="flex items-center gap-2">
        <BigKey>⌘</BigKey>
        <span className="text-[14px] text-graphite">+</span>
        <BigKey>1</BigKey>
        <ArrowRight className="mx-1 size-4 text-faint" />
        <span className="rounded-full bg-red px-3 py-1 text-[12px] font-medium text-white">{t("Atlas is ready")}</span>
      </div>
    ),
  },
];

/** A short first-run tour. Skippable at any point, and replayable from Help. */
export function WelcomeTour({ aiReady, onDone, onConnect }: { aiReady: boolean; onDone: () => void; onConnect: () => void }) {
  const [i, setI] = useState(0);
  const all = steps();
  const last = i === all.length - 1;
  const step = all[i];

  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if (e.key === "Escape") onDone();
      else if (e.key === "ArrowRight" || e.key === "Enter") last ? onDone() : setI((n) => n + 1);
      else if (e.key === "ArrowLeft") setI((n) => Math.max(0, n - 1));
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [last, onDone]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/30 px-4">
      <div
        role="dialog"
        aria-modal="true"
        aria-labelledby="tour-title"
        className="animate-pop-in w-full max-w-[520px] overflow-hidden rounded-[24px] border border-hairline bg-surface shadow-pop"
      >
        <div key={i} className="animate-pop-in flex flex-col items-center px-10 pb-6 pt-10 text-center">
          <div className="flex h-[170px] items-center justify-center">{step.art}</div>
          <h2 id="tour-title" className="mt-6 text-[22px] font-semibold tracking-[-0.02em]">
            {step.title}
          </h2>
          <p className="mt-2 max-w-[400px] text-[14px] leading-relaxed text-graphite">{step.body}</p>
        </div>

        <div className="flex items-center gap-3 border-t border-hairline px-6 py-4">
          <div className="flex gap-1.5" aria-label={t("Step {n} of {total}", { n: i + 1, total: all.length })}>
            {all.map((_, n) => (
              <button
                key={n}
                aria-label={t("Go to step {n}", { n: n + 1 })}
                onClick={() => setI(n)}
                className={`h-1.5 rounded-full transition-all ${n === i ? "w-5 bg-red" : "w-1.5 bg-fill-strong hover:bg-graphite/40"}`}
              />
            ))}
          </div>
          <div className="flex-1" />
          {!last && (
            <button onClick={onDone} className="rounded-full px-3 py-1.5 text-[13px] text-graphite hover:text-ink">
              {t("Skip")}
            </button>
          )}
          {i > 0 && (
            <button onClick={() => setI(i - 1)} className="h-8 rounded-full bg-fill px-4 text-[13px] font-medium hover:bg-fill-strong">
              {t("Back")}
            </button>
          )}
          {last && !aiReady && (
            <button
              onClick={onConnect}
              className="h-8 rounded-full border border-red-text/40 px-4 text-[13px] font-medium text-red-text hover:bg-red-tint"
            >
              {t("Connect an AI")}
            </button>
          )}
          <button
            onClick={() => (last ? onDone() : setI(i + 1))}
            autoFocus
            className="h-8 rounded-full bg-red px-4 text-[13px] font-medium text-white hover:bg-red-hover"
          >
            {last ? t("Get Started") : t("Continue")}
          </button>
        </div>
      </div>
    </div>
  );
}
