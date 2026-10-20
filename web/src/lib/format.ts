import { useEffect } from "react";
import type { Language, Stats } from "@/backend/models";
import { getUiLanguage, t } from "@/lib/i18n";

export function useDocumentTitle(title: string, siteName = "Caslingo") {
  useEffect(() => {
    document.title = title ? `${title} | ${siteName}` : siteName;
  }, [title, siteName]);
}

export function formatNumber(n: number) {
  return n.toLocaleString(getUiLanguage());
}

export function percent(part: number, total: number) {
  if (total <= 0) return 0;
  return Math.floor((part / total) * 1000) / 10;
}

/** "just now", "5 minutes ago", "3 hours ago", or the date */
export function formatRelative(time: string) {
  if (!time) return "";
  const date = new Date(time);
  if (Number.isNaN(date.getTime())) return time;
  const minutes = Math.floor((Date.now() - date.getTime()) / 60000);
  if (minutes < 1) return t("general:just now");
  if (minutes < 60) return t("general:{n} min ago", { n: minutes });
  const hours = Math.floor(minutes / 60);
  if (hours < 24) return t("general:{n} h ago", { n: hours });
  return date.toLocaleDateString(getUiLanguage(), { year: "numeric", month: "short", day: "numeric" });
}

export function formatTime(time: string) {
  if (!time) return "";
  const date = new Date(time);
  if (Number.isNaN(date.getTime())) return time;
  return date.toLocaleString(getUiLanguage(), { year: "numeric", month: "short", day: "numeric", hour: "2-digit", minute: "2-digit" });
}

/** "简体中文 (zh)" from the server's list of languages */
export function languageName(languages: Language[] | undefined, code: string) {
  const language = languages?.find(l => l.code.toLowerCase() === code.toLowerCase());
  return language ? language.nativeName : code;
}

export function languageEnglishName(languages: Language[] | undefined, code: string) {
  const language = languages?.find(l => l.code.toLowerCase() === code.toLowerCase());
  return language ? language.name : "";
}

export function findStats(stats: Stats[] | undefined, language: string, component = "") {
  return stats?.find(s => s.language === language && s.component === component);
}

/** a unit key: JSON keys are paths like ["account","Logout"], shown as account › Logout */
export function formatKey(key: string) {
  if (key.startsWith("[")) {
    try {
      return (JSON.parse(key) as string[]).join(" › ");
    } catch {
      return key;
    }
  }
  return "";
}
