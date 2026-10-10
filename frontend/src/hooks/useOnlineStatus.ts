import { useState, useEffect } from "react";

const readOnline = () => {
  try {
    if (typeof navigator !== "undefined" && typeof navigator.onLine === "boolean") {
      return navigator.onLine;
    }
  } catch {}
  return true;
};

export const useOnlineStatus = () => {
  const [isOnline, setIsOnline] = useState<boolean>(() => readOnline());

  useEffect(() => {
    const goOnline = () => setIsOnline(true);
    const goOffline = () => setIsOnline(false);
    window.addEventListener("online", goOnline);
    window.addEventListener("offline", goOffline);
    return () => {
      window.removeEventListener("online", goOnline);
      window.removeEventListener("offline", goOffline);
    };
  }, []);

  return { isOnline };
};
