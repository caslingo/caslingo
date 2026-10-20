import { ArrowRight, CircleAlert, Download, ExternalLink, GitPullRequestArrow, Loader2, RefreshCw, Settings, Sparkles, Upload, X } from "lucide-react";
import { useEffect, useState } from "react";
import { Link, useParams, useSearchParams } from "react-router-dom";
import { toast } from "sonner";
import * as Backend from "@/backend/api";
import type { Component, Job, ProjectInfo, TranslateOptions } from "@/backend/models";
import { State } from "@/backend/models";
import { Page, PageHeader } from "@/components/app/layout";
import { PercentLabel, TranslationBar } from "@/components/app/progress";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Label } from "@/components/ui/label";
import { NativeSelect } from "@/components/ui/native-select";
import { Progress } from "@/components/ui/progress";
import { Switch } from "@/components/ui/switch";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs";
import { useLoad } from "@/hooks/use-load";
import { findStats, formatNumber, formatRelative, languageEnglishName, languageName, useDocumentTitle } from "@/lib/format";
import { t } from "@/lib/i18n";
import { useSite } from "@/lib/site";
import { GlossaryTab } from "./glossary-tab";
import { HistoryTab } from "./history-tab";
import { IntegrationTab } from "./integration-tab";
import { PageError, PageLoading } from "./static-pages";

const tabs = ["languages", "components", "glossary", "history", "integration"] as const;

