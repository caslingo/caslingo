// Collects the t("namespace:Text") calls of src/ into src/locales/en/data.json, sorted, so Caslingo can translate them.
import fs from "node:fs";
import path from "node:path";

const root = path.resolve(import.meta.dirname, "../src");
const pattern = /\bt\(\s*"((?:[^"\\]|\\.)+)"/g;
const res = {};

function walk(dir) {
  for (const entry of fs.readdirSync(dir, { withFileTypes: true })) {
    const p = path.join(dir, entry.name);
    if (entry.isDirectory()) {
      walk(p);
    } else if (/\.(ts|tsx)$/.test(entry.name)) {
      const code = fs.readFileSync(p, "utf8");
      for (const match of code.matchAll(pattern)) {
        const key = JSON.parse(`"${match[1]}"`);
        const index = key.indexOf(":");
        if (index <= 0) {
          console.warn(`${p}: no namespace in "${key}"`);
          continue;
        }
        const namespace = key.slice(0, index);
        const text = key.slice(index + 1);
        res[namespace] ??= {};
        res[namespace][text] = text;
      }
    }
  }
}

walk(root);

const sorted = {};
for (const namespace of Object.keys(res).sort()) {
  sorted[namespace] = {};
  for (const text of Object.keys(res[namespace]).sort()) {
    sorted[namespace][text] = res[namespace][text];
  }
}

const target = path.join(root, "locales/en/data.json");
fs.writeFileSync(target, JSON.stringify(sorted, null, 2) + "\n");
const count = Object.values(sorted).reduce((n, texts) => n + Object.keys(texts).length, 0);
console.log(`${target}: ${count} texts`);
