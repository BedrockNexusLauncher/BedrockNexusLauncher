/**
 * Minecraft character themes for the Bedrock Nexus launcher.
 *
 * Port of the Novelfa character-theme model
 * (android/.../domain/CharacterThemes.kt): each character carries a
 * fa/en name + tagline, a particle kind, and a 16-color palette for
 * both dark and light mode, in the same field order Novelfa uses:
 * bg, sidebar, card, text, muted, hover, border, soft, hero1, hero2,
 * accent, accentDark, grad1, grad2, aura1, aura2.
 *
 * What is deliberately NOT ported in v1:
 * - Novelfa's rank / XP unlock gates and secret-code unlocks
 *   (BEAKO / EMT / ...). All seven themes are open from the start;
 *   an unlock-by-playtime gate is a later decision, not a hidden one.
 * - Novelfa's WPF emblem Geometry. The emblems here are original,
 *   simple geometric silhouettes drawn in a 100x100 box (pixel-face
 *   style), not Mojang artwork. The Nexus emblem is the launcher's
 *   own hexagon badge (logo: vivid violet #773FF9 on dark navy
 *   #151723, `frontend/src/assets/images/ic_nexus_logo.png`).
 *
 * How a theme is applied: the resolved palette's `accent` feeds the
 * existing `generateTheme()` scale in `useThemeColors`, so the whole
 * app (primary colors, themed strokes, background base) follows the
 * character, exactly like Novelfa's plain palette is tinted from one
 * accent. `grad1`/`grad2`/`aura1`/`aura2` are exposed as CSS vars
 * (`--char-grad-1`, `--char-grad-2`, `--char-aura-1`, `--char-aura-2`)
 * for the emblem watermark and the particle layer.
 */

export type CharacterParticleKind =
  | "blocks"
  | "spores"
  | "ender"
  | "bubbles"
  | "snow"
  | "echo"
  | "hex";

export interface CharacterPalette {
  bg: string;
  sidebar: string;
  card: string;
  text: string;
  muted: string;
  hover: string;
  border: string;
  soft: string;
  hero1: string;
  hero2: string;
  accent: string;
  accentDark: string;
  grad1: string;
  grad2: string;
  aura1: string;
  aura2: string;
}

export interface CharacterTheme {
  id: string;
  nameFa: string;
  taglineFa: string;
  nameEn: string;
  taglineEn: string;
  particle: CharacterParticleKind;
  dark: CharacterPalette;
  light: CharacterPalette;
  /** Simple geometric emblem path data in a 100x100 box. */
  emblemData: string;
  /** True when the emblem path uses even-odd holes (eyes, mouth). */
  emblemEvenOdd?: boolean;
}

const parsePalette = (spaced: string): CharacterPalette => {
  const v = spaced.split(/\s+/).filter(Boolean);
  if (v.length < 16) {
    throw new Error(`character palette needs 16 colors, got ${v.length}`);
  }
  const [
    bg,
    sidebar,
    card,
    text,
    muted,
    hover,
    border,
    soft,
    hero1,
    hero2,
    accent,
    accentDark,
    grad1,
    grad2,
    aura1,
    aura2,
  ] = v;
  return {
    bg,
    sidebar,
    card,
    text,
    muted,
    hover,
    border,
    soft,
    hero1,
    hero2,
    accent,
    accentDark,
    grad1,
    grad2,
    aura1,
    aura2,
  };
};

