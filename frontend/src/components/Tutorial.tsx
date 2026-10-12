import {
  AppWindow,
  ArrowRight,
  Bot,
  Clock,
  FolderClock,
  FolderOpen,
  Keyboard,
  LayoutGrid,
  LayoutPanelLeft,
  Play,
  Plus,
  Sparkles,
  SquareTerminal,
} from "lucide-react";
import { useEffect, useRef, useState, type ComponentType, type ReactNode } from "react";
import { language, t } from "../i18n";
import { Toolbar } from "./Toolbar";
import { Button, CueMark } from "./ui";

/** What a guide's buttons can do; App carries them out. */
export type TutorialAction =
  | "settings"
  | "tour"
  | "new-blank-cue"
  | "new-cue"
  | "accessibility"
  | "shortcuts"
  | "customize-home"
  | "connect"
  | "reveal-folder"
  | "recents";

type Text = { en: string; fr: string };

type Topic = {
  id: string;
  icon: ComponentType<{ className?: string; strokeWidth?: number | string }>;
  title: Text;
  summary: Text;
  steps: Text[];
  tip?: Text;
  actions?: { label: Text; action: TutorialAction }[];
};

/*
 * The guides. Text supports `code` for keys, commands and menu paths, and
 * **bold** for the words to look for on screen.
 */
