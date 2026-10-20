import { t } from "@/lib/i18n";
import type {
  Change,
  Component,
  Contributor,
  Job,
  Project,
  ProjectInfo,
  SiteConfig,
  Suggestion,
  Term,
  Translation,
  TranslateOptions,
  Unit,
  UnitHelp,
  User,
} from "./models";

// Every endpoint answers {status: "ok" | "error", msg, data, data2} (controllers/base.go)

export class ApiError extends Error {}

type Params = Record<string, string | number | undefined>;

function toQuery(params: Params) {
  const entries = Object.entries(params).filter(([, v]) => v !== undefined && v !== "") as [string, string | number][];
  const query = new URLSearchParams(entries.map(([k, v]) => [k, String(v)])).toString();
  return query ? `?${query}` : "";
}

async function call<T, T2 = unknown>(method: "GET" | "POST", path: string, params: Params = {}, body?: object): Promise<[T, T2]> {
  let res: Response;
  try {
    res = await fetch(`/api/${path}${toQuery(params)}`, {
      method,
      credentials: "same-origin",
      body: body === undefined ? undefined : JSON.stringify(body),
    });
  } catch {
    throw new ApiError(t("general:Cannot reach the server, please try again"));
  }

  let json: { status: string; msg: string; data: T; data2: T2 };
  try {
    json = await res.json();
  } catch {
    throw new ApiError(t("general:Server error ({status}), please try again", { status: res.status }));
  }
  if (json.status !== "ok") {
    throw new ApiError(json.msg || t("general:The request failed"));
  }
  return [json.data, json.data2];
}

async function request<T>(method: "GET" | "POST", path: string, params: Params = {}, body?: object): Promise<T> {
  const [data] = await call<T>(method, path, params, body);
  return data;
}

export const getConfig = () => request<SiteConfig>("GET", "get-config");
export const getAccount = () => request<User | null>("GET", "get-account");
export const signin = (code: string) => request<User>("POST", "signin", { code });
export const devSignin = () => request<User>("POST", "dev-signin");
export const signout = () => request<null>("POST", "signout");

export const getProjects = () => request<ProjectInfo[]>("GET", "get-projects");
export async function getProject(project: string) {
  const [info, extra] = await call<ProjectInfo, { components: Component[]; jobs: Job[] }>("GET", "get-project", { project });
  return { project: info, components: extra.components ?? [], jobs: extra.jobs ?? [] };
}
export const getProjectSecrets = (project: string) => request<{ webhookSecret: string; accessToken: string }>("GET", "get-project-secrets", { project });
export const addProject = (project: Partial<Project>) => request<string>("POST", "add-project", {}, project);
export const updateProject = (project: string, data: Project) => request<null>("POST", "update-project", { project }, data);
export const regenerateProjectTokens = (project: string) => request<null>("POST", "regenerate-project-tokens", { project });
export const deleteProject = (project: string) => request<null>("POST", "delete-project", { project });

export const addComponent = (project: string, component: Partial<Component>) => request<string>("POST", "add-component", { project }, component);
export const updateComponent = (project: string, component: string, data: Partial<Component>) =>
  request<null>("POST", "update-component", { project, component }, data);
export const deleteComponent = (project: string, component: string) => request<null>("POST", "delete-component", { project, component });

export interface UnitQuery {
  project: string;
  language: string;
  component?: string;
  file?: string;
  filter?: string;
  search?: string;
  page?: number;
  pageSize?: number;
}

export async function getUnits(query: UnitQuery) {
  const [units, total] = await call<Unit[], number>("GET", "get-units", { ...query });
  return { units: units ?? [], total: total ?? 0 };
}
export const getFiles = (project: string, component?: string) => request<string[]>("GET", "get-files", { project, component });
export const getUnitHelp = (project: string, language: string, id: number) => request<UnitHelp>("GET", "get-unit-help", { project, language, id });

/** saves the text; for someone who is not a translator of the project it becomes a suggestion (the second value) */
export async function saveTranslation(project: string, language: string, id: number, text: string, state: number) {
  const [translation, suggestion] = await call<Translation | null, Suggestion | null>("POST", "save-translation", { project, language, id }, { text, state });
  return { translation, suggestion };
}
export const acceptSuggestion = (project: string, id: number) => request<Translation>("POST", "accept-suggestion", { project, id });
export const deleteSuggestion = (project: string, id: number) => request<null>("POST", "delete-suggestion", { project, id });
export const machineTranslateUnit = (project: string, language: string, id: number) => request<string>("POST", "machine-translate-unit", { project, language, id });

export const startJob = (project: string, kind: Job["kind"], languages: string[] = [], options?: TranslateOptions) =>
  request<Job>("POST", "start-job", { project, kind, languages: languages.join(",") }, options);
export const getJobs = (project: string) => request<Job[]>("GET", "get-jobs", { project });
export const cancelJob = (project: string) => request<null>("POST", "cancel-job", { project });

export async function getChanges(project: string, filter: { language?: string; author?: string; page?: number }) {
  const [changes, extra] = await call<Change[], { total: number; contributors: Contributor[] }>("GET", "get-changes", { project, ...filter });
  return { changes: changes ?? [], total: extra?.total ?? 0, contributors: extra?.contributors ?? [] };
}

export const getTerms = (project: string, language?: string) => request<Term[]>("GET", "get-terms", { project, language });
export const addTerm = (project: string, term: Partial<Term>) => request<Term>("POST", "add-term", { project }, term);
export const updateTerm = (project: string, id: number, term: Partial<Term>) => request<null>("POST", "update-term", { project, id }, term);
export const deleteTerm = (project: string, id: number) => request<null>("POST", "delete-term", { project, id });

export const getDownloadUrl = (project: string, language?: string) => `/api/download-translations${toQuery({ project, language })}`;
export const getFileDownloadUrl = (project: string, component: string, file: string, language: string) =>
  `/api/download-file${toQuery({ project, component, file, language })}`;
