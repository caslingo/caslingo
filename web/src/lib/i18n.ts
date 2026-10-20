import en from "@/locales/en/data.json";
import zh from "@/locales/zh/data.json";

// The texts of the interface are written in English in the code, as t('namespace:Text'); `yarn i18n` collects them
// into locales/en/data.json, and the other languages are translated in Caslingo itself.

type Resource = Record<string, Record<string, string>>;

const resources: Record<string, Resource> = { en, zh };

export const uiLanguages = [
  { code: "en", name: "English" },
  { code: "zh", name: "简体中文" },
];

const storageKey = "caslingo:language";

function detectLanguage() {
  try {
    const saved = localStorage.getItem(storageKey);
    if (saved && resources[saved]) return saved;
  } catch {
    // storage blocked: use the browser's language
  }
  const browser = (navigator.language || "en").toLowerCase();
  return browser.startsWith("zh") ? "zh" : "en";
}

let current = detectLanguage();
document.documentElement.lang = current;

export function getUiLanguage() {
  return current;
}

export function setUiLanguage(language: string) {
  try {
    localStorage.setItem(storageKey, language);
  } catch {
    // the choice lasts until the page reloads
  }
  current = language;
  location.reload();
}

/** t('general:Save') or t('project:{count} strings', { count: 3 }) */
export function t(key: string, vars?: Record<string, string | number>) {
  const index = key.indexOf(":");
  const namespace = key.slice(0, index);
  const text = key.slice(index + 1);
  let res = resources[current]?.[namespace]?.[text] || text;
  if (vars) {
    for (const [name, value] of Object.entries(vars)) {
      res = res.replaceAll(`{${name}}`, String(value));
    }
  }
  return res;
}
