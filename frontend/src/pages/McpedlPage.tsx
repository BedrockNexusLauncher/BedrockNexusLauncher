import React, { useCallback, useEffect, useState } from "react";
import { addToast, Button, Card, CardBody, Chip, Input, Spinner } from "@heroui/react";
import { useTranslation } from "react-i18next";
import { Browser } from "@wailsio/runtime";
import { motion } from "framer-motion";
import {
  FaDownload,
  FaExternalLinkAlt,
  FaExclamationTriangle,
  FaSearch,
  FaStar,
  FaStore,
} from "react-icons/fa";
import { PageHeader } from "@/components/PageHeader";
import { PageContainer } from "@/components/PageContainer";
import { LAYOUT } from "@/constants/layout";
import { COMPONENT_STYLES } from "@/constants/componentStyles";
import { cn } from "@/utils/cn";
import * as minecraft from "bindings/github.com/BedrockNexusLauncher/BedrockNexusLauncher/minecraft";

const PAGE_SIZE = 30;

type SubmissionFile = {
  name: string;
  filename: string;
  downloadUrl: string;
};

type Submission = {
  slug: string;
  title: string;
  summary: string;
  image: string;
  url: string;
  source: string;
  downloadCount: number;
  average_rating: string;
  update_date: string;
  downloads: SubmissionFile[];
};

const formatCount = (value: number): string => {
  if (!Number.isFinite(value) || value <= 0) return "0";
  if (value >= 1_000_000) return `${(value / 1_000_000).toFixed(1)}M`;
  if (value >= 1_000) return `${(value / 1_000).toFixed(1)}K`;
  return String(value);
};

