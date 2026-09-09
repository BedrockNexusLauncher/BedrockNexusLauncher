import { useState, useRef, useCallback, useEffect } from "react";
import {
  StartFileDownload,
  CancelFileDownload,
} from "bindings/github.com/BedrockNexusLauncher/BedrockNexusLauncher/minecraft";
import {
  GetContentRoots,
  ImportMcpackPath,
  ImportMcpackPathWithPlayer,
  ImportMcaddonPath,
  ImportMcaddonPathWithPlayer,
  ImportMcworldPath,
} from "bindings/github.com/BedrockNexusLauncher/BedrockNexusLauncher/contentservice";
import {
  ListVersionMetasWithRegistered,
  GetVersionLogoDataUrl,
} from "bindings/github.com/BedrockNexusLauncher/BedrockNexusLauncher/versionservice";
import { GetLocalUserGamertag } from "bindings/github.com/BedrockNexusLauncher/BedrockNexusLauncher/userservice";
import { Events } from "@wailsio/runtime";
import { VersionMeta } from "bindings/github.com/BedrockNexusLauncher/BedrockNexusLauncher/internal/versions/models";
import {
  getPlayerGamertagMap,
  listPlayers,
} from "@/utils/content";
import { readCurrentVersionName } from "@/utils/currentVersion";
import { compareVersions } from "@/utils/version";

export type InstallStep =
  | "idle"
  | "downloading"
  | "version_select"
  | "player_select"
  | "importing"
  | "success"
  | "error";

export interface InstallFileInfo {
  name: string;
  path: string;
  type: string;
}

/**
 * دانلود → انتخاب نسخه/بازیکن → ایمپورت: همان چرخه‌ی صفحه‌ی CurseForge
 * برای هر فایل دانلودی مارکت‌پلیس (mcaddon/mcpack/mcworld/skinpack).
 */
