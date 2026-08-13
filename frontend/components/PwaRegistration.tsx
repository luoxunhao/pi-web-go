"use client";

import { useEffect } from "react";
import { APP_VERSION, IS_PROD } from "../src/env";

export function PwaRegistration() {
  useEffect(() => {
    if (!IS_PROD || !("serviceWorker" in navigator)) {
      return;
    }

    const register = () => {
      const appVersion = APP_VERSION;
      const scriptUrl = `/sw.js?v=${encodeURIComponent(appVersion)}`;

      void navigator.serviceWorker.register(scriptUrl, {
        scope: "/",
        updateViaCache: "none",
      }).catch((error: unknown) => {
        console.error("Failed to register the Pi Web service worker:", error);
      });
    };

    if (document.readyState === "complete") {
      register();
      return;
    }

    window.addEventListener("load", register, { once: true });
    return () => window.removeEventListener("load", register);
  }, []);

  return null;
}
