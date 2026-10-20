import { Pencil, Plus, Trash2 } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";
import { Link, useNavigate, useParams } from "react-router-dom";
import { toast } from "sonner";
import * as Backend from "@/backend/api";
import type { Component, Project } from "@/backend/models";
import { LanguagePicker, NamesInput } from "@/components/app/language-picker";
import { Page, PageHeader } from "@/components/app/layout";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { NativeSelect } from "@/components/ui/native-select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useLoad } from "@/hooks/use-load";
import { useDocumentTitle } from "@/lib/format";
import { t } from "@/lib/i18n";
import { useSite } from "@/lib/site";
import { PageError, PageLoading } from "./static-pages";

function Field({ label, hint, children, className }: { label: string; hint?: ReactNode; children: ReactNode; className?: string }) {
  return (
    <div className={className}>
      <Label className="mb-2">{label}</Label>
      {children}
      {hint && <div className="mt-1.5 text-xs text-muted-foreground">{hint}</div>}
    </div>
  );
}

function SwitchField({ label, hint, checked, onChange }: { label: string; hint?: string; checked: boolean; onChange: (checked: boolean) => void }) {
  return (
    <label className="flex items-start justify-between gap-4">
      <div>
        <div className="text-sm font-medium">{label}</div>
        {hint && <div className="mt-0.5 text-xs text-muted-foreground">{hint}</div>}
      </div>
      <Switch checked={checked} onCheckedChange={onChange} className="mt-0.5" />
    </label>
  );
}

function Section({ title, children }: { title: string; children: ReactNode }) {
  return (
    <Card className="gap-5 p-5">
      <div className="font-semibold">{title}</div>
      {children}
    </Card>
  );
}

