import { AlertTriangle, Check, CheckCheck, Copy, Download, Loader2, Search, Sparkles, Undo2 } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import * as Backend from "@/backend/api";
import type { Component, ProjectInfo, Unit } from "@/backend/models";
import { State } from "@/backend/models";
import { Page } from "@/components/app/layout";
import { Pager } from "@/components/app/pager";
import { PercentLabel, TranslationBar } from "@/components/app/progress";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { NativeSelect } from "@/components/ui/native-select";
import { Textarea } from "@/components/ui/textarea";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { useLoad } from "@/hooks/use-load";
import { findStats, formatKey, languageName, useDocumentTitle } from "@/lib/format";
import { t } from "@/lib/i18n";
import { getSigninUrl, rememberReturnPath, useSite } from "@/lib/site";
import { cn } from "@/lib/utils";
import { UnitHelpPanel } from "./unit-help";
import { PageError, PageLoading } from "./static-pages";

const pageSize = 30;

function filterLabel(filter: string) {
  switch (filter) {
    case "untranslated":
      return t("state:Untranslated");
    case "needs-edit":
      return t("state:Needs editing");
    case "translated":
      return t("state:Translated");
    case "unapproved":
      return t("editor:Translated, not approved");
    case "approved":
      return t("state:Approved");
    case "checks":
      return t("editor:Failing checks");
    case "suggestions":
      return t("editor:With suggestions");
  }
  return t("editor:All strings");
}

const filters = ["all", "untranslated", "needs-edit", "translated", "unapproved", "approved", "checks", "suggestions"];

/** the notes the Markdown format gives its texts, in the interface language */
function noteLabel(note: string) {
  if (note === "Heading") return t("editor:Heading");
  if (note === "Table row: keep the | separators") return t("editor:Table row: keep the | separators");
  if (note === "Title of a note box") return t("editor:Title of a note box");
  if (note.startsWith("Front matter: ")) return t("editor:Front matter: {key}", { key: note.slice("Front matter: ".length) });
  return note;
}

export function checkLabel(check: string) {
  switch (check) {
    case "placeholders":
      return t("check:Placeholders differ from the source");
    case "code":
      return t("check:Inline code differs from the source");
    case "tags":
      return t("check:HTML tags differ from the source");
    case "links":
      return t("check:Link targets differ from the source");
    case "whitespace":
      return t("check:Spaces at the start or end differ from the source");
  }
  return check;
}

export function EditorPage() {
  const { name = "", language = "" } = useParams();
  const { account, config } = useSite();
  const project = useLoad(() => Backend.getProject(name), [name, account?.name]);
  useDocumentTitle(`${project.data?.project.displayName ?? ""} ${languageName(config.languages, language)}`, config.siteName);

  if (project.error) return <PageError message={project.error} />;
  if (!project.data) return <PageLoading />;
  if (!project.data.project.languages.includes(language)) return <PageError message={t("editor:The project has no language {language}", { language })} />;
  return <Editor project={project.data.project} components={project.data.components} language={language} onSaved={project.reload} />;
}

