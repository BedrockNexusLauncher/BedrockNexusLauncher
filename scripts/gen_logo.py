"""Generate the Bedrock Nexus logo (SVG + PNG app icon).

The mark is an upright voxel "N" — bedrock-block style squares with a
3D extrusion — in an emerald-to-cyan gradient on a dark rounded square.
"""

import os

ROOT = os.path.dirname(os.path.dirname(os.path.abspath(__file__)))

# ---- geometry -------------------------------------------------------------
PITCH = 56.0   # grid pitch
SIZE = 52.0    # front face of one block
EXT = 14.0     # extrusion depth (up-right)
X0 = 139.0
Y0 = 153.0

# voxel "N" pattern on a 4x4 grid (col, row)
CELLS = [(0, 0), (0, 1), (0, 2), (0, 3), (1, 1), (2, 2),
         (3, 0), (3, 1), (3, 2), (3, 3)]

EMERALD = (52, 211, 153)   # #34D399
CYAN = (34, 211, 238)      # #22D3EE


def lerp(a, b, t):
    return tuple(round(a[i] + (b[i] - a[i]) * t) for i in range(3))


def shade(rgb, t):
    if t >= 0:
        return lerp(rgb, (255, 255, 255), t)
    return lerp(rgb, (0, 0, 0), -t)


def hexc(rgb):
    return "#{:02X}{:02X}{:02X}".format(*rgb)


def blocks():
    out = []
    for c, r in CELLS:
        x = X0 + c * PITCH
        y = Y0 + r * PITCH
        base = lerp(EMERALD, CYAN, (c + (3 - r)) / 6.0)
        front = [(x, y), (x + SIZE, y), (x + SIZE, y + SIZE), (x, y + SIZE)]
        top = [(x, y), (x + SIZE, y), (x + SIZE + EXT, y - EXT), (x + EXT, y - EXT)]
        right = [(x + SIZE, y), (x + SIZE + EXT, y - EXT),
                 (x + SIZE + EXT, y + SIZE - EXT), (x + SIZE, y + SIZE)]
        out.append({
            "front": front, "top": top, "right": right,
            "front_c": hexc(base), "top_c": hexc(shade(base, 0.25)),
            "right_c": hexc(shade(base, -0.35)),
            "front_rgb": base, "top_rgb": shade(base, 0.25),
            "right_rgb": shade(base, -0.35),
        })
    # painter order: top rows first so lower blocks overlap extrusions
    return out


# ---- SVG ------------------------------------------------------------------
def poly_svg(points, fill):
    pts = " ".join("{:.1f},{:.1f}".format(px, py) for px, py in points)
    return '<polygon points="{}" fill="{}"/>'.format(pts, fill)


def build_svg():
    parts = []
    parts.append('<svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 512 512" width="512" height="512">')
    parts.append('<defs>')
    parts.append('<linearGradient id="bg" x1="0" y1="0" x2="0" y2="1">'
                 '<stop offset="0" stop-color="#16233F"/>'
                 '<stop offset="1" stop-color="#0A1220"/>'
                 '</linearGradient>')
    parts.append('<radialGradient id="glow" cx="0.5" cy="0.48" r="0.5">'
                 '<stop offset="0" stop-color="#34D399" stop-opacity="0.16"/>'
                 '<stop offset="0.6" stop-color="#22D3EE" stop-opacity="0.06"/>'
                 '<stop offset="1" stop-color="#22D3EE" stop-opacity="0"/>'
                 '</radialGradient>')
    parts.append('</defs>')
    parts.append('<rect x="16" y="16" width="480" height="480" rx="112" fill="url(#bg)"/>')
    parts.append('<rect x="16" y="16" width="480" height="480" rx="112" fill="url(#glow)"/>')
    parts.append('<rect x="16" y="16" width="480" height="480" rx="112" fill="none" stroke="#FFFFFF" stroke-opacity="0.08" stroke-width="2"/>')
    for b in blocks():
        parts.append(poly_svg(b["right"], b["right_c"]))
        parts.append(poly_svg(b["top"], b["top_c"]))
        parts.append(poly_svg(b["front"], b["front_c"]))
    # nexus nodes: glowing dots on the three peaks of the N
    for c, r in [(0, 0), (3, 0), (3, 3)]:
        x = X0 + c * PITCH
        y = Y0 + r * PITCH
        cx = x + SIZE / 2 + EXT / 2
        cy = y - EXT / 2
        parts.append('<circle cx="{:.1f}" cy="{:.1f}" r="5" fill="#FFFFFF" fill-opacity="0.95"/>'.format(cx, cy))
    parts.append('</svg>')
    return "\n".join(parts)


