import { StrictMode, Suspense } from "react";
import { createRoot } from "react-dom/client";
import { AppShell } from "../components/AppShell";
import { I18nProvider } from "../hooks/useI18n";
import "../app/globals.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <Suspense>
      <I18nProvider>
        <AppShell />
      </I18nProvider>
    </Suspense>
  </StrictMode>,
);
