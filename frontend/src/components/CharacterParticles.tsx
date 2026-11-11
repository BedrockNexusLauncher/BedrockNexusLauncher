import { useEffect, useRef } from "react";
import type { CharacterPalette, CharacterParticleKind } from "@/constants/characterThemes";

/**
 * Animated character-theme particle layer.
 *
 * Direct port of the Novelfa particle model
 * (android/.../theme/ParticleLayer.kt, itself ported from WPF
 * ThemeParticles.cs): per-kind count / size / duration / peak /
 * rising / sway, a fade envelope (0 -> peak -> 0), rotation for
 * snow / echo, and a twinkle pulsation for ender sparkles.
 * Colors come from the active character palette
 * (accent / aura2 / text-or-accent), peak dimmed in light mode.
 * Pure canvas draw, hit-test invisible, and fully skipped when
 * animations are disabled (Settings -> Disable Animations).
 */

interface KindParams {
  count: number;
  sizeMin: number;
  sizeMax: number;
  durMin: number;
  durMax: number;
  peak: number;
  rising: boolean;
  swayAmp: number;
  rotates: boolean;
  shape: "dot" | "square" | "spark" | "bubble" | "flake" | "shard" | "hex";
}

const paramsFor = (type: CharacterParticleKind): KindParams => {
  switch (type) {
    case "blocks":
      return { count: 12, sizeMin: 6, sizeMax: 12, durMin: 12, durMax: 20, peak: 0.7, rising: false, swayAmp: 20, rotates: false, shape: "square" };
    case "spores":
      return { count: 14, sizeMin: 3, sizeMax: 7, durMin: 8, durMax: 16, peak: 0.7, rising: true, swayAmp: 18, rotates: false, shape: "dot" };
    case "ender":
      return { count: 10, sizeMin: 7, sizeMax: 15, durMin: 11, durMax: 22, peak: 0.8, rising: true, swayAmp: 14, rotates: false, shape: "spark" };
    case "bubbles":
      return { count: 12, sizeMin: 5, sizeMax: 11, durMin: 10, durMax: 20, peak: 0.55, rising: true, swayAmp: 24, rotates: false, shape: "bubble" };
    case "snow":
      return { count: 18, sizeMin: 5, sizeMax: 11, durMin: 12, durMax: 24, peak: 0.85, rising: false, swayAmp: 28, rotates: true, shape: "flake" };
    case "echo":
      return { count: 16, sizeMin: 4, sizeMax: 9, durMin: 14, durMax: 26, peak: 0.6, rising: false, swayAmp: 22, rotates: true, shape: "shard" };
    case "hex":
      return { count: 12, sizeMin: 6, sizeMax: 12, durMin: 14, durMax: 24, peak: 0.7, rising: false, swayAmp: 24, rotates: true, shape: "hex" };
  }
};

/** Deterministic per-index random, so particles don't jump on re-render. */
const mulberry32 = (seed: number) => {
  let a = seed >>> 0;
  return () => {
    a |= 0;
    a = (a + 0x6d2b79f5) | 0;
    let t = Math.imul(a ^ (a >>> 15), 1 | a);
    t = (t + Math.imul(t ^ (t >>> 7), 61 | t)) ^ t;
    return ((t ^ (t >>> 14)) >>> 0) / 4294967296;
  };
};

interface Seed {
  x0: number;
  dur: number;
  t0: number;
  size: number;
  swayAmp: number;
  swayPeriod: number;
  swayPhase: number;
  spinDir: number;
  spinDur: number;
  twinkleDur: number;
  colorIdx: number;
}

const makeSeeds = (p: KindParams): Seed[] => {
  const seeds: Seed[] = [];
  for (let i = 0; i < p.count; i++) {
    const rnd = mulberry32(i * 7919 + 13);
    const dur = p.durMin + rnd() * (p.durMax - p.durMin);
    seeds.push({
      x0: rnd(),
      dur,
      t0: -rnd() * dur,
      size: p.sizeMin + rnd() * (p.sizeMax - p.sizeMin),
      swayAmp: p.swayAmp * (0.4 + rnd() * 0.6),
      swayPeriod: 3 + rnd() * 3.5,
      swayPhase: rnd() * 2 * Math.PI,
      spinDir: rnd() < 0.5 ? -1 : 1,
      spinDur: 7 + rnd() * 8,
      twinkleDur: 1 + rnd() * 1.2,
      colorIdx: i,
    });
  }
  return seeds;
};

const fillFor = (
  type: CharacterParticleKind,
  i: number,
  b1: string,
  b2: string,
  b3: string,
): string => {
  switch (type) {
    case "spores":
      return i % 4 === 0 ? b2 : b1;
    case "blocks":
    case "bubbles":
      return i % 2 === 0 ? b1 : b2;
    case "echo":
      return i % 9 === 0 ? b2 : i % 2 === 0 ? b1 : b3;
    default:
      return i % 3 === 0 ? b2 : i % 3 === 1 ? b1 : b3;
  }
};

