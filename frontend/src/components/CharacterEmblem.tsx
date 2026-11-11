import type { CharacterTheme } from "@/constants/characterThemes";

/**
 * Character emblem watermark, ported from Novelfa's EmblemWatermark
 * (theme/ThemeArtwork.kt): the active character's emblem drawn large
 * at low opacity with its gradient, hit-test invisible. In the
 * launcher it sits fixed in the bottom corner, under the content.
 */
export const CharacterEmblem: React.FC<{
  theme: CharacterTheme;
  disabled?: boolean;
}> = ({ theme, disabled }) => {
  if (disabled) return null;
  return (
    <div
      className="fixed -bottom-10 -end-10 z-0 pointer-events-none select-none"
      aria-hidden="true"
    >
      <svg
        width={320}
        height={320}
        viewBox="0 0 100 100"
        style={{ opacity: 0.07 }}
      >
        <defs>
          <linearGradient id="char-emblem-grad" x1="0" y1="0" x2="1" y2="1">
            <stop offset="0%" stopColor="var(--char-grad-1)" />
            <stop offset="100%" stopColor="var(--char-grad-2)" />
          </linearGradient>
        </defs>
        <path
          d={theme.emblemData}
          fill="url(#char-emblem-grad)"
          fillRule={theme.emblemEvenOdd ? "evenodd" : "nonzero"}
        />
      </svg>
    </div>
  );
};