function Editor({ project, components, language, onSaved }: { project: ProjectInfo; components: Component[]; language: string; onSaved: () => void }) {
  const { account, config } = useSite();
  const [params, setParams] = useSearchParams();
  const filter = params.get("filter") ?? "all";
  const component = params.get("component") ?? "";
  const file = params.get("file") ?? "";
  const search = params.get("search") ?? "";
  const page = Number(params.get("page") ?? 1) || 1;
  const [searchInput, setSearchInput] = useState(search);
  const [activeId, setActiveId] = useState<number | null>(null);
  const [translatingFile, setTranslatingFile] = useState(false);

  const setParam = (patch: Record<string, string>) => {
    const next = new URLSearchParams(params);
    for (const [k, v] of Object.entries(patch)) {
      if (v && !(k === "filter" && v === "all") && !(k === "page" && v === "1")) next.set(k, v);
      else next.delete(k);
    }
    if (!("page" in patch)) next.delete("page");
    setParams(next, { replace: true });
  };

  const files = useLoad(() => Backend.getFiles(project.name, component), [project.name, component]);
  const { data, error, setData, reload } = useLoad(
    () => Backend.getUnits({ project: project.name, language, component, file, filter, search, page, pageSize }),
    [project.name, language, component, file, filter, search, page],
  );

  useEffect(() => {
    if (data?.units.length) setActiveId(id => (id && data.units.some(u => u.id === id) ? id : data.units[0].id));
  }, [data]);

  const units = data?.units ?? [];
  const active = units.find(u => u.id === activeId) ?? null;
  const stats = findStats(project.stats, language, component);
  const canEdit = !!account;

  const translateFile = () => {
    setTranslatingFile(true);
    Backend.startJob(project.name, "translate", [language], { component, file, state: State.Translated })
      .then(async () => {
        // the job runs in the background; wait for it, then show the results
        for (;;) {
          await new Promise(resolve => setTimeout(resolve, 2000));
          const jobs = await Backend.getJobs(project.name);
          if (!jobs.some(job => job.status === "running")) {
            const last = jobs[0];
            if (last?.status === "failed") toast.error(last.message);
            else if (last) toast.success(last.message);
            break;
          }
        }
        reload();
        onSaved();
      })
      .catch((e: Error) => toast.error(e.message))
      .finally(() => setTranslatingFile(false));
  };

  const updateUnit = (unit: Unit) => {
    setData(d => (d ? { ...d, units: d.units.map(u => (u.id === unit.id ? unit : u)) } : d));
  };
  const goNext = (id: number) => {
    const index = units.findIndex(u => u.id === id);
    if (index >= 0 && index + 1 < units.length) {
      setActiveId(units[index + 1].id);
      document.getElementById(`unit-${units[index + 1].id}`)?.querySelector("textarea")?.focus();
    } else if (data && page * pageSize < data.total) {
      setParam({ page: String(page + 1) });
      window.scrollTo(0, 0);
    }
  };

  return (
    <Page className="max-w-[96rem]">
      <div className="mb-4 flex flex-wrap items-center gap-x-4 gap-y-2">
        <div className="text-sm text-muted-foreground">
          <Link to="/" className="hover:text-foreground">
            {t("general:Projects")}
          </Link>
          {" / "}
          <Link to={`/projects/${project.name}`} className="hover:text-foreground">
            {project.displayName}
          </Link>
          {" / "}
        </div>
        <h1 className="text-xl font-semibold">{languageName(config.languages, language)}</h1>
        <div className="flex min-w-48 flex-1 items-center gap-3">
          <TranslationBar stats={stats} className="max-w-64" />
          <span className="text-sm text-muted-foreground">
            <PercentLabel stats={stats} />
          </span>
        </div>
        {!account && (
          <Button size="sm" variant="outline" asChild>
            <a href={getSigninUrl(config)} onClick={() => rememberReturnPath()}>
              {t("editor:Sign in to translate")}
            </a>
          </Button>
        )}
        {account && !project.canTranslate && <Badge variant="secondary">{t("editor:Your translations are saved as suggestions")}</Badge>}
      </div>

      <div className="mb-4 flex flex-wrap gap-2">
        <form
          className="relative min-w-56 flex-1"
          onSubmit={e => {
            e.preventDefault();
            setParam({ search: searchInput });
          }}
        >
          <Search className="pointer-events-none absolute top-1/2 left-2.5 size-4 -translate-y-1/2 text-muted-foreground" />
          <Input className="pl-8" value={searchInput} onChange={e => setSearchInput(e.target.value)} placeholder={t("editor:Search the source, key or translation")} />
        </form>
        <NativeSelect value={filter} onChange={e => setParam({ filter: e.target.value })} className="w-52">
          {filters.map(f => (
            <option key={f} value={f}>
              {filterLabel(f)}
            </option>
          ))}
        </NativeSelect>
        {components.length > 1 && (
          <NativeSelect value={component} onChange={e => setParam({ component: e.target.value, file: "" })} className="w-44">
            <option value="">{t("editor:All components")}</option>
            {components.map(c => (
              <option key={c.name} value={c.name}>
                {c.displayName}
              </option>
            ))}
          </NativeSelect>
        )}
        {(files.data?.length ?? 0) > 1 && (
          <NativeSelect value={file} onChange={e => setParam({ file: e.target.value })} className="w-72">
            <option value="">{t("editor:All files")}</option>
            {files.data?.map(f => (
              <option key={f} value={f}>
                {f}
              </option>
            ))}
          </NativeSelect>
        )}
        {file && component && project.canMaintain && config.aiEnabled && (
          <Button variant="outline" onClick={translateFile} disabled={translatingFile}>
            {translatingFile ? <Loader2 className="animate-spin" /> : <Sparkles />}
            {t("editor:Machine translate the file")}
          </Button>
        )}
        {file && component && (
          <Button variant="outline" asChild>
            <a href={Backend.getFileDownloadUrl(project.name, component, file, language)}>
              <Download />
              {t("editor:Download file")}
            </a>
          </Button>
        )}
      </div>

      {error ? (
        <PageError message={error} />
      ) : !data ? (
        <div className="flex justify-center py-20 text-muted-foreground">
          <Loader2 className="size-6 animate-spin" />
        </div>
      ) : units.length === 0 ? (
        <Card className="py-16 text-center text-sm text-muted-foreground">{filter === "untranslated" ? t("editor:Everything here is translated.") : t("editor:No strings match.")}</Card>
      ) : (
        <div className="grid items-start gap-4 lg:grid-cols-[minmax(0,1fr)_22rem]">
          <div className="space-y-3">
            {units.map(unit => (
              <UnitRow
                key={unit.id}
                unit={unit}
                project={project}
                language={language}
                canEdit={canEdit}
                active={unit.id === activeId}
                showFile={!file}
                onFocus={() => setActiveId(unit.id)}
                onSaved={next => {
                  updateUnit(next);
                  onSaved();
                }}
                onNext={() => goNext(unit.id)}
              />
            ))}
            <Pager page={page} pageSize={pageSize} total={data.total} onPage={p => (setParam({ page: String(p) }), window.scrollTo(0, 0))} />
          </div>
          <div className="sticky top-18 hidden lg:block">
            {active && <UnitHelpPanel key={`${active.id}-${active.translation?.updatedTime ?? ""}`} project={project} language={language} unit={active} onUse={text => fillDraft(active.id, text)} onChanged={updateUnit} />}
          </div>
        </div>
      )}
    </Page>
  );
}