# ---- PNG (PIL) ------------------------------------------------------------
def build_png(path, size=1024):
    from PIL import Image, ImageDraw, ImageFilter

    S = 4  # supersample
    W = size * S
    img = Image.new("RGBA", (W, W), (0, 0, 0, 0))
    draw = ImageDraw.Draw(img)
    sc = W / 512.0

    def P(pts):
        return [(px * sc, py * sc) for px, py in pts]

    m = 16 * sc
    # base fill + vertical gradient
    draw.rounded_rectangle([m, m, W - m, W - m], radius=112 * sc,
                           fill=(22, 35, 63, 255))
    grad = Image.new("RGBA", (W, W), (10, 18, 32, 255))
    grad_mask = Image.new("L", (W, W), 0)
    gd = ImageDraw.Draw(grad_mask)
    gd.rounded_rectangle([m, m, W - m, W - m], radius=112 * sc, fill=255)
    fade = Image.new("L", (W, W), 0)
    fd = ImageDraw.Draw(fade)
    for yy in range(int(m), int(W - m), 2):
        t = (yy - m) / (W - 2 * m)
        fd.line([(0, yy), (W, yy)], fill=int(210 * t), width=2)
    img.paste(grad, (0, 0), Image.composite(fade, Image.new("L", (W, W), 0), grad_mask))

    # subtle glow behind the mark
    glow = Image.new("RGBA", (W, W), (0, 0, 0, 0))
    gl = ImageDraw.Draw(glow)
    cx, cy = 256 * sc, 250 * sc
    for rad, alpha in [(190 * sc, 28), (130 * sc, 34), (85 * sc, 42)]:
        gl.ellipse([cx - rad, cy - rad, cx + rad, cy + rad], fill=(52, 211, 153, alpha))
    glow = glow.filter(ImageFilter.GaussianBlur(45 * sc))
    img.paste(glow, (0, 0), Image.composite(glow.split()[3], Image.new("L", (W, W), 0), grad_mask))

    # blocks
    for b in blocks():
        draw.polygon(P(b["right"]), fill=b["right_rgb"] + (255,))
        draw.polygon(P(b["top"]), fill=b["top_rgb"] + (255,))
        draw.polygon(P(b["front"]), fill=b["front_rgb"] + (255,))

    # nexus nodes
    for c, r in [(0, 0), (3, 0), (3, 3)]:
        x = X0 + c * PITCH
        y = Y0 + r * PITCH
        cx = (x + SIZE / 2 + EXT / 2) * sc
        cy = (y - EXT / 2) * sc
        rr = 5 * sc
        draw.ellipse([cx - rr, cy - rr, cx + rr, cy + rr], fill=(255, 255, 255, 242))

    img = img.resize((size, size), Image.LANCZOS)
    img.save(path, "PNG")


if __name__ == "__main__":
    svg_path = os.path.join(ROOT, "assets", "bedrock-nexus-logo.svg")
    os.makedirs(os.path.dirname(svg_path), exist_ok=True)
    with open(svg_path, "w", encoding="utf-8", newline="\n") as f:
        f.write(build_svg() + "\n")
    build_png(os.path.join(ROOT, "build", "appicon.png"), 1024)
    print("wrote", svg_path, "and build/appicon.png")
