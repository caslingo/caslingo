import { useState } from "react";
import * as Backend from "@/backend/api";
import type { Change, ProjectInfo } from "@/backend/models";
import { Pager } from "@/components/app/pager";
import { Badge } from "@/components/ui/badge";
import { Card } from "@/components/ui/card";
import { NativeSelect } from "@/components/ui/native-select";
import { useLoad } from "@/hooks/use-load";
import { formatNumber, formatTime, languageName } from "@/lib/format";
import { t } from "@/lib/i18n";
import { useSite } from "@/lib/site";

function actionLabel(action: string) {
  switch (action) {
    case "translate":
      return t("history:translated");
    case "approve":
      return t("history:approved");
    case "suggest":
      return t("history:suggested");
    case "accept":
      return t("history:accepted a suggestion");
    case "machine":
      return t("history:machine translated");
    case "memory":
      return t("history:from translation memory");
    case "sync":
      return t("history:pulled");
    case "push":
      return t("history:pushed");
  }
  return action;
}

export function HistoryTab({ project }: { project: ProjectInfo }) {
  const { config } = useSite();
  const [language, setLanguage] = useState("");
  const [author, setAuthor] = useState("");
  const [page, setPage] = useState(1);
  const { data, error } = useLoad(() => Backend.getChanges(project.name, { language, author, page }), [project.name, language, author, page]);

  return (
    <div className="grid gap-6 lg:grid-cols-[1fr_16rem]">
      <div className="min-w-0 space-y-3">
        <div className="flex gap-2">
          <NativeSelect value={language} onChange={e => (setLanguage(e.target.value), setPage(1))} className="w-44">
            <option value="">{t("history:All languages")}</option>
            {project.languages.map(code => (
              <option key={code} value={code}>
                {languageName(config.languages, code)}
              </option>
            ))}
          </NativeSelect>
          {author && (
            <Badge variant="secondary" className="cursor-pointer" onClick={() => (setAuthor(""), setPage(1))}>
              {author} ×
            </Badge>
          )}
        </div>
        {error && <div className="text-sm text-destructive">{error}</div>}
        <Card className="gap-0 divide-y py-0">
          {data?.changes.map(change => <ChangeRow key={change.id} change={change} onAuthor={name => (setAuthor(name), setPage(1))} />)}
          {data?.changes.length === 0 && <div className="p-8 text-center text-sm text-muted-foreground">{t("history:Nothing has happened yet.")}</div>}
        </Card>
        {data && <Pager page={page} pageSize={50} total={data.total} onPage={setPage} />}
      </div>

      <Card className="h-fit gap-3 p-4">
        <div className="text-sm font-medium">{t("history:Contributors")}</div>
        {data?.contributors.length === 0 && <div className="text-sm text-muted-foreground">{t("history:No one yet.")}</div>}
        {data?.contributors.map(c => (
          <button key={c.author} onClick={() => (setAuthor(c.author), setPage(1))} className="flex justify-between text-left text-sm hover:text-primary">
            <span className="truncate">{c.author}</span>
            <span className="text-muted-foreground tabular-nums">{formatNumber(c.count)}</span>
          </button>
        ))}
      </Card>
    </div>
  );
}

function ChangeRow({ change, onAuthor }: { change: Change; onAuthor: (name: string) => void }) {
  const { config } = useSite();
  return (
    <div className="space-y-1.5 p-3 text-sm">
      <div className="flex flex-wrap items-center gap-x-2 text-muted-foreground">
        {change.author ? (
          <button onClick={() => onAuthor(change.author)} className="font-medium text-foreground hover:text-primary">
            {change.author}
          </button>
        ) : (
          <span className="font-medium text-foreground">Caslingo</span>
        )}
        <span>{actionLabel(change.action)}</span>
        {change.language && !change.language.includes(",") && <Badge variant="outline">{languageName(config.languages, change.language)}</Badge>}
        <span className="ml-auto text-xs">{formatTime(change.createdTime)}</span>
      </div>
      {change.source && <div className="text-unit line-clamp-2 text-xs text-muted-foreground">{change.source}</div>}
      {change.unitId ? (
        <div className="text-unit">
          {change.oldText && <span className="mr-2 text-muted-foreground line-through">{change.oldText}</span>}
          {change.newText}
        </div>
      ) : (
        change.newText && <div className="text-muted-foreground">{change.newText}</div>
      )}
    </div>
  );
}
