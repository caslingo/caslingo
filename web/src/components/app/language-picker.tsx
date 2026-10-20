import { X } from "lucide-react";
import { useId, useState } from "react";
import { Badge } from "@/components/ui/badge";
import { Input } from "@/components/ui/input";
import { languageName } from "@/lib/format";
import { t } from "@/lib/i18n";
import { useSite } from "@/lib/site";

/** Picks language codes: the known ones are suggested, any other code (as the files name it) can be typed. */
export function LanguagePicker({ value, onChange, exclude }: { value: string[]; onChange: (value: string[]) => void; exclude?: string }) {
  const { config } = useSite();
  const [input, setInput] = useState("");
  const listId = useId();

  const add = (raw: string) => {
    const code = raw.trim().split(/\s/)[0];
    if (code && code !== exclude && !value.includes(code)) onChange([...value, code]);
    setInput("");
  };

  return (
    <div className="space-y-2">
      <div className="flex flex-wrap gap-1.5">
        {value.length === 0 && <span className="text-sm text-muted-foreground">{t("project:No languages yet")}</span>}
        {value.map(code => (
          <Badge key={code} variant="secondary" className="gap-1 pr-1">
            {languageName(config.languages, code)} <span className="text-muted-foreground">{code}</span>
            <button type="button" onClick={() => onChange(value.filter(c => c !== code))} className="rounded-sm p-0.5 hover:bg-background" aria-label={t("general:Remove")}>
              <X className="size-3" />
            </button>
          </Badge>
        ))}
      </div>
      <Input
        list={listId}
        value={input}
        placeholder={t("project:Type a language code like fr or pt-BR, then press Enter")}
        onChange={e => {
          const next = e.target.value;
          // picking from the list fills "code — name"
          if (next.includes(" — ")) add(next);
          else setInput(next);
        }}
        onKeyDown={e => {
          if (e.key === "Enter" || e.key === ",") {
            e.preventDefault();
            add(input);
          }
        }}
        onBlur={() => input && add(input)}
      />
      <datalist id={listId}>
        {config.languages
          ?.filter(l => !value.includes(l.code) && l.code !== exclude)
          .map(l => <option key={l.code} value={`${l.code} — ${l.nativeName} (${l.name})`} />)}
      </datalist>
    </div>
  );
}

/** Comma-separated user names, for maintainers and translators */
export function NamesInput({ value, onChange, placeholder }: { value: string[]; onChange: (value: string[]) => void; placeholder?: string }) {
  const [text, setText] = useState((value ?? []).join(", "));
  return (
    <Input
      value={text}
      placeholder={placeholder}
      onChange={e => {
        setText(e.target.value);
        onChange(
          e.target.value
            .split(/[,\s]+/)
            .map(s => s.trim())
            .filter(Boolean),
        );
      }}
    />
  );
}
