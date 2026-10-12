// Checks that every English string passed to t(), tn() or msg() has a French
// translation with the same {placeholders}, and lists French entries nobody
// uses any more. Run with `npm run check:i18n`.
import { readdirSync, readFileSync, statSync } from "node:fs";
import { join } from "node:path";

const src = new URL("../src/", import.meta.url).pathname;
const str = String.raw`"((?:[^"\\]|\\.)*)"`;
const patterns = [
  new RegExp(String.raw`\b(?:t|msg)\(\s*` + str, "g"),
  new RegExp(String.raw`\btn\(\s*[^,]+,\s*` + str + String.raw`\s*,\s*` + str, "g"),
];

function files(dir) {
  return readdirSync(dir).flatMap((f) => {
    const p = join(dir, f);
    if (statSync(p).isDirectory()) return files(p);
    return /\.(ts|tsx)$/.test(f) && !f.startsWith("i18n") ? [p] : [];
  });
}

const used = new Map();
for (const file of files(src)) {
  const text = readFileSync(file, "utf8");
  for (const re of patterns)
    for (const m of text.matchAll(re))
      for (const k of m.slice(1)) if (k !== undefined) used.set(JSON.parse(`"${k}"`), file.slice(src.length));
}

const frText = readFileSync(join(src, "i18n.fr.ts"), "utf8");
const fr = new Function(frText.replace(/^export const fr[^=]*=/m, "return ").replace(/;\s*$/, ""))();
const holes = (s) => (s.match(/\{\w+\}/g) || []).sort().join(" ");

let problems = 0;
for (const [k, where] of used) {
  if (!(k in fr)) {
    console.log(`missing: ${JSON.stringify(k)}  (${where})`);
    problems++;
  } else if (holes(k) !== holes(fr[k])) {
    console.log(`placeholders differ: ${JSON.stringify(k)} → ${JSON.stringify(fr[k])}`);
    problems++;
  }
}
const unused = Object.keys(fr).filter((k) => !used.has(k));
if (unused.length) console.log(`unused French entries (fine to remove):\n  ${unused.map((k) => JSON.stringify(k)).join("\n  ")}`);
console.log(`${used.size} strings, ${problems} problem(s).`);
process.exit(problems ? 1 : 0);
