export const ROUTES = {
  home: "/",
  download: "/download",
  downloadTasks: "/download/tasks",
  instances: "/instances",
  instanceSettings: "/instances/settings",
  content: "/content",
  settings: "/settings",
  about: "/about",
  mcpedl: "/mcpedl",
  updating: "/updating",
  onboarding: "/onboarding",
  elevationConsent: "/elevation-consent",
} as const;

export const isRouteActive = (pathname: string, route: string): boolean => {
  const currentPath = String(pathname || "");
  const targetRoute = String(route || "");

  if (!targetRoute) return false;
  if (targetRoute === ROUTES.home) return currentPath === ROUTES.home;

  return (
    currentPath === targetRoute || currentPath.startsWith(`${targetRoute}/`)
  );
};