export function SettingsPage() {
  const { name = "" } = useParams();
  const navigate = useNavigate();
  const { account, config } = useSite();
  const { data, error, reload } = useLoad(() => Backend.getProject(name), [name]);
  const [project, setProject] = useState<Project | null>(null);
  const [editing, setEditing] = useState<Partial<Component> | null>(null);
  const [saving, setSaving] = useState(false);
  useDocumentTitle(t("project:Settings"), config.siteName);

  useEffect(() => {
    if (data) setProject(data.project);
  }, [data]);

  if (error) return <PageError message={error} />;
  if (!data || !project) return <PageLoading />;
  if (!data.project.canMaintain) return <PageError message={t("settings:Only the project's maintainers can change its settings")} />;

  const set = (patch: Partial<Project>) => setProject(p => (p ? { ...p, ...patch } : p));
  const save = () => {
    setSaving(true);
    Backend.updateProject(name, project)
      .then(() => {
        toast.success(t("settings:Saved"));
        reload();
      })
      .catch((e: Error) => toast.error(e.message))
      .finally(() => setSaving(false));
  };
  const remove = () => {
    if (!confirm(t("settings:Delete the project {name} with all its strings, translations and history? This cannot be undone.", { name }))) return;
    Backend.deleteProject(name)
      .then(() => navigate("/"))
      .catch((e: Error) => toast.error(e.message));
  };
  const removeComponent = (component: Component) => {
    if (!confirm(t("settings:Delete the component {name} with its strings and translations?", { name: component.displayName }))) return;
    Backend.deleteComponent(name, component.name)
      .then(reload)
      .catch((e: Error) => toast.error(e.message));
  };

  return (
    <Page className="max-w-4xl">
      <PageHeader
        title={t("project:Settings")}
        actions={
          <Button onClick={save} disabled={saving}>
            {t("general:Save")}
          </Button>
        }
      >
        <Link to={`/projects/${name}`} className="mb-1 inline-block text-sm text-muted-foreground hover:text-foreground">
          {data.project.displayName} /
        </Link>
      </PageHeader>

      <div className="space-y-6">
        <Section title={t("settings:Project")}>
          <div className="grid gap-5 sm:grid-cols-2">
            <Field label={t("project:Name")} hint={t("settings:Used in addresses, cannot be changed")}>
              <Input value={project.name} disabled />
            </Field>
            <Field label={t("project:Display name")}>
              <Input value={project.displayName} onChange={e => set({ displayName: e.target.value })} />
            </Field>
            <Field label={t("settings:Description")} className="sm:col-span-2" hint={t("settings:Also tells the model what the product is")}>
              <Input value={project.description} onChange={e => set({ description: e.target.value })} />
            </Field>
            <Field label={t("settings:Website")}>
              <Input value={project.websiteUrl} onChange={e => set({ websiteUrl: e.target.value })} placeholder="https://casdoor.ai" />
            </Field>
            <Field label={t("project:Source language")}>
              <Input value={project.sourceLanguage} onChange={e => set({ sourceLanguage: e.target.value })} />
            </Field>
            <Field label={t("project:Languages to translate into")} className="sm:col-span-2" hint={t("settings:Use the codes the files use, like zh, pt-BR or zh-CN")}>
              <LanguagePicker value={project.languages} onChange={languages => set({ languages })} exclude={project.sourceLanguage} />
            </Field>
          </div>
        </Section>

        <Section title={t("settings:People")}>
          <SwitchField label={t("settings:Public")} hint={t("settings:Anyone can see the project and its translations, and signed-in users can suggest translations")} checked={project.isPublic} onChange={isPublic => set({ isPublic })} />
          <SwitchField label={t("settings:Anyone signed in can translate")} hint={t("settings:Otherwise only the translators below can, and others make suggestions")} checked={project.openEditing} onChange={openEditing => set({ openEditing })} />
          <div className="grid gap-5 sm:grid-cols-2">
            <Field label={t("settings:Maintainers")} hint={t("settings:Change the settings, approve, pull, push and machine translate. Casdoor user names, comma-separated")}>
              <NamesInput value={project.maintainers} onChange={maintainers => set({ maintainers })} placeholder={account?.name} />
            </Field>
            <Field label={t("settings:Translators")} hint={t("settings:Translate and edit the glossary")}>
              <NamesInput value={project.translators} onChange={translators => set({ translators })} />
            </Field>
          </div>
        </Section>

        <Section title={t("settings:Repository")}>
          <div className="grid gap-5 sm:grid-cols-2">
            <Field label={t("project:Repository URL")} className="sm:col-span-2" hint={t("settings:HTTPS URL of a git repository; Caslingo clones it to read the source files and push the translations")}>
              <Input value={project.repoUrl} onChange={e => set({ repoUrl: e.target.value })} placeholder="https://github.com/casdoor/casdoor" />
            </Field>
            <Field label={t("project:Branch")}>
              <Input value={project.branch} onChange={e => set({ branch: e.target.value })} />
            </Field>
            <Field label={t("settings:Access token")} hint={t("settings:For private repositories and for pushing, e.g. a GitHub fine-grained token with Contents (and Pull requests) write access")}>
              <Input type="password" value={project.repoToken} onChange={e => set({ repoToken: e.target.value })} autoComplete="off" />
            </Field>
            <Field label={t("settings:Push")}>
              <NativeSelect value={project.pushMode} onChange={e => set({ pushMode: e.target.value as Project["pushMode"] })}>
                <option value="">{t("settings:Off: only download or take them at build time")}</option>
                <option value="direct">{t("settings:Commit to the branch")}</option>
                <option value="pull-request">{t("settings:Open a pull request (GitHub)")}</option>
              </NativeSelect>
            </Field>
            {project.pushMode === "pull-request" && (
              <Field label={t("settings:Pull request branch")}>
                <Input value={project.pushBranch} onChange={e => set({ pushBranch: e.target.value })} />
              </Field>
            )}
            <Field label={t("settings:Commit message")} className="sm:col-span-2" hint={t("settings:Follow the repository's convention, e.g. feat: update translations")}>
              <Input value={project.commitMessage} onChange={e => set({ commitMessage: e.target.value })} />
            </Field>
            <Field label={t("settings:Commit author name")}>
              <Input value={project.commitName} onChange={e => set({ commitName: e.target.value })} />
            </Field>
            <Field label={t("settings:Commit author email")}>
              <Input value={project.commitEmail} onChange={e => set({ commitEmail: e.target.value })} />
            </Field>
          </div>
        </Section>

        <Section title={t("settings:Machine translation")}>
          {!config.aiEnabled && <div className="rounded-md bg-muted p-3 text-sm text-muted-foreground">{t("settings:Machine translation is not set up on this server: set aiBaseUrl, aiModel and aiApiKey in conf/app.conf.")}</div>}
          <Field label={t("settings:Instructions for the model")} hint={t("settings:Style, tone, words to keep in English; the glossary is added on its own")}>
            <Textarea rows={4} value={project.instructions} onChange={e => set({ instructions: e.target.value })} placeholder={t("settings:e.g. Keep the names of the console's pages and buttons in English and in bold.")} />
          </Field>
          <SwitchField label={t("settings:Translate new strings after each pull")} hint={t("settings:From the translation memory, then with the model, into every language")} checked={project.autoTranslate} onChange={autoTranslate => set({ autoTranslate })} />
          <SwitchField label={t("settings:Push after each pull that changed strings")} checked={project.autoPush} onChange={autoPush => set({ autoPush })} />
        </Section>

        <Section title={t("project:Components")}>
          <p className="-mt-2 text-sm text-muted-foreground">{t("settings:Each component is a set of files of one format: the UI strings, the docs, the blog. Changes take effect on the next pull.")}</p>
          <div className="divide-y rounded-md border">
            {data.components.map(component => (
              <div key={component.name} className="flex items-center gap-3 p-3">
                <div className="min-w-0 flex-1">
                  <div className="font-medium">
                    {component.displayName} <span className="text-xs font-normal text-muted-foreground">{component.format}</span>
                  </div>
                  <div className="truncate font-mono text-xs text-muted-foreground">
                    {component.sourcePattern} → {component.translationPattern}
                  </div>
                </div>
                <Button variant="ghost" size="icon-sm" onClick={() => setEditing(component)} aria-label={t("general:Edit")}>
                  <Pencil />
                </Button>
                <Button variant="ghost" size="icon-sm" onClick={() => removeComponent(component)} aria-label={t("general:Delete")}>
                  <Trash2 />
                </Button>
              </div>
            ))}
            {data.components.length === 0 && <div className="p-6 text-center text-sm text-muted-foreground">{t("project:No components yet.")}</div>}
          </div>
          <div className="flex flex-wrap gap-2">
            <Button variant="outline" onClick={() => setEditing({ format: "json", fallbackToSource: true, sourcePattern: "web/src/locales/en/data.json", translationPattern: "web/src/locales/{lang}/data.json", name: "ui", displayName: "UI" })}>
              <Plus />
              {t("settings:JSON strings")}
            </Button>
            <Button variant="outline" onClick={() => setEditing({ format: "markdown", headingIds: true, sourcePattern: "docs/**/*.{md,mdx}", translationPattern: "i18n/{lang}/docusaurus-plugin-content-docs/current/{path}", name: "docs", displayName: "Docs" })}>
              <Plus />
              {t("settings:Docusaurus docs")}
            </Button>
            <Button variant="outline" onClick={() => setEditing({ format: "docusaurus-json", uploadedSource: true, sourcePattern: "i18n/en/**/*.json", translationPattern: "i18n/{lang}/{path}", name: "theme", displayName: "Site texts" })}>
              <Plus />
              {t("settings:Docusaurus site texts")}
            </Button>
          </div>
        </Section>

        {account?.isAdmin && (
          <Section title={t("settings:Danger zone")}>
            <div className="flex items-center justify-between gap-4">
              <span className="text-sm text-muted-foreground">{t("settings:Deletes the project with all its strings, translations and history. The repository is not touched.")}</span>
              <Button variant="destructive" onClick={remove}>
                {t("settings:Delete project")}
              </Button>
            </div>
          </Section>
        )}
      </div>

      {editing && <ComponentDialog project={name} component={editing} onClose={() => setEditing(null)} onSaved={() => (setEditing(null), reload())} />}
    </Page>
  );
}

