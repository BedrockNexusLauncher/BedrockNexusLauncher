import React from "react";
import {
  Button,
  Chip,
  Table,
  TableBody,
  TableCell,
  TableColumn,
  TableHeader,
  TableRow,
  Tooltip,
} from "@heroui/react";
import { useTranslation } from "react-i18next";
import { LuFileDigit } from "react-icons/lu";
import { COMPONENT_STYLES } from "@/constants/componentStyles";
import type { LIPPackageFileInfo } from "@/utils/content";

export type BedrinthFileState = {
  file: LIPPackageFileInfo;
  supportedGameVersions: string[];
  hasLLRequirement: boolean;
};

interface BedrinthVersionListProps {
  files: BedrinthFileState[];
  mappingReady: boolean;
  mappingAvailable: boolean;
  installDisabled: boolean;
  actionRunning: boolean;
  onInstall: (version: string) => void;
}

/**
 * Bedrinth package version table: version, LeviLamina requirement,
 * other dependencies, supported game versions, install action.
 * Extracted from LIPPackagePage so the Bedrinth surface owns its list.
 */
export const BedrinthVersionList: React.FC<BedrinthVersionListProps> = ({
  files,
  mappingReady,
  mappingAvailable,
  installDisabled,
  actionRunning,
  onInstall,
}) => {
  const { t } = useTranslation();

  if (files.length === 0) {
    return (
      <div className="flex flex-col items-center justify-center py-12 text-default-400 border border-dashed border-default-200 rounded-xl">
        <LuFileDigit size={48} className="mb-4 opacity-50" />
        <p className="text-lg font-medium">{t("common.no_results")}</p>
      </div>
    );
  }

  return (
    <Table
      aria-label={t("lip.files.table_aria_label")}
      removeWrapper
      classNames={COMPONENT_STYLES.table}
    >
      <TableHeader>
        <TableColumn>{t("common.version")}</TableColumn>
        <TableColumn>{t("lip.files.ll_requirement")}</TableColumn>
        <TableColumn>{t("lip.files.dependencies")}</TableColumn>
        <TableColumn>{t("lip.files.game_versions_label")}</TableColumn>
        <TableColumn>{t("lip.files.action")}</TableColumn>
      </TableHeader>
      <TableBody>
        {files.map((fileState) => {
          const { file } = fileState;
          const dependencyEntries = Object.entries(file.otherDependencies);

          return (
            <TableRow key={file.version}>
              <TableCell>
                <Chip size="sm" variant="flat" className="font-mono">
                  {file.version}
                </Chip>
              </TableCell>
              <TableCell>
                {file.llDependencyRanges.length > 0 ? (
                  <div className="text-xs text-default-700 dark:text-zinc-300 break-all">
                    {file.llDependencyRanges.join(", ")}
                  </div>
                ) : (
                  <span className="text-default-400">-</span>
                )}
              </TableCell>
              <TableCell>
                {dependencyEntries.length > 0 ? (
                  <Tooltip
                    content={
                      <div className="max-w-sm p-2 space-y-1">
                        {dependencyEntries.map(([key, value]) => (
                          <div key={key} className="text-xs break-all">
                            <span className="font-semibold">{key}</span>:{" "}
                            {value}
                          </div>
                        ))}
                      </div>
                    }
                  >
                    <span className="cursor-help text-xs text-default-700 dark:text-zinc-300">
                      {t("lip.files.dependencies_count", {
                        count: dependencyEntries.length,
                      })}
                    </span>
                  </Tooltip>
                ) : (
                  <span className="text-default-400">-</span>
                )}
              </TableCell>
              <TableCell>
                {!fileState.hasLLRequirement ? (
                  <Chip size="sm" variant="flat" color="default">
                    {t("lip.files.game_versions_unrestricted")}
                  </Chip>
                ) : !mappingReady ? (
                  <Chip size="sm" variant="flat" color="default">
                    {t("common.loading")}
                  </Chip>
                ) : !mappingAvailable ? (
                  <Chip size="sm" variant="flat" color="warning">
                    {t("lip.files.game_versions_unavailable")}
                  </Chip>
                ) : fileState.supportedGameVersions.length > 0 ? (
                  <div className="flex flex-wrap gap-1">
                    {fileState.supportedGameVersions.map((gameVersion) => (
                      <Chip
                        key={`${file.version}-${gameVersion}`}
                        size="sm"
                        variant="flat"
                        color="default"
                      >
                        {gameVersion}
                      </Chip>
                    ))}
                  </div>
                ) : (
                  <Chip size="sm" variant="flat" color="danger">
                    {t("contentpage.none")}
                  </Chip>
                )}
              </TableCell>
              <TableCell>
                <Button
                  size="sm"
                  color="primary"
                  variant="flat"
                  onPress={() => onInstall(file.version)}
                  isDisabled={
                    installDisabled ||
                    (fileState.hasLLRequirement && !mappingAvailable) ||
                    actionRunning
                  }
                >
                  {t("lip.files.install")}
                </Button>
              </TableCell>
            </TableRow>
          );
        })}
      </TableBody>
    </Table>
  );
};
