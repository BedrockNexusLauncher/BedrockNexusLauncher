import React from "react";
import {
  Button,
  Slider,
  Switch,
  Tabs,
  Tab,
} from "@heroui/react";
import { useTranslation } from "react-i18next";
import { useTheme } from "next-themes";
import {
  LuCheck,
  LuLayers,
  LuMoon,
  LuRotateCcw,
  LuSun,
} from "react-icons/lu";
import { useBackgroundAppearance } from "@/hooks/useBackgroundAppearance";
import { useCurrentBackground } from "@/utils/BackgroundContext";
import {
  getAppearanceStyle,
  getEffectiveSurfaceOpacity,
  getMaterialPreset,
  type AppearanceMode,
  type MaterialPreset,
} from "@/utils/backgroundAppearance";
import { BackgroundLayers } from "./BackgroundLayers";
import { COMPONENT_STYLES } from "@/constants/componentStyles";

const PRESETS: MaterialPreset[] = ["balanced", "clear", "solid"];

/**
 * Material personalization: presets, surface opacity/blur, wallpaper
 * overlay and readability guard, saved separately per light/dark theme.
 */
export const MaterialSettings: React.FC = () => {
  const { t } = useTranslation();
  const { resolvedTheme } = useTheme();
  const [mode, setMode] = React.useState<AppearanceMode>(
    resolvedTheme === "dark" ? "dark" : "light",
  );
  const { profiles, updateProfile } = useBackgroundAppearance();
  const background = useCurrentBackground();
  const profile = profiles[mode];
  const hasImage =
    Boolean(background?.bgData) && (background?.backgroundOpacity ?? 0) > 0;
  const effectiveOpacity = getEffectiveSurfaceOpacity(
    profile,
    mode,
    background?.backgroundBrightness ?? 100,
    background?.backgroundBlur ?? 0,
    background?.backgroundOpacity ?? 100,
  );
  const preset = PRESETS.find((name) => {
    const values = getMaterialPreset(mode, name);
    return (Object.keys(values) as (keyof typeof values)[]).every(
      (key) => values[key] === profile[key],
    );
  });
  const copy = (key: string) => t(`settings.appearance.material.${key}`);

  return (
    <div className="flex flex-col gap-4 p-5 mt-6 rounded-3xl bg-default-200/10 border border-default-200/50">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div className="flex items-center gap-2">
          <LuLayers className="text-primary-500" size={18} />
          <p className="text-small font-bold text-default-700">
            {copy("title")}
          </p>
        </div>
        <Tabs
          aria-label={copy("edit_mode")}
          selectedKey={mode}
          onSelectionChange={(k) => setMode(k as AppearanceMode)}
          variant="solid"
          size="sm"
          classNames={COMPONENT_STYLES.tabs}
        >
          <Tab
            key="light"
            title={
              <div className="flex items-center gap-1.5">
                <LuSun size={14} />
                {t("settings.appearance.theme_light")}
              </div>
            }
          />
          <Tab
            key="dark"
            title={
              <div className="flex items-center gap-1.5">
                <LuMoon size={14} />
                {t("settings.appearance.theme_dark")}
              </div>
            }
          />
        </Tabs>
      </div>
      <p className="text-tiny text-default-500">{copy("description")}</p>

      {background && (
        <div
          className="appearance-preview"
          data-testid="appearance-preview"
          data-material-scope=""
          data-wallpaper-active={hasImage ? "true" : undefined}
          data-readability={profile.readability ? "true" : undefined}
          style={{
            ...getAppearanceStyle(
              profile,
              mode,
              background.backgroundBrightness,
              background.backgroundBlur,
              background.backgroundOpacity,
            ),
            ...(!hasImage ? { "--wallpaper-surface-opacity": 1 } : {}),
          } as React.CSSProperties}
          role="img"
          aria-label={copy("preview_label")}
        >
          <BackgroundLayers background={background} mode={mode} />
          <div className="relative flex flex-col gap-3 p-4">
            <div className="flex items-center justify-between gap-2 text-xs font-semibold">
              <span className="material-preview-pill rounded-full px-3 py-1.5">
                {copy("preview")}
              </span>
              <span className="material-preview-pill rounded-full px-3 py-1.5">
                {t(`settings.appearance.theme_${mode}`)}
              </span>
            </div>
            <div className="material-preview-card rounded-2xl border p-4">
              <p className="text-xl font-black">Minecraft</p>
              <p className="text-xs opacity-80">Bedrock Edition</p>
              <p className="mt-3 text-xs opacity-80">{copy("preview_text")}</p>
            </div>
          </div>
        </div>
      )}

      {!hasImage && (
        <p className="text-tiny text-default-500" role="status">
          {copy("no_image")}
        </p>
      )}

      <div
        className="grid grid-cols-1 gap-2 sm:grid-cols-3"
        role="group"
        aria-label={copy("presets")}
      >
        {PRESETS.map((name) => (
          <Button
            key={name}
            variant="flat"
            color={preset === name ? "primary" : "default"}
            onPress={() => updateProfile(mode, getMaterialPreset(mode, name))}
            className="h-auto min-h-16 w-full justify-start whitespace-normal rounded-xl border border-default-200/50 p-3 text-left"
            aria-pressed={preset === name}
          >
            <span className="flex-1">
              <span className="block text-sm font-semibold">
                {copy(name)}
              </span>
              <span className="mt-1 block text-xs font-normal opacity-70">
                {copy(`${name}_desc`)}
              </span>
            </span>
            {preset === name && <LuCheck className="shrink-0" />}
          </Button>
        ))}
      </div>

      <div className="grid grid-cols-1 gap-5 sm:grid-cols-2">
        {(
          [
            ["surfaceOpacity", "opacity", 100, "%"],
            ["surfaceBlur", "blur", 32, "px"],
            ["overlayOpacity", "overlay", 80, "%"],
          ] as const
        ).map(([key, label, max, unit]) => (
          <div key={key} className="flex flex-col gap-2">
            <div className="flex items-center justify-between">
              <p className="text-tiny font-medium text-default-600">
                {copy(label)}
              </p>
              <span className="text-tiny font-mono text-primary-500">
                {profile[key]}
                {unit}
              </span>
            </div>
            <Slider
              size="sm"
              step={1}
              maxValue={max}
              minValue={0}
              aria-label={copy(label)}
              value={profile[key]}
              classNames={{
                filler: "bg-primary-500",
                thumb: "bg-primary-500",
              }}
              onChange={(v) =>
                updateProfile(mode, { [key]: Number(v) })
              }
            />
          </div>
        ))}
      </div>

      <div className="py-4 border-t border-default-200/50 flex items-center justify-between">
        <div className="flex flex-col gap-1">
          <p className="font-medium text-default-700 dark:text-zinc-200">
            {copy("readability")}
          </p>
          <p className="text-tiny text-default-500 dark:text-zinc-400">
            {copy("readability_desc")}
          </p>
        </div>
        <Switch
          size="sm"
          aria-label={copy("readability")}
          isSelected={profile.readability}
          onValueChange={(readability) =>
            updateProfile(mode, { readability })
          }
          classNames={{
            wrapper: "group-data-[selected=true]:bg-primary-500",
          }}
        />
      </div>

      <div className="flex flex-wrap items-center justify-between gap-3">
        <p className="text-tiny text-default-500" role="status">
          {hasImage && effectiveOpacity > profile.surfaceOpacity / 100 + 0.005
            ? t("settings.appearance.material.protected_opacity", {
                value: Math.round(effectiveOpacity * 100),
              })
            : copy(profile.readability ? "protected" : "unprotected")}
        </p>
        <Button
          size="sm"
          variant="light"
          onPress={() => updateProfile(mode, getMaterialPreset(mode))}
        >
          <LuRotateCcw size={14} />
          {copy("reset")}
        </Button>
      </div>

    </div>
  );
};
