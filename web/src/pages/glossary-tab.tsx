import { Pencil, Plus, Trash2 } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import * as Backend from "@/backend/api";
import type { ProjectInfo, Term } from "@/backend/models";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { NativeSelect } from "@/components/ui/native-select";
import { Table, TableBody, TableCell, TableHead, TableHeader, TableRow } from "@/components/ui/table";
import { useLoad } from "@/hooks/use-load";
import { languageName } from "@/lib/format";
import { t } from "@/lib/i18n";
import { useSite } from "@/lib/site";

export function GlossaryTab({ project }: { project: ProjectInfo }) {
  const { config } = useSite();
  const [language, setLanguage] = useState("");
  const { data: terms, error, reload } = useLoad(() => Backend.getTerms(project.name, language), [project.name, language]);
  const [editing, setEditing] = useState<Partial<Term> | null>(null);

  const remove = (term: Term) => {
    if (!confirm(t("glossary:Delete the term {term}?", { term: term.source }))) return;
    Backend.deleteTerm(project.name, term.id)
      .then(reload)
      .catch((e: Error) => toast.error(e.message));
  };

  return (
    <div className="space-y-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="max-w-2xl text-sm text-muted-foreground">
          {t("glossary:The words of the product and how each language says them. The editor shows the ones a string contains, and machine translation follows them. Leave the translation empty for terms that stay as they are, like product names.")}
        </p>
        <div className="flex items-center gap-2">
          <NativeSelect value={language} onChange={e => setLanguage(e.target.value)} className="w-44">
            <option value="">{t("glossary:All languages")}</option>
            {project.languages.map(code => (
              <option key={code} value={code}>
                {languageName(config.languages, code)}
              </option>
            ))}
          </NativeSelect>
          {project.canTranslate && (
            <Button onClick={() => setEditing({ language: language || project.languages[0] })} disabled={project.languages.length === 0}>
              <Plus />
              {t("glossary:Add term")}
            </Button>
          )}
        </div>
      </div>

      <Card className="gap-0 py-0">
        {error ? (
          <div className="p-8 text-center text-sm text-destructive">{error}</div>
        ) : (
          <Table>
            <TableHeader>
              <TableRow className="hover:bg-transparent">
                <TableHead>{t("glossary:Term")}</TableHead>
                <TableHead>{t("glossary:Language")}</TableHead>
                <TableHead>{t("glossary:Translation")}</TableHead>
                <TableHead>{t("glossary:Note")}</TableHead>
                <TableHead />
              </TableRow>
            </TableHeader>
            <TableBody>
              {terms?.map(term => (
                <TableRow key={term.id}>
                  <TableCell className="font-medium">{term.source}</TableCell>
                  <TableCell className="text-muted-foreground">{languageName(config.languages, term.language)}</TableCell>
                  <TableCell>{term.target || <span className="text-muted-foreground italic">{t("glossary:(kept as is)")}</span>}</TableCell>
                  <TableCell className="text-muted-foreground">{term.note}</TableCell>
                  <TableCell className="text-right whitespace-nowrap">
                    {project.canTranslate && (
                      <>
                        <Button variant="ghost" size="icon-sm" onClick={() => setEditing(term)} aria-label={t("general:Edit")}>
                          <Pencil />
                        </Button>
                        <Button variant="ghost" size="icon-sm" onClick={() => remove(term)} aria-label={t("general:Delete")}>
                          <Trash2 />
                        </Button>
                      </>
                    )}
                  </TableCell>
                </TableRow>
              ))}
            </TableBody>
          </Table>
        )}
        {terms?.length === 0 && <div className="p-8 text-center text-sm text-muted-foreground">{t("glossary:No terms yet.")}</div>}
      </Card>

      {editing && <TermDialog project={project} term={editing} onClose={() => setEditing(null)} onSaved={() => (setEditing(null), reload())} />}
    </div>
  );
}

function TermDialog({ project, term, onClose, onSaved }: { project: ProjectInfo; term: Partial<Term>; onClose: () => void; onSaved: () => void }) {
  const { config } = useSite();
  const [value, setValue] = useState<Partial<Term>>(term);
  const set = (patch: Partial<Term>) => setValue(v => ({ ...v, ...patch }));

  const save = () => {
    const request = value.id ? Backend.updateTerm(project.name, value.id, value) : Backend.addTerm(project.name, value);
    request.then(onSaved).catch((e: Error) => toast.error(e.message));
  };

  return (
    <Dialog open onOpenChange={open => !open && onClose()}>
      <DialogContent className="sm:max-w-md">
        <DialogHeader>
          <DialogTitle>{value.id ? t("glossary:Edit term") : t("glossary:Add term")}</DialogTitle>
          <DialogDescription>{t("glossary:Terms match whole words, without regard to case.")}</DialogDescription>
        </DialogHeader>
        <div className="grid gap-4">
          <div className="grid gap-2">
            <Label>{t("glossary:Term")}</Label>
            <Input value={value.source ?? ""} onChange={e => set({ source: e.target.value })} placeholder="Organization" autoFocus />
          </div>
          <div className="grid gap-2">
            <Label>{t("glossary:Language")}</Label>
            <NativeSelect value={value.language ?? ""} onChange={e => set({ language: e.target.value })}>
              {project.languages.map(code => (
                <option key={code} value={code}>
                  {languageName(config.languages, code)}
                </option>
              ))}
            </NativeSelect>
          </div>
          <div className="grid gap-2">
            <Label>{t("glossary:Translation")}</Label>
            <Input value={value.target ?? ""} onChange={e => set({ target: e.target.value })} placeholder={t("glossary:Empty: keep the term as is")} />
          </div>
          <div className="grid gap-2">
            <Label>{t("glossary:Note")}</Label>
            <Input value={value.note ?? ""} onChange={e => set({ note: e.target.value })} />
          </div>
        </div>
        <DialogFooter>
          <Button variant="outline" onClick={onClose}>
            {t("general:Cancel")}
          </Button>
          <Button onClick={save} disabled={!value.source}>
            {t("general:Save")}
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
