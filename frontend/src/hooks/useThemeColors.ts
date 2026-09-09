import { useState, useEffect } from "react";
import { THEMES, hexToRgb, hexToHsl, generateTheme } from "@/constants/themes";

/** Default accent palette (matches the launcher's purple logo). */
export const DEFAULT_THEME_COLOR = "violet";
export const DEFAULT_CUSTOM_THEME_COLOR = "#8b5cf6";

type ModeKey = "light" | "dark";

const otherMode = (mode: ModeKey): ModeKey =>
  mode === "light" ? "dark" : "light";

// Theme colors are stored per mode (app.lightThemeColor / app.darkThemeColor).
// If the mode being resolved has no saved color yet, fall back to the other
// mode's color so a color picked in one mode carries over to both until the
// user customizes them separately; then the legacy app.themeColor, then the
// default. This keeps switching dark/light from "resetting" to the default.
export const readThemeColorPref = (mode: ModeKey): string => {
  try {
    const own = localStorage.getItem(`app.${mode}ThemeColor`);
    if (own) return own;
    const other = localStorage.getItem(
      `app.${otherMode(mode)}ThemeColor`,
    );
    if (other) return other;
    const legacy = localStorage.getItem("app.themeColor");
    if (legacy === "rose") return "pink";
    return legacy || DEFAULT_THEME_COLOR;
  } catch {
    return DEFAULT_THEME_COLOR;
  }
};

export const readCustomThemeColorPref = (mode: ModeKey): string => {
  try {
    const own = localStorage.getItem(`app.${mode}CustomThemeColor`);
    if (own) return own;
    const other = localStorage.getItem(
      `app.${otherMode(mode)}CustomThemeColor`,
    );
    if (other) return other;
    return (
      localStorage.getItem("app.customThemeColor") ||
      DEFAULT_CUSTOM_THEME_COLOR
    );
  } catch {
    return DEFAULT_CUSTOM_THEME_COLOR;
  }
};

export const useThemeColors = (resolvedTheme: string | undefined) => {
  const [themeColorsReady, setThemeColorsReady] = useState<boolean>(false);
  const [lightThemeColor, setLightThemeColor] = useState<string>(() =>
    readThemeColorPref("light"),
  );

  const [lightCustomThemeColor, setLightCustomThemeColor] = useState<string>(
    () => readCustomThemeColorPref("light"),
  );

  const [darkThemeColor, setDarkThemeColor] = useState<string>(() =>
    readThemeColorPref("dark"),
  );

  const [darkCustomThemeColor, setDarkCustomThemeColor] = useState<string>(
    () => readCustomThemeColorPref("dark"),
  );

  useEffect(() => {
    const handler = () => {
      setLightThemeColor(readThemeColorPref("light"));
      setLightCustomThemeColor(readCustomThemeColorPref("light"));
      setDarkThemeColor(readThemeColorPref("dark"));
      setDarkCustomThemeColor(readCustomThemeColorPref("dark"));
    };
    window.addEventListener("app-theme-changed", handler);
    return () => window.removeEventListener("app-theme-changed", handler);
  }, []);

  useEffect(() => {
    const isDark = resolvedTheme === "dark";
    const currentColor = isDark ? darkThemeColor : lightThemeColor;
    const currentCustomColor = isDark
      ? darkCustomThemeColor
      : lightCustomThemeColor;

    let theme = THEMES[currentColor];
    if (currentColor === "custom") {
      theme = generateTheme(currentCustomColor);
    }
    if (!theme) theme = THEMES[DEFAULT_THEME_COLOR];

    const root = document.documentElement;

    Object.keys(theme).forEach((key) => {
      const k = Number(key);
      root.style.setProperty(`--theme-${k}`, hexToRgb(theme[k]));
    });
    // Accent used by the optional "themed strokes" setting (style.css);
    // a lighter shade in dark mode keeps the tint visible on black.
    root.style.setProperty(
      "--theme-stroke-hsl",
      hexToHsl(theme[isDark ? 400 : 500] || theme[500]),
    );
    setThemeColorsReady(true);
  }, [
    resolvedTheme,
    lightThemeColor,
    lightCustomThemeColor,
    darkThemeColor,
    darkCustomThemeColor,
  ]);

  return { themeColorsReady };
};
