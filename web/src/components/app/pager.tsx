import { ChevronLeft, ChevronRight } from "lucide-react";
import { Button } from "@/components/ui/button";
import { formatNumber } from "@/lib/format";
import { t } from "@/lib/i18n";

export function Pager({ page, pageSize, total, onPage }: { page: number; pageSize: number; total: number; onPage: (page: number) => void }) {
  const pages = Math.max(1, Math.ceil(total / pageSize));
  if (total <= pageSize) return null;
  return (
    <div className="flex items-center justify-between text-sm text-muted-foreground">
      <span>
        {t("general:{from}–{to} of {total}", {
          from: formatNumber((page - 1) * pageSize + 1),
          to: formatNumber(Math.min(page * pageSize, total)),
          total: formatNumber(total),
        })}
      </span>
      <div className="flex items-center gap-1">
        <Button variant="outline" size="icon-sm" disabled={page <= 1} onClick={() => onPage(page - 1)} aria-label={t("general:Previous page")}>
          <ChevronLeft />
        </Button>
        <span className="px-2 tabular-nums">
          {page} / {pages}
        </span>
        <Button variant="outline" size="icon-sm" disabled={page >= pages} onClick={() => onPage(page + 1)} aria-label={t("general:Next page")}>
          <ChevronRight />
        </Button>
      </div>
    </div>
  );
}
