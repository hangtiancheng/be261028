/** Shared helpers: trade-number generation, money/time formatting, countdown parsing. */

/** Generates an n-digit random numeric string (crypto-seeded). */
export function randomNumeric(n = 12): string {
  let s = '';
  const buf = new Uint32Array(n);
  crypto.getRandomValues(buf);
  for (let i = 0; i < n; i++) s += String(buf[i] % 10);
  return s;
}

/** Unique external trade number per attempt: OTN-<timestamp>-<rand>. */
export function newOutTradeNo(): string {
  return `OTN-${Date.now()}-${randomNumeric(6)}`;
}

/** Money formatting: always two decimal places. */
export function fmtPrice(v: number | string | null | undefined): string {
  const n = typeof v === 'string' ? Number(v) : (v ?? 0);
  return Number.isFinite(n) ? n.toFixed(2) : '0.00';
}

/** "2026-10-28T19:02:00" / ISO → Date (values without a timezone suffix parse as local time). */
export function parseTime(v: string | null | undefined): Date | null {
  if (!v) return null;
  const d = new Date(v);
  return Number.isNaN(d.getTime()) ? null : d;
}

export interface Countdown {
  hours: number;
  minutes: number;
  seconds: number;
  total: number;
  ended: boolean;
}

/** Builds a countdown from a remaining-seconds value. */
export function fromSeconds(total: number): Countdown {
  const t = Math.max(0, Math.floor(total));
  return {
    hours: Math.floor(t / 3600),
    minutes: Math.floor((t % 3600) / 60),
    seconds: t % 60,
    total: t,
    ended: t <= 0,
  };
}

/**
 * Parses the backend countdown string ("HH:MM:SS", optionally prefixed with
 * "Nd " for windows of a day or more). Returns null for "Ended"/"Invalid time".
 */
export function parseCountdown(hhmmss: string | null | undefined): Countdown | null {
  if (!hhmmss) return null;
  const m = /^(?:(\d+)d\s+)?(\d+):(\d{2}):(\d{2})$/.exec(hhmmss.trim());
  if (!m) return null;
  const days = m[1] ? Number(m[1]) : 0;
  return fromSeconds(days * 86400 + Number(m[2]) * 3600 + Number(m[3]) * 60 + Number(m[4]));
}

export function fmtCountdown(c: Countdown): string {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${p(c.hours)}:${p(c.minutes)}:${p(c.seconds)}`;
}

/** Date → "YYYY-MM-DDTHH:mm:ss" (local time; the format the backend parses). */
export function toLocalInput(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
}

/** datetime-local value ("YYYY-MM-DDTHH:mm") → backend format (seconds appended). */
export function datetimeLocalToBackend(v: string): string {
  if (!v) return v;
  return v.length === 16 ? `${v}:00` : v;
}

/** Backend time string → datetime-local input value. */
export function backendToDatetimeLocal(v: string | null | undefined): string {
  const d = parseTime(v);
  if (!d) return '';
  return toLocalInput(d).slice(0, 16);
}

/** Display date-time (YYYY-MM-DD HH:mm). */
export function fmtDateTime(v: string | null | undefined): string {
  const d = parseTime(v);
  if (!d) return '—';
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}
