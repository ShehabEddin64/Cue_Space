import { Check, FolderOpen, KeyRound } from "lucide-react";
import { useEffect, useState } from "react";
import { AIService, Config, errorMessage, SystemService, type AISettings } from "../api";
import { Button, Dialog, Input, Label } from "./ui";

const presets = [
  { name: "OpenAI", url: "https://api.openai.com/v1", key: true },
  { name: "OpenRouter", url: "https://openrouter.ai/api/v1", key: true },
  { name: "Ollama", url: "http://localhost:11434/v1", key: false },
  { name: "LM Studio", url: "http://localhost:1234/v1", key: false },
];

type Provider = "" | "anthropic" | "openai";

export function SettingsDialog({ folder, onClose, onSaved }: { folder: string; onClose: () => void; onSaved: (s: AISettings) => void }) {
  const [loaded, setLoaded] = useState<AISettings | null>(null);
  const [provider, setProvider] = useState<Provider>("");
  const [model, setModel] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [key, setKey] = useState("");
  const [status, setStatus] = useState<{ kind: "ok" | "error"; text: string } | null>(null);
  const [busy, setBusy] = useState<"save" | "test" | null>(null);

  useEffect(() => {
    AIService.GetSettings().then((s) => {
      setLoaded(s);
      setProvider(s.config.provider as Provider);
      setModel(s.config.model);
      setBaseUrl(s.config.baseUrl);
    });
  }, []);

  const switchProvider = (p: Provider) => {
    setProvider(p);
    setStatus(null);
    setKey("");
    if (p === "anthropic") {
      setModel(loaded?.config.provider === "anthropic" ? loaded.config.model : "claude-opus-5-5");
      setBaseUrl("");
    } else if (p === "openai") {
      setModel(loaded?.config.provider === "openai" ? loaded.config.model : "");
      setBaseUrl(loaded?.config.provider === "openai" ? loaded.config.baseUrl : presets[2].url);
    }
  };

  const persist = async () => {
    await AIService.SaveSettings(new Config({ provider, model, baseUrl }), key, false);
    const s = await AIService.GetSettings();
    setLoaded(s);
    setKey("");
    onSaved(s);
    return s;
  };

  const save = async () => {
    setBusy("save");
    setStatus(null);
    try {
      await persist();
      onClose();
    } catch (e) {
      setStatus({ kind: "error", text: errorMessage(e) });
    } finally {
      setBusy(null);
    }
  };

  const test = async () => {
    setBusy("test");
    setStatus(null);
    try {
      await persist();
      await AIService.TestConnection();
      setStatus({ kind: "ok", text: "Connected" });
    } catch (e) {
      setStatus({ kind: "error", text: errorMessage(e) });
    } finally {
      setBusy(null);
    }
  };

  const removeKey = async () => {
    await AIService.SaveSettings(new Config({ provider, model, baseUrl }), "", true);
    const s = await AIService.GetSettings();
    setLoaded(s);
    onSaved(s);
  };

  const hasKey = loaded?.hasKey && loaded.config.provider === provider;
  const preset = presets.find((p) => p.url === baseUrl);
  const needsKey = provider === "anthropic" || (provider === "openai" && (preset?.key ?? true));

  return (
    <Dialog
      title="Settings"
      onClose={onClose}
      footer={
        <>
          {status && (
            <span className={`mr-auto flex items-center gap-1.5 self-center text-xs ${status.kind === "ok" ? "text-ok" : "text-danger"}`}>
              {status.kind === "ok" && <Check className="size-3.5" />}
              {status.text}
            </span>
          )}
          {provider && (
            <Button busy={busy === "test"} disabled={!!busy} onClick={test}>
              Test connection
            </Button>
          )}
          <Button variant="primary" busy={busy === "save"} disabled={!!busy} onClick={save}>
            Save
          </Button>
        </>
      }
    >
      <section>
        <Label hint="Used only to propose Cues">AI provider</Label>
        <div className="grid grid-cols-3 gap-1 rounded-lg bg-surface-2 p-1">
          {(
            [
              ["", "None"],
              ["anthropic", "Anthropic"],
              ["openai", "OpenAI-compatible"],
            ] as const
          ).map(([value, name]) => (
            <button
              key={value}
              onClick={() => switchProvider(value)}
              className={`h-7 rounded-md text-xs font-medium transition ${
                provider === value ? "bg-surface text-text shadow-card" : "text-muted hover:text-text"
              }`}
            >
              {name}
            </button>
          ))}
        </div>
      </section>

      {provider === "openai" && (
        <section className="mt-4">
          <Label>Server</Label>
          <div className="mb-2 flex flex-wrap gap-1.5">
            {presets.map((p) => (
              <button
                key={p.name}
                onClick={() => setBaseUrl(p.url)}
                className={`rounded-full border px-2.5 py-0.5 text-[11px] transition ${
                  baseUrl === p.url ? "border-accent bg-accent-soft text-accent" : "border-border-strong text-muted hover:text-text"
                }`}
              >
                {p.name}
              </button>
            ))}
          </div>
          <Input value={baseUrl} onChange={(e) => setBaseUrl(e.target.value)} placeholder="https://…/v1" className="font-mono text-xs" />
        </section>
      )}

      {provider && (
        <section className="mt-4">
          <Label hint={provider === "anthropic" ? "Default: claude-opus-5-5" : "e.g. gpt-5, llama3.2, qwen3"}>Model</Label>
          <Input value={model} onChange={(e) => setModel(e.target.value)} className="font-mono text-xs" />
        </section>
      )}

      {provider && needsKey && (
        <section className="mt-4">
          <Label hint={hasKey ? "Saved in Keychain" : undefined}>API key</Label>
          <div className="flex gap-1.5">
            <div className="relative flex-1">
              <KeyRound className="pointer-events-none absolute left-2.5 top-2.5 size-3.5 text-faint" />
              <Input
                type="password"
                value={key}
                onChange={(e) => setKey(e.target.value)}
                placeholder={hasKey ? "•••••••••••• (leave empty to keep)" : provider === "anthropic" ? "sk-ant-…" : "sk-…"}
                className="pl-8 font-mono text-xs"
              />
            </div>
            {hasKey && (
              <Button variant="ghost" onClick={removeKey}>
                Remove
              </Button>
            )}
          </div>
        </section>
      )}

      {!provider && (
        <p className="mt-3 text-xs leading-relaxed text-muted">
          Without an AI provider you can still build and start Cues by hand. Nothing about running a Cue depends on AI.
        </p>
      )}

      <section className="mt-6 border-t border-border pt-4">
        <Label>Your Cues</Label>
        <div className="flex items-center gap-2">
          <code className="min-w-0 flex-1 truncate rounded-md bg-surface-2 px-2 py-1.5 font-mono text-[11px] text-muted">{folder}</code>
          <Button size="sm" onClick={() => SystemService.RevealCuesFolder()}>
            <FolderOpen className="size-3.5" />
            Show in Finder
          </Button>
        </div>
      </section>
    </Dialog>
  );
}
