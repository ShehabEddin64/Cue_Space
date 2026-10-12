import type { CancellablePromise } from "@wailsio/runtime";
import { useCallback, useRef, useState } from "react";
import { AIService, Cue, CueService, errorMessage, type Proposal } from "./api";

/**
 * The lifecycle of an AI proposal: ask, refine, retry, undo, discard, save.
 * `base` is the saved Cue being changed, or null when creating a new one.
 */
export function useProposal(base: Cue | null, onSaved: (c: Cue) => void) {
  const [proposal, setProposal] = useState<Proposal | null>(null);
  const [history, setHistory] = useState<Proposal[]>([]);
  const [busy, setBusy] = useState<"compose" | "refine" | null>(null);
  const [error, setError] = useState("");
  const [updated, setUpdated] = useState(false);
  const lastPrompt = useRef("");
  const pending = useRef<CancellablePromise<Proposal> | null>(null);

  const run = useCallback(async (text: string, from: Cue | null, kind: "compose" | "refine") => {
    setBusy(kind);
    setError("");
    const call = AIService.Propose(text, from);
    pending.current = call;
    try {
      const p = await call;
      if (pending.current !== call) return null;
      return p;
    } catch (e) {
      if (pending.current === call) setError(errorMessage(e));
      return null;
    } finally {
      if (pending.current === call) {
        pending.current = null;
        setBusy(null);
      }
    }
  }, []);

  const compose = useCallback(
    async (text: string) => {
      lastPrompt.current = text;
      const p = await run(text, base, "compose");
      if (p) {
        setProposal(p);
        setHistory([]);
        setUpdated(false);
      }
    },
    [base, run],
  );

  const refine = useCallback(
    async (text: string) => {
      if (!proposal) return;
      const before = proposal;
      const p = await run(text, proposal.cue, "refine");
      if (p) {
        setHistory((h) => [...h, before]);
        setProposal(p);
        // Acknowledge that the correction took effect.
        setUpdated(true);
        setTimeout(() => setUpdated(false), 2200);
      }
    },
    [proposal, run],
  );

  const retry = useCallback(() => lastPrompt.current && compose(lastPrompt.current), [compose]);

  const undo = useCallback(() => {
    setHistory((h) => {
      if (!h.length) return h;
      setProposal(h[h.length - 1]);
      return h.slice(0, -1);
    });
  }, []);

  const cancel = useCallback(() => {
    pending.current?.cancel();
    pending.current = null;
    setBusy(null);
  }, []);

  const discard = useCallback(() => {
    cancel();
    setProposal(null);
    setHistory([]);
    setError("");
  }, [cancel]);

  const save = useCallback(async () => {
    if (!proposal) return;
    try {
      const saved = await CueService.Save(proposal.cue);
      setProposal(null);
      setHistory([]);
      onSaved(saved);
    } catch (e) {
      setError(errorMessage(e));
    }
  }, [proposal, onSaved]);

  return {
    proposal,
    setProposal,
    busy,
    error,
    updated,
    canUndo: history.length > 0,
    lastPrompt: lastPrompt.current,
    compose,
    refine,
    retry,
    undo,
    cancel,
    discard,
    save,
  };
}
