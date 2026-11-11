import { useCallback, useEffect, useState } from "react";
import {
  CHARACTER_THEME_CHANGED_EVENT,
  CHARACTER_THEME_KEY,
  findCharacterTheme,
  type CharacterTheme,
} from "@/constants/characterThemes";

const readCharacterThemeId = (): string | null => {
  try {
    return localStorage.getItem(CHARACTER_THEME_KEY) || null;
  } catch {
    return null;
  }
};

/**
 * Active Minecraft character theme, persisted in localStorage.
 * Empty id = no character theme: the saved light/dark color theme
 * applies untouched. Selection is broadcast on
 * `app-character-theme-changed` so `useThemeColors` (which owns the
 * --theme-* vars) and the picker stay in sync without sharing state.
 */
export const useCharacterTheme = () => {
  const [characterThemeId, setCharacterThemeIdState] = useState<
    string | null
  >(readCharacterThemeId);

  useEffect(() => {
    const handler = () => setCharacterThemeIdState(readCharacterThemeId());
    window.addEventListener(CHARACTER_THEME_CHANGED_EVENT, handler);
    return () =>
      window.removeEventListener(CHARACTER_THEME_CHANGED_EVENT, handler);
  }, []);

  const setCharacterTheme = useCallback((id: string | null) => {
    try {
      if (id) {
        localStorage.setItem(CHARACTER_THEME_KEY, id);
      } else {
        localStorage.removeItem(CHARACTER_THEME_KEY);
      }
    } catch {
      // Storage unavailable: keep the in-memory selection only.
    }
    setCharacterThemeIdState(id);
    window.dispatchEvent(new CustomEvent(CHARACTER_THEME_CHANGED_EVENT));
  }, []);

  const characterTheme: CharacterTheme | null =
    findCharacterTheme(characterThemeId);

  return { characterThemeId, characterTheme, setCharacterTheme };
};
