import React, { useState } from "react";
import { useTranslation } from "react-i18next";
import { useNavigate } from "react-router-dom";
import { Button, Spinner } from "@heroui/react";
import { LuShieldCheck, LuTriangleAlert } from "react-icons/lu";
import * as minecraft from "bindings/github.com/BedrockNexusLauncher/BedrockNexusLauncher/minecraft";
import { ROUTES } from "@/constants/routes";

type ConsentStatus = "idle" | "restarting" | "failed";

/**
 * ElevationConsentPage
 * صفحه‌ی گرافیکی رضایت برای اجرای لانچر با دسترسی مدیر (جایگزین MessageBox ویندوز).
 * پس از پذیرش، بک‌اند درخواست UAC ارسال می‌کند و پروسه‌ی فعلی بسته می‌شود.
 */
export default function ElevationConsentPage() {
  const { t } = useTranslation();
  const navigate = useNavigate();
  const [status, setStatus] = useState<ConsentStatus>("idle");

  const handleAccept = async () => {
    try {
      const result = String(
        (await minecraft?.AcceptElevationConsent?.()) || "FAILED",
      );
      if (result === "OK_RESTARTING") {
        setStatus("restarting");
      } else if (result === "ALREADY_ELEVATED") {
        navigate(ROUTES.home, { replace: true });
      } else {
        setStatus("failed");
      }
    } catch {
      setStatus("failed");
    }
  };

  const handleDecline = () => {
    try {
      minecraft?.DeclineElevationConsent?.();
    } catch {}
    navigate(ROUTES.home, { replace: true });
  };

  const handleRetry = () => {
    setStatus("idle");
  };

  return (
    <div className="w-full h-full min-h-[60vh] flex items-center justify-center p-6">
      <div
        className={`w-full max-w-md rounded-3xl border border-default-200/60 dark:border-default-100/20 bg-content1/80 backdrop-blur-xl shadow-2xl shadow-black/10 dark:shadow-black/40 overflow-hidden transition-opacity duration-300 ${
          status === "restarting" ? "pointer-events-none opacity-90" : ""
        }`}
      >
        {/* نوار بالایی گرادیانی با آیکون سپر */}
        <div className="relative flex flex-col items-center gap-4 px-8 pt-10 pb-6">
          <div className="absolute inset-x-0 top-0 h-32 bg-gradient-to-b from-[rgb(var(--theme-500,34_197_94))]/15 to-transparent pointer-events-none" />
          <div className="relative">
            <div className="absolute inset-0 rounded-full bg-[rgb(var(--theme-500,34_197_94))]/25 blur-xl scale-125" />
            <div className="relative w-20 h-20 rounded-full bg-gradient-to-br from-[rgb(var(--theme-400,74_222_128))] to-[rgb(var(--theme-600,22_163_74))] flex items-center justify-center shadow-lg shadow-[rgb(var(--theme-500,34_197_94))]/30">
              <LuShieldCheck className="w-10 h-10 text-white" strokeWidth={1.8} />
            </div>
          </div>

          <h1 className="relative text-2xl font-bold text-foreground text-center">
            {t("elevation.title")}
          </h1>
          <p className="relative text-sm text-default-500 text-center leading-relaxed">
            {t("elevation.description")}
          </p>
        </div>

        {/* بدنه: وضعیت‌ها و دکمه‌ها */}
        <div className="px-8 pb-8 flex flex-col gap-3">
          {status === "restarting" ? (
            <div className="flex flex-col items-center gap-4 py-4">
              <Spinner size="lg" color="primary" />
              <p className="text-sm text-default-500 text-center">
                {t("elevation.restarting")}
              </p>
            </div>
          ) : status === "failed" ? (
            <>
              <div className="flex items-start gap-3 rounded-2xl bg-warning-100/60 dark:bg-warning-100/10 border border-warning-200/60 dark:border-warning-200/20 px-4 py-3">
                <LuTriangleAlert className="w-5 h-5 text-warning-500 shrink-0 mt-0.5" />
                <p className="text-xs text-warning-600 dark:text-warning-400 leading-relaxed">
                  {t("elevation.failed")}
                </p>
              </div>
              <Button
                color="primary"
                size="lg"
                className="w-full font-semibold"
                onPress={handleAccept}
              >
                {t("elevation.retry")}
              </Button>
              <Button
                variant="flat"
                size="lg"
                className="w-full font-medium text-default-600"
                onPress={handleDecline}
              >
                {t("elevation.decline")}
              </Button>
            </>
          ) : (
            <>
              <Button
                color="primary"
                size="lg"
                className="w-full font-semibold"
                startContent={<LuShieldCheck className="w-5 h-5" />}
                onPress={handleAccept}
              >
                {t("elevation.accept")}
              </Button>
              <Button
                variant="flat"
                size="lg"
                className="w-full font-medium text-default-600"
                onPress={handleDecline}
              >
                {t("elevation.decline")}
              </Button>
              <p className="text-xs text-default-400 text-center mt-2">
                {t("elevation.note")}
              </p>
            </>
          )}
        </div>
      </div>
    </div>
  );
}
