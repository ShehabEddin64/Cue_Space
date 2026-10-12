import { AlertCircle, X } from "lucide-react";
import { useEffect } from "react";
import { Status, SystemService, type Result } from "../api";
import { CueMark } from "./ui";

export type RunReport = { cueName: string; result: Result } | { cueName: string; error: string };

/** Feedback for Start Cue, in the interface rather than an alert. */
export function RunToast({ report, onClose, onFix }: { report: RunReport; onClose: () => void; onFix: () => void }) {
  const failed = "result" in report ? report.result.steps.filter((s) => s.status === Status.StatusFailed) : [];
  const opened = "result" in report ? report.result.steps.filter((s) => s.status === Status.StatusOK).length : 0;
  const ok = "result" in report && failed.length === 0 && report.result.notes.length === 0;

  // Successful runs dismiss themselves; problems stay until read.
  useEffect(() => {
    if (!ok) return;
    const t = setTimeout(onClose, 3200);
    return () => clearTimeout(t);
  }, [ok, onClose]);

  return (
    <div
      role="status"
      aria-live="polite"
      className="animate-pop-in fixed bottom-6 left-1/2 z-50 w-[min(440px,calc(100vw-32px))] -translate-x-1/2 rounded-2xl border border-hairline bg-surface/95 p-4 shadow-pop backdrop-blur-xl"
    >
      <div className="flex items-start gap-3">
        {ok ? <CueMark size={18} className="mt-px" /> : <AlertCircle className="mt-px size-[18px] shrink-0 text-red-text" />}
        <div className="min-w-0 flex-1">
          <p className="text-[13px] font-semibold">
            {"error" in report
              ? `Couldn’t start ${report.cueName}`
              : ok
                ? `${report.cueName} is ready`
                : `${report.cueName} started with ${failed.length} problem${failed.length === 1 ? "" : "s"}`}
          </p>
          {"error" in report ? (
            <p className="mt-0.5 text-[12px] text-graphite">{report.error}</p>
          ) : (
            <>
              <p className="mt-0.5 text-[12px] text-graphite">
                Opened {opened} item{opened === 1 ? "" : "s"}
              </p>
              {failed.length > 0 && (
                <ul className="mt-2 space-y-1">
                  {failed.map((s) => (
                    <li key={s.resourceId} className="text-[12px]">
                      <span className="font-medium">{s.label}</span> <span className="text-graphite">— {s.error}</span>
                    </li>
                  ))}
                </ul>
              )}
              {report.result.steps.some((s) => s.note) && (
                <ul className="mt-2 space-y-0.5">
                  {report.result.steps
                    .filter((s) => s.note && s.status === Status.StatusOK)
                    .map((s) => (
                      <li key={s.resourceId} className="truncate text-[12px] text-graphite">
                        <span className="font-medium text-ink">{s.label}</span> · {s.note}
                      </li>
                    ))}
                </ul>
              )}
              {report.result.notes.map((n) => (
                <p key={n} className="mt-2 text-[12px] text-red-on-tint">
                  {n}
                </p>
              ))}
              <div className="mt-2.5 flex gap-3">
                {failed.length > 0 && (
                  <button onClick={onFix} className="text-[12px] font-medium text-red-text hover:underline">
                    Fix in Editor
                  </button>
                )}
                {report.result.needsAccessibility && (
                  <button onClick={() => SystemService.RequestArrangeWindows()} className="text-[12px] font-medium text-red-text hover:underline">
                    Allow Accessibility…
                  </button>
                )}
              </div>
            </>
          )}
        </div>
        <button aria-label="Dismiss" onClick={onClose} className="rounded-full p-0.5 text-faint hover:bg-fill hover:text-ink">
          <X className="size-3.5" />
        </button>
      </div>
    </div>
  );
}
