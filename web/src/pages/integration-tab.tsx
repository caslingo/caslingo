import { Copy, Eye, KeyRound } from "lucide-react";
import { useState } from "react";
import { toast } from "sonner";
import * as Backend from "@/backend/api";
import type { ProjectInfo } from "@/backend/models";
import { Button } from "@/components/ui/button";
import { Card } from "@/components/ui/card";
import { t } from "@/lib/i18n";

function CodeBlock({ code }: { code: string }) {
  return (
    <div className="group relative">
      <pre className="overflow-x-auto rounded-md bg-muted p-3 font-mono text-xs leading-relaxed">{code}</pre>
      <Button
        variant="ghost"
        size="icon-xs"
        className="absolute top-2 right-2 opacity-0 group-hover:opacity-100"
        onClick={() => navigator.clipboard.writeText(code).then(() => toast.success(t("general:Copied")))}
        aria-label={t("general:Copy")}
      >
        <Copy />
      </Button>
    </div>
  );
}

export function IntegrationTab({ project }: { project: ProjectInfo }) {
  const [secrets, setSecrets] = useState<{ webhookSecret: string; accessToken: string } | null>(null);
  const origin = location.origin;
  const token = secrets?.accessToken ?? "<access token>";
  const name = project.name;

  const show = () =>
    Backend.getProjectSecrets(name)
      .then(setSecrets)
      .catch((e: Error) => toast.error(e.message));
  const regenerate = () => {
    if (!confirm(t("integration:Make a new webhook secret and access token? The old ones stop working at once."))) return;
    Backend.regenerateProjectTokens(name)
      .then(show)
      .catch((e: Error) => toast.error(e.message));
  };

  return (
    <div className="grid gap-4 lg:grid-cols-2">
      <Card className="gap-3 p-5">
        <div className="flex items-center justify-between gap-3">
          <div className="font-medium">{t("integration:Secrets")}</div>
          <div className="flex gap-2">
            <Button variant="outline" size="sm" onClick={show}>
              <Eye />
              {t("integration:Show")}
            </Button>
            <Button variant="outline" size="sm" onClick={regenerate}>
              <KeyRound />
              {t("integration:Regenerate")}
            </Button>
          </div>
        </div>
        <div className="grid gap-2 text-sm">
          <div className="flex justify-between gap-3">
            <span className="text-muted-foreground">{t("integration:Webhook secret")}</span>
            <code className="font-mono text-xs break-all">{secrets?.webhookSecret ?? "••••••••"}</code>
          </div>
          <div className="flex justify-between gap-3">
            <span className="text-muted-foreground">{t("integration:Access token")}</span>
            <code className="font-mono text-xs break-all">{secrets?.accessToken ?? "••••••••"}</code>
          </div>
        </div>
        <p className="text-xs text-muted-foreground">{t("integration:The access token lets CI jobs download translations, upload generated source files and start a pull of this project. Keep it in a CI secret.")}</p>
      </Card>

      <Card className="gap-3 p-5">
        <div className="font-medium">{t("integration:GitHub webhook")}</div>
        <p className="text-sm text-muted-foreground">
          {t("integration:In the repository's Settings > Webhooks, add a webhook for push events with content type application/json and the secret above. Caslingo then pulls the new source strings on every push to {branch}.", { branch: project.branch })}
        </p>
        <CodeBlock code={`${origin}/api/webhook?project=${name}`} />
      </Card>

      <Card className="gap-3 p-5">
        <div className="font-medium">{t("integration:Take the translations at build time")}</div>
        <p className="text-sm text-muted-foreground">
          {t("integration:For sites that build every language, like Docusaurus, instead of committing the translations: download them all before the build. The files land where the components' translation patterns put them.")}
        </p>
        <CodeBlock
          code={`curl -fsSL -H "Authorization: Bearer ${token}" \\\n  "${origin}/api/download-translations?project=${name}" -o translations.zip\nunzip -o translations.zip && rm translations.zip`}
        />
      </Card>

      <Card className="gap-3 p-5">
        <div className="font-medium">{t("integration:Upload generated source files")}</div>
        <p className="text-sm text-muted-foreground">
          {t("integration:For components whose source is uploaded (files made at build time, like the i18n/en/*.json of docusaurus write-translations). Caslingo pulls after each upload.")}
        </p>
        <CodeBlock
          code={`for f in i18n/en/*.json; do\n  curl -fsSL -H "Authorization: Bearer ${token}" -H "Content-Type: application/octet-stream" \\\n    --data-binary @"$f" "${origin}/api/upload-source?project=${name}&path=$f&sync=false"\ndone\ncurl -fsSL -X POST -H "Authorization: Bearer ${token}" "${origin}/api/start-job?project=${name}&kind=sync"`}
        />
      </Card>
    </div>
  );
}
