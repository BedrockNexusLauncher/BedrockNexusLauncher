import { useState, useEffect } from "react";

const STORAGE_KEY = "app.themedStrokes";
const EVENT_NAME = "app-themed-strokes-changed";

const readPref = (): boolean => {
  try {
    return localStorage.getItem(STORAGE_KEY) === "true";
  } catch {
    return false;
  }
};

/** Applies the optional "themed strokes" personalization setting: toggles
    the `themed-strokes` class on <html>; the visual effect lives in
    style.css (accent-tinted borders, see html.themed-strokes rules). */
export const useThemedStrokes = () => {
  const [themedStrokes, setThemedStrokes] = useState<boolean>(readPref);

  useEffect(() => {
    document.documentElement.classList.toggle("themed-strokes", themedStrokes);
  }, [themedStrokes]);

  useEffect(() => {
    const handler = () => setThemedStrokes(readPref());
    window.addEventListener(EVENT_NAME, handler);
    return () => window.removeEventListener(EVENT_NAME, handler);
  }, []);

  return { themedStrokes };
};
