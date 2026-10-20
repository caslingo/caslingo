import { Check, Languages, LogOut, User as UserIcon } from "lucide-react";
import type { ReactNode } from "react";
import { Link, NavLink } from "react-router-dom";
import { toast } from "sonner";
import * as Backend from "@/backend/api";
import { Avatar, AvatarFallback, AvatarImage } from "@/components/ui/avatar";
import { Button } from "@/components/ui/button";
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu";
import { getUiLanguage, setUiLanguage, t, uiLanguages } from "@/lib/i18n";
import { getProfileUrl, getSigninUrl, rememberReturnPath, useSite } from "@/lib/site";
import { cn } from "@/lib/utils";

export function Logo({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 32 32" className={className} aria-hidden>
      <rect width="32" height="32" rx="7" fill="var(--primary)" />
      <path d="M9 10h9M13.5 8v2M16 10c-.6 3.6-3 6.6-6.5 8.4M11.5 13.5c1.4 2 3.4 3.6 5.8 4.6" fill="none" stroke="#fff" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M17 24l3.5-9 3.5 9M18.2 21h4.6" fill="none" stroke="#fff" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  );
}

export function SiteHeader() {
  const { config } = useSite();
  const siteName = config.siteName || "Caslingo";
  return (
    <header className="sticky top-0 z-30 border-b bg-card/95 backdrop-blur">
      <div className="wrap flex h-14 items-center gap-6">
        <Link to="/" className="flex shrink-0 items-center gap-2" aria-label={siteName}>
          {config.logoUrl ? <img src={config.logoUrl} alt={siteName} className="h-7 w-auto" /> : <Logo className="size-7" />}
          {!config.logoUrl && <span className="text-lg font-semibold tracking-tight">{siteName}</span>}
        </Link>
        <nav className="flex h-full items-stretch">
          <NavLink
            to="/"
            end
            className={({ isActive }) =>
              cn("flex items-center px-3 text-sm text-muted-foreground transition-colors hover:text-foreground", isActive && "font-medium text-foreground")
            }
          >
            {t("general:Projects")}
          </NavLink>
        </nav>
        <div className="ml-auto flex items-center gap-2">
          <LanguageMenu />
          <AccountMenu />
        </div>
      </div>
    </header>
  );
}

function LanguageMenu() {
  const current = getUiLanguage();
  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <Button variant="ghost" size="sm" aria-label={t("general:Interface language")}>
          <Languages />
          <span className="hidden sm:inline">{uiLanguages.find(l => l.code === current)?.name}</span>
        </Button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end">
        {uiLanguages.map(language => (
          <DropdownMenuItem key={language.code} onSelect={() => language.code !== current && setUiLanguage(language.code)}>
            <Check className={cn(language.code !== current && "invisible")} />
            {language.name}
          </DropdownMenuItem>
        ))}
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

function AccountMenu() {
  const { config, account, accountLoaded, signout, setAccount } = useSite();
  if (!accountLoaded) return <div className="h-8 w-16" />;
  if (!account) {
    if (!config.casdoor?.serverUrl && config.devSignin) {
      return (
        <Button
          size="sm"
          onClick={() =>
            Backend.devSignin()
              .then(setAccount)
              .catch((e: Error) => toast.error(e.message))
          }
        >
          {t("general:Sign in")}
        </Button>
      );
    }
    return (
      <Button asChild size="sm">
        <a href={getSigninUrl(config)} onClick={() => rememberReturnPath()}>
          {t("general:Sign in")}
        </a>
      </Button>
    );
  }

  return (
    <DropdownMenu>
      <DropdownMenuTrigger asChild>
        <button className="flex items-center gap-2 rounded-full outline-none focus-visible:ring-[3px] focus-visible:ring-ring/50" aria-label={t("general:Account")}>
          <Avatar className="size-8 border">
            <AvatarImage src={account.avatar} alt="" />
            <AvatarFallback className="bg-secondary text-xs text-secondary-foreground">{account.displayName.slice(0, 2)}</AvatarFallback>
          </Avatar>
        </button>
      </DropdownMenuTrigger>
      <DropdownMenuContent align="end" className="w-56">
        <DropdownMenuLabel className="font-normal">
          <div className="font-medium">{account.displayName}</div>
          <div className="truncate text-xs text-muted-foreground">
            {account.name}
            {account.isAdmin && ` (${t("general:Administrator")})`}
          </div>
        </DropdownMenuLabel>
        <DropdownMenuSeparator />
        {config.casdoor?.serverUrl && (
          <DropdownMenuItem asChild>
            <a href={getProfileUrl(config)} target="_blank" rel="noopener">
              <UserIcon />
              {t("general:My account")}
            </a>
          </DropdownMenuItem>
        )}
        <DropdownMenuItem
          onSelect={() =>
            signout()
              .then(() => location.assign("/"))
              .catch((e: Error) => toast.error(e.message))
          }
        >
          <LogOut />
          {t("general:Sign out")}
        </DropdownMenuItem>
      </DropdownMenuContent>
    </DropdownMenu>
  );
}

export function SiteFooter() {
  const { config } = useSite();
  return (
    <footer className="mt-12 border-t bg-card">
      <div className="wrap flex flex-col items-center justify-between gap-2 py-5 text-xs text-muted-foreground sm:flex-row">
        {config.footerHtml ? (
          <div dangerouslySetInnerHTML={{ __html: config.footerHtml }} />
        ) : (
          <div>
            {t("general:Powered by")}{" "}
            <a href="https://github.com/caslingo/caslingo" target="_blank" rel="noopener" className="font-medium hover:text-foreground">
              Caslingo
            </a>
          </div>
        )}
        <div>{t("general:Open-source translation platform for software and docs")}</div>
      </div>
    </footer>
  );
}

export function Page({ children, className }: { children: ReactNode; className?: string }) {
  return <main className={cn("wrap py-6", className)}>{children}</main>;
}

export function PageHeader({ title, description, actions, children }: { title: ReactNode; description?: ReactNode; actions?: ReactNode; children?: ReactNode }) {
  return (
    <div className="mb-6 flex flex-col gap-4 sm:flex-row sm:items-start sm:justify-between">
      <div className="min-w-0">
        {children}
        <h1 className="text-2xl font-semibold tracking-tight">{title}</h1>
        {description && <div className="mt-1 text-sm text-muted-foreground">{description}</div>}
      </div>
      {actions && <div className="flex shrink-0 flex-wrap items-center gap-2">{actions}</div>}
    </div>
  );
}
