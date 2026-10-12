import { ArrowUp, Square } from "lucide-react";
import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react";
import type { AISettings } from "../api";
import { CueMark } from "./ui";

export type ComposerHandle = { focus: () => void; setText: (t: string) => void; clear: () => void };

// Specific, truthful progress instead of "Processing…" (HIG › Generative AI › Outputs).
const progress = ["Reading what you want to do", "Matching apps on this Mac", "Putting your Cue together"];

/**
 * The place to say what you want to do. Large on Home, compact when changing
 * an existing Cue. Return sends, Shift-Return adds a line.
 */
export const Composer = forwardRef<
  ComposerHandle,
  {
    ai: AISettings | null;
    busy: boolean;
    placeholder: string;
    compact?: boolean;
    initialText?: string;
    onSubmit: (text: string) => void;
    onCancel?: () => void;
    onConnect: () => void;
  }
>(function Composer({ ai, busy, placeholder, compact, initialText = "", onSubmit, onCancel, onConnect }, ref) {
  const [text, setText] = useState(initialText);
  const [step, setStep] = useState(0);
  const area = useRef<HTMLTextAreaElement>(null);

  useImperativeHandle(ref, () => ({
    focus: () => area.current?.focus(),
    clear: () => setText(""),
    setText: (t: string) => {
      setText(t);
      requestAnimationFrame(() => {
        area.current?.focus();
        area.current?.setSelectionRange(t.length, t.length);
      });
    },
  }));

  // Grow with the text, up to a limit.
  useEffect(() => {
    const el = area.current;
    if (!el) return;
    el.style.height = "0px";
    el.style.height = `${Math.min(el.scrollHeight, compact ? 120 : 180)}px`;
  }, [text, compact]);

  useEffect(() => {
    if (!busy) return setStep(0);
    const t = setInterval(() => setStep((s) => Math.min(s + 1, progress.length - 1)), 1700);
    return () => clearInterval(t);
  }, [busy]);

  const ready = !!ai?.ready;
  const submit = () => {
    if (busy) return;
    if (!ready) return onConnect();
    if (text.trim()) onSubmit(text.trim());
  };

  return (
    <div
      className={`group relative rounded-[22px] border border-hairline bg-surface shadow-composer transition-[border-color,box-shadow] focus-within:border-red-text/45 focus-within:ring-4 focus-within:ring-red-tint ${
        compact ? "rounded-2xl" : ""
      }`}
    >
      <div className={`flex items-start gap-3 ${compact ? "px-3.5 pt-3" : "px-5 pt-[18px]"}`}>
        <CueMark size={compact ? 20 : 26} thinking={busy} className={compact ? "mt-[3px]" : "mt-[2px]"} />
        <textarea
          ref={area}
          rows={1}
          value={text}
          disabled={busy}
          aria-label={placeholder}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (e.key === "Enter" && !e.shiftKey && !e.nativeEvent.isComposing) {
              e.preventDefault();
              submit();
            } else if (e.key === "Escape" && busy && onCancel) {
              onCancel();
            }
          }}
          placeholder={placeholder}
          spellCheck
          className={`min-h-0 flex-1 resize-none bg-transparent leading-[1.45] placeholder:text-faint focus:outline-none disabled:opacity-60 ${
            compact ? "py-[1px] text-[14px]" : "py-[2px] text-[17px]"
          }`}
        />
      </div>

      <div className={`flex items-center justify-between gap-3 ${compact ? "px-3.5 pb-3 pt-2" : "px-5 pb-4 pt-3"}`}>
        <div className="min-w-0 text-[12px] text-graphite" aria-live="polite">
          {busy ? (
            <span className="font-medium text-ink">{progress[step]}…</span>
          ) : ready ? (
            <button onClick={onConnect} className="truncate rounded-md hover:text-ink" title="AI settings">
              {ai!.disclosure.model}
              <span className="text-faint"> · {ai!.disclosure.local ? "On this Mac" : ai!.disclosure.vendor}</span>
            </button>
          ) : (
            <span>Bring your own AI: Anthropic, OpenAI-compatible or a local model</span>
          )}
        </div>

        {busy ? (
          <button
            onClick={onCancel}
            aria-label="Stop"
            title="Stop"
            className="flex size-8 shrink-0 items-center justify-center rounded-full bg-fill text-ink hover:bg-fill-strong"
          >
            <Square className="size-3 fill-current" />
          </button>
        ) : ready ? (
          <button
            onClick={submit}
            disabled={!text.trim()}
            aria-label="Propose Cue"
            title="Propose Cue (Return)"
            className="flex size-8 shrink-0 items-center justify-center rounded-full bg-red text-white shadow-[0_1px_2px_rgb(217_48_37/0.3)] transition-opacity hover:bg-red-hover disabled:opacity-30"
          >
            <ArrowUp className="size-4" strokeWidth={2.5} />
          </button>
        ) : (
          <button
            onClick={onConnect}
            className="h-8 shrink-0 rounded-full border border-red-text/40 px-3.5 text-[13px] font-medium text-red-text hover:bg-red-tint"
          >
            Connect AI
          </button>
        )}
      </div>
    </div>
  );
});