export const CharacterParticles: React.FC<{
  particle: CharacterParticleKind;
  palette: CharacterPalette;
  light: boolean;
  disabled?: boolean;
}> = ({ particle, palette, light, disabled }) => {
  const canvasRef = useRef<HTMLCanvasElement | null>(null);

  useEffect(() => {
    if (disabled) return;
    const canvas = canvasRef.current;
    if (!canvas) return;
    const ctx = canvas.getContext("2d");
    if (!ctx) return;

    const p = paramsFor(particle);
    const seeds = makeSeeds(p);
    const b1 = palette.accent;
    const b2 = palette.aura2;
    const b3 = light ? palette.accent : palette.text;
    const peak = light ? p.peak * 0.8 : p.peak;

    let raf = 0;
    let start = 0;
    const resize = () => {
      const dpr = Math.min(window.devicePixelRatio || 1, 2);
      canvas.width = Math.floor(window.innerWidth * dpr);
      canvas.height = Math.floor(window.innerHeight * dpr);
      ctx.setTransform(dpr, 0, 0, dpr, 0, 0);
    };
    resize();
    window.addEventListener("resize", resize);

    const draw = (now: number) => {
      if (!start) start = now;
      const t = (now - start) / 1000;
      const w = window.innerWidth;
      const h = window.innerHeight;
      ctx.clearRect(0, 0, w, h);
      for (const s of seeds) {
        let cyc = (((t + s.t0) % s.dur) / s.dur + 1) % 1;
        const span = h + 70;
        const y = p.rising ? h + 30 - cyc * span : -40 + cyc * span;
        const x =
          s.x0 * w +
          s.swayAmp *
            Math.sin((2 * Math.PI * (t + s.t0)) / s.swayPeriod + s.swayPhase);
        const alpha =
          peak *
          (cyc < 0.12
            ? cyc / 0.12
            : cyc > 0.88
              ? (1 - cyc) / 0.12
              : 1);
        if (alpha <= 0.01) continue;
        const color = fillFor(particle, s.colorIdx, b1, b2, b3);
        ctx.save();
        ctx.globalAlpha = alpha;
        ctx.translate(x, y);
        if (p.rotates) {
          ctx.rotate(
            (s.spinDir * 2 * Math.PI * ((t + s.t0) / s.spinDur)) % (2 * Math.PI),
          );
        }
        const size = s.size;
        switch (p.shape) {
          case "dot":
            ctx.fillStyle = color;
            ctx.globalAlpha = alpha * 0.25;
            ctx.beginPath();
            ctx.arc(0, 0, size * 1.8, 0, 2 * Math.PI);
            ctx.fill();
            ctx.globalAlpha = alpha;
            ctx.beginPath();
            ctx.arc(0, 0, size * 0.6, 0, 2 * Math.PI);
            ctx.fill();
            break;
          case "square":
            ctx.fillStyle = color;
            ctx.fillRect(-size / 2, -size / 2, size, size);
            break;
          case "spark": {
            const tw =
              0.45 +
              0.65 * (0.5 + 0.5 * Math.sin((2 * Math.PI * t) / s.twinkleDur + s.swayPhase));
            ctx.fillStyle = color;
            ctx.scale(tw, tw);
            ctx.beginPath();
            ctx.moveTo(0, -size);
            ctx.quadraticCurveTo(size * 0.18, -size * 0.18, size, 0);
            ctx.quadraticCurveTo(size * 0.18, size * 0.18, 0, size);
            ctx.quadraticCurveTo(-size * 0.18, size * 0.18, -size, 0);
            ctx.quadraticCurveTo(-size * 0.18, -size * 0.18, 0, -size);
            ctx.fill();
            break;
          }
          case "bubble":
            ctx.strokeStyle = color;
            ctx.lineWidth = 1.5;
            ctx.beginPath();
            ctx.arc(0, 0, size / 2, 0, 2 * Math.PI);
            ctx.stroke();
            break;
          case "flake":
            ctx.strokeStyle = color;
            ctx.lineWidth = 1.5;
            for (let k = 0; k < 3; k++) {
              const a = (k * Math.PI) / 3;
              ctx.beginPath();
              ctx.moveTo(-Math.cos(a) * size * 0.6, -Math.sin(a) * size * 0.6);
              ctx.lineTo(Math.cos(a) * size * 0.6, Math.sin(a) * size * 0.6);
              ctx.stroke();
            }
            break;
          case "shard":
            ctx.fillStyle = color;
            ctx.beginPath();
            ctx.moveTo(0, -size * 0.7);
            ctx.lineTo(size * 0.45, 0);
            ctx.lineTo(0, size * 0.7);
            ctx.lineTo(-size * 0.45, 0);
            ctx.closePath();
            ctx.fill();
            break;
          case "hex": {
            ctx.strokeStyle = color;
            ctx.lineWidth = 1.5;
            ctx.beginPath();
            for (let k = 0; k < 6; k++) {
              const a = (k * Math.PI) / 3 - Math.PI / 2;
              const px = Math.cos(a) * size * 0.6;
              const py = Math.sin(a) * size * 0.6;
              if (k === 0) ctx.moveTo(px, py);
              else ctx.lineTo(px, py);
            }
            ctx.closePath();
            ctx.stroke();
            break;
          }
        }
        ctx.restore();
      }
      raf = requestAnimationFrame(draw);
    };
    raf = requestAnimationFrame(draw);
    return () => {
      cancelAnimationFrame(raf);
      window.removeEventListener("resize", resize);
    };
  }, [particle, palette, light, disabled]);

  if (disabled) return null;
  return (
    <canvas
      ref={canvasRef}
      className="fixed inset-0 z-0 pointer-events-none"
      aria-hidden="true"
    />
  );
};
