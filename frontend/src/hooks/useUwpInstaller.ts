import { useState, useRef, useCallback, useEffect } from "react";
import {
  StartFileDownload,
  CancelFileDownload,
} from "bindings/github.com/BedrockNexusLauncher/BedrockNexusLauncher/minecraft";
import {
  UwpListVersions,
  UwpPrepareInstall,
  UwpFinishInstall,
  UwpGetPrereqs,
} from "bindings/github.com/BedrockNexusLauncher/BedrockNexusLauncher/versionservice";
import { Events } from "@wailsio/runtime";

export type UwpInstallStep =
  | "idle"
  | "downloading"
  | "installing"
  | "success"
  | "error";

export interface UwpCatalogEntry {
  version: string;
  uuid: string;
  type: string;
  packageType: string;
}

/**
 * UWP install flow: prepare (validate + resolve URL) → download with
 * progress (shared file downloader) → finish (verify + register).
 * Mirrors usePackInstaller event conventions.
 */
export const useUwpInstaller = (onInstalled?: () => void) => {
  const [step, setStep] = useState<UwpInstallStep>("idle");
  const [entries, setEntries] = useState<UwpCatalogEntry[]>([]);
  const [devMode, setDevMode] = useState<boolean | null>(null);
  const [progress, setProgress] = useState<{
    downloaded: number;
    total: number;
  } | null>(null);
  const [installError, setInstallError] = useState("");
  const isCancelling = useRef(false);
  const cleanupRef = useRef<() => void>(() => {});
  const finishRef = useRef<{
    dest: string;
    updateID: string;
    instanceName: string;
  } | null>(null);

  useEffect(() => {
    return () => {
      cleanupRef.current();
    };
  }, []);

  const loadChannel = useCallback(async (channel: string) => {
    setEntries([]);
    try {
      const list = (await UwpListVersions(channel)) as any;
      setEntries(Array.isArray(list) ? list : []);
    } catch {
      setEntries([]);
    }
    try {
      const pre = (await UwpGetPrereqs()) as any;
      setDevMode(!!pre?.developerMode);
    } catch {
      setDevMode(null);
    }
  }, []);

  const startInstall = useCallback(
    async (updateID: string, instanceName: string) => {
      const name = instanceName.trim();
      if (!updateID || !name) return;
      setInstallError("");
      setProgress(null);
      isCancelling.current = false;
      finishRef.current = null;

      let plan: any;
      try {
        plan = await UwpPrepareInstall(updateID, name);
      } catch (e: any) {
        setInstallError(e?.message || "Prepare failed");
        setStep("error");
        return;
      }
      if (!plan || plan.Error) {
        setInstallError(String(plan?.Error || "Prepare failed"));
        setStep("error");
        return;
      }

      setStep("downloading");
      try {
        const dest = (await StartFileDownload(
          String(plan.URL),
          String(plan.Filename),
        )) as unknown as string;
        finishRef.current = { dest: String(dest), updateID, instanceName: name };

        const cleanup = () => {
          Events.Off("file.download.progress");
          Events.Off("file.download.done");
          Events.Off("file.download.error");
        };
        cleanupRef.current = cleanup;

        Events.On("file.download.progress", (event: any) => {
          const data = event.data || {};
          setProgress({
            downloaded: Number(data.Downloaded || 0),
            total: Number(data.Total || 0),
          });
        });

        Events.On("file.download.done", async () => {
          cleanup();
          const fin = finishRef.current;
          if (!fin) {
            setInstallError("Download finished without context");
            setStep("error");
            return;
          }
          setStep("installing");
          try {
            const code = (await UwpFinishInstall(
              fin.dest,
              fin.updateID,
              fin.instanceName,
            )) as unknown as string;
            if (code) {
              setInstallError(String(code));
              setStep("error");
              return;
            }
            setStep("success");
            try {
              onInstalled && onInstalled();
            } catch {}
          } catch (e: any) {
            setInstallError(e?.message || "Install failed");
            setStep("error");
          }
        });

        Events.On("file.download.error", (event: any) => {
          cleanup();
          if (isCancelling.current) return;
          setInstallError(String(event.data || "Download failed"));
          setStep("error");
        });
      } catch (e: any) {
        if (isCancelling.current) return;
        setInstallError(e?.message || "Download start failed");
        setStep("error");
      }
    },
    [onInstalled],
  );

  const cancel = useCallback(async () => {
    isCancelling.current = true;
    cleanupRef.current();
    try {
      await CancelFileDownload();
    } catch {}
    setStep("idle");
  }, []);

  const reset = useCallback(() => {
    setStep("idle");
    setProgress(null);
    setInstallError("");
    finishRef.current = null;
  }, []);

  return {
    step,
    entries,
    devMode,
    progress,
    installError,
    loadChannel,
    startInstall,
    cancel,
    reset,
  };
};
