import React, { useEffect, useMemo, useState } from "react";
import { useParams, useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { GetMarketplaceItem } from "bindings/github.com/BedrockNexusLauncher/BedrockNexusLauncher/minecraft";
import { getIndexURL } from "@/utils/MarketplaceContext";
import { Browser } from "@wailsio/runtime";
import { UnifiedModal } from "@/components/UnifiedModal";
import { motion } from "framer-motion";
import { PageContainer } from "@/components/PageContainer";
import { LAYOUT } from "@/constants/layout";
import {
  Button,
  Chip,
  Image,
  Card,
  CardBody,
  Tabs,
  Tab,
  Table,
  TableHeader,
  TableColumn,
  TableBody,
  TableRow,
  TableCell,
  Tooltip,
  Select,
  SelectItem,
  Progress,
  Spinner,
  Skeleton,
} from "@heroui/react";
import { FiCheckCircle } from "react-icons/fi";
import {
  LuDownload,
  LuCalendar,
  LuGlobe,
  LuGithub,
  LuGamepad2,
  LuUser,
  LuFileDigit,
  LuPackage,
} from "react-icons/lu";
import { COMPONENT_STYLES } from "@/constants/componentStyles";
import { formatDateStr, formatFileSize } from "@/utils/formatting";
import {
  usePackInstaller,
} from "@/hooks/usePackInstaller";
import ReactMarkdown from "react-markdown";
import remarkGfm from "remark-gfm";

const MarketplaceItemPage: React.FC = () => {
  const { id } = useParams<{ id: string }>();
  const navigate = useNavigate();
  const { t } = useTranslation();

  const [item, setItem] = useState<any | null>(null);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState(false);
  const [selectedTab, setSelectedTab] = useState<string>("description");

  const installer = usePackInstaller();

  useEffect(() => {
    if (!id) return;
    let cancelled = false;
    setLoading(true);
    setLoadError(false);
    GetMarketplaceItem(getIndexURL(), id)
      .then((result) => {
        if (cancelled) return;
        setItem(result || null);
      })
      .catch((e) => {
        console.error("Failed to load marketplace item:", e);
        if (cancelled) return;
        setItem(null);
        setLoadError(true);
      })
      .finally(() => {
        if (!cancelled) setLoading(false);
      });
    return () => {
      cancelled = true;
    };
  }, [id]);

  const typeKey = `marketplace.types.${item?.type || ""}`;
  const typeLabel = useMemo(() => {
    const raw = item?.type || "";
    if (!raw) return "";
    const translated = t(typeKey);
    return translated === typeKey ? raw : translated;
  }, [item?.type, t]);

  const sortedVersions = useMemo(() => {
    const versions = [...(item?.versions || [])];
    versions.sort((a: any, b: any) =>
      String(b.date || "").localeCompare(String(a.date || "")),
    );
    return versions;
  }, [item]);

  const handleInstall = (file: any) => {
    const downloadUrl = file.downloadUrl || "";
    if (!downloadUrl) return;
    const fileName = file.fileName || `${item?.id || "download"}-${file.version}`;
    void installer.start(downloadUrl, fileName, file.fileType || item?.type || "");
  };

  if (loading) {
    return (
      <PageContainer animate={false}>
        <Card className={LAYOUT.GLASS_CARD.BASE}>
          <CardBody className="p-6">
            <div className="flex flex-col md:flex-row gap-6">
              <div className="flex items-start gap-4 flex-1">
                <Skeleton className="w-24 h-24 rounded-2xl shrink-0" />
                <div className="flex flex-col gap-3 w-full max-w-lg">
                  <Skeleton className="h-8 w-3/4 rounded-lg" />
                  <div className="flex gap-2">
                    <Skeleton className="h-4 w-20 rounded-md" />
                    <Skeleton className="h-4 w-20 rounded-md" />
                  </div>
                  <div className="flex gap-2">
                    <Skeleton className="h-6 w-16 rounded-full" />
                    <Skeleton className="h-6 w-16 rounded-full" />
                  </div>
                </div>
              </div>
            </div>
          </CardBody>
        </Card>
        <Card className={`${LAYOUT.GLASS_CARD.BASE} min-h-[300px]`}>
          <CardBody className="p-6">
            <div className="space-y-4">
              <Skeleton className="h-4 w-full rounded-md" />
              <Skeleton className="h-4 w-full rounded-md" />
              <Skeleton className="h-4 w-3/4 rounded-md" />
            </div>
          </CardBody>
        </Card>
      </PageContainer>
    );
  }

  if (!item) {
    return (
      <div className="w-full h-full min-h-0 flex flex-col p-4 sm:p-6 gap-4 items-center justify-center">
        <Card className="bg-white/50 dark:bg-zinc-900/40 backdrop-blur-md rounded-4xl p-8">
          <CardBody className="flex flex-col items-center gap-4">
            <p className="text-xl font-bold">{t("marketplace.item_not_found")}</p>
            <Button
              onPress={() => navigate("/marketplace")}
              color="primary"
              className="bg-primary-500 hover:bg-primary-500 text-white font-bold shadow-lg shadow-primary-900/20"
            >
              {t("marketplace.go_back")}
            </Button>
          </CardBody>
        </Card>
      </div>
    );
  }

  const latest = sortedVersions[0];

  return (
    <PageContainer animate={false}>
      {/* Header Card */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.4 }}
      >
        <Card className={LAYOUT.GLASS_CARD.BASE}>
          <CardBody className="p-6">
            <div className="flex flex-col md:flex-row gap-6">
              <div className="shrink-0">
                <Image
                  src={item.icon}
                  alt={item.name}
                  className="w-32 h-32 object-cover rounded-2xl shadow-lg bg-content2"
                />
              </div>

              <div className="flex flex-col grow gap-3">
                <h1 className="text-3xl sm:text-4xl font-black tracking-tight text-default-900 dark:text-zinc-100 pb-1">
                  {item.name}
                </h1>

                <div className="flex items-center gap-3 text-default-500 dark:text-zinc-400 text-sm flex-wrap">
                  <span className="flex items-center gap-1">
                    {t("marketplace.by")}
                    <span className="text-default-700 dark:text-zinc-200 font-medium">
                      {item.author || "Unknown"}
                    </span>
                  </span>
                  {latest && (
                    <>
                      <span className="w-1 h-1 rounded-full bg-default-300"></span>
                      <span className="flex items-center gap-1">
                        <LuCalendar size={14} />
                        {t("marketplace.updated_date", {
                          date: formatDateStr(latest.date),
                        })}
                      </span>
                    </>
                  )}
                  {latest && (
                    <>
                      <span className="w-1 h-1 rounded-full bg-default-300"></span>
                      <span className="flex items-center gap-1">
                        <LuDownload size={14} />v{latest.version}
                      </span>
                    </>
                  )}
                </div>

                <div className="flex flex-wrap gap-2 mt-1">
                  <Chip
                    size="sm"
                    variant="flat"
                    startContent={<LuPackage size={12} />}
                  >
                    {typeLabel}
                  </Chip>
                  {(item.tags || []).map((tag: string) => (
                    <Chip key={tag} size="sm" variant="flat">
                      {tag}
                    </Chip>
                  ))}
                </div>

                <p className="text-default-600 dark:text-zinc-300 mt-2 text-sm leading-relaxed max-w-4xl">
                  {item.summary}
                </p>
              </div>

              {/* Actions */}
              <div className="flex flex-col gap-3 min-w-[240px] md:border-s md:border-default-100 md:ps-8 justify-center">
                {latest && (
                  <Button
                    className="w-full font-semibold shadow-md shadow-primary-900/20 text-white bg-primary-500 hover:bg-primary-500"
                    startContent={<LuDownload size={20} />}
                    size="lg"
                    onPress={() => handleInstall(latest)}
                  >
                    {t("marketplace.install_action")}
                  </Button>
                )}
                <div className="flex gap-2 justify-center">
                  {item.homepage && (
                    <Button
                      onPress={() => Browser.OpenURL(item.homepage)}
                      isIconOnly
                      variant="flat"
                      aria-label={t("marketplace.homepage")}
                    >
                      {String(item.homepage).includes("github.com") ? (
                        <LuGithub size={20} />
                      ) : (
                        <LuGlobe size={20} />
                      )}
                    </Button>
                  )}
                </div>
              </div>
            </div>
          </CardBody>
        </Card>
      </motion.div>

      {/* Content Card */}
      <motion.div
        initial={{ opacity: 0 }}
        animate={{ opacity: 1 }}
        transition={{ duration: 0.4, delay: 0.1 }}
      >
        <Card className={`${LAYOUT.GLASS_CARD.BASE} min-h-[500px]`}>
          <CardBody className="p-6">
            <Tabs
              aria-label={t("marketplace.item_details_aria_label")}
              variant="underlined"
              color="primary"
              selectedKey={selectedTab}
              onSelectionChange={(key) => setSelectedTab(key as string)}
              classNames={{
                tabList:
                  "gap-6 w-full relative rounded-none p-0 border-b border-default-200 mb-6",
                cursor:
                  "w-full bg-linear-to-r from-primary-500 to-primary-400 h-[3px]",
                tab: "max-w-fit px-0 h-12 text-base font-medium text-default-500 dark:text-zinc-400",
                tabContent:
                  "group-data-[selected=true]:text-primary-600 dark:group-data-[selected=true]:text-primary-500 font-bold",
              }}
            >
              <Tab key="description" title={t("marketplace.tabs.description")}>
                <div className="flex flex-col gap-6">
                  {(item.screenshots || []).length > 0 && (
                    <div className="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-3">
                      {item.screenshots.map((shot: string, index: number) => (
                        <Image
                          key={index}
                          src={shot}
                          alt={`${item.name} #${index + 1}`}
                          className="w-full object-cover rounded-xl bg-content2"
                          loading="lazy"
                        />
                      ))}
                    </div>
                  )}
                  <div className="prose dark:prose-invert max-w-none prose-img:rounded-xl prose-img:mx-auto prose-a:text-primary-600 dark:prose-a:text-primary-500">
                    <ReactMarkdown remarkPlugins={[remarkGfm]}>
                      {item.description || item.summary || ""}
                    </ReactMarkdown>
                  </div>
                </div>
              </Tab>
              <Tab key="files" title={t("marketplace.tabs.files")}>
                <div className="flex flex-col gap-4">
                  <h3 className="text-lg font-semibold">
                    {t("marketplace.files.title")}
                  </h3>

                  {sortedVersions.length > 0 ? (
                    <Table
                      aria-label={t("marketplace.files.table_aria_label")}
                      removeWrapper
                      classNames={COMPONENT_STYLES.table}
                    >
                      <TableHeader>
                        <TableColumn>
                          {t("marketplace.files.columns.version")}
                        </TableColumn>
                        <TableColumn>
                          {t("marketplace.files.columns.date")}
                        </TableColumn>
                        <TableColumn>
                          {t("marketplace.files.columns.size")}
                        </TableColumn>
                        <TableColumn>
                          {t("marketplace.files.columns.changelog")}
                        </TableColumn>
                        <TableColumn>
                          {t("marketplace.files.columns.actions")}
                        </TableColumn>
                      </TableHeader>
                      <TableBody>
                        {sortedVersions.map((file: any, index: number) => (
                          <TableRow key={`${file.version}-${index}`}>
                            <TableCell>
                              <Chip
                                size="sm"
                                color={index === 0 ? "success" : "default"}
                                variant="flat"
                              >
                                v{file.version}
                              </Chip>
                            </TableCell>
                            <TableCell>
                              <span className="text-default-500 dark:text-zinc-400">
                                {formatDateStr(file.date)}
                              </span>
                            </TableCell>
                            <TableCell>
                              <span className="text-default-500 dark:text-zinc-400">
                                {file.size > 0 ? formatFileSize(file.size) : "-"}
                              </span>
                            </TableCell>
                            <TableCell>
                              <span className="text-default-500 dark:text-zinc-400 line-clamp-2 max-w-xs">
                                {file.changelog || "-"}
                              </span>
                            </TableCell>
                            <TableCell>
                              <Tooltip content={t("marketplace.install_action")}>
                                <Button
                                  isIconOnly
                                  variant="light"
                                  size="sm"
                                  className="text-default-500 dark:text-zinc-400 hover:text-primary"
                                  onPress={() => handleInstall(file)}
                                >
                                  <LuDownload size={20} />
                                </Button>
                              </Tooltip>
                            </TableCell>
                          </TableRow>
                        ))}
                      </TableBody>
                    </Table>
                  ) : (
                    <div className="flex flex-col items-center justify-center py-12 text-default-400 border border-dashed border-default-200 rounded-xl">
                      <LuFileDigit size={48} className="mb-4 opacity-50" />
                      <p className="text-lg font-medium">
                        {t("marketplace.no_files_found")}
                      </p>
                    </div>
                  )}
                </div>
              </Tab>
            </Tabs>
          </CardBody>
        </Card>
      </motion.div>

      {/* Install Modal */}
      <UnifiedModal
        isOpen={installer.step !== "idle"}
        onOpenChange={(open) => {
          if (!open) {
            if (installer.step === "downloading") {
              void installer.cancelDownload();
            } else {
              installer.dismiss();
            }
          }
        }}
        isDismissable={false}
        hideCloseButton={
          installer.step === "downloading" || installer.step === "importing"
        }
        contentKey={installer.step}
        motionProps={{
          variants: {
            enter: {
              scale: 1,
              opacity: 1,
              transition: { duration: 0.3, ease: "easeOut" },
            },
            exit: {
              scale: 0.95,
              opacity: 0,
              transition: { duration: 0.2, ease: "easeIn" },
            },
          },
        }}
        type={installer.step === "error" ? "error" : "primary"}
        icon={
          installer.step === "downloading" ? (
            <LuDownload size={24} />
          ) : installer.step === "version_select" ? (
            <LuGamepad2 size={24} />
          ) : installer.step === "player_select" ? (
            <LuUser size={24} />
          ) : installer.step === "importing" ? (
            <LuFileDigit size={24} />
          ) : installer.step === "success" ? (
            <FiCheckCircle size={24} />
          ) : undefined
        }
        title={
          <>
            {installer.step === "downloading" &&
              t("marketplace.install.downloading_title")}
            {installer.step === "version_select" &&
              t("marketplace.install.select_version_title")}
            {installer.step === "player_select" &&
              t("marketplace.install.select_player_title")}
            {installer.step === "importing" &&
              t("marketplace.install.importing_title")}
            {installer.step === "success" &&
              t("marketplace.install.success_title")}
            {installer.step === "error" && t("marketplace.install.error_title")}
          </>
        }
        footer={
          <>
            {installer.step === "downloading" && (
              <Button
                color="danger"
                variant="flat"
                onPress={() => void installer.cancelDownload()}
              >
                {t("common.cancel")}
              </Button>
            )}
            {(installer.step === "version_select" ||
              installer.step === "player_select") && (
              <>
                <Button variant="flat" onPress={installer.dismiss}>
                  {t("common.cancel")}
                </Button>
                <Button
                  color="primary"
                  onPress={() => void installer.next()}
                  className="bg-primary-500 hover:bg-primary-500 text-white font-bold shadow-lg shadow-primary-900/20"
                >
                  {t("marketplace.install.next")}
                </Button>
              </>
            )}
            {(installer.step === "success" || installer.step === "error") && (
              <Button
                color={installer.step === "error" ? "danger" : "primary"}
                onPress={installer.dismiss}
                className={
                  installer.step === "error"
                    ? "font-bold shadow-lg shadow-danger-500/20"
                    : "bg-primary-500 hover:bg-primary-500 text-white font-bold shadow-lg shadow-primary-900/20"
                }
              >
                {t("marketplace.install.close")}
              </Button>
            )}
          </>
        }
      >
        {installer.step === "downloading" && (
          <div className="flex flex-col items-center gap-4 py-4 w-full">
            <p className="text-default-500 dark:text-zinc-400">
              {t("marketplace.install.downloading_body")}
            </p>
            {installer.downloadProgress ? (
              <Progress
                aria-label="Downloading..."
                value={
                  (installer.downloadProgress.downloaded /
                    installer.downloadProgress.total) *
                  100
                }
                className="max-w-md w-full"
                color="primary"
                classNames={{
                  indicator: "bg-primary-500 dark:bg-primary-500",
                }}
                showValueLabel={true}
              />
            ) : (
              <Spinner size="lg" color="primary" />
            )}
            {installer.downloadProgress && (
              <p className="text-tiny text-default-400">
                {formatFileSize(installer.downloadProgress.downloaded)} /{" "}
                {formatFileSize(installer.downloadProgress.total)}
              </p>
            )}
          </div>
        )}

        {installer.step === "version_select" && (
          <div className="flex flex-col gap-4">
            <p className="text-small text-default-500 dark:text-zinc-400">
              {t("marketplace.install.select_version_body")}
            </p>
            <Select
              label={t("marketplace.install.local_installation")}
              placeholder={t("marketplace.install.select_version_placeholder")}
              selectedKeys={
                installer.selectedVersion ? [installer.selectedVersion] : []
              }
              onChange={(e) => installer.setSelectedVersion(e.target.value)}
              classNames={COMPONENT_STYLES.select}
              items={installer.availableVersions}
            >
              {(ver: any) => (
                <SelectItem key={ver.name} textValue={ver.name}>
                  <div className="flex gap-2 items-center">
                    <div className="w-8 h-8 rounded bg-default-200 flex items-center justify-center overflow-hidden">
                      <img
                        src={
                          installer.versionLogos[ver.name] ||
                          "https://raw.githubusercontent.com/BedrockNexusLauncher/BedrockNexusLauncher/main/build/appicon.png"
                        }
                        alt="icon"
                        className="w-full h-full object-cover"
                        onError={(e) =>
                          (e.currentTarget.style.display = "none")
                        }
                      />
                    </div>
                    <div className="flex flex-col">
                      <span className="text-small">{ver.name}</span>
                      <span className="text-tiny text-default-400">
                        {ver.gameVersion}
                      </span>
                    </div>
                    {ver.registered && (
                      <Chip
                        size="sm"
                        color="success"
                        variant="flat"
                        className="ml-auto"
                      >
                        {t("marketplace.install.registered")}
                      </Chip>
                    )}
                  </div>
                </SelectItem>
              )}
            </Select>
          </div>
        )}

        {installer.step === "player_select" && (
          <div className="flex flex-col gap-4">
            <p className="text-small text-default-500 dark:text-zinc-400">
              {t("marketplace.install.select_player_body")}
            </p>
            <Select
              label={t("marketplace.install.player_label")}
              placeholder={t("marketplace.install.select_player_placeholder")}
              selectedKeys={
                installer.selectedPlayer ? [installer.selectedPlayer] : []
              }
              onChange={(e) => installer.setSelectedPlayer(e.target.value)}
              classNames={COMPONENT_STYLES.select}
            >
              {installer.availablePlayers.map((player) => (
                <SelectItem key={player}>
                  {installer.playerGamertagMap[player] || player}
                </SelectItem>
              ))}
            </Select>
          </div>
        )}

        {installer.step === "importing" && (
          <div className="flex flex-col items-center gap-4 py-4">
            <Spinner size="lg" color="primary" />
            <p className="text-default-500 dark:text-zinc-400">
              {t("marketplace.install.importing_body")}
            </p>
          </div>
        )}

        {installer.step === "success" && (
          <div className="flex flex-col items-center gap-4 py-4">
            <div className="text-center">
              <p className="text-xl font-bold text-default-900 dark:text-white">
                {t("marketplace.install.success_msg")}
              </p>
              <p className="text-default-500 dark:text-zinc-400 mt-1">
                {t("marketplace.install.success_desc")}
              </p>
            </div>
          </div>
        )}

        {installer.step === "error" && (
          <div className="flex flex-col items-center gap-4 py-4">
            <div className="text-center w-full">
              <p className="text-xl font-bold text-danger-600 dark:text-danger-500 mb-2">
                {t("marketplace.install.failed_msg")}
              </p>
              <div className="text-default-500 dark:text-zinc-400 px-4 bg-default-100 dark:bg-zinc-800 rounded-lg py-3 font-mono text-xs break-all mx-auto max-w-[90%]">
                {installer.installError}
              </div>
            </div>
          </div>
        )}
      </UnifiedModal>

      {/* Duplicate/overwrite modal */}
      <UnifiedModal
        size="md"
        isOpen={installer.dupOpen}
        onOpenChange={(open) => {
          if (!open) {
            installer.resolveDuplicate(false);
          }
        }}
        hideCloseButton
        motionProps={{
          variants: {
            enter: {
              scale: 1,
              opacity: 1,
              transition: { duration: 0.3, ease: "easeOut" },
            },
            exit: {
              scale: 0.95,
              opacity: 0,
              transition: { duration: 0.2, ease: "easeIn" },
            },
          },
        }}
        type="warning"
        title={t("mods.overwrite_modal_title")}
        footer={
          <>
            <Button
              variant="light"
              onPress={() => installer.resolveDuplicate(false)}
            >
              {t("common.cancel")}
            </Button>
            <Button
              color="warning"
              className="font-bold shadow-lg shadow-warning-500/20 text-white"
              onPress={() => installer.resolveDuplicate(true)}
            >
              {t("common.confirm")}
            </Button>
          </>
        }
      >
        <div className="text-sm text-default-700 dark:text-zinc-300">
          {t("mods.overwrite_modal_body")}
        </div>
        {installer.dupName ? (
          <div className="mt-1 rounded-md bg-default-100/60 dark:bg-zinc-800/60 border border-default-200 dark:border-zinc-700 px-3 py-2 text-default-800 dark:text-zinc-100 text-sm wrap-break-word whitespace-pre-wrap">
            {installer.dupName}
          </div>
        ) : null}
      </UnifiedModal>
    </PageContainer>
  );
};

export default MarketplaceItemPage;
