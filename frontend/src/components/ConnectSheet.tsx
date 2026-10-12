import { Clipboard } from "@wailsio/runtime";
import { Check, Copy, MessageSquare, SquareTerminal, TriangleAlert } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";
import { ConnectService, errorMessage, type Connections } from "../api";
import { Button, CueMark, Sheet } from "./ui";

const example =
  "Make a Cue for studying math: Moodle, the newest PDF in my Downloads, and my Math folder in VS Code. Put Moodle and the PDF side by side on a new desktop.";

/**
 * Connect Cue Space to AI apps through its MCP server (cuespace-cli), so
 * Claude or ChatGPT can build Cues from a conversation.
 */
export function ConnectSheet({ onClose }: { onClose: () => void }) {
  const [c, setC] = useState<Connections | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [justAdded, setJustAdded] = useState(false);

  useEffect(() => {
    ConnectService.Status().then(setC);
  }, []);

  const change = async (add: boolean) => {
    setBusy(true);
    setError("");
    try {
      setC(add ? await ConnectService.AddToClaudeDesktop() : await ConnectService.RemoveFromClaudeDesktop());
      setJustAdded(add);
    } catch (e) {
      setError(errorMessage(e));
    } finally {
      setBusy(false);
    }
  };

  return (
    <Sheet
      title="Connect AI Apps"
      onClose={onClose}
      width="max-w-xl"
      footer={
        <Button variant="primary" onClick={onClose}>
          Done
        </Button>
      }
    >
      <p className="text-[13px] leading-relaxed text-graphite">
        Let Claude or ChatGPT build Cues with you. They can see your Cues and installed apps, find folders by name, and create or change Cues. A Cue
        only starts when you ask, and starting it never uses AI.
      </p>

      {c && !c.cli && (
        <p className="mt-4 flex gap-2 rounded-xl bg-red-tint px-3 py-2.5 text-[12px] text-red-on-tint">
          <TriangleAlert className="mt-px size-3.5 shrink-0" />
          This build doesn’t include the cuespace-cli tool. Build the app with ./scripts/build.sh.
        </p>
      )}

      {c?.cli && (
        <div className="mt-4 space-y-3">
          <Card icon={<span className="text-[15px] font-semibold text-red-text">C</span>} title="Claude Desktop" status={claudeStatus(c)}>
            {!c.claudeDesktop ? (
              <p className="text-[12px] text-graphite">Install Claude Desktop and open it once, then come back here.</p>
            ) : (
              <>
                <div className="flex flex-wrap items-center gap-2">
                  {c.claudeDesktopStatus !== "added" ? (
                    <Button variant="primary" size="sm" busy={busy} onClick={() => change(true)}>
                      {c.claudeDesktopStatus === "outdated" ? "Update in Claude Desktop" : "Add to Claude Desktop"}
                    </Button>
                  ) : (
                    <Button variant="ghost" size="sm" busy={busy} onClick={() => change(false)}>
                      Remove from Claude Desktop
                    </Button>
                  )}
                </div>
                {c.claudeDesktopStatus === "outdated" && (
                  <p className="mt-2 text-[12px] text-graphite">Claude points at another copy of Cue Space, for example before the app was moved.</p>
                )}
                {(justAdded || c.claudeDesktopStatus === "added") && (
                  <p className="mt-2 text-[12px] leading-relaxed text-graphite">
                    {justAdded && <span className="font-medium text-ink">Added. Quit Claude (⌘Q) and open it again to load Cue Space. </span>}
                    Then ask Claude to make a Cue, for example: <span className="italic">“{example}”</span>
                  </p>
                )}
              </>
            )}
          </Card>

          <Card icon={<SquareTerminal className="size-4 text-red-text" />} title="Claude Code">
            <p className="mb-2 text-[12px] text-graphite">Run this once in Terminal:</p>
            <Command text={c.claudeCode} />
          </Card>

          <Card icon={<MessageSquare className="size-4 text-red-text" />} title="ChatGPT" status="Advanced">
            <p className="text-[12px] leading-relaxed text-graphite">
              ChatGPT connects to tools by web address, so it needs Cue Space’s server running and reachable from the internet through a tunnel
              (for example cloudflared). Start the server in Terminal:
            </p>
            <div className="mt-2">
              <Command text={c.httpCommand} />
            </div>
            <p className="mt-3 text-[12px] leading-relaxed text-graphite">
              Point the tunnel at port 8765, then add a connector in ChatGPT’s developer mode with the tunnel’s https address followed by this
              path, and no authentication:
            </p>
            <div className="mt-2">
              <Command text={c.httpUrl ? new URL(c.httpUrl).pathname : ""} secret />
            </div>
            <p className="mt-2 flex gap-1.5 text-[11.5px] text-red-on-tint">
              <TriangleAlert className="mt-px size-3 shrink-0" />
              Keep this address private: anyone with it can change and start your Cues. Stop the server (Ctrl-C) when you’re done.
            </p>
          </Card>

          <Card icon={<SquareTerminal className="size-4 text-red-text" />} title="Terminal">
            <p className="mb-2 text-[12px] text-graphite">
              To use <span className="font-mono text-ink">cuespace-cli list</span>, <span className="font-mono text-ink">start</span> and{" "}
              <span className="font-mono text-ink">create</span> from any shell, link it once:
            </p>
            <Command text={c.terminal} />
          </Card>
        </div>
      )}

      {error && <p className="mt-3 text-[12px] font-medium text-red-text">{error}</p>}
    </Sheet>
  );
}

