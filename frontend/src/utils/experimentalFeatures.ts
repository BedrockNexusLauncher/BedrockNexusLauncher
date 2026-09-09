import { useEffect, useState } from "react";

export const EXPERIMENTAL_INSTANCE_BACKUP_KEY =
  "app.experimental.instanceBackup";
export const EXPERIMENTAL_MCPEDL_KEY = "app.experimental.mcpedl";
export const EXPERIMENTAL_LIP_BEDRINTH_FALLBACK_KEY =
  "app.experimental.lipBedrinthFallback";
export const EXPERIMENTAL_FEATURES_EVENT_NAME =
  "app-experimental-features-changed";

export const readExperimentalInstanceBackupEnabled = (): boolean => {
  try {
    return localStorage.getItem(EXPERIMENTAL_INSTANCE_BACKUP_KEY) === "true";
  } catch {
    return false;
  }
};

export const persistExperimentalInstanceBackupEnabled = (enabled: boolean) => {
  try {
    localStorage.setItem(
      EXPERIMENTAL_INSTANCE_BACKUP_KEY,
      enabled ? "true" : "false",
    );
    window.dispatchEvent(
      new CustomEvent(EXPERIMENTAL_FEATURES_EVENT_NAME, {
        detail: { instanceBackup: enabled },
      }),
    );
  } catch {}
};

export const readExperimentalMcpedlEnabled = (): boolean => {
  try {
    return localStorage.getItem(EXPERIMENTAL_MCPEDL_KEY) === "true";
  } catch {
    return false;
  }
};

export const persistExperimentalMcpedlEnabled = (enabled: boolean) => {
  try {
    localStorage.setItem(EXPERIMENTAL_MCPEDL_KEY, enabled ? "true" : "false");
    window.dispatchEvent(
      new CustomEvent(EXPERIMENTAL_FEATURES_EVENT_NAME, {
        detail: { mcpedl: enabled },
      }),
    );
  } catch {}
};

export const readExperimentalLipBedrinthFallbackEnabled = (): boolean => {
  try {
    return (
      localStorage.getItem(EXPERIMENTAL_LIP_BEDRINTH_FALLBACK_KEY) === "true"
    );
  } catch {
    return false;
  }
};

export const persistExperimentalLipBedrinthFallbackEnabled = (
  enabled: boolean,
) => {
  try {
    localStorage.setItem(
      EXPERIMENTAL_LIP_BEDRINTH_FALLBACK_KEY,
      enabled ? "true" : "false",
    );
    window.dispatchEvent(
      new CustomEvent(EXPERIMENTAL_FEATURES_EVENT_NAME, {
        detail: { lipBedrinthFallback: enabled },
      }),
    );
  } catch {}
};

/** Subscribe to a feature flag and stay in sync with changes made in Settings. */
export const useExperimentalFeature = (readEnabled: () => boolean): boolean => {
  const [enabled, setEnabled] = useState<boolean>(() => readEnabled());

  useEffect(() => {
    const handler = () => setEnabled(readEnabled());
    handler();
    window.addEventListener(EXPERIMENTAL_FEATURES_EVENT_NAME, handler);
    return () =>
      window.removeEventListener(EXPERIMENTAL_FEATURES_EVENT_NAME, handler);
  }, [readEnabled]);

  return enabled;
};
