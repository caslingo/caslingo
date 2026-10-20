import { BookA, Check, History, Languages, Lightbulb, Loader2, Trash2 } from "lucide-react";
import type { ReactNode } from "react";
import { toast } from "sonner";
import * as Backend from "@/backend/api";
import type { ProjectInfo, Unit } from "@/backend/models";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { useLoad } from "@/hooks/use-load";
import { formatKey, formatRelative, languageName } from "@/lib/format";
import { t } from "@/lib/i18n";
import { useSite } from "@/lib/site";

function Section({ icon, title, children }: { icon: ReactNode; title: string; children: ReactNode }) {
  return (
    <div className="space-y-2">
      <div className="flex items-center gap-1.5 text-xs font-medium text-muted-foreground [&_svg]:size-3.5">
        {icon}
        {title}
      </div>
      {children}
    </div>
  );
}

/** What helps translate the selected string: suggestions, the translation memory, the glossary, other languages, history. */
export function UnitHelpPanel({
  project,
  language,
  unit,
  onUse,
  onChanged,
}: {
  project: ProjectInfo;
  language: string;
  unit: Unit;
  onUse: (text: string) => void;
  onChanged: (unit: Unit) => void;
}) {
  const { account, config } = useSite();
  const { data, error } = useLoad(() => Backend.getUnitHelp(project.name, language, unit.id), [project.name, language, unit.id]);
  const canUse = !!account;

  const accept = (id: number) =>
    Backend.acceptSuggestion(project.name, id)
      .then(translation => onChanged({ ...unit, translation, suggestions: unit.suggestions?.filter(s => s.id !== id) ?? null }))
      .catch((e: Error) => toast.error(e.message));
  const reject = (id: number) =>
    Backend.deleteSuggestion(project.name, id)
      .then(() => onChanged({ ...unit, suggestions: unit.suggestions?.filter(s => s.id !== id) ?? null }))
      .catch((e: Error) => toast.error(e.message));

  return (
    <Card className="max-h-[calc(100svh-6rem)] gap-5 overflow-y-auto p-4 text-sm">
      <div className="space-y-1">
        <div className="text-xs text-muted-foreground">{t("editor:File")}</div>
        <div className="font-mono text-xs break-all">{unit.file}</div>
        {formatKey(unit.key) && <div className="font-mono text-xs break-all text-muted-foreground">{formatKey(unit.key)}</div>}
      </div>

      {unit.suggestions?.length ? (
        <Section icon={<Lightbulb />} title={t("editor:Suggestions")}>
          {unit.suggestions.map(s => (
            <div key={s.id} className="space-y-1.5 rounded-md border p-2.5">
              <div className="text-unit">{s.text}</div>
              <div className="flex items-center gap-1 text-xs text-muted-foreground">
                {s.author}, {formatRelative(s.createdTime)}
                <div className="ml-auto flex gap-1">
                  {project.canTranslate && (
                    <Button variant="ghost" size="icon-xs" onClick={() => accept(s.id)} title={t("editor:Accept")}>
                      <Check />
                    </Button>
                  )}
                  {(project.canTranslate || s.author === account?.name) && (
                    <Button variant="ghost" size="icon-xs" onClick={() => reject(s.id)} title={t("general:Delete")}>
                      <Trash2 />
                    </Button>
                  )}
                </div>
              </div>
            </div>
          ))}
        </Section>
      ) : null}

      {error && <div className="text-destructive">{error}</div>}
      {!data && !error && <Loader2 className="size-4 animate-spin text-muted-foreground" />}
      {data && (
        <>
          <Section icon={<History />} title={t("editor:Translation memory")}>
            {data.memory?.length ? (
              data.memory.map((m, i) => (
                <button key={i} disabled={!canUse} onClick={() => onUse(m.text)} className="block w-full space-y-1 rounded-md border p-2.5 text-left hover:border-primary/50 hover:bg-accent/40 disabled:pointer-events-none">
                  <div className="text-unit">{m.text}</div>
                  <div className="flex items-center gap-2 text-xs text-muted-foreground">
                    <Badge variant={m.quality === 100 ? "default" : "secondary"} className="px-1.5 py-0">
                      {m.quality}%
                    </Badge>
                    {m.project !== project.name && m.project}
                    {m.quality < 100 && <span className="line-clamp-1">{m.source}</span>}
                  </div>
                </button>
              ))
            ) : (
              <div className="text-xs text-muted-foreground">{t("editor:No matches")}</div>
            )}
          </Section>

          {data.terms?.length ? (
            <Section icon={<BookA />} title={t("editor:Glossary")}>
              {data.terms.map(term => (
                <div key={term.id} className="flex items-baseline justify-between gap-3">
                  <span className="font-medium">{term.source}</span>
                  <span className="text-right">
                    {term.target || <span className="text-muted-foreground italic">{t("glossary:(kept as is)")}</span>}
                    {term.note && <span className="block text-xs text-muted-foreground">{term.note}</span>}
                  </span>
                </div>
              ))}
            </Section>
          ) : null}

          {data.others?.length ? (
            <Section icon={<Languages />} title={t("editor:Other languages")}>
              {data.others.map(other => (
                <div key={other.id} className="space-y-0.5">
                  <div className="text-xs text-muted-foreground">{languageName(config.languages, other.language)}</div>
                  <div className="text-unit">{other.text}</div>
                </div>
              ))}
            </Section>
          ) : null}

          {data.history?.length ? (
            <Section icon={<History />} title={t("editor:History")}>
              {data.history.map(change => (
                <div key={change.id} className="space-y-0.5">
                  <div className="text-xs text-muted-foreground">
                    {change.author || "Caslingo"}, {formatRelative(change.createdTime)}
                  </div>
                  <div className="text-unit line-clamp-3">{change.newText}</div>
                </div>
              ))}
            </Section>
          ) : null}
        </>
      )}
    </Card>
  );
}
