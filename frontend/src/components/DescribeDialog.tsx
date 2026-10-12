import { Settings2, Sparkles } from "lucide-react";
import { useState } from "react";
import { AIService, errorMessage, type Proposal } from "../api";
import { Button, Dialog, Kbd } from "./ui";

const examples = [
  "Create a Cue for working on Atlas. I need my GitHub repo, IDE, Claude, localhost and documentation.",
  "Studying math: my textbook PDF, the exercise list, my notes folder, Calculator and the course website.",
  "Design time: Figma, my moodboard folder and Dribbble.",
];

export function DescribeDialog({
  aiReady,
  onClose,
  onProposal,
  onOpenSettings,
}: {
  aiReady: boolean;
  onClose: () => void;
  onProposal: (p: Proposal) => void;
  onOpenSettings: () => void;
}) {
  const [text, setText] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  const submit = async () => {
    if (!text.trim() || busy) return;
    setBusy(true);
    setError("");
    try {
      onProposal(await AIService.Propose(text, ""));
    } catch (e) {
      setError(errorMessage(e));
      setBusy(false);
    }
  };

  if (!aiReady) {
    return (
      <Dialog
        title="Describe a Cue"
        onClose={onClose}
        footer={
          <>
            <Button variant="ghost" onClick={onClose}>
              Not now
            </Button>
            <Button variant="primary" onClick={onOpenSettings}>
              <Settings2 className="size-4" />
              Set up AI
            </Button>
          </>
        }
      >
        <p className="text-[13px] leading-relaxed text-muted">
          Describing a Cue uses your own AI provider: an Anthropic API key, or any OpenAI-compatible server, including a local model in
          Ollama or LM Studio. Your key is kept in the macOS Keychain.
        </p>
        <p className="mt-3 text-[13px] leading-relaxed text-muted">The AI only proposes Cues. Starting a Cue never uses it.</p>
      </Dialog>
    );
  }

  return (
    <Dialog
      title={
        <span className="flex items-center gap-2">
          <Sparkles className="size-4 text-accent" />
          Describe a Cue
        </span>
      }
      onClose={onClose}
      width="max-w-xl"
      footer={
        <>
          <span className="mr-auto self-center text-[11px] text-faint">
            <Kbd>⌘</Kbd> <Kbd>↵</Kbd> to propose · you review before anything is saved
          </span>
          <Button variant="ghost" onClick={onClose}>
            Cancel
          </Button>
          <Button variant="primary" busy={busy} disabled={!text.trim()} onClick={submit}>
            {busy ? "Thinking…" : "Propose Cue"}
          </Button>
        </>
      }
    >
      <textarea
        autoFocus
        value={text}
        onChange={(e) => setText(e.target.value)}
        onKeyDown={(e) => (e.metaKey || e.ctrlKey) && e.key === "Enter" && submit()}
        rows={4}
        disabled={busy}
        placeholder="What do you want to do, and what do you need for it?"
        className="w-full resize-none rounded-xl border border-border-strong bg-surface px-3 py-2.5 text-[14px] leading-relaxed placeholder:text-faint focus:border-accent focus:outline-none focus:ring-3 focus:ring-accent-soft"
      />
      {error && <p className="mt-2 text-xs text-danger">{error}</p>}
      <div className="mt-3 space-y-1">
        <p className="text-[11px] font-medium text-faint">Try</p>
        {examples.map((ex) => (
          <button
            key={ex}
            onClick={() => setText(ex)}
            disabled={busy}
            className="block w-full truncate rounded-md px-2 py-1 text-left text-xs text-muted hover:bg-surface-2 hover:text-text"
          >
            {ex}
          </button>
        ))}
      </div>
    </Dialog>
  );
}
