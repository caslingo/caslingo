// The shapes of what the API returns (object/*.go)

export interface User {
  owner: string;
  name: string;
  displayName: string;
  avatar: string;
  email: string;
  isAdmin: boolean;
}

export interface Language {
  code: string;
  name: string;
  nativeName: string;
}

export interface FormatInfo {
  name: string;
  displayName: string;
  extensions: string;
}

export interface SiteConfig {
  siteName?: string;
  logoUrl?: string;
  faviconUrl?: string;
  footerHtml?: string;
  casdoor?: { serverUrl: string; clientId: string; organizationName: string; appName: string };
  devSignin?: boolean;
  aiEnabled?: boolean;
  aiModel?: string;
  languages?: Language[];
  formats?: FormatInfo[];
}

export const PushModes = ["", "direct", "pull-request"] as const;
export type PushMode = (typeof PushModes)[number];

export interface Project {
  name: string;
  displayName: string;
  description: string;
  websiteUrl: string;
  createdTime: string;
  sourceLanguage: string;
  languages: string[];
  isPublic: boolean;
  openEditing: boolean;
  maintainers: string[];
  translators: string[];
  repoUrl: string;
  branch: string;
  repoToken: string;
  pushMode: PushMode;
  pushBranch: string;
  commitMessage: string;
  commitName: string;
  commitEmail: string;
  instructions: string;
  autoTranslate: boolean;
  autoPush: boolean;
  lastSyncTime: string;
  lastSyncCommit: string;
  lastPushTime: string;
  lastError: string;
}

export interface Stats {
  language: string;
  component: string;
  total: number;
  totalWords: number;
  translated: number;
  translatedWords: number;
  approved: number;
  needsEdit: number;
  checks: number;
  suggestions: number;
}

export interface ProjectInfo extends Project {
  stats: Stats[];
  canTranslate: boolean;
  canMaintain: boolean;
}

export interface Component {
  project: string;
  name: string;
  displayName: string;
  position: number;
  format: string;
  sourcePattern: string;
  translationPattern: string;
  excludePattern: string;
  uploadedSource: boolean;
  fallbackToSource: boolean;
  headingIds: boolean;
}

export const State = { Empty: 0, NeedsEdit: 10, Translated: 20, Approved: 30 } as const;

export interface Translation {
  id: number;
  unitId: number;
  language: string;
  text: string;
  state: number;
  origin: string;
  author: string;
  checks: string;
  updatedTime: string;
}

export interface Suggestion {
  id: number;
  unitId: number;
  language: string;
  text: string;
  author: string;
  createdTime: string;
}

export interface Unit {
  id: number;
  project: string;
  component: string;
  file: string;
  key: string;
  position: number;
  source: string;
  note: string;
  words: number;
  translation: Translation | null;
  suggestions: Suggestion[] | null;
}

export interface MemoryMatch {
  source: string;
  text: string;
  project: string;
  quality: number;
}

export interface Term {
  id: number;
  project: string;
  source: string;
  language: string;
  target: string;
  note: string;
}

export interface Change {
  id: number;
  project: string;
  component: string;
  unitId: number;
  language: string;
  action: string;
  author: string;
  oldText: string;
  newText: string;
  createdTime: string;
  source: string;
}

export interface UnitHelp {
  memory: MemoryMatch[];
  terms: Term[];
  others: Translation[];
  history: Change[];
}

export interface Job {
  id: string;
  project: string;
  kind: "sync" | "push" | "translate" | "import";
  languages: string[];
  author: string;
  status: "running" | "done" | "failed" | "canceled";
  progress: number;
  total: number;
  message: string;
  url: string;
  startedTime: string;
  finishedTime: string;
}

export interface TranslateOptions {
  component?: string;
  file?: string;
  includeNeedsEdit?: boolean;
  state?: number;
  memoryOnly?: boolean;
}

export interface Contributor {
  author: string;
  count: number;
}
