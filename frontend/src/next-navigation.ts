import { useMemo } from "react";

type Router = {
  push: (href: string, options?: unknown) => void;
  replace: (href: string, options?: unknown) => void;
  back: () => void;
  prefetch: (href: string, options?: unknown) => void;
  refresh: () => void;
};

export function useRouter(): Router {
  return useMemo<Router>(() => ({
    push: () => {},
    replace: () => {},
    back: () => {},
    prefetch: () => {},
    refresh: () => {},
  }), []);
}

export function useSearchParams(): URLSearchParams {
  return useMemo(() => new URLSearchParams(window.location.search), []);
}

export function usePathname(): string {
  return window.location.pathname;
}