export function ProjectPage() {
  const { name = "" } = useParams();
  const { account, config } = useSite();
  const { data, error, reload, setData } = useLoad(() => Backend.getProject(name), [name, account?.name]);
  const [params, setParams] = useSearchParams();
  const tab = tabs.find(t => t === params.get("tab")) ?? "languages";
  const [translating, setTranslating] = useState<string[] | null>(null);
  useDocumentTitle(data?.project.displayName ?? "", config.siteName);

  const running = data?.jobs.find(job => job.status === "running");
  // follow the running job, then load the new numbers
  useEffect(() => {
    if (!running) return;
    const timer = setInterval(() => {
      Backend.getJobs(name)
        .then(jobs => {
          if (jobs.some(job => job.status === "running")) setData(d => (d ? { ...d, jobs } : d));
          else reload();
        })
        .catch(() => {});
    }, 2000);
    return () => clearInterval(timer);
  }, [running?.id, name, reload, setData]);

  if (error) return <PageError message={error} />;
  if (!data) return <PageLoading />;
  const { project, components, jobs } = data;

  const start = (kind: Job["kind"], languages: string[] = [], options?: TranslateOptions) =>
    Backend.startJob(name, kind, languages, options)
      .then(() => reload())
      .catch((e: Error) => toast.error(e.message));

  return (
    <Page>
      <PageHeader
        title={project.displayName}
        description={
          <div className="flex flex-wrap items-center gap-x-4 gap-y-1">
            {project.description && <span>{project.description}</span>}
            {project.repoUrl && (
              <a href={project.repoUrl} target="_blank" rel="noopener" className="inline-flex items-center gap-1 hover:text-foreground">
                {project.repoUrl.replace(/^https?:\/\//, "")} ({project.branch})
                <ExternalLink className="size-3" />
              </a>
            )}
            {project.websiteUrl && (
              <a href={project.websiteUrl} target="_blank" rel="noopener" className="inline-flex items-center gap-1 hover:text-foreground">
                {project.websiteUrl.replace(/^https?:\/\//, "")}
                <ExternalLink className="size-3" />
              </a>
            )}
          </div>
        }
        actions={
          project.canMaintain && (
            <>
              <Button variant="outline" size="sm" disabled={!!running} onClick={() => start("sync")}>
                <RefreshCw />
                {t("project:Pull")}
              </Button>
              <Button variant="outline" size="sm" disabled={!!running || !project.pushMode} onClick={() => start("push")} title={!project.pushMode ? t("project:Pushing is off in the settings") : undefined}>
                <Upload />
                {t("project:Push")}
              </Button>
              <Button variant="outline" size="sm" disabled={!!running || !config.aiEnabled} onClick={() => setTranslating(project.languages)}>
                <Sparkles />
                {t("project:Machine translate")}
              </Button>
              <Button variant="outline" size="sm" asChild>
                <a href={Backend.getDownloadUrl(name)}>
                  <Download />
                  {t("project:Download")}
                </a>
              </Button>
              <Button size="sm" asChild>
                <Link to={`/projects/${name}/settings`}>
                  <Settings />
                  {t("project:Settings")}
                </Link>
              </Button>
            </>
          )
        }
      >
        <Link to="/" className="mb-1 inline-block text-sm text-muted-foreground hover:text-foreground">
          {t("general:Projects")} /
        </Link>
      </PageHeader>

      <JobPanel project={project} jobs={jobs} />

      {components.length === 0 && (
        <Card className="mb-6 flex-row items-center gap-3 border-dashed p-4 text-sm">
          <CircleAlert className="size-5 shrink-0 text-state-needs-edit" />
          <span className="flex-1">{t("project:This project has no components yet: tell Caslingo which files of the repository to translate.")}</span>
          {project.canMaintain && (
            <Button size="sm" asChild>
              <Link to={`/projects/${name}/settings`}>{t("project:Add a component")}</Link>
            </Button>
          )}
        </Card>
      )}

      <Tabs value={tab} onValueChange={value => setParams(value === "languages" ? {} : { tab: value }, { replace: true })}>
        <TabsList className="mb-4">
          <TabsTrigger value="languages">{t("project:Languages")}</TabsTrigger>
          <TabsTrigger value="components">{t("project:Components")}</TabsTrigger>
          <TabsTrigger value="glossary">{t("project:Glossary")}</TabsTrigger>
          <TabsTrigger value="history">{t("project:History")}</TabsTrigger>
          {project.canMaintain && <TabsTrigger value="integration">{t("project:Integration")}</TabsTrigger>}
        </TabsList>
        <TabsContent value="languages">
          <LanguagesTab project={project} onTranslate={language => setTranslating([language])} busy={!!running} />
        </TabsContent>
        <TabsContent value="components">
          <ComponentsTab project={project} components={components} />
        </TabsContent>
        <TabsContent value="glossary">
          <GlossaryTab project={project} />
        </TabsContent>
        <TabsContent value="history">
          <HistoryTab project={project} />
        </TabsContent>
        {project.canMaintain && (
          <TabsContent value="integration">
            <IntegrationTab project={project} />
          </TabsContent>
        )}
      </Tabs>

      {translating && (
        <TranslateDialog
          project={project}
          components={components}
          languages={translating}
          onClose={() => setTranslating(null)}
          onStart={(languages, options) => {
            setTranslating(null);
            start("translate", languages, options);
          }}
        />
      )}
    </Page>
  );
}

function jobLabel(kind: Job["kind"]) {
  if (kind === "sync") return t("project:Pulling the repository");
  if (kind === "push") return t("project:Pushing the translations");
  if (kind === "import") return t("project:Importing translations");
  return t("project:Machine translating");
}

function jobStatus(status: Job["status"]) {
  if (status === "done") return t("job:Done");
  if (status === "failed") return t("job:Failed");
  if (status === "canceled") return t("job:Canceled");
  return t("job:Running");
}

function JobPanel({ project, jobs }: { project: ProjectInfo; jobs: Job[] }) {
  const running = jobs.find(job => job.status === "running");
  const last = jobs.find(job => job.status !== "running");

  if (running) {
    const value = running.total > 0 ? (running.progress / running.total) * 100 : undefined;
    return (
      <Card className="mb-6 gap-3 p-4">
        <div className="flex items-center gap-3 text-sm">
          <Loader2 className="size-4 animate-spin text-primary" />
          <span className="font-medium">{jobLabel(running.kind)}</span>
          <span className="min-w-0 flex-1 truncate text-muted-foreground">{running.message}</span>
          {project.canMaintain && (
            <Button
              variant="ghost"
              size="sm"
              onClick={() => Backend.cancelJob(project.name).catch((e: Error) => toast.error(e.message))}
            >
              <X />
              {t("general:Cancel")}
            </Button>
          )}
        </div>
        {value !== undefined ? <Progress value={value} /> : <Progress value={100} className="animate-pulse opacity-40" />}
      </Card>
    );
  }

  if (project.lastError) {
    return (
      <Card className="mb-6 flex-row items-start gap-3 border-destructive/40 bg-destructive/5 p-4 text-sm">
        <CircleAlert className="mt-0.5 size-4 shrink-0 text-destructive" />
        <div className="min-w-0 flex-1">
          <div className="font-medium text-destructive">{t("project:The last pull or push failed")}</div>
          <div className="mt-1 font-mono text-xs break-all whitespace-pre-wrap text-muted-foreground">{project.lastError}</div>
        </div>
      </Card>
    );
  }

  if (last && Date.now() - new Date(last.finishedTime).getTime() < 10 * 60000) {
    return (
      <Card className="mb-6 flex-row items-center gap-3 p-4 text-sm">
        <Badge variant={last.status === "done" ? "secondary" : "destructive"}>{jobStatus(last.status)}</Badge>
        <span className="font-medium">{jobLabel(last.kind)}</span>
        <span className="min-w-0 flex-1 truncate text-muted-foreground">{last.message}</span>
        {last.url && (
          <a href={last.url} target="_blank" rel="noopener" className="inline-flex items-center gap-1 text-primary hover:underline">
            <GitPullRequestArrow className="size-4" />
            {t("project:Pull request")}
          </a>
        )}
        <span className="text-xs text-muted-foreground">{formatRelative(last.finishedTime)}</span>
      </Card>
    );
  }

  return (
    <div className="mb-4 text-xs text-muted-foreground">
      {project.lastSyncTime
        ? project.lastSyncCommit
          ? t("project:Pulled {time}, commit {commit}", { time: formatRelative(project.lastSyncTime), commit: project.lastSyncCommit.slice(0, 7) })
          : t("project:Read {time}", { time: formatRelative(project.lastSyncTime) })
        : project.repoUrl
          ? t("project:Not pulled yet: press Pull to read the repository")
          : ""}
      {project.lastPushTime && `; ${t("project:pushed {time}", { time: formatRelative(project.lastPushTime) })}`}
    </div>
  );
}

function LanguagesTab({ project, onTranslate, busy }: { project: ProjectInfo; onTranslate: (language: string) => void; busy: boolean }) {
  const { config } = useSite();
  const source = findStats(project.stats, "");
  const sorted = [...project.languages].sort((a, b) => (findStats(project.stats, b)?.translated ?? 0) - (findStats(project.stats, a)?.translated ?? 0));

  return (
    <Card className="gap-0 py-0">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead>{t("project:Language")}</TableHead>
            <TableHead className="w-64">{t("project:Progress")}</TableHead>
            <TableHead className="text-right">{t("state:Untranslated")}</TableHead>
            <TableHead className="text-right">{t("state:Needs editing")}</TableHead>
            <TableHead className="text-right">{t("project:Failing checks")}</TableHead>
            <TableHead className="text-right">{t("project:Suggestions")}</TableHead>
            <TableHead />
          </TableRow>
        </TableHeader>
        <TableBody>
          <TableRow className="hover:bg-transparent">
            <TableCell className="text-muted-foreground">
              {languageName(config.languages, project.sourceLanguage)} <span className="text-xs">({t("project:source")})</span>
            </TableCell>
            <TableCell colSpan={6} className="text-xs text-muted-foreground">
              {t("project:{strings} strings, {words} words", { strings: formatNumber(source?.total ?? 0), words: formatNumber(source?.totalWords ?? 0) })}
            </TableCell>
          </TableRow>
          {sorted.map(language => {
            const stats = findStats(project.stats, language);
            const untranslated = (stats?.total ?? 0) - (stats?.translated ?? 0) - (stats?.needsEdit ?? 0);
            const editor = `/projects/${project.name}/${language}`;
            return (
              <TableRow key={language}>
                <TableCell>
                  <Link to={editor} className="font-medium hover:text-primary">
                    {languageName(config.languages, language)}
                  </Link>
                  <div className="text-xs text-muted-foreground">
                    {language}
                    {languageEnglishName(config.languages, language) && ` (${languageEnglishName(config.languages, language)})`}
                  </div>
                </TableCell>
                <TableCell>
                  <div className="flex items-center gap-3">
                    <TranslationBar stats={stats} className="flex-1" />
                    <span className="w-12 text-right text-xs">
                      <PercentLabel stats={stats} />
                    </span>
                  </div>
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {untranslated > 0 ? <Link to={`${editor}?filter=untranslated`} className="hover:text-primary">{formatNumber(untranslated)}</Link> : "0"}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {stats?.needsEdit ? <Link to={`${editor}?filter=needs-edit`} className="text-state-needs-edit hover:underline">{formatNumber(stats.needsEdit)}</Link> : "0"}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {stats?.checks ? <Link to={`${editor}?filter=checks`} className="text-destructive hover:underline">{formatNumber(stats.checks)}</Link> : "0"}
                </TableCell>
                <TableCell className="text-right tabular-nums">
                  {stats?.suggestions ? <Link to={`${editor}?filter=suggestions`} className="text-primary hover:underline">{formatNumber(stats.suggestions)}</Link> : "0"}
                </TableCell>
                <TableCell className="text-right whitespace-nowrap">
                  {project.canMaintain && config.aiEnabled && untranslated > 0 && (
                    <Button variant="ghost" size="sm" disabled={busy} onClick={() => onTranslate(language)} title={t("project:Machine translate")}>
                      <Sparkles />
                    </Button>
                  )}
                  <Button variant="ghost" size="sm" asChild>
                    <Link to={editor}>
                      {project.canTranslate ? t("project:Translate") : t("project:View")}
                      <ArrowRight />
                    </Link>
                  </Button>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
      {project.languages.length === 0 && <div className="p-8 text-center text-sm text-muted-foreground">{t("project:No languages yet: add them in the settings.")}</div>}
    </Card>
  );
}

function ComponentsTab({ project, components }: { project: ProjectInfo; components: Component[] }) {
  const { config } = useSite();
  return (
    <Card className="gap-0 py-0">
      <Table>
        <TableHeader>
          <TableRow className="hover:bg-transparent">
            <TableHead>{t("project:Component")}</TableHead>
            <TableHead>{t("project:Format")}</TableHead>
            <TableHead>{t("project:Files")}</TableHead>
            <TableHead className="text-right">{t("project:Strings")}</TableHead>
            <TableHead className="w-56">{t("project:Translated, all languages")}</TableHead>
          </TableRow>
        </TableHeader>
        <TableBody>
          {components.map(component => {
            const source = findStats(project.stats, "", component.name);
            const perLanguage = project.languages.map(language => findStats(project.stats, language, component.name));
            const total = (source?.total ?? 0) * project.languages.length;
            const sum = (key: "translated" | "approved" | "needsEdit") => perLanguage.reduce((n, s) => n + (s?.[key] ?? 0), 0);
            const stats = { ...(source ?? findStats(project.stats, "")!), total, translated: sum("translated"), approved: sum("approved"), needsEdit: sum("needsEdit") };
            return (
              <TableRow key={component.name}>
                <TableCell className="font-medium">{component.displayName}</TableCell>
                <TableCell className="text-muted-foreground">{config.formats?.find(f => f.name === component.format)?.displayName ?? component.format}</TableCell>
                <TableCell className="font-mono text-xs text-muted-foreground">
                  <div>{component.sourcePattern}</div>
                  <div>→ {component.translationPattern}</div>
                </TableCell>
                <TableCell className="text-right tabular-nums">{formatNumber(source?.total ?? 0)}</TableCell>
                <TableCell>
                  <div className="flex items-center gap-3">
                    <TranslationBar stats={stats} className="flex-1" />
                    <span className="w-12 text-right text-xs">
                      <PercentLabel stats={stats} />
                    </span>
                  </div>
                </TableCell>
              </TableRow>
            );
          })}
        </TableBody>
      </Table>
      {components.length === 0 && <div className="p-8 text-center text-sm text-muted-foreground">{t("project:No components yet.")}</div>}
    </Card>
  );
}

function TranslateDialog({
  project,
  components,
  languages,
  onClose,
  onStart,
}: {
  project: ProjectInfo;
  components: Component[];
  languages: string[];
  onClose: () => void;
  onStart: (languages: string[], options: TranslateOptions) => void;
}) {
  const { config } = useSite();
  const [component, setComponent] = useState("");
  const [review, setReview] = useState(false);
  const [includeNeedsEdit, setIncludeNeedsEdit] = useState(false);
  const [memoryOnly, setMemoryOnly] = useState(false);
  const pending = languages.reduce((n, language) => {
    const stats = findStats(project.stats, language, component);
    return n + (stats ? stats.total - stats.translated - (includeNeedsEdit ? 0 : stats.needsEdit) : 0);
  }, 0);

  return (
    <Dialog open onOpenChange={open => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{t("project:Machine translate")}</DialogTitle>
          <DialogDescription>
            {languages.length === 1
              ? t("project:Fills in the untranslated strings of {language}: first from the translation memory, then with {model}.", {
                  language: languageName(config.languages, languages[0]),
                  model: config.aiModel ?? "",
                })
              : t("project:Fills in the untranslated strings of all {count} languages: first from the translation memory, then with {model}.", {
                  count: languages.length,
                  model: config.aiModel ?? "",
                })}
          </DialogDescription>
        </DialogHeader>
        <div className="grid gap-4">
          <div className="grid gap-2">
            <Label>{t("project:Component")}</Label>
            <NativeSelect value={component} onChange={e => setComponent(e.target.value)}>
              <option value="">{t("project:All components")}</option>
              {components.map(c => (
                <option key={c.name} value={c.name}>
                  {c.displayName}
                </option>
              ))}
            </NativeSelect>
          </div>
          <Label className="justify-between font-normal">
            {t("project:Mark the results as needing a review")}
            <Switch checked={review} onCheckedChange={setReview} />
          </Label>
          <Label className="justify-between font-normal">
            {t("project:Also redo the strings that need editing")}
            <Switch checked={includeNeedsEdit} onCheckedChange={setIncludeNeedsEdit} />
          </Label>
          <Label className="justify-between font-normal">
            {t("project:Only use the translation memory (free)")}
            <Switch checked={memoryOnly} onCheckedChange={setMemoryOnly} />
          </Label>
          <div className="rounded-md bg-muted p-3 text-sm">{t("project:About {count} strings to translate.", { count: formatNumber(Math.max(0, pending)) })}</div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t("general:Cancel")}
          </Button>
          <Button
            onClick={() => onStart(languages, { component, includeNeedsEdit, memoryOnly, state: review ? State.NeedsEdit : State.Translated })}
            disabled={pending <= 0}
          >
            <Sparkles />
            {t("project:Start")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
