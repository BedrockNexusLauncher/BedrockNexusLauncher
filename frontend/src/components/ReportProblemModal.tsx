import React, { useEffect, useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { addToast, Button, Switch, Textarea } from "@heroui/react";
import { Browser } from "@wailsio/runtime";
import { FaCopy, FaGithub, FaSearch } from "react-icons/fa";
import * as minecraft from "bindings/github.com/BedrockNexusLauncher/BedrockNexusLauncher/minecraft";
import { UnifiedModal } from "@/components/UnifiedModal";

const REPO_URL = "https://github.com/BedrockNexusLauncher/BedrockNexusLauncher";
// GitHub's issue prefill drops long query strings; past this, the logs field
// is omitted and the user is told to paste the diagnostics manually.
const MAX_PREFILL_URL_LENGTH = 7000;

interface ReportProblemModalProps {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  /** Optional context (e.g. error details) pre-filled into the description. */
  initialDescription?: string;
}

export const ReportProblemModal: React.FC<ReportProblemModalProps> = ({
  isOpen,
  onOpenChange,
  initialDescription,
}) => {
  const { t } = useTranslation();
  const [description, setDescription] = useState("");
  const [includeDiagnostics, setIncludeDiagnostics] = useState(true);
  const [diagnostics, setDiagnostics] = useState("");
  const [diagnosticsFailed, setDiagnosticsFailed] = useState(false);
  const [loadingDiagnostics, setLoadingDiagnostics] = useState(false);

  useEffect(() => {
    if (!isOpen) return;
    setDescription(initialDescription ?? "");
    let cancelled = false;
    setLoadingDiagnostics(true);
    setDiagnosticsFailed(false);
    try {
      minecraft
        ?.GetDiagnosticsReport?.()
        .then((res: unknown) => {
          if (cancelled) return;
          const text = String(res ?? "");
          setDiagnostics(text);
          setDiagnosticsFailed(text.length === 0);
        })
        .catch(() => {
          if (!cancelled) setDiagnosticsFailed(true);
        })
        .finally(() => {
          if (!cancelled) setLoadingDiagnostics(false);
        });
    } catch {
      if (!cancelled) {
        setDiagnosticsFailed(true);
        setLoadingDiagnostics(false);
      }
    }
    return () => {
      cancelled = true;
    };
    // Re-seed (and reload diagnostics) whenever the modal opens; the parent
    // computes initialDescription at open time.
  }, [isOpen, initialDescription]);

  const title = useMemo(() => {
    const firstLine = description.trim().split("\n")[0]?.trim() ?? "";
    return firstLine.slice(0, 80) || t("reportProblem.default_title");
  }, [description, t]);

  const osLine = useMemo(
    () => /^- OS: (.*)$/m.exec(diagnostics)?.[1]?.trim() ?? "",
    [diagnostics],
  );
  const appVersion = useMemo(
    () => /^- App version: (.*)$/m.exec(diagnostics)?.[1]?.trim() ?? "",
    [diagnostics],
  );

  const buildIssueUrl = (withLogs: boolean) => {
    const params = new URLSearchParams();
    params.set("template", "bug_report.yml");
    params.set("title", `[Bug] ${title}`);
    params.set("description", description.trim());
    if (osLine) params.set("platform", osLine);
    if (appVersion) params.set("version", appVersion);
    if (withLogs && diagnostics) params.set("logs", diagnostics);
    return `${REPO_URL}/issues/new?${params.toString()}`;
  };

  const logsTooLong =
    includeDiagnostics &&
    Boolean(diagnostics) &&
    buildIssueUrl(true).length > MAX_PREFILL_URL_LENGTH;

  const openIssue = () => {
    if (!description.trim()) return;
    Browser.OpenURL(buildIssueUrl(!logsTooLong));
  };

  const searchIssues = () => {
    const query = description.trim().slice(0, 120);
    const suffix = query ? `?q=is%3Aissue+${encodeURIComponent(query)}` : "";
    Browser.OpenURL(`${REPO_URL}/issues${suffix}`);
  };

  const copyDiagnostics = async () => {
    try {
      await navigator.clipboard.writeText(diagnostics);
      addToast({ title: t("reportProblem.copied"), color: "success" });
    } catch {
      addToast({ title: t("reportProblem.copy_failed"), color: "warning" });
    }
  };

  return (
    <UnifiedModal
      size="lg"
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      type="info"
      title={t("reportProblem.title")}
      isDismissable
      hideCloseButton={false}
      showCancelButton={false}
      showConfirmButton={false}
      footer={
        <div className="flex w-full flex-wrap justify-between gap-2">
          <Button
            variant="light"
            startContent={<FaSearch />}
            onPress={searchIssues}
          >
            {t("reportProblem.search_issues")}
          </Button>
          <Button
            color="primary"
            startContent={<FaGithub />}
            isDisabled={!description.trim()}
            onPress={openIssue}
          >
            {t("reportProblem.open_issue")}
          </Button>
        </div>
      }
    >
      <div className="flex flex-col gap-4">
        <Textarea
          label={t("reportProblem.description_label")}
          labelPlacement="outside"
          placeholder={t("reportProblem.description_placeholder")}
          value={description}
          onValueChange={setDescription}
          minRows={4}
          maxRows={8}
          variant="bordered"
          radius="sm"
        />

        <Switch
          size="sm"
          isSelected={includeDiagnostics}
          onValueChange={setIncludeDiagnostics}
        >
          <span className="text-sm text-default-600 dark:text-zinc-300">
            {t("reportProblem.include_diagnostics")}
          </span>
        </Switch>

        {includeDiagnostics && (
          <div className="flex flex-col gap-2">
            <div className="flex items-center justify-between gap-2">
              <p className="text-tiny text-default-500 dark:text-zinc-400">
                {t("reportProblem.diagnostics_preview")}
              </p>
              <Button
                size="sm"
                variant="light"
                startContent={<FaCopy />}
                isDisabled={!diagnostics}
                onPress={copyDiagnostics}
              >
                {t("reportProblem.copy")}
              </Button>
            </div>
            <pre className="max-h-44 overflow-auto rounded-xl border border-default-100 bg-default-50 p-3 text-tiny font-mono whitespace-pre-wrap break-all text-default-600 dark:border-white/5 dark:bg-white/5 dark:text-zinc-300">
              {loadingDiagnostics
                ? t("reportProblem.loading")
                : diagnostics || t("reportProblem.error")}
            </pre>
            {logsTooLong && (
              <p className="text-tiny text-warning-600 dark:text-warning-400">
                {t("reportProblem.too_long_note")}
              </p>
            )}
          </div>
        )}

        <p className="text-tiny text-default-500 dark:text-zinc-400">
          {t("reportProblem.privacy_note")}
        </p>
      </div>
    </UnifiedModal>
  );
};