const topics: Topic[] = [
  {
    id: "start",
    icon: Sparkles,
    title: { en: "Get started", fr: "Premiers pas" },
    summary: {
      en: "A Cue is one activity, like “Study math” or “Work on Atlas”: the links, apps, files and folders it needs, opened together in one click.",
      fr: "Un Cue, c’est une activité, comme « Étudier les maths » ou « Travailler sur Atlas » : les liens, apps, fichiers et dossiers dont elle a besoin, ouverts ensemble en un clic.",
    },
    steps: [
      {
        en: "**Home** is where you start: describe what you want to do in the box at the top, or click **Create Manually**.",
        fr: "L’**Accueil** est votre point de départ : décrivez ce que vous voulez faire dans le champ en haut, ou cliquez sur **Créer manuellement**.",
      },
      {
        en: "Your Cues are listed in the sidebar and in **Cues**. Click one to see what it opens, then **Start Cue**.",
        fr: "Vos Cues sont listés dans la barre latérale et dans **Cues**. Cliquez sur l’un d’eux pour voir ce qu’il ouvre, puis sur **Lancer le Cue**.",
      },
      {
        en: "Starting a Cue never uses AI: it opens exactly what’s saved, every time.",
        fr: "Lancer un Cue n’utilise jamais l’IA : il ouvre exactement ce qui est enregistré, à chaque fois.",
      },
    ],
    tip: {
      en: "Change the language and appearance any time in Settings (`⌘,`).",
      fr: "Changez la langue et l’apparence quand vous voulez dans les Réglages (`⌘,`).",
    },
    actions: [
      { label: { en: "Replay the Welcome Tour", fr: "Revoir la visite de bienvenue" }, action: "tour" },
      { label: { en: "Open Settings", fr: "Ouvrir les Réglages" }, action: "settings" },
    ],
  },
  {
    id: "ai",
    icon: Bot,
    title: { en: "Create a Cue with AI", fr: "Créer un Cue avec l’IA" },
    summary: {
      en: "Describe an activity in your own words and your AI proposes a Cue for you to review. You bring the AI; Gemini has a free tier.",
      fr: "Décrivez une activité avec vos mots et votre IA propose un Cue que vous vérifiez. Vous choisissez l’IA ; Gemini a une offre gratuite.",
    },
    steps: [
      {
        en: "Open **Settings** (`⌘,`) and choose an AI: **Gemini**, **Anthropic**, **OpenAI-compatible** (including Ollama or LM Studio on your Mac).",
        fr: "Ouvrez les **Réglages** (`⌘,`) et choisissez une IA : **Gemini**, **Anthropic**, **Compatible OpenAI** (y compris Ollama ou LM Studio sur votre Mac).",
      },
      {
        en: "For Gemini, click **Get a free key**, create a key in Google AI Studio, paste it in **API key**, then click **Load Models** and **Save**.",
        fr: "Pour Gemini, cliquez sur **Obtenir une clé gratuite**, créez une clé dans Google AI Studio, collez-la dans **Clé d’API**, puis cliquez sur **Charger les modèles** et **Enregistrer**.",
      },
      {
        en: "Click **Test Connection** to check it works. Your key is kept in the macOS Keychain.",
        fr: "Cliquez sur **Tester la connexion** pour vérifier. Votre clé est conservée dans le trousseau de macOS.",
      },
      {
        en: "On Home, describe the activity, e.g. “Study math: Moodle, my textbook PDF and Calculator”, and press `Return`.",
        fr: "Sur l’Accueil, décrivez l’activité, par exemple « Étudier les maths : Moodle, mon manuel PDF et Calculette », puis appuyez sur `Retour`.",
      },
      {
        en: "Review the proposal: fill in anything it asks, refine it with a sentence or a command, then click **Save Cue**.",
        fr: "Vérifiez la proposition : complétez ce qu’elle demande, affinez-la avec une phrase ou une commande, puis cliquez sur **Enregistrer le Cue**.",
      },
    ],
    tip: {
      en: "Press `Tab` in the empty box to use the idea shown there, and `↑` `↓` to see others.",
      fr: "Appuyez sur `Tab` dans le champ vide pour utiliser l’idée affichée, et sur `↑` `↓` pour en voir d’autres.",
    },
    actions: [
      { label: { en: "Open Settings", fr: "Ouvrir les Réglages" }, action: "settings" },
      { label: { en: "Describe a Cue", fr: "Décrire un Cue" }, action: "new-cue" },
    ],
  },
  {
    id: "manual",
    icon: Plus,
    title: { en: "Create a Cue manually", fr: "Créer un Cue manuellement" },
    summary: {
      en: "No AI needed: pick every link, app, file and folder yourself.",
      fr: "Aucune IA nécessaire : choisissez vous-même chaque lien, app, fichier et dossier.",
    },
    steps: [
      {
        en: "Click **Create Manually** on Home, the **+** in the sidebar, or press `⌥⌘N`.",
        fr: "Cliquez sur **Créer manuellement** sur l’Accueil, sur le **+** de la barre latérale, ou appuyez sur `⌥⌘N`.",
      },
      {
        en: "Give it a name and an emoji icon.",
        fr: "Donnez-lui un nom et une icône emoji.",
      },
      {
        en: "Use **Add Link**, **Add App**, **Add File** and **Add Folder**. You can also paste a link anywhere on the page.",
        fr: "Utilisez **Ajouter un lien**, **Ajouter une app**, **Ajouter un fichier** et **Ajouter un dossier**. Vous pouvez aussi coller un lien n’importe où sur la page.",
      },
      {
        en: "Drag items to set the order they open in. The switch on each item turns it off without deleting it.",
        fr: "Faites glisser les éléments pour choisir leur ordre d’ouverture. L’interrupteur de chaque élément le désactive sans le supprimer.",
      },
      {
        en: "Files and folders can open in a specific app with **Open with**, e.g. a project folder in VS Code.",
        fr: "Les fichiers et dossiers peuvent s’ouvrir dans une app précise avec **Ouvrir avec**, par exemple un dossier de projet dans VS Code.",
      },
      { en: "Click **Save** or press `⌘S`.", fr: "Cliquez sur **Enregistrer** ou appuyez sur `⌘S`." },
    ],
    actions: [{ label: { en: "Create Manually", fr: "Créer manuellement" }, action: "new-blank-cue" }],
  },
  {
    id: "run",
    icon: Play,
    title: { en: "Start a Cue", fr: "Lancer un Cue" },
    summary: {
      en: "Everything in a Cue opens in order, in one click or one keystroke.",
      fr: "Tout ce que contient un Cue s’ouvre dans l’ordre, en un clic ou une touche.",
    },
    steps: [
      {
        en: "Click **Start Cue** on a Cue, on its card, or the ▶ next to it in the sidebar.",
        fr: "Cliquez sur **Lancer le Cue** sur un Cue, sur sa carte, ou sur le ▶ à côté de lui dans la barre latérale.",
      },
      {
        en: "Your first nine Cues get `⌘1` to `⌘9` (pinned Cues first). The **Cues** menu lists them.",
        fr: "Vos neuf premiers Cues ont `⌘1` à `⌘9` (les Cues épinglés d’abord). Le menu **Cues** les liste.",
      },
      {
        en: "A short report confirms what opened. If something failed, click **Fix in Editor**.",
        fr: "Un court rapport confirme ce qui s’est ouvert. Si quelque chose a échoué, cliquez sur **Corriger dans l’éditeur**.",
      },
    ],
    tip: {
      en: "On a Cue’s page, `⌘Return` starts it and `⌘E` edits it.",
      fr: "Sur la page d’un Cue, `⌘Retour` le lance et `⌘E` le modifie.",
    },
  },
  {
    id: "desktops",
    icon: LayoutPanelLeft,
    title: { en: "Arrange windows on desktops", fr: "Disposer les fenêtres sur des bureaux" },
    summary: {
      en: "Put an app in full screen on its own desktop, two windows side by side on a new desktop, quarters, thirds and more.",
      fr: "Mettez une app en plein écran sur son propre bureau, deux fenêtres côte à côte sur un nouveau bureau, en quarts, en tiers, et plus.",
    },
    steps: [
      {
        en: "Allow Cue Space once: **System Settings › Privacy & Security › Accessibility**, then turn on **Cue Space**.",
        fr: "Autorisez Cue Space une fois : **Réglages Système › Confidentialité et sécurité › Accessibilité**, puis activez **Cue Space**.",
      },
      {
        en: "Open a Cue, click **Edit Cue**, and in **Desktops** click **Add Desktop**.",
        fr: "Ouvrez un Cue, cliquez sur **Modifier le Cue**, puis dans **Bureaux** cliquez sur **Ajouter un bureau**.",
      },
      {
        en: "Pick an arrangement (Full Screen, Left & Right, Quarters…) and choose **On a new desktop** or **On the current desktop**.",
        fr: "Choisissez une disposition (Plein écran, Gauche et droite, Quarts…) puis **Sur un nouveau bureau** ou **Sur le bureau actuel**.",
      },
      {
        en: "For each numbered slot, choose which item goes there. The preview shows the result, like Mission Control.",
        fr: "Pour chaque emplacement numéroté, choisissez l’élément qui y va. L’aperçu montre le résultat, comme Mission Control.",
      },
      { en: "Save, then start the Cue.", fr: "Enregistrez, puis lancez le Cue." },
    ],
    tip: {
      en: "If a new desktop can’t be created, the windows are arranged on the current one and the report says so.",
      fr: "Si un nouveau bureau ne peut pas être créé, les fenêtres sont disposées sur le bureau actuel et le rapport l’indique.",
    },
    actions: [{ label: { en: "Allow Accessibility…", fr: "Autoriser l’accessibilité…" }, action: "accessibility" }],
  },
  {
    id: "newest",
    icon: FolderClock,
    title: { en: "Open the newest file in a folder", fr: "Ouvrir le fichier le plus récent d’un dossier" },
    summary: {
      en: "Great for your latest download, the newest scan or the last export: the Cue opens whatever arrived most recently.",
      fr: "Idéal pour votre dernier téléchargement, le scan le plus récent ou la dernière exportation : le Cue ouvre ce qui est arrivé en dernier.",
    },
    steps: [
      {
        en: "Edit a Cue and click **Add Folder**, e.g. your Downloads folder.",
        fr: "Modifiez un Cue et cliquez sur **Ajouter un dossier**, par exemple votre dossier Téléchargements.",
      },
      {
        en: "Tick **Open the newest file inside, not the folder**.",
        fr: "Cochez **Ouvrir le fichier le plus récent, pas le dossier**.",
      },
      {
        en: "Optionally type file types in **Only**, e.g. `pdf` or `pdf, docx`.",
        fr: "Si vous voulez, indiquez des types de fichiers dans **Seulement**, par exemple `pdf` ou `pdf, docx`.",
      },
      {
        en: "Save. Each start opens the newest matching file, and the report names it.",
        fr: "Enregistrez. Chaque lancement ouvre le fichier correspondant le plus récent, et le rapport le nomme.",
      },
    ],
  },
  {
    id: "commands",
    icon: Keyboard,
    title: { en: "Commands and shortcuts", fr: "Commandes et raccourcis" },
    summary: {
      en: "Type `/` in a box for commands. Most run on your Mac without any AI, so they’re instant and free.",
      fr: "Tapez `/` dans un champ pour les commandes. La plupart fonctionnent sur votre Mac sans IA : elles sont instantanées et gratuites.",
    },
    steps: [
      {
        en: "On Home: `/create` proposes a Cue, `/new` creates one manually, `/start` and `/open` find a Cue by name.",
        fr: "Sur l’Accueil : `/create` propose un Cue, `/new` en crée un manuellement, `/start` et `/open` trouvent un Cue par son nom.",
      },
      {
        en: "On a Cue: `/add figma, ~/Notes, github.com/me/repo` adds items, finding them on your Mac first.",
        fr: "Sur un Cue : `/add figma, ~/Notes, github.com/moi/projet` ajoute des éléments, en les cherchant d’abord sur votre Mac.",
      },
      {
        en: "`/remove`, `/off` and `/on` change items; `/rename`, `/icon`, `/describe` and `/pin` change the Cue.",
        fr: "`/remove`, `/off` et `/on` modifient les éléments ; `/rename`, `/icon`, `/describe` et `/pin` modifient le Cue.",
      },
      {
        en: "Press `⌘/` to see every shortcut and command.",
        fr: "Appuyez sur `⌘/` pour voir tous les raccourcis et commandes.",
      },
    ],
    actions: [{ label: { en: "Show All Shortcuts", fr: "Voir tous les raccourcis" }, action: "shortcuts" }],
  },
  {
    id: "home",
    icon: LayoutGrid,
    title: { en: "Pins, Recently Used and Home", fr: "Épingles, récents et Accueil" },
    summary: {
      en: "Keep your favorite Cues in front and arrange Home the way you like.",
      fr: "Gardez vos Cues préférés en avant et organisez l’Accueil à votre goût.",
    },
    steps: [
      {
        en: "Open a Cue and click **Pin** (or press `⇧⌘P`). Pinned Cues come first on Home, in the sidebar and in `⌘1`–`⌘9`.",
        fr: "Ouvrez un Cue et cliquez sur **Épingler** (ou appuyez sur `⇧⌘P`). Les Cues épinglés passent en premier sur l’Accueil, dans la barre latérale et dans `⌘1`–`⌘9`.",
      },
      {
        en: "**Recently Used** in the sidebar lists the Cues you started, grouped by day.",
        fr: "**Utilisés récemment**, dans la barre latérale, liste les Cues que vous avez lancés, regroupés par jour.",
      },
      {
        en: "Click **Customize** on Home to show, hide and reorder sections and pick Small, Medium or Large for each, like widgets.",
        fr: "Cliquez sur **Personnaliser** sur l’Accueil pour afficher, masquer et réordonner les sections, et choisir Petit, Moyen ou Grand pour chacune, comme des widgets.",
      },
    ],
    tip: {
      en: "Hover a section’s title on Home to change its size right there.",
      fr: "Survolez le titre d’une section sur l’Accueil pour changer sa taille directement.",
    },
    actions: [
      { label: { en: "Customize Home", fr: "Personnaliser l’accueil" }, action: "customize-home" },
      { label: { en: "Recently Used", fr: "Utilisés récemment" }, action: "recents" },
    ],
  },
  {
    id: "widget",
    icon: AppWindow,
    title: { en: "Add the widget", fr: "Ajouter le widget" },
    summary: {
      en: "Start Cues from your desktop or Notification Center without opening the app.",
      fr: "Lancez vos Cues depuis le bureau ou le centre de notifications sans ouvrir l’app.",
    },
    steps: [
      {
        en: "Right-click an empty spot on your desktop and choose **Edit Widgets…**",
        fr: "Faites un clic droit sur un endroit vide du bureau et choisissez **Modifier les widgets…**",
      },
      {
        en: "Search for **Cue Space** and pick a size: Small shows one Cue, Medium four, Large and Extra Large more.",
        fr: "Recherchez **Cue Space** et choisissez une taille : Petit affiche un Cue, Moyen quatre, Grand et Très grand davantage.",
      },
      {
        en: "Drag it onto the desktop or into Notification Center, then click **Done**.",
        fr: "Faites-le glisser sur le bureau ou dans le centre de notifications, puis cliquez sur **Terminé**.",
      },
      {
        en: "Click a Cue in the widget to start it. The widget follows your pins.",
        fr: "Cliquez sur un Cue dans le widget pour le lancer. Le widget suit vos épingles.",
      },
    ],
    tip: {
      en: "Links like `cuespace://start/<id>` also start a Cue from Shortcuts, Raycast or Alfred.",
      fr: "Les liens comme `cuespace://start/<id>` lancent aussi un Cue depuis Raccourcis, Raycast ou Alfred.",
    },
  },
  {
    id: "connect",
    icon: CueMarkIcon,
    title: { en: "Build Cues with Claude or ChatGPT", fr: "Créer des Cues avec Claude ou ChatGPT" },
    summary: {
      en: "Connect Cue Space to an AI app and build Cues by chatting. It can see your Cues and apps, find folders, and create or change Cues.",
      fr: "Connectez Cue Space à une app d’IA et créez des Cues en discutant. Elle voit vos Cues et vos apps, trouve des dossiers, et crée ou modifie des Cues.",
    },
    steps: [
      {
        en: "Open **File › Connect AI Apps…** (or **Settings › Connect…**).",
        fr: "Ouvrez **Fichier › Connecter des apps d’IA…** (ou **Réglages › Connecter…**).",
      },
      {
        en: "Claude Desktop: click **Add to Claude Desktop**, then quit Claude with `⌘Q` and open it again.",
        fr: "Claude Desktop : cliquez sur **Ajouter à Claude Desktop**, puis quittez Claude avec `⌘Q` et rouvrez-le.",
      },
      {
        en: "In a new chat, ask: “Make me a Cue for studying math with Moodle and my Concordia folder.” Allow Cue Space when Claude asks.",
        fr: "Dans une nouvelle conversation, demandez : « Crée-moi un Cue pour étudier les maths avec Moodle et mon dossier Concordia. » Autorisez Cue Space quand Claude le demande.",
      },
      {
        en: "Claude Code: copy the command shown in the sheet and run it once in Terminal.",
        fr: "Claude Code : copiez la commande affichée dans la fenêtre et exécutez-la une fois dans le Terminal.",
      },
      {
        en: "ChatGPT (advanced): run the server shown in the sheet and expose it through a tunnel. Keep its address private.",
        fr: "ChatGPT (avancé) : lancez le serveur indiqué dans la fenêtre et rendez-le accessible avec un tunnel. Gardez son adresse privée.",
      },
    ],
    tip: {
      en: "The AI only starts a Cue when you ask, and the run is the same as clicking Start.",
      fr: "L’IA ne lance un Cue que si vous le demandez, et le lancement est le même qu’un clic sur Lancer.",
    },
    actions: [{ label: { en: "Connect AI Apps…", fr: "Connecter des apps d’IA…" }, action: "connect" }],
  },
  {
    id: "terminal",
    icon: SquareTerminal,
    title: { en: "Use the terminal", fr: "Utiliser le terminal" },
    summary: {
      en: "`cuespace-cli` lists, shows, starts and creates Cues from any shell or script.",
      fr: "`cuespace-cli` liste, affiche, lance et crée des Cues depuis n’importe quel shell ou script.",
    },
    steps: [
      {
        en: "Open **File › Connect AI Apps…** and copy the **Terminal** command to make `cuespace-cli` available everywhere.",
        fr: "Ouvrez **Fichier › Connecter des apps d’IA…** et copiez la commande **Terminal** pour rendre `cuespace-cli` disponible partout.",
      },
      {
        en: "`cuespace-cli list` shows your Cues; `cuespace-cli start math` starts one.",
        fr: "`cuespace-cli list` affiche vos Cues ; `cuespace-cli start math` en lance un.",
      },
      {
        en: "`cuespace-cli create cue.json` creates a Cue from JSON; `cuespace-cli help` shows the format.",
        fr: "`cuespace-cli create cue.json` crée un Cue à partir de JSON ; `cuespace-cli help` montre le format.",
      },
    ],
    actions: [{ label: { en: "Connect AI Apps…", fr: "Connecter des apps d’IA…" }, action: "connect" }],
  },
  {
    id: "files",
    icon: FolderOpen,
    title: { en: "Your Cues and your privacy", fr: "Vos Cues et votre vie privée" },
    summary: {
      en: "No account and no cloud. Your Cues are plain files on your Mac, and they belong to you.",
      fr: "Pas de compte, pas de cloud. Vos Cues sont de simples fichiers sur votre Mac, et ils vous appartiennent.",
    },
    steps: [
      {
        en: "Each Cue is a JSON file in `~/Library/Application Support/CueSpace/cues`. Back it up, sync it, or edit it: the app updates live.",
        fr: "Chaque Cue est un fichier JSON dans `~/Library/Application Support/CueSpace/cues`. Sauvegardez-le, synchronisez-le ou modifiez-le : l’app se met à jour en direct.",
      },
      {
        en: "API keys stay in the macOS Keychain, never in those files.",
        fr: "Les clés d’API restent dans le trousseau de macOS, jamais dans ces fichiers.",
      },
      {
        en: "When you describe a Cue, only your description and the names of your apps go to the AI you chose. Starting a Cue sends nothing.",
        fr: "Quand vous décrivez un Cue, seuls votre description et les noms de vos apps vont à l’IA choisie. Lancer un Cue n’envoie rien.",
      },
    ],
    actions: [{ label: { en: "Show Cues Folder", fr: "Afficher le dossier des Cues" }, action: "reveal-folder" }],
  },
];

