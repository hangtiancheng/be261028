/**
 * Programmatically generated visuals — image files are forbidden app-wide
 * (a hard user requirement).
 *
 * The algorithm matches yukino-chat's utils/avatar.ts:
 *   fnv1a string hash → xorShift32 PRNG → deterministic pattern drawn on an
 *   offscreen canvas → canvas.toDataURL() → used as an <img src> / CSS background.
 * The same seed always yields the same artwork; results are memoized per key
 * to avoid redundant drawing.
 */

function fnv1a(str: string): number {
  let hash = 0x811c9dc5;
  for (let i = 0; i < str.length; i++) {
    hash ^= str.charCodeAt(i);
    hash = Math.imul(hash, 0x01000193);
  }
  return hash >>> 0;
}

function xorShift32(seed: number): () => number {
  let state = seed || 1;
  return () => {
    state ^= state << 13;
    state ^= state >>> 17;
    state ^= state << 5;
    state >>>= 0;
    return state / 0xffffffff;
  };
}

/** Single-argument memoization (replaces es-toolkit memoize to avoid a new dependency). */
function memoize(fn: (key: string) => string): (key: string) => string {
  const cache = new Map<string, string>();
  return (key: string): string => {
    const hit = cache.get(key);
    if (hit !== undefined) return hit;
    const value = fn(key);
    cache.set(key, value);
    return value;
  };
}

function makeCanvas(
  size: number,
): [HTMLCanvasElement, CanvasRenderingContext2D] {
  const canvas = document.createElement("canvas");
  canvas.width = size;
  canvas.height = size;
  return [canvas, canvas.getContext("2d")!];
}

type Shape = "round" | "circle" | "ring" | "triangle" | "diamond";

const SHAPES: Shape[] = ["round", "circle", "ring", "triangle", "diamond"];

function drawShape(
  ctx: CanvasRenderingContext2D,
  shape: Shape,
  x: number,
  y: number,
  s: number,
): void {
  ctx.beginPath();
  switch (shape) {
    case "round":
      ctx.roundRect(x, y, s, s, s * 0.32);
      ctx.fill();
      break;
    case "circle":
      ctx.arc(x + s / 2, y + s / 2, s / 2, 0, Math.PI * 2);
      ctx.fill();
      break;
    case "ring":
      ctx.arc(x + s / 2, y + s / 2, s / 2, 0, Math.PI * 2);
      ctx.lineWidth = s * 0.18;
      ctx.stroke();
      break;
    case "triangle":
      ctx.moveTo(x + s / 2, y);
      ctx.lineTo(x + s, y + s);
      ctx.lineTo(x, y + s);
      ctx.closePath();
      ctx.fill();
      break;
    case "diamond":
      ctx.moveTo(x + s / 2, y);
      ctx.lineTo(x + s, y + s / 2);
      ctx.lineTo(x + s / 2, y + s);
      ctx.lineTo(x, y + s / 2);
      ctx.closePath();
      ctx.fill();
      break;
  }
}

/**
 * Product poster art (carousel / hero slots): diagonal gradient background +
 * soft light blobs + mirrored rounded geometric modules. The seed (goodsId,
 * activity id, …) determines hue and layout; the same seed always renders the
 * same artwork.
 */