export const CHARACTER_THEMES: CharacterTheme[] = [
  {
    // The launcher's own brand theme, first in the list. Shares the
    // violet family with Enderman by brand (logo #773FF9), told apart
    // by its darker navy base, hexagon emblem and hex particles.
    id: "nexus",
    nameFa: "بدراک نکسوس",
    taglineFa: "نشان شش‌ضلعی، بنفشِ خودِ نکسوس",
    nameEn: "Bedrock Nexus",
    taglineEn: "The hexagon badge, Nexus violet itself",
    particle: "hex",
    dark: parsePalette(
      "#0C0A16 #110D22 #191233 #EFE9FF #9A8FC4 #241B47 #2E2458 #2A2150 #221A45 #140E2C #8B5CF6 #7C3AED #A78BFA #4C1D95 #8B5CF6 #C4B5FD",
    ),
    light: parsePalette(
      "#F5F3FF #FFFFFF #FFFFFF #1D1445 #6C63A6 #EDE9FE #DDD6FE #E6E0FF #EFEAFF #E0D8FF #6D28D9 #5B21B6 #7C3AED #4C1D95 #8B5CF6 #C4B5FD",
    ),
    emblemData:
      "M50 4 L92 27 V73 L50 96 L8 73 V27 Z M34 36h12v10H34z M54 36h12v10H54z M44 52h12v6H44z M47 58h6v10H47z",
    emblemEvenOdd: true,
  },
  {
    id: "steve",
    nameFa: "استیو",
    taglineFa: "اولین قدم‌ها در دنیای بلوکی",
    nameEn: "Steve",
    taglineEn: "First steps in a blocky world",
    particle: "blocks",
    dark: parsePalette(
      "#0B1210 #0E1815 #14201C #E8F0EA #8AA095 #1F2F27 #26382F #1E3A32 #17302A #101F1B #0EA5E9 #0284C7 #38BDF8 #2563EB #38BDF8 #60A5FA",
    ),
    light: parsePalette(
      "#F2F7F5 #FFFFFF #FFFFFF #12211C #5E7A6B #E9F3EC #D5E5DA #DFF3EA #EAF6EE #D8EFE6 #0284C7 #0369A1 #0EA5E9 #2563EB #38BDF8 #3B82F6",
    ),
    emblemData:
      "M22 18h56v56H22z M34 38h10v8H34z M56 38h10v8H56z M40 54h20v5H40z",
    emblemEvenOdd: true,
  },
  {
    id: "creeper",
    nameFa: "کریپر",
    taglineFa: "بی‌صدا، سبز، و همیشه نزدیک",
    nameEn: "Creeper",
    taglineEn: "Silent, green, and always close",
    particle: "spores",
    dark: parsePalette(
      "#0A120A #0E1A0E #152415 #EAF6E6 #8FB58A #1E331D #27402A #1B3A1E #16301A #0F2210 #4ADE80 #22C55E #4ADE80 #A3E635 #4ADE80 #86EFAC",
    ),
    light: parsePalette(
      "#F3F8F1 #FFFFFF #FFFFFF #14260F #5F8058 #E8F4E4 #D3E6CC #DFF2D8 #E9F6E4 #D6EECB #16A34A #15803D #22C55E #65A30D #4ADE80 #86EFAC",
    ),
    emblemData:
      "M22 18h56v56H22z M32 36h12v12H32z M56 36h12v12H56z M44 48h12v14H44z M38 62h24v12H38z",
    emblemEvenOdd: true,
  },
  {
    id: "enderman",
    nameFa: "اندرمن",
    taglineFa: "از تاریکی می‌آید، به تاریکی برمی‌گردد",
    nameEn: "Enderman",
    taglineEn: "Comes from the dark, returns to the dark",
    particle: "ender",
    dark: parsePalette(
      "#0D0A18 #120E24 #1A1433 #EFE9FF #9A8FC4 #241C47 #2E2458 #2A2150 #221A45 #150F2E #A78BFA #8B5CF6 #A78BFA #7C3AED #A78BFA #C4B5FD",
    ),
    light: parsePalette(
      "#F5F3FF #FFFFFF #FFFFFF #1D1445 #6C63A6 #EDE9FE #DDD6FE #E6E0FF #EFEAFF #E0D8FF #7C3AED #6D28D9 #8B5CF6 #6D28D9 #A78BFA #C4B5FD",
    ),
    emblemData: "M26 36h18v10H26z M56 36h18v10H56z M44 62h12v4H44z",
  },
  {
    id: "axolotl",
    nameFa: "آکسولوتل",
    taglineFa: "صورتیِ آرومِ آب‌های عمیق",
    nameEn: "Axolotl",
    taglineEn: "The calm pink of deep waters",
    particle: "bubbles",
    dark: parsePalette(
      "#140A12 #1B0E19 #271423 #FDEDF5 #B58AA5 #341A2E #43203C #3A1B34 #301627 #1E0C18 #F472B6 #EC4899 #F472B6 #FB7185 #F472B6 #F9A8D4",
    ),
    light: parsePalette(
      "#FFF4F8 #FFFFFF #FFFFFF #3A1226 #9A6580 #FFE6F0 #F8D2E3 #FFE0EE #FFEAF3 #FFD9E8 #DB2777 #BE185D #EC4899 #F43F5E #F472B6 #F9A8D4",
    ),
    emblemData:
      "M30 30h40v34H30z M18 28h8v10H18z M18 44h8v10H18z M74 28h8v10H74z M74 44h8v10H74z M40 42h6v6H40z M54 42h6v6H54z",
    emblemEvenOdd: true,
  },
  {
    id: "wolf",
    nameFa: "گرگ",
    taglineFa: "وفادار، زیر نور ماه",
    nameEn: "Wolf",
    taglineEn: "Loyal, under the moonlight",
    particle: "snow",
    dark: parsePalette(
      "#0C0E14 #10131C #171B28 #E8ECF5 #8B93A8 #202634 #2A3040 #232838 #1B2030 #12151F #94A3B8 #64748B #CBD5E1 #64748B #E2E8F0 #94A3B8",
    ),
    light: parsePalette(
      "#F4F6F9 #FFFFFF #FFFFFF #1B2233 #64748B #E8ECF3 #D5DCE6 #E2E8F0 #EDF1F6 #DDE4EC #475569 #334155 #64748B #334155 #94A3B8 #CBD5E1",
    ),
    emblemData:
      "M20 18l16 12h28l16-12-4 44H24z M38 44h8v6h-8z M54 44h8v6h-8z",
    emblemEvenOdd: true,
  },
  {
    id: "warden",
    nameFa: "واردن",
    taglineFa: "سکوتِ عمیق، قلبِ تپنده‌ی تاریکی",
    nameEn: "Warden",
    taglineEn: "Deep silence, the beating heart of the dark",
    particle: "echo",
    dark: parsePalette(
      "#05090C #081116 #0C1A20 #E6F7F5 #6FA8A0 #122A30 #1A3A42 #0F2E35 #0E242B #08161B #22D3EE #0891B2 #22D3EE #0D9488 #22D3EE #67E8F9",
    ),
    light: parsePalette(
      "#F0F7F6 #FFFFFF #FFFFFF #0B2422 #5B8078 #E2F2EF #C6E2DC #D2EDE8 #E6F4F1 #CFEAE4 #0E7490 #155E75 #06B6D4 #0D9488 #22D3EE #67E8F9",
    ),
    emblemData:
      "M18 30c8-14 20-20 32-18-10 4-18 10-22 20z M82 30c-8-14-20-20-32-18 10 4 18 10 22 20z M34 34h32v30H34z M46 44h8v8h-8z",
    emblemEvenOdd: true,
  },
];

export const findCharacterTheme = (
  id: string | null | undefined,
): CharacterTheme | null => {
  if (!id) return null;
  return CHARACTER_THEMES.find((c) => c.id === id) ?? null;
};

export const characterPalette = (
  theme: CharacterTheme | null,
  isDark: boolean,
): CharacterPalette | null => {
  if (!theme) return null;
  return isDark ? theme.dark : theme.light;
};

export const characterDisplayName = (
  theme: CharacterTheme,
  isEnglish: boolean,
): string => (isEnglish ? theme.nameEn : theme.nameFa);

export const characterDisplayTagline = (
  theme: CharacterTheme,
  isEnglish: boolean,
): string => (isEnglish ? theme.taglineEn : theme.taglineFa);

/** localStorage key + change event, mirroring app.themeMode / app.themeColor. */
export const CHARACTER_THEME_KEY = "app.characterTheme";
export const CHARACTER_THEME_CHANGED_EVENT = "app-character-theme-changed";
