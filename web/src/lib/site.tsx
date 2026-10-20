import { createContext, useCallback, useContext, useEffect, useMemo, useState, type ReactNode } from "react";
import * as Backend from "@/backend/api";
import type { SiteConfig, User } from "@/backend/models";

// Site-wide state: the server config and the signed-in user.

interface SiteState {
  config: SiteConfig;
  account: User | null;
  /** false until the first get-account answer */
  accountLoaded: boolean;
  signout: () => Promise<void>;
  setAccount: (user: User | null) => void;
}

const SiteContext = createContext<SiteState | null>(null);

export function SiteProvider({ children }: { children: ReactNode }) {
  const [config, setConfig] = useState<SiteConfig>({});
  const [account, setAccount] = useState<User | null>(null);
  const [accountLoaded, setAccountLoaded] = useState(false);

  useEffect(() => {
    Backend.getConfig().then(setConfig).catch(() => {});
    Backend.getAccount()
      .then(setAccount)
      .catch(() => setAccount(null))
      .finally(() => setAccountLoaded(true));
  }, []);

  const signout = useCallback(async () => {
    await Backend.signout();
    setAccount(null);
  }, []);

  const value = useMemo(() => ({ config, account, accountLoaded, signout, setAccount }), [config, account, accountLoaded, signout]);
  return <SiteContext.Provider value={value}>{children}</SiteContext.Provider>;
}

export function useSite() {
  const site = useContext(SiteContext);
  if (!site) throw new Error("useSite must be used inside <SiteProvider>");
  return site;
}

const signinStateKey = "caslingo:signin-state";
const returnPathKey = "caslingo:signin-return";

/** The OAuth state, one per browser session, so every sign-in link on the page carries the same one */
function getSigninState() {
  try {
    let state = sessionStorage.getItem(signinStateKey);
    if (!state) {
      state = Math.random().toString(36).slice(2);
      sessionStorage.setItem(signinStateKey, state);
    }
    return state;
  } catch {
    return "";
  }
}

/** Casdoor sign-in page; call rememberReturnPath when the link is followed so /callback comes back to the page */
export function getSigninUrl(config: SiteConfig) {
  const casdoor = config.casdoor;
  if (!casdoor?.serverUrl) return "/";

  const params = new URLSearchParams({
    client_id: casdoor.clientId,
    response_type: "code",
    redirect_uri: `${location.origin}/callback`,
    scope: "read",
    state: getSigninState(),
  });
  return `${casdoor.serverUrl}/login/oauth/authorize?${params}`;
}

export function rememberReturnPath(returnPath = location.pathname + location.search) {
  try {
    sessionStorage.setItem(returnPathKey, returnPath);
  } catch {
    // ignore: the callback then goes to the home page
  }
}

/** Where /callback should go next, or null when the state is not the one this browser session sent */
export function takeSigninReturnPath(state: string | null): string | null {
  try {
    const saved = sessionStorage.getItem(signinStateKey);
    const returnPath = sessionStorage.getItem(returnPathKey) || "/";
    sessionStorage.removeItem(signinStateKey);
    sessionStorage.removeItem(returnPathKey);
    // no saved state (another tab, or storage blocked): accept, as the code alone is only good for this application
    if (!saved) return returnPath;
    return saved === state ? returnPath : null;
  } catch {
    return "/";
  }
}

export function getSignupUrl(config: SiteConfig) {
  const casdoor = config.casdoor;
  return casdoor?.serverUrl ? `${casdoor.serverUrl}/signup/${casdoor.appName}` : "/";
}

export function getProfileUrl(config: SiteConfig) {
  return config.casdoor?.serverUrl ? `${config.casdoor.serverUrl}/account` : "/";
}
