import type { Stats } from "@/backend/models";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { formatNumber, percent } from "@/lib/format";
import { t } from "@/lib/i18n";
import { cn } from "@/lib/utils";

/** a bar split into approved, translated and needing an edit */
export function TranslationBar({ stats, className }: { stats?: Stats; className?: string }) {
  const total = stats?.total ?? 0;
  const approved = stats?.approved ?? 0;
  const translated = (stats?.translated ?? 0) - approved;
  const needsEdit = stats?.needsEdit ?? 0;
  const parts = [
    { value: approved, className: "bg-state-approved", label: t("state:Approved") },
    { value: translated, className: "bg-state-translated", label: t("state:Translated") },
    { value: needsEdit, className: "bg-state-needs-edit", label: t("state:Needs editing") },
  ];

  return (
    <Tooltip>
      <TooltipTrigger asChild>
        <div className={cn("flex h-2 w-full overflow-hidden rounded-full bg-muted", className)}>
          {parts.map(part =>
            part.value > 0 ? <div key={part.label} className={part.className} style={{ width: `${percent(part.value, total)}%` }} /> : null,
          )}
        </div>
      </TooltipTrigger>
      <TooltipContent>
        <div className="space-y-0.5">
          {parts.map(part => (
            <div key={part.label} className="flex items-center gap-2">
              <span className={cn("size-2 rounded-full", part.className)} />
              {part.label}: {formatNumber(part.value)}
            </div>
          ))}
          <div>
            {t("state:Untranslated")}: {formatNumber(Math.max(0, total - approved - translated - needsEdit))}
          </div>
        </div>
      </TooltipContent>
    </Tooltip>
  );
}

export function PercentLabel({ stats }: { stats?: Stats }) {
  return <span className="tabular-nums">{percent(stats?.translated ?? 0, stats?.total ?? 0)}%</span>;
}