export const genProductArt = memoize((seed: string): string => {
  const SIZE = 560;
  const [canvas, ctx] = makeCanvas(SIZE);
  const rand = xorShift32(fnv1a(`art:${seed}`));

  const hue = Math.floor(rand() * 360);
  const hue2 = (hue + 55 + Math.floor(rand() * 90)) % 360;

  // Background: diagonal two-color gradient.
  const bg = ctx.createLinearGradient(0, 0, SIZE, SIZE);
  bg.addColorStop(0, `hsl(${hue}, 88%, 66%)`);
  bg.addColorStop(1, `hsl(${hue2}, 80%, 42%)`);
  ctx.fillStyle = bg;
  ctx.fillRect(0, 0, SIZE, SIZE);

  // Soft light blobs: three large radial gradients.
  for (let i = 0; i < 3; i++) {
    const cx = rand() * SIZE;
    const cy = rand() * SIZE;
    const r = SIZE * (0.2 + rand() * 0.32);
    const glow = ctx.createRadialGradient(cx, cy, 0, cx, cy, r);
    glow.addColorStop(
      0,
      `hsla(${(hue2 + i * 36) % 360}, 96%, 78%, ${(0.14 + rand() * 0.14).toFixed(3)})`,
    );
    glow.addColorStop(1, "hsla(0, 0%, 100%, 0)");
    ctx.fillStyle = glow;
    ctx.fillRect(0, 0, SIZE, SIZE);
  }

  // Mirrored geometric modules (identicon style, left-right symmetric).
  const GRID = 6;
  const CELL = SIZE / GRID;
  for (let col = 0; col < Math.ceil(GRID / 2); col++) {
    for (let row = 0; row < GRID; row++) {
      if (rand() < 0.45) continue;
      const shape = SHAPES[Math.floor(rand() * SHAPES.length)];
      const pad = CELL * (0.12 + rand() * 0.14);
      const alpha = 0.1 + rand() * 0.24;
      const tone =
        rand() < 0.22
          ? `hsla(${hue}, 70%, 22%, ${(alpha + 0.08).toFixed(3)})`
          : `hsla(0, 0%, 100%, ${alpha.toFixed(3)})`;
      ctx.fillStyle = tone;
      ctx.strokeStyle = tone;
      const s = CELL - pad * 2;
      drawShape(ctx, shape, pad + col * CELL, pad + row * CELL, s);
      const mirrorCol = GRID - 1 - col;
      if (mirrorCol !== col) {
        drawShape(ctx, shape, pad + mirrorCol * CELL, pad + row * CELL, s);
      }
    }
  }

  // Center badge: concentric ring + solid dot to anchor the visual focus.
  const c = SIZE / 2;
  ctx.strokeStyle = "hsla(0, 0%, 100%, 0.5)";
  ctx.lineWidth = SIZE * 0.014;
  ctx.beginPath();
  ctx.arc(c, c, SIZE * 0.2, 0, Math.PI * 2);
  ctx.stroke();
  ctx.fillStyle = "hsla(0, 0%, 100%, 0.16)";
  ctx.beginPath();
  ctx.arc(c, c, SIZE * 0.15, 0, Math.PI * 2);
  ctx.fill();
  ctx.fillStyle = "hsla(0, 0%, 100%, 0.85)";
  ctx.beginPath();
  ctx.arc(c, c, SIZE * 0.035, 0, Math.PI * 2);
  ctx.fill();

  // Top sheen fade for a poster-like finish.
  const sheen = ctx.createLinearGradient(0, 0, 0, SIZE * 0.5);
  sheen.addColorStop(0, "hsla(0, 0%, 100%, 0.14)");
  sheen.addColorStop(1, "hsla(0, 0%, 100%, 0)");
  ctx.fillStyle = sheen;
  ctx.fillRect(0, 0, SIZE, SIZE * 0.5);

  return canvas.toDataURL();
});

/**
 * Deterministic user avatar (identicon style, circular clip): the seed (e.g.
 * userId) picks the hue pair and the mirrored 5×5 module pattern.
 */
