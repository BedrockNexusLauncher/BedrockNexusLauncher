import React from "react";
import { useTranslation } from "react-i18next";
import { useTheme } from "next-themes";
import { LuCheck, LuSparkles } from "react-icons/lu";
import {
  CHARACTER_THEMES,
  characterDisplayName,
  characterDisplayTagline,
  type CharacterParticleKind,
  type CharacterTheme,
} from "@/constants/characterThemes";
import { useCharacterTheme } from "@/hooks/useCharacterTheme";

const PARTICLE_FA: Record<CharacterParticleKind, string> = {
  blocks: "بلوک",
  spores: "هاگ",
  ender: "جرقه اندر",
  bubbles: "حباب",
  snow: "برف",
  echo: "پژواک",
};

const PARTICLE_EN: Record<CharacterParticleKind, string> = {
  blocks: "Blocks",
  spores: "Spores",
  ender: "Ender sparks",
  bubbles: "Bubbles",
  snow: "Snow",
  echo: "Echo",
};

/**
 * Character theme picker, rendered in Settings -> Appearance
 * (inside MaterialSettings). Modelled on Novelfa's ThemePreviewCard:
 * each card shows the character's own palette (background, text
 * lines, accent bar) so the user sees the theme before picking it.
 * No locale-file strings: names and taglines live in
 * characterThemes.ts, and the few chrome labels below are picked
 * from the current language directly, so no other locale file
 * needs to change when a character is added.
 */
export const CharacterThemePicker: React.FC = () => {
  const { i18n } = useTranslation();
  const { resolvedTheme } = useTheme();
  const { characterThemeId, setCharacterTheme } = useCharacterTheme();
  const isEnglish = (i18n.language || "en").toLowerCase().startsWith("en");
  const isDark = resolvedTheme === "dark";

  const card = (theme: CharacterTheme | null) => {
    const pal = theme ? (isDark ? theme.dark : theme.light) : null;
    const selected =
      (theme === null && !characterThemeId) ||
      (theme !== null && characterThemeId === theme.id);
    const name = theme
      ? characterDisplayName(theme, isEnglish)
      : isEnglish
        ? "No character"
        : "بدون تم شخصیتی";
    const tagline = theme
      ? characterDisplayTagline(theme, isEnglish)
      : isEnglish
        ? "Keep the saved color theme"
        : "تم رنگی ذخیره‌شده باقی می‌ماند";
    return (
      <button
        key={theme?.id ?? "none"}
        type="button"
        onClick={() => setCharacterTheme(theme?.id ?? null)}
        aria-pressed={selected}
        className={`overflow-hidden rounded-2xl border text-start transition-transform hover:scale-[1.02] focus-visible:outline-2 focus-visible:outline-primary-500 ${
          selected
            ? "border-primary-500 ring-2 ring-primary-500/40"
            : "border-default-200/50"
        }`}
      >
        <div
          className="h-[72px] p-3"
          style={{ backgroundColor: pal?.bg ?? "transparent" }}
        >
          <div
            className="h-2 w-4/5 rounded-sm"
            style={{
              backgroundColor: pal?.text ?? "var(--color-default-400)",
              opacity: 0.9,
            }}
          />
          <div
            className="mt-1.5 h-2 w-3/5 rounded-sm"
            style={{
              backgroundColor: pal?.text ?? "var(--color-default-400)",
              opacity: 0.55,
            }}
          />
          <div
            className="mt-2 h-[3px] w-full"
            style={{
              backgroundColor: pal?.accent ?? "var(--color-primary-500)",
            }}
          />
        </div>
        <div className="flex items-start justify-between gap-2 p-3">
          <div className="min-w-0">
            <p className="truncate text-sm font-bold text-default-700">
              {name}
            </p>
            <p className="mt-0.5 line-clamp-2 text-xs text-default-500">
              {tagline}
            </p>
            {theme && (
              <p className="mt-1 text-[11px] text-default-400">
                {isEnglish
                  ? PARTICLE_EN[theme.particle]
                  : PARTICLE_FA[theme.particle]}
              </p>
            )}
          </div>
          {selected && (
            <LuCheck className="mt-0.5 shrink-0 text-primary-500" size={16} />
          )}
        </div>
      </button>
    );
  };

  return (
    <div className="flex flex-col gap-4 p-5 rounded-3xl bg-default-200/10 border border-default-200/50">
      <div className="flex items-center gap-2">
        <LuSparkles className="text-primary-500" size={18} />
        <p className="text-small font-bold text-default-700">
          {isEnglish ? "Character themes" : "تم‌های شخصیتی"}
        </p>
      </div>
      <p className="text-tiny text-default-500">
        {isEnglish
          ? "A Minecraft character tints the whole launcher — accent, particles and emblem — in the style of Novelfa's character themes. All six are open from the start."
          : "یک کاراکتر ماینکرفت کل لانچر را رنگ می‌کند — اکسنت، ذره‌ها و نماد — به سبک تم‌های شخصیتی ناول‌فا. هر شش تم از اول باز هستند."}
      </p>
      <div
        className="grid grid-cols-2 gap-3 sm:grid-cols-3"
        role="group"
        aria-label={isEnglish ? "Character themes" : "تم‌های شخصیتی"}
      >
        {card(null)}
        {CHARACTER_THEMES.map((theme) => card(theme))}
      </div>
    </div>
  );
};
