import { Loader2 } from "lucide-react";
import { useEffect, useState, type ReactNode } from "react";
import { Link, useNavigate, useSearchParams } from "react-router-dom";
import * as Backend from "@/backend/api";
import { Page } from "@/components/app/layout";
import { Button } from "@/components/ui/button";
import { useDocumentTitle } from "@/lib/format";
import { t } from "@/lib/i18n";
import { getSigninUrl, rememberReturnPath, takeSigninReturnPath, useSite } from "@/lib/site";

export function CallbackPage() {
  const [params] = useSearchParams();
  const navigate = useNavigate();
  const { setAccount, config } = useSite();
  const [error, setError] = useState("");
  useDocumentTitle(t("general:Sign in"));

  useEffect(() => {
    const code = params.get("code");
    const returnPath = takeSigninReturnPath(params.get("state"));
    if (!code || returnPath === null) {
      setError(t("general:The sign-in link is not valid, please sign in again"));
      return;
    }
    Backend.signin(code)
      .then(user => {
        setAccount(user);
        navigate(returnPath, { replace: true });
      })
      .catch((e: Error) => setError(e.message));
    // runs once for the code in the address
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, []);

  return (
    <Page className="flex min-h-[50svh] flex-col items-center justify-center gap-4 text-sm text-muted-foreground">
      {error ? (
        <>
          <span>{error}</span>
          <Button asChild>
            <a href={getSigninUrl(config)} onClick={() => rememberReturnPath("/")}>
              {t("general:Sign in again")}
            </a>
          </Button>
        </>
      ) : (
        <>
          <Loader2 className="size-6 animate-spin" />
          {t("general:Signing in…")}
        </>
      )}
    </Page>
  );
}

export function SigninRedirect() {
  const { config } = useSite();
  useEffect(() => {
    if (config.casdoor?.serverUrl) {
      rememberReturnPath("/");
      location.replace(getSigninUrl(config));
    }
  }, [config]);
  return <PageLoading />;
}

export function PageLoading() {
  return (
    <Page className="flex min-h-[50svh] items-center justify-center text-muted-foreground">
      <Loader2 className="size-6 animate-spin" />
    </Page>
  );
}

export function PageError({ message }: { message: string }) {
  return (
    <Page className="flex min-h-[50svh] flex-col items-center justify-center gap-4 text-center">
      <div className="text-muted-foreground">{message}</div>
      <Button asChild variant="outline">
        <Link to="/">{t("general:Back to the projects")}</Link>
      </Button>
    </Page>
  );
}

export function NotFoundPage() {
  useDocumentTitle(t("general:Page not found"));
  return <PageError message={t("general:Page not found")} />;
}

/** Shows the page only to a signed-in user, sending others to sign in. */
export function RequireSignin({ children, admin = false }: { children: ReactNode; admin?: boolean }) {
  const { account, accountLoaded, config } = useSite();
  useEffect(() => {
    if (accountLoaded && !account && config.casdoor?.serverUrl) {
      rememberReturnPath();
      location.replace(getSigninUrl(config));
    }
  }, [accountLoaded, account, config]);

  if (!accountLoaded) return <PageLoading />;
  if (!account) return <PageError message={t("general:Please sign in first")} />;
  if (admin && !account.isAdmin) return <PageError message={t("general:Only administrators can see this page")} />;
  return children;
}
