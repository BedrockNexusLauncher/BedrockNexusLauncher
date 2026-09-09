import React, { useEffect, useMemo, useState } from "react";
import {
  Button,
  Chip,
  Input,
  Select,
  SelectItem,
  Pagination,
  Skeleton,
  Card,
  CardBody,
} from "@heroui/react";
import { PageHeader } from "@/components/PageHeader";
import { useNavigate } from "react-router-dom";
import { useTranslation } from "react-i18next";
import { PageContainer } from "@/components/PageContainer";
import { LAYOUT } from "@/constants/layout";
import { COMPONENT_STYLES } from "@/constants/componentStyles";
import { cn } from "@/utils/cn";
import { formatDateStr, formatFileSize } from "@/utils/formatting";
import {
  useMarketplace,
  getIndexURL,
  setIndexURL,
  MarketplaceProvider,
} from "@/utils/MarketplaceContext";
import {
  LuSearch,
  LuDownload,
  LuClock,
  LuPackage,
  LuRefreshCw,
  LuSettings2,
} from "react-icons/lu";
import { motion } from "framer-motion";

const PAGE_SIZE = 20;

type SortKey = "newest" | "name";

export const MarketplacePage: React.FC = () => {
  return (
    <MarketplaceProvider>
      <MarketplacePageContent />
    </MarketplaceProvider>
  );
};