// the help panel fills the textarea of a unit through this
const drafts = new Map<number, (text: string) => void>();
function fillDraft(id: number, text: string) {
  drafts.get(id)?.(text);
}

function stateOf(unit: Unit) {
  return unit.translation?.text ? unit.translation.state : State.Empty;
}

function StateDot({ state }: { state: number }) {
  const label =
    state === State.Approved ? t("state:Approved") : state === State.Translated ? t("state:Translated") : state === State.NeedsEdit ? t("state:Needs editing") : t("state:Untranslated");
  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <span
          className={cn(
            "inline-block size-2.5 shrink-0 rounded-full",
            state === State.Approved ? "bg-state-approved" : state === State.Translated ? "bg-state-translated" : state === State.NeedsEdit ? "bg-state-needs-edit" : "bg-state-empty",
          )}
        />
      </TooltipTrigger>
      <TooltipContent>{label}</TooltipContent>
    </Tooltip>
  );
}

function UnitRow({
  unit,
  project,
  language,
  canEdit,
  active,
  showFile,
  onFocus,
  onSaved,
  onNext,
}: {
  unit: Unit;
  project: ProjectInfo;
  language: string;
  canEdit: boolean;
  active: boolean;
  showFile: boolean;
  onFocus: () => void;
  onSaved: (unit: Unit) => void;
  onNext: () => void;
}) {
  const saved = unit.translation?.text ?? "";
  const [text, setText] = useState(saved);
  const [busy, setBusy] = useState<"" | "save" | "machine">("");
  const textareaRef = useRef<HTMLTextAreaElement>(null);
  const state = stateOf(unit);
  const dirty = text !== saved;
  const checks = useMemo(() => (unit.translation?.checks ? unit.translation.checks.split(",") : []), [unit.translation?.checks]);
  const key = formatKey(unit.key);
  const isSuggesting = !project.canTranslate;

  useEffect(() => setText(saved), [saved]);
  useEffect(() => {
    drafts.set(unit.id, value => {
      setText(value);
      textareaRef.current?.focus();
    });
    return () => {
      drafts.delete(unit.id);
    };
  }, [unit.id]);

  const save = (nextState: number, goOn: boolean) => {
    setBusy("save");
    Backend.saveTranslation(project.name, language, unit.id, text, nextState)
      .then(({ translation, suggestion }) => {
        if (suggestion) {
          toast.success(t("editor:Suggestion saved; a translator will review it"));
          onSaved({ ...unit, suggestions: [...(unit.suggestions ?? []), suggestion] });
          setText(saved);
        } else {
          onSaved({ ...unit, translation });
        }
        if (goOn) onNext();
      })
      .catch((e: Error) => toast.error(e.message))
      .finally(() => setBusy(""));
  };

  const machine = () => {
    setBusy("machine");
    Backend.machineTranslateUnit(project.name, language, unit.id)
      .then(result => {
        setText(result);
        textareaRef.current?.focus();
      })
      .catch((e: Error) => toast.error(e.message))
      .finally(() => setBusy(""));
  };

  return (
    <Card
      id={`unit-${unit.id}`}
      onClick={onFocus}
      className={cn("gap-0 overflow-hidden py-0 transition-shadow", active && "ring-2 ring-primary/30")}
    >
      <div className="grid md:grid-cols-2">
        <div className="space-y-2 border-b p-4 md:border-r md:border-b-0">
          <div className="flex items-center gap-2 text-xs text-muted-foreground">
            <StateDot state={state} />
            <span className="truncate" title={unit.file}>
              {key || (showFile ? unit.file : "")}
            </span>
            {unit.suggestions?.length ? (
              <Badge variant="secondary" className="ml-auto">
                {t("editor:{count} suggestions", { count: unit.suggestions.length })}
              </Badge>
            ) : null}
          </div>
          <div className="text-unit text-[15px] leading-relaxed">{unit.source}</div>
          {unit.note && <div className="text-xs text-muted-foreground">{noteLabel(unit.note)}</div>}
        </div>
        <div className="space-y-2 p-4">
          <Textarea
            ref={textareaRef}
            value={text}
            disabled={!canEdit}
            onChange={e => setText(e.target.value)}
            onFocus={onFocus}
            placeholder={canEdit ? t("editor:Translation") : t("editor:Not translated yet")}
            className="text-unit resize-none text-[15px] leading-relaxed"
            onKeyDown={e => {
              if (e.key === "Enter" && (e.ctrlKey || e.metaKey)) {
                e.preventDefault();
                if (text) save(e.shiftKey ? State.NeedsEdit : State.Translated, true);
              }
            }}
          />
          {checks.length > 0 && !dirty && (
            <div className="flex flex-wrap gap-1.5">
              {checks.map(check => (
                <Badge key={check} variant="outline" className="gap-1 border-destructive/40 text-destructive">
                  <AlertTriangle className="size-3" />
                  {checkLabel(check)}
                </Badge>
              ))}
            </div>
          )}
          {canEdit && (
            <div className="flex flex-wrap items-center gap-1.5">
              <Button size="sm" disabled={!text || busy !== "" || (!dirty && state >= State.Translated)} onClick={() => save(State.Translated, false)} title="Ctrl+Enter">
                {busy === "save" ? <Loader2 className="animate-spin" /> : <Check />}
                {isSuggesting ? t("editor:Suggest") : t("general:Save")}
              </Button>
              {!isSuggesting && (
                <Button variant="outline" size="sm" disabled={!text || busy !== "" || (!dirty && state === State.NeedsEdit)} onClick={() => save(State.NeedsEdit, false)} title="Ctrl+Shift+Enter">
                  {t("editor:Needs editing")}
                </Button>
              )}
              {project.canMaintain && state !== State.Approved && (
                <Button variant="outline" size="sm" disabled={!text || busy !== ""} onClick={() => save(State.Approved, false)}>
                  <CheckCheck />
                  {t("editor:Approve")}
                </Button>
              )}
              <div className="ml-auto flex items-center gap-0.5">
                {dirty && (
                  <Button variant="ghost" size="icon-sm" onClick={() => setText(saved)} title={t("editor:Undo changes")}>
                    <Undo2 />
                  </Button>
                )}
                <Button variant="ghost" size="icon-sm" onClick={() => setText(unit.source)} title={t("editor:Copy the source")}>
                  <Copy />
                </Button>
                {project.canTranslate && (
                  <Button variant="ghost" size="icon-sm" disabled={busy !== ""} onClick={machine} title={t("editor:Machine translate this string")}>
                    {busy === "machine" ? <Loader2 className="animate-spin" /> : <Sparkles />}
                  </Button>
                )}
              </div>
            </div>
          )}
        </div>
      </div>
    </Card>
  );
}
