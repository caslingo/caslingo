import { FolderGit2, Globe2, Lock, Plus } from "lucide-react";
import { useState } from "react";
import { Link, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import * as Backend from "@/backend/api";
import type { ProjectInfo } from "@/backend/models";
import { LanguagePicker } from "@/components/app/language-picker";
import { Page, PageHeader } from "@/components/app/layout";
import { PercentLabel, TranslationBar } from "@/components/app/progress";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Skeleton } from "@/components/ui/skeleton";
import { useLoad } from "@/hooks/use-load";
import { findStats, formatNumber, useDocumentTitle } from "@/lib/format";
import { t } from "@/lib/i18n";
import { useSite } from "@/lib/site";
import { PageError } from "./static-pages";

export function HomePage() {
  const { account, config } = useSite();
  const { data: projects, error } = useLoad(Backend.getProjects, [account?.name]);
  const [creating, setCreating] = useState(false);
  useDocumentTitle("", config.siteName);

  if (error) return <PageError message={error} />;

  return (
    <Page>
      <PageHeader
        title={t("home:Projects")}
        description={t("home:Translate the interface and documentation of each project, in the browser or with machine translation.")}
        actions={
          account?.isAdmin && (
            <Button onClick={() => setCreating(true)}>
              <Plus />
              {t("home:New project")}
            </Button>
          )
        }
      />

      {!projects ? (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {[0, 1, 2].map(i => (
            <Skeleton key={i} className="h-40" />
          ))}
        </div>
      ) : projects.length === 0 ? (
        <Card className="items-center gap-3 py-16 text-center text-sm text-muted-foreground">
          <FolderGit2 className="size-8" />
          {account?.isAdmin ? t("home:No projects yet. Create one to start translating.") : t("home:There are no public projects yet.")}
        </Card>
      ) : (
        <div className="grid gap-4 md:grid-cols-2 lg:grid-cols-3">
          {projects.map(project => (
            <ProjectCard key={project.name} project={project} />
          ))}
        </div>
      )}

      <NewProjectDialog open={creating} onOpenChange={setCreating} />
    </Page>
  );
}

function ProjectCard({ project }: { project: ProjectInfo }) {
  const stats = findStats(project.stats, "");
  const languageStats = project.languages.map(language => findStats(project.stats, language));
  const translated = languageStats.reduce((n, s) => n + (s?.translated ?? 0), 0);
  const total = (stats?.total ?? 0) * project.languages.length;

  return (
    <Link to={`/projects/${project.name}`} className="group">
      <Card className="h-full gap-4 p-5 transition-shadow group-hover:shadow-md">
        <div className="flex items-start justify-between gap-3">
          <div className="min-w-0">
            <div className="truncate font-semibold group-hover:text-primary">{project.displayName}</div>
            <div className="mt-1 line-clamp-2 min-h-10 text-sm text-muted-foreground">{project.description || project.repoUrl || project.name}</div>
          </div>
          <Badge variant="outline" className="shrink-0 gap-1">
            {project.isPublic ? <Globe2 className="size-3" /> : <Lock className="size-3" />}
            {project.isPublic ? t("home:Public") : t("home:Private")}
          </Badge>
        </div>
        <div className="mt-auto space-y-2">
          <TranslationBar
            stats={{
              language: "",
              component: "",
              total,
              totalWords: 0,
              translated,
              translatedWords: 0,
              approved: languageStats.reduce((n, s) => n + (s?.approved ?? 0), 0),
              needsEdit: languageStats.reduce((n, s) => n + (s?.needsEdit ?? 0), 0),
              checks: 0,
              suggestions: 0,
            }}
          />
          <div className="flex justify-between text-xs text-muted-foreground">
            <span>
              {t("home:{languages} languages, {strings} strings, {words} words", {
                languages: project.languages.length,
                strings: formatNumber(stats?.total ?? 0),
                words: formatNumber(stats?.totalWords ?? 0),
              })}
            </span>
            <PercentLabel stats={{ ...(stats ?? emptyStats), total, translated }} />
          </div>
        </div>
      </Card>
    </Link>
  );
}

const emptyStats = { language: "", component: "", total: 0, totalWords: 0, translated: 0, translatedWords: 0, approved: 0, needsEdit: 0, checks: 0, suggestions: 0 };

function NewProjectDialog({ open, onOpenChange }: { open: boolean; onOpenChange: (open: boolean) => void }) {
  const navigate = useNavigate();
  const [name, setName] = useState("");
  const [displayName, setDisplayName] = useState("");
  const [repoUrl, setRepoUrl] = useState("");
  const [branch, setBranch] = useState("master");
  const [sourceLanguage, setSourceLanguage] = useState("en");
  const [languages, setLanguages] = useState<string[]>(["zh"]);
  const [saving, setSaving] = useState(false);

  const submit = () => {
    setSaving(true);
    Backend.addProject({ name, displayName, repoUrl, branch, sourceLanguage, languages, isPublic: true })
      .then(created => {
        onOpenChange(false);
        navigate(`/projects/${created}/settings`);
        toast.success(t("home:Project created. Now add its components: the files to translate."));
      })
      .catch((e: Error) => toast.error(e.message))
      .finally(() => setSaving(false));
  };

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-lg">
        <DialogHeader>
          <DialogTitle>{t("home:New project")}</DialogTitle>
          <DialogDescription>{t("home:A project is usually one git repository. You can change everything later in its settings.")}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-4">
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="project-name">{t("project:Name")}</Label>
              <Input id="project-name" value={name} onChange={e => setName(e.target.value.toLowerCase())} placeholder="casdoor" />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="project-display-name">{t("project:Display name")}</Label>
              <Input id="project-display-name" value={displayName} onChange={e => setDisplayName(e.target.value)} placeholder="Casdoor" />
            </div>
          </div>
          <div className="grid gap-2">
            <Label htmlFor="project-repo">{t("project:Repository URL")}</Label>
            <Input id="project-repo" value={repoUrl} onChange={e => setRepoUrl(e.target.value)} placeholder="https://github.com/casdoor/casdoor" />
          </div>
          <div className="grid gap-4 sm:grid-cols-2">
            <div className="grid gap-2">
              <Label htmlFor="project-branch">{t("project:Branch")}</Label>
              <Input id="project-branch" value={branch} onChange={e => setBranch(e.target.value)} />
            </div>
            <div className="grid gap-2">
              <Label htmlFor="project-source">{t("project:Source language")}</Label>
              <Input id="project-source" value={sourceLanguage} onChange={e => setSourceLanguage(e.target.value)} />
            </div>
          </div>
          <div className="grid gap-2">
            <Label>{t("project:Languages to translate into")}</Label>
            <LanguagePicker value={languages} onChange={setLanguages} exclude={sourceLanguage} />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={() => onOpenChange(false)}>
            {t("general:Cancel")}
          </Button>
          <Button onClick={submit} disabled={saving || !name}>
            {t("general:Create")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