function CueMarkIcon({ className }: { className?: string }) {
  return <CueMark size={16} ring="currentColor" dot="currentColor" className={className} />;
}

/** Renders `code` and **bold** inside a guide's text. */
function Rich({ text }: { text: string }) {
  const parts: ReactNode[] = [];
  const re = /`([^`]+)`|\*\*([^*]+)\*\*/g;
  let last = 0;
  let m: RegExpExecArray | null;
  while ((m = re.exec(text))) {
    if (m.index > last) parts.push(text.slice(last, m.index));
    parts.push(
      m[1] !== undefined ? (
        <code key={m.index} className="rounded-[5px] bg-fill px-1 py-px font-mono text-[0.9em] text-ink">
          {m[1]}
        </code>
      ) : (
        <strong key={m.index} className="font-semibold text-ink">
          {m[2]}
        </strong>
      ),
    );
    last = m.index + m[0].length;
  }
  if (last < text.length) parts.push(text.slice(last));
  return <>{parts}</>;
}

/** A guide to every feature, with buttons that take you to the right place. */
export function Tutorial({ onAction }: { onAction: (a: TutorialAction) => void }) {
  const lang = language();
  const say = (x: Text) => x[lang] || x.en;
  const [active, setActive] = useState(topics[0].id);
  const detail = useRef<HTMLDivElement>(null);
  const index = topics.findIndex((x) => x.id === active);
  const topic = topics[index];
  const next = topics[index + 1];

  useEffect(() => detail.current?.scrollTo({ top: 0 }), [active]);

  return (
    <div className="flex h-full flex-col">
      <Toolbar title={t("Tutorial")} />
      <div className="flex min-h-0 flex-1">
        <nav aria-label={t("Tutorial")} className="w-[236px] shrink-0 overflow-y-auto border-r border-hairline px-2.5 pb-6 pt-1">
          <p className="px-2.5 pb-2 text-[11.5px] leading-relaxed text-graphite">{t("Everything Cue Space can do, and how to set it up.")}</p>
          {topics.map((x, i) => {
            const Icon = x.icon;
            const on = x.id === active;
            return (
              <button
                key={x.id}
                onClick={() => setActive(x.id)}
                aria-current={on ? "page" : undefined}
                className={`flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-left text-[12.5px] ${
                  on ? "bg-selected font-medium text-ink" : "text-ink/85 hover:bg-selected/60"
                }`}
              >
                <span className="w-4 shrink-0 text-right font-mono text-[10.5px] text-faint">{i + 1}</span>
                <Icon className="size-4 shrink-0 text-red-text" strokeWidth={1.9} />
                <span className="min-w-0 flex-1 leading-snug">{say(x.title)}</span>
              </button>
            );
          })}
        </nav>

        <div ref={detail} className="min-w-0 flex-1 overflow-y-auto">
          <article key={topic.id} className="animate-pop-in mx-auto max-w-[620px] px-10 pb-16 pt-6">
            <div className="flex size-12 items-center justify-center rounded-[14px] bg-red-tint text-red-text">
              <topic.icon className="size-6" strokeWidth={1.75} />
            </div>
            <p className="mt-4 text-[12px] font-medium text-faint">{t("Guide {n} of {total}", { n: index + 1, total: topics.length })}</p>
            <h1 className="mt-1 text-[26px] font-semibold leading-tight tracking-[-0.025em]">{say(topic.title)}</h1>
            <p className="mt-2.5 text-[15px] leading-relaxed text-graphite">
              <Rich text={say(topic.summary)} />
            </p>

            <ol className="mt-7 space-y-3.5">
              {topic.steps.map((s, i) => (
                <li key={i} className="flex gap-3.5">
                  <span className="mt-px flex size-6 shrink-0 items-center justify-center rounded-full bg-ink text-[11.5px] font-semibold text-surface">
                    {i + 1}
                  </span>
                  <p className="text-[14px] leading-relaxed text-ink/90">
                    <Rich text={say(s)} />
                  </p>
                </li>
              ))}
            </ol>

            {topic.tip && (
              <p className="mt-6 flex gap-2.5 rounded-2xl bg-fill/70 px-4 py-3 text-[13px] leading-relaxed text-graphite">
                <Sparkles className="mt-[3px] size-3.5 shrink-0 text-red-text" />
                <span>
                  <Rich text={say(topic.tip)} />
                </span>
              </p>
            )}

            {topic.actions && (
              <div className="mt-6 flex flex-wrap gap-2">
                {topic.actions.map((a, i) => (
                  <Button key={a.action} variant={i === 0 ? "primary" : "secondary"} onClick={() => onAction(a.action)}>
                    {say(a.label)}
                  </Button>
                ))}
              </div>
            )}

            {next && (
              <button
                onClick={() => setActive(next.id)}
                className="group mt-10 flex w-full items-center gap-3 rounded-2xl border border-hairline bg-surface px-4 py-3.5 text-left shadow-card transition-transform hover:-translate-y-px"
              >
                <span className="min-w-0 flex-1">
                  <span className="block text-[11.5px] text-faint">{t("Next")}</span>
                  <span className="block text-[14px] font-semibold">{say(next.title)}</span>
                </span>
                <ArrowRight className="size-4 text-faint transition-colors group-hover:text-red-text" />
              </button>
            )}
            {!next && (
              <p className="mt-10 flex items-center gap-2 text-[13px] text-graphite">
                <Clock className="size-3.5" />
                {t("That’s everything. Come back any time from Help › Tutorial (⇧⌘T).")}
              </p>
            )}
          </article>
        </div>
      </div>
    </div>
  );
}