function ComponentDialog({ project, component, onClose, onSaved }: { project: string; component: Partial<Component>; onClose: () => void; onSaved: () => void }) {
  const { config } = useSite();
  const [value, setValue] = useState<Partial<Component>>(component);
  const isNew = !component.project;
  const set = (patch: Partial<Component>) => setValue(v => ({ ...v, ...patch }));

  const save = () => {
    const request = isNew ? Backend.addComponent(project, value) : Backend.updateComponent(project, component.name!, value);
    request
      .then(() => {
        toast.success(t("settings:Saved. Pull the repository to read the files."));
        onSaved();
      })
      .catch((e: Error) => toast.error(e.message));
  };

  return (
    <Dialog open onOpenChange={open => !open && onClose()}>
      <DialogContent className="sm:max-w-xl">
        <DialogHeader>
          <DialogTitle>{isNew ? t("settings:Add component") : t("settings:Edit component")}</DialogTitle>
          <DialogDescription>{t("settings:Paths are relative to the repository root. * matches within a folder, ** any number of folders, {md,mdx} either.")}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-4 sm:grid-cols-2">
          <Field label={t("project:Name")}>
            <Input value={value.name ?? ""} onChange={e => set({ name: e.target.value.toLowerCase() })} disabled={!isNew} />
          </Field>
          <Field label={t("project:Display name")}>
            <Input value={value.displayName ?? ""} onChange={e => set({ displayName: e.target.value })} />
          </Field>
          <Field label={t("project:Format")} className="sm:col-span-2">
            <NativeSelect value={value.format ?? "json"} onChange={e => set({ format: e.target.value })}>
              {config.formats?.map(f => (
                <option key={f.name} value={f.name}>
                  {f.displayName}
                </option>
              ))}
            </NativeSelect>
          </Field>
          <Field label={t("settings:Source files")} className="sm:col-span-2">
            <Input className="font-mono" value={value.sourcePattern ?? ""} onChange={e => set({ sourcePattern: e.target.value })} />
          </Field>
          <Field
            label={t("settings:Translation files")}
            className="sm:col-span-2"
            hint={t("settings:{lang} is the language code; {path} is the source file's path below the part of the source pattern before the first wildcard")}
          >
            <Input className="font-mono" value={value.translationPattern ?? ""} onChange={e => set({ translationPattern: e.target.value })} />
          </Field>
          <Field label={t("settings:Exclude")} className="sm:col-span-2" hint={t("settings:Patterns of source files to leave out, comma-separated")}>
            <Input className="font-mono" value={value.excludePattern ?? ""} onChange={e => set({ excludePattern: e.target.value })} />
          </Field>
          <div className="space-y-4 sm:col-span-2">
            <SwitchField label={t("settings:Write untranslated strings in the source language")} hint={t("settings:Keeps every key in the translation files (what apps that load one file per language need); otherwise they are left out")} checked={!!value.fallbackToSource} onChange={fallbackToSource => set({ fallbackToSource })} />
            {value.format === "markdown" && (
              <SwitchField label={t("settings:Keep the heading anchors")} hint={t("settings:Adds {#anchor} of the English heading to translated headings, so links to #sections keep working")} checked={!!value.headingIds} onChange={headingIds => set({ headingIds })} />
            )}
            <SwitchField label={t("settings:The source files are uploaded")} hint={t("settings:For files generated at build time and not in the repository; see the project's Integration tab")} checked={!!value.uploadedSource} onChange={uploadedSource => set({ uploadedSource })} />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t("general:Cancel")}
          </Button>
          <Button onClick={save} disabled={!value.name || !value.sourcePattern || !value.translationPattern}>
            {t("general:Save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