const McpedlPage: React.FC = () => {
  const { t } = useTranslation();
  const [queryInput, setQueryInput] = useState("");
  const [query, setQuery] = useState("");
  const [page, setPage] = useState(1);
  const [items, setItems] = useState<Submission[]>([]);
  const [hasMore, setHasMore] = useState(false);
  const [loading, setLoading] = useState(false);
  const [initialLoaded, setInitialLoaded] = useState(false);
  const [expandedSlug, setExpandedSlug] = useState<string | null>(null);

  const load = useCallback(
    async (searchQuery: string, targetPage: number, append: boolean) => {
      setLoading(true);
      try {
        const res: any = await minecraft?.McpedlSearchSubmissions?.(
          searchQuery,
          targetPage,
        );
        const list: Submission[] = Array.isArray(res)
          ? res
          : Array.isArray(res?.data)
            ? res.data
            : [];
        setItems((prev) => (append ? [...prev, ...list] : list));
        setHasMore(list.length >= PAGE_SIZE);
      } catch (err) {
        console.error("MCPEDL search failed", err);
        if (!append) setItems([]);
        addToast({ title: t("mcpedl.load_failed"), color: "danger" });
      } finally {
        setLoading(false);
        setInitialLoaded(true);
      }
    },
    [t],
  );

  useEffect(() => {
    void load("", 1, false);
  }, [load]);

  const runSearch = () => {
    const trimmed = queryInput.trim();
    setQuery(trimmed);
    setPage(1);
    setExpandedSlug(null);
    void load(trimmed, 1, false);
  };

  const loadMore = () => {
    const next = page + 1;
    setPage(next);
    void load(query, next, true);
  };

  const startDownload = (file: SubmissionFile) => {
    const filename =
      file.filename || file.name || `${file.downloadUrl.split("/").pop() || "content.mcpack"}`;
    try {
      minecraft?.StartFileDownload?.(file.downloadUrl, filename);
      addToast({
        title: t("mcpedl.download_started"),
        description: filename,
        color: "success",
      });
    } catch (err) {
      console.error("Download failed", err);
      addToast({ title: t("mcpedl.load_failed"), color: "danger" });
    }
  };

  return (
    <PageContainer>
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, ease: "easeOut" }}
      >
        <Card className={LAYOUT.GLASS_CARD.BASE}>
          <CardBody className="p-6 flex flex-col gap-4">
            <PageHeader
              title={t("mcpedl.title")}
              description={t("mcpedl.desc")}
              endContent={
                <Chip
                  variant="flat"
                  color="warning"
                  startContent={<FaExclamationTriangle className="text-xs" />}
                >
                  {t("mcpedl.experimental_badge")}
                </Chip>
              }
            />
            <div className="flex items-center gap-2">
              <Input
                value={queryInput}
                onValueChange={setQueryInput}
                placeholder={t("mcpedl.search_placeholder")}
                startContent={<FaSearch className="text-default-400" />}
                classNames={COMPONENT_STYLES.input}
                onKeyDown={(e) => {
                  if (e.key === "Enter") runSearch();
                }}
              />
              <Button
                color="primary"
                variant="flat"
                onPress={runSearch}
                isLoading={loading && initialLoaded && items.length === 0}
              >
                {t("mcpedl.search")}
              </Button>
            </div>
            <p className="text-tiny text-default-500 dark:text-zinc-400">
              {t("mcpedl.experimental_note")}
            </p>
          </CardBody>
        </Card>
      </motion.div>

      <div className="flex-1 min-h-0 flex flex-col">
        {loading && items.length === 0 ? (
          <div className="flex-1 flex items-center justify-center">
            <Spinner size="lg" color="primary" />
          </div>
        ) : items.length === 0 ? (
          <div className="flex-1 flex items-center justify-center text-default-500 dark:text-zinc-400">
            {t("mcpedl.no_results")}
          </div>
        ) : (
          <>
            <div className="grid grid-cols-1 md:grid-cols-2 xl:grid-cols-3 gap-4">
              {items.map((item, idx) => {
                const fileCount = Array.isArray(item.downloads)
                  ? item.downloads.length
                  : 0;
                const isExpanded = expandedSlug === item.slug;
                return (
                  <motion.div
                    key={`${item.slug}-${idx}`}
                    initial={{ opacity: 0, y: 16 }}
                    animate={{ opacity: 1, y: 0 }}
                    transition={{ delay: Math.min(idx * 0.04, 0.4) }}
                  >
                    <Card
                      className={cn(
                        "h-full flex flex-col",
                        LAYOUT.GLASS_CARD.BASE,
                        LAYOUT.CARD_HOVER,
                        "rounded-3xl overflow-hidden",
                      )}
                    >
                      {item.image && (
                        <img
                          src={item.image}
                          alt={item.title}
                          loading="lazy"
                          className="w-full h-36 object-cover"
                          onError={(e) => {
                            (e.target as HTMLImageElement).style.display =
                              "none";
                          }}
                        />
                      )}
                      <CardBody className="p-4 flex flex-col gap-3 flex-1">
                        <div className="flex items-start justify-between gap-2">
                          <h3 className="font-bold text-default-800 dark:text-zinc-100 leading-snug">
                            {item.title}
                          </h3>
                          {item.source && (
                            <Chip
                              size="sm"
                              variant="flat"
                              color="default"
                              className="shrink-0 h-6"
                            >
                              {item.source}
                            </Chip>
                          )}
                        </div>
                        {item.summary && (
                          <p className="text-tiny text-default-500 dark:text-zinc-400 line-clamp-3 leading-relaxed">
                            {item.summary}
                          </p>
                        )}
                        <div className="flex items-center gap-3 text-xs text-default-500 dark:text-zinc-400">
                          <span className="flex items-center gap-1">
                            <FaDownload className="text-[10px]" />
                            {formatCount(item.downloadCount)}
                          </span>
                          {item.average_rating && Number(item.average_rating) > 0 && (
                            <span className="flex items-center gap-1">
                              <FaStar className="text-warning-500 text-[10px]" />
                              {Number(item.average_rating).toFixed(1)}
                            </span>
                          )}
                        </div>
                        <div className="flex items-center gap-2 mt-auto">
                          {fileCount > 0 && (
                            <Button
                              size="sm"
                              variant="flat"
                              color="primary"
                              startContent={<FaDownload />}
                              onPress={() =>
                                setExpandedSlug(isExpanded ? null : item.slug)
                              }
                            >
                              {t("mcpedl.files", { count: fileCount })}
                            </Button>
                          )}
                          {item.url && (
                            <Button
                              size="sm"
                              variant="light"
                              startContent={<FaExternalLinkAlt />}
                              onPress={() => Browser.OpenURL(item.url)}
                            >
                              {t("mcpedl.open_page")}
                            </Button>
                          )}
                        </div>
                        {isExpanded && (
                          <div className="flex flex-col gap-2 border-t border-default-100 dark:border-white/5 pt-3">
                            {item.downloads.map((file, fileIdx) => (
                              <div
                                key={`${item.slug}-${fileIdx}`}
                                className="flex items-center justify-between gap-2 p-2 rounded-xl bg-default-100/50 dark:bg-white/5"
                              >
                                <div className="min-w-0">
                                  <p className="text-xs font-medium text-default-700 dark:text-zinc-200 truncate">
                                    {file.name || file.filename}
                                  </p>
                                  {file.filename && file.name !== file.filename && (
                                    <p className="text-[10px] text-default-400 truncate">
                                      {file.filename}
                                    </p>
                                  )}
                                </div>
                                <Button
                                  size="sm"
                                  isIconOnly
                                  variant="flat"
                                  color="primary"
                                  aria-label={t("mcpedl.download")}
                                  onPress={() => startDownload(file)}
                                >
                                  <FaDownload />
                                </Button>
                              </div>
                            ))}
                          </div>
                        )}
                      </CardBody>
                    </Card>
                  </motion.div>
                );
              })}
            </div>

            {hasMore && (
              <div className="flex justify-center py-4">
                <Button
                  variant="flat"
                  className="bg-default-100 dark:bg-white/10"
                  isLoading={loading}
                  onPress={loadMore}
                >
                  {t("mcpedl.load_more")}
                </Button>
              </div>
            )}
          </>
        )}
      </div>
    </PageContainer>
  );
};

export default McpedlPage;
