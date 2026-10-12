import { ArrowUp, Square } from "lucide-react";
import { forwardRef, useEffect, useImperativeHandle, useRef, useState } from "react";
import type { AISettings } from "../api";
import { parseCommand, suggestCommands, type CommandDef } from "../commands";
import { CueMark } from "./ui";

export type ComposerHandle = { focus: () => void; setText: (t: string) => void; clear: () => void };

// Specific, truthful progress instead of "Processing…" (HIG › Generative AI › Outputs).
const progress = ["Reading what you want to do", "Matching apps on this Mac", "Putting your Cue together"];

const aiLabel = { always: "Uses AI", sometimes: "AI only if needed" } as const;

/**
 * The place to say what you want to do. Large on Home, compact when changing
 * an existing Cue. Return sends, Shift-Return adds a line, "/" lists commands.
 */
export const Composer = forwardRef<
  ComposerHandle,
  {
    ai: AISettings | null;
    busy: boolean;
    placeholder: string;
    compact?: boolean;
    initialText?: string;
    commands?: CommandDef[];
    /** Short status from the last command, e.g. "Added Figma". */
    status?: string;
    /** Shown instead of the AI progress steps while busy (e.g. for local commands). */
    busyLabel?: string;
    onSubmit: (text: string) => void;
    onCancel?: () => void;
    onConnect: () => void;
  }
>(function Composer({ ai, busy, placeholder, compact, initialText = "", commands = [], status, busyLabel, onSubmit, onCancel, onConnect }, ref) {
  const [text, setText] = useState(initialText);
  const [step, setStep] = useState(0);
  const [active, setActive] = useState(0);
  const area = useRef<HTMLTextAreaElement>(null);

  const focusEnd = (t: string) =>
    requestAnimationFrame(() => {
      area.current?.focus();
      area.current?.setSelectionRange(t.length, t.length);
    });

  useImperativeHandle(ref, () => ({
    focus: () => area.current?.focus(),
    clear: () => setText(""),
    setText: (t: string) => {
      setText(t);
      focusEnd(t);
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

  const suggestions = suggestCommands(text, commands);
  useEffect(() => setActive(0), [text]);

  const typed = parseCommand(text);
  const typedDef = typed ? commands.find((c) => c.name === typed.name) : undefined;
  const isCommand = text.trim().startsWith("/");
  const ready = !!ai?.ready;

  const complete = (def: CommandDef) => {
    const t = `/${def.name}${def.args ? " " : ""}`;
    setText(t);
    focusEnd(t);
    if (!def.args) onSubmit(t.trim());
  };

  const submit = () => {
    if (busy || !text.trim()) return;
    // Commands are handled locally even without an AI; sentences need one.
    if (!ready && !isCommand) return onConnect();
    onSubmit(text.trim());
  };

  return (
    <div
      className={`group relative border border-hairline bg-surface shadow-composer transition-[border-color,box-shadow] focus-within:border-red-text/45 focus-within:ring-4 focus-within:ring-red-tint ${
        compact ? "rounded-2xl" : "rounded-[22px]"
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
          aria-autocomplete="list"
          aria-expanded={suggestions.length > 0}
          onChange={(e) => setText(e.target.value)}
          onKeyDown={(e) => {
            if (suggestions.length) {
              if (e.key === "ArrowDown" || e.key === "ArrowUp") {
                e.preventDefault();
                const d = e.key === "ArrowDown" ? 1 : -1;
                return setActive((i) => (i + d + suggestions.length) % suggestions.length);
              }
              if (e.key === "Tab" || (e.key === "Enter" && !e.shiftKey)) {
                e.preventDefault();
                return complete(suggestions[active]);
              }
              if (e.key === "Escape") return setText("");
            }
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
        <div className="min-w-0 truncate text-[12px] text-graphite" aria-live="polite">
          {busy ? (
            <span className="font-medium text-ink">{busyLabel ?? progress[step]}…</span>
          ) : typedDef ? (
            <span>
              <span className="font-medium text-ink">/{typedDef.name}</span> {typedDef.description}
              <span className="text-faint"> · {typedDef.usesAI ? aiLabel[typedDef.usesAI] : "No AI"}</span>
            </span>
          ) : status ? (
            <span className="font-medium text-ink">{status}</span>
          ) : ready ? (
            <button onClick={onConnect} className="truncate rounded-md hover:text-ink" title="AI settings">
              {ai!.disclosure.model}
              <span className="text-faint"> · {ai!.disclosure.local ? "On this Mac" : ai!.disclosure.vendor}</span>
              {commands.length > 0 && <span className="text-faint"> · Type / for commands</span>}
            </button>
          ) : (
            <span>
              {commands.length > 0 ? "Type / for commands, or connect an AI to describe things" : "Bring your own AI: Anthropic, OpenAI-compatible or a local model"}
            </span>
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
        ) : ready || isCommand ? (
          <button
            onClick={submit}
            disabled={!text.trim()}
            aria-label={isCommand ? "Run command" : "Propose Cue"}
            title={isCommand ? "Run command (Return)" : "Propose Cue (Return)"}
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

      {suggestions.length > 0 && (
        <ul
          role="listbox"
          aria-label="Commands"
          className="animate-pop-in absolute inset-x-0 top-full z-30 mt-2 overflow-hidden rounded-2xl border border-hairline bg-surface p-1.5 shadow-pop"
        >
          {suggestions.map((c, i) => (
            <li
              key={c.name}
              role="option"
              aria-selected={i === active}
              onMouseEnter={() => setActive(i)}
              onMouseDown={(e) => {
                e.preventDefault();
                complete(c);
              }}
              className={`flex items-center gap-3 rounded-xl px-3 py-2 ${i === active ? "bg-fill" : ""}`}
            >
              <span className="w-[76px] shrink-0 font-mono text-[12.5px] font-medium text-ink">/{c.name}</span>
              <span className="min-w-0 flex-1 truncate text-[12.5px] text-graphite">
                {c.description}
                {c.args && <span className="text-faint"> · {c.args}</span>}
              </span>
              <span
                className={`shrink-0 rounded-full px-2 py-0.5 text-[10.5px] font-medium ${
                  c.usesAI ? "bg-red-tint text-red-on-tint" : "bg-fill text-graphite"
                }`}
              >
                {c.usesAI ? aiLabel[c.usesAI] : "No AI"}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
});