export const usePackInstaller = () => {
  const [step, setStep] = useState<InstallStep>("idle");
  const [installFile, setInstallFile] = useState<InstallFileInfo | null>(null);
  const [downloadProgress, setDownloadProgress] = useState<{
    downloaded: number;
    total: number;
  } | null>(null);
  const [availableVersions, setAvailableVersions] = useState<VersionMeta[]>([]);
  const [versionLogos, setVersionLogos] = useState<Record<string, string>>({});
  const [availablePlayers, setAvailablePlayers] = useState<string[]>([]);
  const [playerGamertagMap, setPlayerGamertagMap] = useState<
    Record<string, string>
  >({});
  const [selectedVersion, setSelectedVersion] = useState("");
  const [selectedPlayer, setSelectedPlayer] = useState("");
  const [installError, setInstallError] = useState("");
  const [dupOpen, setDupOpen] = useState(false);
  const [dupName, setDupName] = useState("");
  const dupResolveRef = useRef<((overwrite: boolean) => void) | null>(null);
  const isCancelling = useRef(false);
  const cleanupRef = useRef<() => void>(() => {});

  useEffect(() => {
    return () => {
      cleanupRef.current();
    };
  }, []);

  const detectType = async (fileName: string, path: string, declaredType: string) => {
    if (declaredType === "skinpack") return "skin_pack";
    const lowerName = fileName.toLowerCase();
    if (lowerName.endsWith(".mcworld")) return "mcworld";
    if (lowerName.endsWith(".mcaddon")) return "mcaddon";
    if (lowerName.endsWith(".mcpack")) {
      try {
        const { IsMcpackSkinPackPath } = await import(
          "bindings/github.com/BedrockNexusLauncher/BedrockNexusLauncher/contentservice"
        );
        if (await IsMcpackSkinPackPath(path)) return "skin_pack";
      } catch {}
      return "mcpack";
    }
    if (declaredType === "mcworld") return "mcworld";
    if (declaredType === "mcaddon") return "mcaddon";
    return "mcpack";
  };

  const start = useCallback(
    async (downloadUrl: string, fileName: string, declaredType = "") => {
      if (!downloadUrl) return;

      setInstallFile(null);
      setDownloadProgress(null);
      setInstallError("");
      isCancelling.current = false;
      setStep("downloading");

      try {
        const dest = await StartFileDownload(downloadUrl, fileName);

        const cleanup = () => {
          Events.Off("file.download.progress");
          Events.Off("file.download.done");
          Events.Off("file.download.error");
        };
        cleanupRef.current = cleanup;

        Events.On("file.download.progress", (event: any) => {
          const data = event.data || {};
          setDownloadProgress({
            downloaded: Number(data.Downloaded || 0),
            total: Number(data.Total || 0),
          });
        });

        Events.On("file.download.done", async () => {
          cleanup();
          try {
            const type = await detectType(fileName, dest, declaredType);
            setInstallFile({ name: fileName, path: dest, type });

            const metas = (await ListVersionMetasWithRegistered()) || [];
            metas.sort((a, b) =>
              -compareVersions(a.gameVersion || "0", b.gameVersion || "0"),
            );
            setAvailableVersions(metas);

            const currentName = readCurrentVersionName();
            let defaultSelect = "";
            if (currentName && metas.some((m) => m.name === currentName)) {
              defaultSelect = currentName;
            } else if (metas.length > 0) {
              defaultSelect = metas[0].name;
            }
            setSelectedVersion(defaultSelect);

            const logoMap: Record<string, string> = {};
            await Promise.all(
              metas.map(async (m) => {
                try {
                  const url = await GetVersionLogoDataUrl(m.name);
                  if (url) logoMap[m.name] = url;
                } catch {}
              }),
            );
            setVersionLogos(logoMap);

            setStep("version_select");
          } catch (e: any) {
            setInstallError(e?.message || "Detection failed");
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
    [],
  );

  const cancelDownload = useCallback(async () => {
    isCancelling.current = true;
    cleanupRef.current();
    try {
      await CancelFileDownload();
    } catch {}
    setStep("idle");
  }, []);

  const next = useCallback(async () => {
    if (!installFile) return;

    if (step === "player_select") {
      await executeImport();
      return;
    }

    let skipPlayerSelect = false;
    if (installFile.type === "skin_pack") {
      const targetMeta = availableVersions.find(
        (v) => v.name === selectedVersion,
      );
      if (
        targetMeta &&
        compareVersions(targetMeta.gameVersion || "0", "1.26.0") > 0
      ) {
        skipPlayerSelect = true;
      }
    }

    if (
      installFile.type === "mcworld" ||
      (installFile.type === "skin_pack" && !skipPlayerSelect)
    ) {
      setStep("player_select");
      try {
        const roots = await GetContentRoots(selectedVersion);
        if (roots && roots.usersRoot) {
          const players = await listPlayers(roots.usersRoot);
          setAvailablePlayers(players);
          if (players.length > 0) {
            setSelectedPlayer(players[0]);
          } else {
            setSelectedPlayer("");
          }

          (async () => {
            try {
              const map = await getPlayerGamertagMap(roots.usersRoot);
              setPlayerGamertagMap(map);

              const tag = await GetLocalUserGamertag();
              if (tag) {
                for (const p of players) {
                  if (map[p] === tag) {
                    if (p !== players[0]) setSelectedPlayer(p);
                    break;
                  }
                }
              }
            } catch {}
          })();
        } else {
          setAvailablePlayers([]);
          setPlayerGamertagMap({});
        }
      } catch (e) {
        console.error(e);
        setAvailablePlayers([]);
        setPlayerGamertagMap({});
      }
    } else {
      await executeImport();
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [installFile, step, availableVersions, selectedVersion]);

  const executeImport = async () => {
    if (!installFile || !selectedVersion) return;

    setStep("importing");
    setInstallError("");

    try {
      const { name, path, type } = installFile;
      const runImport = async (overwrite: boolean): Promise<string> => {
        if (type === "mcworld") {
          if (!selectedPlayer) throw new Error("No player selected");
          return String(
            await ImportMcworldPath(
              selectedVersion,
              selectedPlayer,
              path,
              overwrite,
            ),
          );
        }
        if (type === "mcaddon") {
          if (selectedPlayer) {
            return String(
              await ImportMcaddonPathWithPlayer(
                selectedVersion,
                selectedPlayer,
                path,
                overwrite,
              ),
            );
          }
          return String(
            await ImportMcaddonPath(selectedVersion, path, overwrite),
          );
        }
        if (selectedPlayer && type === "skin_pack") {
          return String(
            await ImportMcpackPathWithPlayer(
              selectedVersion,
              selectedPlayer,
              path,
              overwrite,
            ),
          );
        }
        return String(await ImportMcpackPath(selectedVersion, path, overwrite));
      };

      let err = await runImport(false);
      if (
        err &&
        (String(err) === "ERR_DUPLICATE_FOLDER" ||
          String(err) === "ERR_DUPLICATE_UUID")
      ) {
        setDupName(name);
        const overwrite = await new Promise<boolean>((resolve) => {
          dupResolveRef.current = resolve;
          setDupOpen(true);
        });
        if (!overwrite) {
          setStep("idle");
          return;
        }
        err = await runImport(true);
      }

      if (err) {
        throw new Error(err);
      }

      setStep("success");
    } catch (e: any) {
      setInstallError(e?.message || "Import failed");
      setStep("error");
    }
  };

  const resolveDuplicate = useCallback((overwrite: boolean) => {
    try {
      if (dupResolveRef.current) dupResolveRef.current(overwrite);
    } finally {
      setDupOpen(false);
    }
  }, []);

  const dismiss = useCallback(() => {
    setStep("idle");
  }, []);

  return {
    step,
    installFile,
    downloadProgress,
    availableVersions,
    versionLogos,
    availablePlayers,
    playerGamertagMap,
    selectedVersion,
    selectedPlayer,
    installError,
    dupOpen,
    dupName,
    setSelectedVersion,
    setSelectedPlayer,
    start,
    next,
    cancelDownload,
    resolveDuplicate,
    dismiss,
  };
};
