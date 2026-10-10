import { useTranslation } from "react-i18next";
import { useOnlineStatus } from "@/hooks/useOnlineStatus";

export const OfflineBanner = () => {
  const { t } = useTranslation();
  const { isOnline } = useOnlineStatus();

  if (isOnline) return null;

  return (
    <div
      role="alert"
      className="absolute top-0 start-0 end-0 z-20 px-4 py-1.5 text-center text-small font-medium bg-warning-500/15 text-warning-700 dark:text-warning-400 border-b border-warning-500/30 backdrop-blur"
    >
      {t("common.offline.banner")}
    </div>
  );
};
