import { Browser } from "@wailsio/runtime";
import { Check, ExternalLink, FolderOpen, KeyRound, Lock, RefreshCw } from "lucide-react";
import { useEffect, useState } from "react";
import { AIService, Config, disclosureSentence, errorMessage, Prefs, SystemService, type AISettings } from "../api";
import { usePrefs } from "../prefs";
import { Button, Input, Label, Sheet } from "./ui";

const presets = [
  { name: "OpenAI", url: "https://api.openai.com/v1", key: true },
  { name: "OpenRouter", url: "https://openrouter.ai/api/v1", key: true },
  { name: "Ollama", url: "http://localhost:11434/v1", key: false },
  { name: "LM Studio", url: "http://localhost:1234/v1", key: false },
];

type Provider = "" | "anthropic" | "gemini" | "openai";

const providers: [Provider, string][] = [
  ["anthropic", "Anthropic"],
  ["gemini", "Gemini"],
  ["openai", "OpenAI-compatible"],
  ["", "None"],
];

const defaultModel: Record<string, string> = { anthropic: "claude-opus-5-5", gemini: "gemini-2.5-flash", openai: "" };
const GEMINI_KEY_URL = "https://aistudio.google.com/apikey";

const appearances: [string, string][] = [
  ["system", "System"],
  ["light", "Light"],
  ["dark", "Dark"],
];

export function SettingsDialog({ folder, onClose, onSaved }: { folder: string; onClose: () => void; onSaved: (s: AISettings) => void }) {
  const { prefs, save: savePrefs } = usePrefs();
  const [loaded, setLoaded] = useState<AISettings | null>(null);
  const [provider, setProvider] = useState<Provider>("anthropic");
  const [model, setModel] = useState("");
  const [baseUrl, setBaseUrl] = useState("");
  const [key, setKey] = useState("");
  const [status, setStatus] = useState<{ kind: "ok" | "error"; text: string } | null>(null);
  const [busy, setBusy] = useState<"save" | "test" | null>(null);
  const [models, setModels] = useState<string[]>([]);
  const [modelsBusy, setModelsBusy] = useState(false);
  const [modelsError, setModelsError] = useState("");

  useEffect(() => {
    AIService.GetSettings()
      .then((s) => {
        setLoaded(s);
        const p = (s.config.provider || "anthropic") as Provider;
        setProvider(p);
        setModel(s.config.model || defaultModel[p] || "");
        setBaseUrl(s.config.baseUrl);
      })
      .catch((e) => {
        // Still open, with defaults, and say what went wrong.
        setLoaded({ config: new Config({ provider: "", model: "", baseUrl: "" }), hasKey: false, ready: false } as AISettings);
        setModel(defaultModel.anthropic);
        setStatus({ kind: "error", text: errorMessage(e) });
      });
  }, []);

  const switchProvider = (p: Provider) => {
    setProvider(p);
    setStatus(null);
    setKey("");
    setModels([]);
    setModelsError("");
    const same = loaded?.config.provider === p;
    if (p === "anthropic" || p === "gemini") {
      setModel(same ? loaded!.config.model : defaultModel[p]);
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

  // Ask the provider which models this key can use, so nobody has to guess ids.
  const loadModels = async () => {
    setModelsBusy(true);
    setModelsError("");
    try {
      const list = await AIService.ListModels(new Config({ provider, model, baseUrl }), key);
      setModels(list);
      if (!list.length) setModelsError("No models found.");
    } catch (e) {
      setModels([]);
      setModelsError(errorMessage(e));
    } finally {
      setModelsBusy(false);
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
  const needsKey = provider === "anthropic" || provider === "gemini" || (provider === "openai" && (preset?.key ?? true));
  const canList = provider !== "" && (!needsKey || hasKey || key.trim() !== "") && (provider !== "openai" || baseUrl !== "");
  const local = provider === "openai" && /^https?:\/\/(localhost|127\.0\.0\.1|\[::1\])/.test(baseUrl);
  const where =
    provider === ""
      ? ""
      : disclosureSentence({
          model: model || "your model",
          vendor:
            provider === "anthropic"
              ? "Anthropic"
              : provider === "gemini"
                ? "Google"
                : (preset?.name ?? baseUrl.replace(/^https?:\/\//, "").split("/")[0]),
          local,
          note: provider === "gemini" ? "On Google’s free tier, Google may use your requests to improve its products." : undefined,
        });

  // Settings load from a local file in a moment; wait rather than briefly
  // showing defaults (it used to flash "Anthropic" before the real choice).
  if (!loaded) return null;

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
        <Label hint="System follows your Mac">Appearance</Label>
        <div role="radiogroup" aria-label="Appearance" className="grid grid-cols-3 gap-1 rounded-full bg-fill p-1">
          {appearances.map(([value, name]) => (
            <button
              key={value}
              role="radio"
              aria-checked={prefs.appearance === value}
              onClick={() => savePrefs((p) => new Prefs({ ...p, appearance: value }))}
              className={`h-7 rounded-full text-[12px] font-medium transition ${
                prefs.appearance === value ? "bg-surface text-ink shadow-card" : "text-graphite hover:text-ink"
              }`}
            >
              {name}
            </button>
          ))}
        </div>
      </section>

      <section className="mt-5">
        <Label>AI for describing Cues</Label>
        <div role="radiogroup" className="grid grid-cols-4 gap-1 rounded-full bg-fill p-1">
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

      {provider === "gemini" && (
        <p className="mt-3 text-[12px] leading-relaxed text-graphite">
          Gemini has a free tier. Get an API key in Google AI Studio, paste it below, then load the models your key can use.{" "}
          <button onClick={() => Browser.OpenURL(GEMINI_KEY_URL)} className="inline-flex items-center gap-1 font-medium text-red-text hover:underline">
            Get a free key
            <ExternalLink className="size-3" />
          </button>
        </p>
      )}

      {provider && (
        <section className="mt-5">
          <Label
            hint={
              provider === "gemini"
                ? "If it’s busy, Cue Space switches to Gemini 2.5 Flash automatically"
                : defaultModel[provider]
                  ? `Default: ${defaultModel[provider]}`
                  : "e.g. gpt-5, llama3.2, qwen3"
            }
          >
            Model
          </Label>
          <div className="flex gap-1.5">
            {models.length > 0 ? (
              <select
                value={model}
                onChange={(e) => setModel(e.target.value)}
                aria-label="Model"
                className="h-8 min-w-0 flex-1 rounded-lg border border-hairline bg-surface px-2 font-mono text-xs focus:border-red-text/60 focus:outline-none"
              >
                {!models.includes(model) && model && <option value={model}>{model}</option>}
                {models.map((m) => (
                  <option key={m} value={m}>
                    {m}
                  </option>
                ))}
              </select>
            ) : (
              <Input value={model} onChange={(e) => setModel(e.target.value)} className="font-mono text-xs" />
            )}
            <Button onClick={loadModels} busy={modelsBusy} disabled={!canList} title={canList ? "Load the models this key can use" : "Add your API key first"}>
              {!modelsBusy && <RefreshCw className="size-3.5" />}
              Load Models
            </Button>
          </div>
          {modelsError && <p className="mt-1.5 text-[11.5px] text-red-text">{modelsError}</p>}
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
                placeholder={
                  hasKey ? "Leave empty to keep the saved key" : provider === "anthropic" ? "sk-ant-…" : provider === "gemini" ? "AIza…" : "sk-…"
                }
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
