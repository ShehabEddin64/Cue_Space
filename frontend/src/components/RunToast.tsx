import { AlertCircle, CheckCircle2, X } from "lucide-react";
import { useEffect } from "react";
import { Status, type Result } from "../api";

export type RunReport = { cueName: string; result: Result } | { cueName: string; error: string };

export function RunToast({ report, onClose, onFix }: { report: RunReport; onClose: () => void; onFix: () => void }) {
  const failed = "result" in report ? report.result.steps.filter((s) => s.status === Status.StatusFailed) : [];
  const opened = "result" in report ? report.result.steps.filter((s) => s.status === Status.StatusOK).length : 0;
  const ok = "result" in report && failed.length === 0;

  // Successful runs dismiss themselves; problems stay until read.
  useEffect(() => {
    if (!ok) return;
    const t = setTimeout(onClose, 3500);
    return () => clearTimeout(t);
  }, [ok, onClose]);

  return (
    <div
      role="status"
      aria-live="polite"
      className="no-drag animate-pop-in fixed bottom-5 left-1/2 z-50 w-[min(420px,calc(100vw-32px))] -translate-x-1/2 rounded-xl border border-border bg-surface p-3.5 shadow-pop"
    >
      <div className="flex items-start gap-2.5">
        {ok ? <CheckCircle2 className="mt-px size-4 shrink-0 text-ok" /> : <AlertCircle className="mt-px size-4 shrink-0 text-danger" />}
        <div className="min-w-0 flex-1">
          <p className="text-[13px] font-medium">
            {"error" in report
              ? `Couldn’t start ${report.cueName}`
              : ok
                ? `${report.cueName} started`
                : `${report.cueName} started with ${failed.length} problem${failed.length === 1 ? "" : "s"}`}
          </p>
          {"error" in report ? (
            <p className="mt-0.5 text-xs text-muted">{report.error}</p>
          ) : (
            <>
              <p className="mt-0.5 text-xs text-muted">
                Opened {opened} item{opened === 1 ? "" : "s"}
              </p>
              {failed.length > 0 && (
                <ul className="mt-2 space-y-1">
                  {failed.map((s) => (
                    <li key={s.resourceId} className="text-xs">
                      <span className="font-medium">{s.label}</span> <span className="text-muted">— {s.error}</span>
                    </li>
                  ))}
                </ul>
              )}
              {failed.length > 0 && (
                <button onClick={onFix} className="mt-2 text-xs font-medium text-accent hover:underline">
                  Fix in editor
                </button>
              )}
            </>
          )}
        </div>
        <button aria-label="Dismiss" onClick={onClose} className="rounded p-0.5 text-faint hover:text-text">
          <X className="size-3.5" />
        </button>
      </div>
    </div>
  );
}