const MarketplacePageContent: React.FC = () => {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const { items, loading, error, reload } = useMarketplace();

  const [query, setQuery] = useState("");
  const [searchToken, setSearchToken] = useState(0);
  const [selectedType, setSelectedType] = useState("");
  const [selectedTag, setSelectedTag] = useState("");
  const [sortKey, setSortKey] = useState<SortKey>("newest");
  const [currentPage, setCurrentPage] = useState(1);
  const [urlInput, setUrlInput] = useState("");
  const [showUrlEditor, setShowUrlEditor] = useState(false);

  useEffect(() => {
    setUrlInput(getIndexURL());
  }, []);

  useEffect(() => {
    setCurrentPage(1);
  }, [searchToken, selectedType, selectedTag, sortKey]);

  const typeLabels = useMemo(() => {
    const labels = new Map<string, string>();
    items.forEach((item) => {
      const type = item.type || "unknown";
      if (!labels.has(type)) {
        const key = `marketplace.types.${type}`;
        const translated = t(key);
        labels.set(type, translated === key ? type : translated);
      }
    });
    return labels;
  }, [items, t]);

  const tags = useMemo(() => {
    const set = new Set<string>();
    items.forEach((item) => {
      (item.tags || []).forEach((tag: string) => {
        if (tag) set.add(tag);
      });
    });
    return Array.from(set).sort((a, b) => a.localeCompare(b));
  }, [items]);

  const latestVersionOf = (item: any) => {
    const versions = item.versions || [];
    if (versions.length === 0) return null;
    const sorted = [...versions].sort((a: any, b: any) =>
      String(b.date || "").localeCompare(String(a.date || "")),
    );
    return sorted[0];
  };

  const filtered = useMemo(() => {
    const q = query.trim().toLowerCase();
    let list = [...items];

    if (q) {
      list = list.filter(
        (item) =>
          String(item.name || "").toLowerCase().includes(q) ||
          String(item.summary || "").toLowerCase().includes(q) ||
          String(item.author || "").toLowerCase().includes(q) ||
          String(item.id || "").toLowerCase().includes(q),
      );
    }
    if (selectedType) {
      list = list.filter((item) => (item.type || "unknown") === selectedType);
    }
    if (selectedTag) {
      list = list.filter((item) => (item.tags || []).includes(selectedTag));
    }

    list.sort((a, b) => {
      if (sortKey === "name") {
        return String(a.name || "").localeCompare(String(b.name || ""));
      }
      const av = latestVersionOf(a)?.date || "";
      const bv = latestVersionOf(b)?.date || "";
      return String(bv).localeCompare(String(av));
    });

    return list;
  }, [items, query, searchToken, selectedType, selectedTag, sortKey]);

  const totalCount = filtered.length;
  const totalPages = Math.ceil(totalCount / PAGE_SIZE);
  const pageItems = useMemo(() => {
    const start = (currentPage - 1) * PAGE_SIZE;
    return filtered.slice(start, start + PAGE_SIZE);
  }, [filtered, currentPage]);

  const handleSearch = () => {
    setSearchToken((v) => v + 1);
  };

  const handleKeyPress = (e: React.KeyboardEvent) => {
    if (e.key === "Enter") {
      handleSearch();
    }
  };

  const applyIndexURL = () => {
    setIndexURL(urlInput.trim());
    void reload();
    setShowUrlEditor(false);
  };

  const renderSkeletons = () => {
    return Array(5)
      .fill(0)
      .map((_, index) => (
        <div key={index} className="w-full flex items-center gap-3 p-3">
          <Skeleton className="flex rounded-lg w-20 h-20 sm:w-24 sm:h-24" />
          <div className="w-full flex flex-col gap-2">
            <Skeleton className="h-3 w-3/5 rounded-lg" />
            <Skeleton className="h-3 w-4/5 rounded-lg" />
            <div className="flex gap-2 pt-2">
              <Skeleton className="h-2 w-1/4 rounded-lg" />
              <Skeleton className="h-2 w-1/4 rounded-lg" />
            </div>
          </div>
        </div>
      ));
  };

  return (
    <PageContainer animate={false} className="min-h-0 !overflow-hidden">
      <motion.div
        initial={{ opacity: 0, y: -20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4 }}
      >
        <Card className={cn("shrink-0", LAYOUT.GLASS_CARD.BASE)}>
          <CardBody className="p-6 flex flex-col gap-6">
            <PageHeader title={t("marketplace.title")} />

            <div className="flex flex-col sm:flex-row gap-3">
              <Input
                placeholder={t("marketplace.search_placeholder")}
                value={query}
                onValueChange={setQuery}
                onKeyPress={handleKeyPress}
                startContent={<LuSearch />}
                className="flex-1"
                size="sm"
                classNames={COMPONENT_STYLES.input}
              />
              <Button
                color="primary"
                onPress={handleSearch}
                startContent={<LuSearch />}
                size="sm"
                className="bg-primary-500 hover:bg-primary-500 text-white font-bold shadow-lg shadow-primary-900/20"
              >
                {t("marketplace.search")}
              </Button>
              <Button
                variant="flat"
                onPress={() => void reload()}
                startContent={<LuRefreshCw />}
                size="sm"
              >
                {t("common.refresh")}
              </Button>
              <Button
                variant="flat"
                isIconOnly
                onPress={() => setShowUrlEditor((v) => !v)}
                aria-label={t("marketplace.index_source")}
              >
                <LuSettings2 />
              </Button>
            </div>

            {showUrlEditor && (
              <div className="flex flex-col sm:flex-row gap-2">
                <Input
                  placeholder="https://cdn.jsdelivr.net/gh/<user>/<repo>@main/index.json"
                  value={urlInput}
                  onValueChange={setUrlInput}
                  size="sm"
                  classNames={COMPONENT_STYLES.input}
                />
                <Button color="primary" variant="flat" onPress={applyIndexURL} size="sm">
                  {t("marketplace.apply")}
                </Button>
              </div>
            )}

            <div className="grid grid-cols-1 sm:grid-cols-3 gap-3">
              <Select
                label={t("marketplace.type")}
                placeholder={t("marketplace.select_type")}
                selectedKeys={selectedType ? [selectedType] : []}
                onSelectionChange={(keys) => {
                  setSelectedType((Array.from(keys)[0] as string) || "");
                }}
                size="sm"
                classNames={COMPONENT_STYLES.select}
                items={[
                  { key: "", label: t("marketplace.all_types") },
                  ...Array.from(typeLabels.entries()).map(([key, label]) => ({
                    key,
                    label,
                  })),
                ]}
              >
                {(item) => <SelectItem key={item.key}>{item.label}</SelectItem>}
              </Select>

              <Select
                label={t("marketplace.tag")}
                placeholder={t("marketplace.select_tag")}
                selectedKeys={selectedTag ? [selectedTag] : []}
                onSelectionChange={(keys) => {
                  setSelectedTag((Array.from(keys)[0] as string) || "");
                }}
                size="sm"
                classNames={COMPONENT_STYLES.select}
                items={[
                  { key: "", label: t("marketplace.all_tags") },
                  ...tags.map((tag) => ({ key: tag, label: tag })),
                ]}
              >
                {(item) => <SelectItem key={item.key}>{item.label}</SelectItem>}
              </Select>

              <Select
                label={t("marketplace.sort_by")}
                placeholder={t("marketplace.select_sort")}
                selectedKeys={[sortKey]}
                onSelectionChange={(keys) => {
                  setSortKey((Array.from(keys)[0] as SortKey) || "newest");
                }}
                size="sm"
                classNames={COMPONENT_STYLES.select}
                items={[
                  { key: "newest", label: t("marketplace.sort.newest") },
                  { key: "name", label: t("marketplace.sort.name") },
                ]}
              >
                {(item) => <SelectItem key={item.key}>{item.label}</SelectItem>}
              </Select>
            </div>
          </CardBody>
        </Card>
      </motion.div>

      <motion.div
        className="flex-1 min-h-0 flex flex-col"
        initial={{ opacity: 0, y: 20 }}
        animate={{ opacity: 1, y: 0 }}
        transition={{ duration: 0.4, delay: 0.1 }}
      >
        <Card className={cn("flex-1 min-h-0", LAYOUT.GLASS_CARD.BASE)}>
          <CardBody className="p-0 overflow-hidden flex flex-col">
            <div
              className="flex-1 overflow-y-auto p-4 relative [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]"
            >
              {error ? (
                <div className="flex flex-col items-center justify-center h-full gap-4 text-danger">
                  <p>{t("marketplace.load_error")}</p>
                  <Button color="danger" variant="flat" onPress={() => void reload()}>
                    {t("common.retry")}
                  </Button>
                </div>
              ) : loading ? (
                <div className="flex flex-col gap-3">{renderSkeletons()}</div>
              ) : pageItems.length === 0 ? (
                <div className="flex items-center justify-center h-full text-default-500 dark:text-zinc-400">
                  <p>{t("marketplace.no_results")}</p>
                </div>
              ) : (
                <div className="flex flex-col gap-3">
                  {pageItems.map((item) => {
                    const latest = latestVersionOf(item);
                    return (
                      <motion.div
                        key={item.id}
                        initial={{ opacity: 0 }}
                        animate={{ opacity: 1 }}
                        transition={{ duration: 0.2 }}
                      >
                        <div
                          className="w-full p-4 bg-default-50/50 dark:bg-white/5 hover:bg-default-100/50 dark:hover:bg-white/10 transition cursor-pointer rounded-2xl flex gap-4 group shadow-sm hover:shadow-md border border-default-100 dark:border-white/5"
                          onClick={() => navigate(`/marketplace/item/${item.id}`)}
                        >
                          <div className="shrink-0">
                            <img
                              src={item.icon || ""}
                              alt={item.name}
                              className="w-20 h-20 sm:w-24 sm:h-24 rounded-2xl object-cover bg-content3 shadow-sm"
                              loading="lazy"
                              onError={(e) => {
                                (e.currentTarget.style.visibility = "hidden");
                              }}
                            />
                          </div>

                          <div className="flex flex-col flex-1 min-w-0 gap-1">
                            <div className="flex items-baseline gap-2 truncate">
                              <h3 className="text-base sm:text-lg font-bold text-foreground truncate">
                                {item.name}
                              </h3>
                              <span className="text-xs sm:text-sm text-default-500 dark:text-zinc-400 truncate">
                                | {t("marketplace.by_author", { author: item.author || "Unknown" })}
                              </span>
                            </div>

                            <p className="text-xs sm:text-sm text-default-500 dark:text-zinc-400 line-clamp-2 w-full">
                              {item.summary || t("marketplace.no_description")}
                            </p>

                            <div className="flex flex-wrap items-center gap-x-4 gap-y-1 text-xs text-default-400 mt-1">
                              <div className="flex items-center gap-1" title={t("marketplace.type")}>
                                <LuPackage />
                                <span>
                                  {typeLabels.get(item.type || "unknown") || item.type}
                                </span>
                              </div>
                              {latest && (
                                <>
                                  <div className="flex items-center gap-1" title={t("marketplace.version")}>
                                    <LuDownload />
                                    <span>v{latest.version}</span>
                                  </div>
                                  <div className="flex items-center gap-1" title={t("marketplace.updated")}>
                                    <LuClock />
                                    <span>{formatDateStr(latest.date)}</span>
                                  </div>
                                  {latest.size > 0 && (
                                    <span>{formatFileSize(latest.size)}</span>
                                  )}
                                </>
                              )}
                            </div>

                            <div className="flex flex-wrap gap-1 mt-2">
                              <Chip
                                size="sm"
                                variant="flat"
                                radius="sm"
                                className="h-5 text-[10px] bg-primary/10 text-primary font-medium"
                              >
                                {typeLabels.get(item.type || "unknown") || item.type}
                              </Chip>
                              {(item.tags || []).map((tag: string) => (
                                <Chip
                                  key={tag}
                                  size="sm"
                                  variant="flat"
                                  radius="sm"
                                  className="h-5 text-[10px] bg-default-100 dark:bg-zinc-800 text-default-500 dark:text-zinc-400 group-hover:bg-default-200 dark:group-hover:bg-zinc-700 transition-colors"
                                >
                                  {tag}
                                </Chip>
                              ))}
                            </div>
                          </div>
                        </div>
                      </motion.div>
                    );
                  })}
                </div>
              )}
            </div>

            {totalPages > 1 && (
              <div className="flex justify-center p-4 border-t border-default-100 dark:border-white/5 bg-white/50 dark:bg-zinc-900/50 backdrop-blur-md shrink-0">
                <Pagination
                  total={totalPages}
                  page={currentPage}
                  onChange={setCurrentPage}
                  showControls
                  color="primary"
                  className="gap-2"
                  radius="full"
                  classNames={{
                    cursor:
                      "bg-primary-500 hover:bg-primary-500 shadow-lg shadow-primary-900/20 font-bold",
                  }}
                />
              </div>
            )}
          </CardBody>
        </Card>
      </motion.div>
    </PageContainer>
  );
};

export default MarketplacePage;