function claudeStatus(c: Connections): string | undefined {
  if (!c.claudeDesktop) return "Not installed";
  if (c.claudeDesktopStatus === "added") return "Connected";
  if (c.claudeDesktopStatus === "outdated") return "Needs update";
  return undefined;
}

function Card({ icon, title, status, children }: { icon: ReactNode; title: string; status?: string; children: ReactNode }) {
  return (
    <section className="rounded-2xl border border-hairline bg-surface p-3.5 shadow-card">
      <div className="mb-2.5 flex items-center gap-2.5">
        <span className="flex size-7 shrink-0 items-center justify-center rounded-lg bg-red-tint">{icon}</span>
        <h3 className="flex-1 text-[13.5px] font-semibold">{title}</h3>
        {status && (
          <span
            className={`flex items-center gap-1 rounded-full px-2 py-0.5 text-[11px] font-medium ${
              status === "Connected" ? "bg-red-tint text-red-on-tint" : "bg-fill text-graphite"
            }`}
          >
            {status === "Connected" && <CueMark size={10} />}
            {status}
          </span>
        )}
      </div>
      {children}
    </section>
  );
}

/** A command or address with a Copy button. Secrets stay hidden until shown. */
function Command({ text, secret }: { text: string; secret?: boolean }) {
  const [copied, setCopied] = useState(false);
  const [shown, setShown] = useState(!secret);
  const copy = async () => {
    await Clipboard.SetText(text);
    setCopied(true);
    setTimeout(() => setCopied(false), 1600);
  };
  return (
    <div className="flex items-center gap-1.5">
      <code className="min-w-0 flex-1 truncate rounded-lg bg-fill px-2.5 py-1.5 font-mono text-[11px] text-ink" title={shown ? text : undefined}>
        {shown ? text : text.replace(/[0-9a-f]{16,}/, "•".repeat(12))}
      </code>
      {secret && (
        <Button size="sm" variant="ghost" onClick={() => setShown((s) => !s)}>
          {shown ? "Hide" : "Show"}
        </Button>
      )}
      <Button size="sm" onClick={copy} aria-label="Copy">
        {copied ? <Check className="size-3.5" /> : <Copy className="size-3.5" />}
        {copied ? "Copied" : "Copy"}
      </Button>
    </div>
  );
}