export const genAvatarArt = memoize((seed: string): string => {
  const SIZE = 128;
  const [canvas, ctx] = makeCanvas(SIZE);
  const rand = xorShift32(fnv1a(`avatar:${seed}`));

  const hue = Math.floor(rand() * 360);
  const hue2 = (hue + 40 + Math.floor(rand() * 80)) % 360;

  ctx.save();
  ctx.beginPath();
  ctx.arc(SIZE / 2, SIZE / 2, SIZE / 2, 0, Math.PI * 2);
  ctx.clip();

  const bg = ctx.createLinearGradient(0, 0, SIZE, SIZE);
  bg.addColorStop(0, `hsl(${hue}, 78%, 62%)`);
  bg.addColorStop(1, `hsl(${hue2}, 72%, 44%)`);
  ctx.fillStyle = bg;
  ctx.fillRect(0, 0, SIZE, SIZE);

  // Mirrored identicon modules.
  const GRID = 5;
  const CELL = SIZE / GRID;
  for (let col = 0; col < Math.ceil(GRID / 2); col++) {
    for (let row = 0; row < GRID; row++) {
      if (rand() >= 0.55) continue;
      ctx.fillStyle = `hsla(0, 0%, 100%, ${(0.16 + rand() * 0.3).toFixed(3)})`;
      const pad = CELL * 0.12;
      ctx.fillRect(
        col * CELL + pad,
        row * CELL + pad,
        CELL - pad * 2,
        CELL - pad * 2,
      );
      const mirrorCol = GRID - 1 - col;
      if (mirrorCol !== col) {
        ctx.fillRect(
          mirrorCol * CELL + pad,
          row * CELL + pad,
          CELL - pad * 2,
          CELL - pad * 2,
        );
      }
    }
  }

  // Subtle top-left highlight for depth.
  const sheen = ctx.createRadialGradient(
    SIZE * 0.3,
    SIZE * 0.25,
    0,
    SIZE * 0.3,
    SIZE * 0.25,
    SIZE * 0.75,
  );
  sheen.addColorStop(0, "hsla(0, 0%, 100%, 0.22)");
  sheen.addColorStop(1, "hsla(0, 0%, 100%, 0)");
  ctx.fillStyle = sheen;
  ctx.fillRect(0, 0, SIZE, SIZE);

  ctx.restore();
  return canvas.toDataURL();
});

/**
 * Simulated payment QR code: the seed (e.g. outTradeNo) drives the module
 * distribution; includes the standard QR visual elements (three finder
 * patterns + timing lines + pseudo-random data modules). Not a real QR code.
 */
export const genQrArt = memoize((seed: string): string => {
  const MODULES = 25;
  const QUIET = 2;
  const CELL = 8;
  const SIZE = (MODULES + QUIET * 2) * CELL;
  const [canvas, ctx] = makeCanvas(SIZE);
  const rand = xorShift32(fnv1a(`qr:${seed}`));

  const DARK = "#18181b";
  ctx.fillStyle = "#ffffff";
  ctx.fillRect(0, 0, SIZE, SIZE);
  ctx.fillStyle = DARK;

  const at = (row: number, col: number): void => {
    ctx.fillRect((QUIET + col) * CELL, (QUIET + row) * CELL, CELL, CELL);
  };
  const inFinder = (row: number, col: number): boolean =>
    (row < 8 && col < 8) ||
    (row < 8 && col >= MODULES - 8) ||
    (row >= MODULES - 8 && col < 8);

  // Data modules.
  for (let row = 0; row < MODULES; row++) {
    for (let col = 0; col < MODULES; col++) {
      if (!inFinder(row, col) && rand() > 0.5) at(row, col);
    }
  }
  // Timing lines (alternating modules on row/column 6).
  for (let i = 8; i < MODULES - 8; i++) {
    if (i % 2 === 0) {
      at(6, i);
      at(i, 6);
    }
  }
  // The three finder patterns.
  const finder = (topRow: number, leftCol: number): void => {
    ctx.fillStyle = DARK;
    ctx.fillRect(
      (QUIET + leftCol) * CELL,
      (QUIET + topRow) * CELL,
      CELL * 7,
      CELL * 7,
    );
    ctx.fillStyle = "#ffffff";
    ctx.fillRect(
      (QUIET + leftCol + 1) * CELL,
      (QUIET + topRow + 1) * CELL,
      CELL * 5,
      CELL * 5,
    );
    ctx.fillStyle = DARK;
    ctx.fillRect(
      (QUIET + leftCol + 2) * CELL,
      (QUIET + topRow + 2) * CELL,
      CELL * 3,
      CELL * 3,
    );
  };
  finder(0, 0);
  finder(0, MODULES - 7);
  finder(MODULES - 7, 0);

  return canvas.toDataURL();
});
