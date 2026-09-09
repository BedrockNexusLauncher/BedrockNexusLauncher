import React, {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useState,
} from "react";
import { GetMarketplaceIndex } from "bindings/github.com/BedrockNexusLauncher/BedrockNexusLauncher/minecraft";

const INDEX_URL_KEY = "marketplace.indexUrl";

export const getIndexURL = (): string => {
  try {
    return localStorage.getItem(INDEX_URL_KEY) || "";
  } catch {
    return "";
  }
};

export const setIndexURL = (url: string) => {
  try {
    if (url) localStorage.setItem(INDEX_URL_KEY, url);
    else localStorage.removeItem(INDEX_URL_KEY);
  } catch {}
};

type MarketItem = any;

interface MarketplaceState {
  items: MarketItem[];
  updatedAt: string;
  loading: boolean;
  error: string | null;
  reload: () => Promise<void>;
}

const MarketplaceContext = createContext<MarketplaceState | null>(null);

export const MarketplaceProvider: React.FC<{ children: React.ReactNode }> = ({
  children,
}) => {
  const [items, setItems] = useState<MarketItem[]>([]);
  const [updatedAt, setUpdatedAt] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState<string | null>(null);

  const load = useCallback(async () => {
    setLoading(true);
    setError(null);
    try {
      const index = await GetMarketplaceIndex(getIndexURL());
      setItems(index?.items || []);
      setUpdatedAt(index?.updatedAt || "");
    } catch (e) {
      console.error("Failed to load marketplace index:", e);
      setItems([]);
      setError("Failed to load marketplace index");
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  return (
    <MarketplaceContext.Provider
      value={{ items, updatedAt, loading, error, reload: load }}
    >
      {children}
    </MarketplaceContext.Provider>
  );
};

export const useMarketplace = (): MarketplaceState => {
  const ctx = useContext(MarketplaceContext);
  if (!ctx) {
    throw new Error("useMarketplace must be used within MarketplaceProvider");
  }
  return ctx;
};
