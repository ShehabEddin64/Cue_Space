import { Check, FolderOpen, KeyRound, Lock } from "lucide-react";
import { useEffect, useState } from "react";
import { AIService, Config, disclosureSentence, errorMessage, SystemService, type AISettings } from "../api";
import { Button, Input, Label, Sheet } from "./ui";

const presets = [
  { name: "OpenAI", url: "https://api.openai.com/v1", key: true },
  { name: "OpenRouter", url: "https://openrouter.ai/api/v1", key: true },
  { name: "Ollama", url: "http://localhost:11434/v1", key: false },
  { name: "LM Studio", url: "http://localhost:1234/v1", key: false },
];

type Provider = "" | "anthropic" | "openai";

const providers: [Provider, string][] = [
  ["anthropic", "Anthropic"],
  ["openai", "OpenAI-compatible"],
  ["", "None"],
];

export function SettingsDialog({ folder, onClose, onSaved }: { folder: string; onClose: () => void; onSaved: (s: AISettings) => void }) {
  const [loaded, setLoaded] = useState<AISettings | null>(null);
  const [provider, setProvider] = useState<Provider>("anthropic");
  const [model, setModel] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [key, setKey] = useState("");
  const [status, setStatus] = useState<{ kind: "ok" | "error"; text: string } | null>(null);
  const [busy, setBusy] = useState<"save" | "test" | null>(null);

  useEffect(() => {
    AIService.GetSettings().then((s) => {
      setLoaded(s);
      const p = (s.config.provider || "anthropic") as Provider;
      setProvider(p);
      setModel(s.config.model || (p === "anthropic" ? "claude-opus-5-5" : ""));
      setBaseUrl(s.config.baseUrl);
    });
  }, []);

  const switchProvider = (p: Provider) => {
    setProvider(p);
    setStatus(null);
    setKey("");
    const same = loaded?.config.provider === p;
    if (p === "anthropic") {
      setModel(same ? loaded!.config.model : "claude-opus-5-5");
      setBaseUrl("");
    } else if (p === "openai") {
      setModel(same ? loaded!.config.model : "");
      setBaseUrl(same ? loaded!.config.baseUrl : presets[2].url);
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
  const local = provider === "openai" && /^https?:\/\/(localhost|127\.0\.0\.1|\[::1\])/.test(baseUrl);
  const where =
    provider === ""
      ? ""
      : disclosureSentence({
          model: model || "your model",
          vendor: provider === "anthropic" ? "Anthropic" : (preset?.name ?? baseUrl.replace(/^https?:\/\//, "").split("/")[0]),
          local,
        });

  return (
    <Sheet
      title="Settings"
      onClose={onClose}
      footer={
        <>
          {status && (
            <span
              className={`mr-auto flex items-center gap-1.5 text-[12px] ${status.kind === "ok" ? "text-graphite" : "text-red-text"}`}
              role="status"
            >
              {status.kind === "ok" && <Check className="size-3.5 text-red-text" />}
              {status.text}
            </span>
          )}
          {provider && (
            <Button busy={busy === "test"} disabled={!!busy} onClick={test}>
              Test Connection
            </Button>
          )}
          <Button variant="primary" busy={busy === "save"} disabled={!!busy} onClick={save}>
            Save
          </Button>
        </>
      }
    >
      <section>
        <Label>AI for describing Cues</Label>
        <div role="radiogroup" className="grid grid-cols-3 gap-1 rounded-full bg-fill p-1">
          {providers.map(([value, name]) => (
            <button
              key={value || "none"}
              role="radio"
              aria-checked={provider === value}
              onClick={() => switchProvider(value)}
              className={`h-7 rounded-full text-[12px] font-medium transition ${
                provider === value ? "bg-surface text-ink shadow-card" : "text-graphite hover:text-ink"
              }`}
            >
              {name}
            </button>
          ))}
        </div>
      </section>

      {provider === "openai" && (
        <section className="mt-5">
          <Label>Server</Label>
          <div className="mb-2 flex flex-wrap gap-1.5">
            {presets.map((p) => (
              <button
                key={p.name}
                onClick={() => setBaseUrl(p.url)}
                className={`rounded-full border px-2.5 py-0.5 text-[11.5px] transition ${
                  baseUrl === p.url ? "border-red-text/50 bg-red-tint text-red-on-tint" : "border-hairline text-graphite hover:text-ink"
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
        <section className="mt-5">
          <Label hint={provider === "anthropic" ? "Default: claude-opus-5-5" : "e.g. gpt-5, llama3.2, qwen3"}>Model</Label>
          <Input value={model} onChange={(e) => setModel(e.target.value)} className="font-mono text-xs" />
        </section>
      )}

      {provider && needsKey && (
        <section className="mt-5">
          <Label hint={hasKey ? "Saved in your Keychain" : undefined}>API key</Label>
          <div className="flex gap-1.5">
            <div className="relative flex-1">
              <KeyRound className="pointer-events-none absolute left-2.5 top-2.5 size-3.5 text-faint" />
              <Input
                type="password"
                value={key}
                onChange={(e) => setKey(e.target.value)}
                placeholder={hasKey ? "Leave empty to keep the saved key" : provider === "anthropic" ? "sk-ant-…" : "sk-…"}
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

      <p className="mt-4 flex gap-2 rounded-xl bg-fill/70 px-3 py-2.5 text-[12px] leading-relaxed text-graphite">
        <Lock className="mt-[2px] size-3.5 shrink-0" />
        <span>
          {provider
            ? `${where} The AI only proposes Cues for you to review. Starting a Cue never uses it.`
            : "Without an AI you can still build and start Cues by hand. Nothing about starting a Cue depends on AI."}
        </span>
      </p>

      <section className="mt-6">
        <Label>Your Cues</Label>
        <div className="flex items-center gap-2">
          <code className="min-w-0 flex-1 truncate rounded-lg bg-fill px-2.5 py-1.5 font-mono text-[11px] text-graphite">{folder}</code>
          <Button size="sm" onClick={() => SystemService.RevealCuesFolder()}>
            <FolderOpen className="size-3.5" />
            Show in Finder
          </Button>
        </div>
      </section>
    </Sheet>
  );
}
