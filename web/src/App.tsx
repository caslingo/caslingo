import { useEffect } from "react";
import { BrowserRouter, Route, Routes, useLocation } from "react-router-dom";
import { Toaster } from "sonner";
import { SiteFooter, SiteHeader } from "@/components/app/layout";
import { TooltipProvider } from "@/components/ui/tooltip";
import { SiteProvider } from "@/lib/site";
import { EditorPage } from "@/pages/editor-page";
import { HomePage } from "@/pages/home-page";
import { ProjectPage } from "@/pages/project-page";
import { SettingsPage } from "@/pages/settings-page";
import { CallbackPage, NotFoundPage, RequireSignin, SigninRedirect } from "@/pages/static-pages";

/** a new page starts at the top */
function ScrollOnNavigate() {
  const { pathname } = useLocation();
  useEffect(() => {
    window.scrollTo(0, 0);
  }, [pathname]);
  return null;
}

export default function App() {
  return (
    <BrowserRouter>
      <SiteProvider>
        <TooltipProvider delayDuration={200}>
          <ScrollOnNavigate />
          <div className="flex min-h-svh flex-col">
            <SiteHeader />
            <div className="flex flex-1 flex-col">
              <Routes>
                <Route path="/" element={<HomePage />} />
                <Route path="/projects/:name" element={<ProjectPage />} />
                <Route path="/projects/:name/settings" element={<RequireSignin><SettingsPage /></RequireSignin>} />
                <Route path="/projects/:name/:language" element={<EditorPage />} />
                <Route path="/signin" element={<SigninRedirect />} />
                <Route path="/callback" element={<CallbackPage />} />
                <Route path="*" element={<NotFoundPage />} />
              </Routes>
            </div>
            <SiteFooter />
          </div>
          <Toaster richColors position="top-center" />
        </TooltipProvider>
      </SiteProvider>
    </BrowserRouter>
  );
}
