/** 通用工具：随机单号、金额/时间格式化、倒计时解析。 */

/** 生成 n 位随机数字串（默认 12 位外部交易单号）。 */
export function randomNumeric(n = 12): string {
  let s = '';
  const buf = new Uint32Array(n);
  crypto.getRandomValues(buf);
  for (let i = 0; i < n; i++) s += String(buf[i] % 10);
  return s;
}

/** 金额格式化：始终两位小数。 */
export function fmtPrice(v: number | string | null | undefined): string {
  const n = typeof v === 'string' ? Number(v) : (v ?? 0);
  return Number.isFinite(n) ? n.toFixed(2) : '0.00';
}

/** "2026-10-28T19:02:00" / ISO → Date（无时区后缀按本地时区解析）。 */
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

/** 由剩余秒数构造倒计时。 */
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

/** 解析后端 "HH:MM:SS" 倒计时字符串。 */
export function parseCountdown(hhmmss: string | null | undefined): Countdown | null {
  if (!hhmmss) return null;
  const m = /^(\d+):(\d{2}):(\d{2})$/.exec(hhmmss.trim());
  if (!m) return null;
  return fromSeconds(Number(m[1]) * 3600 + Number(m[2]) * 60 + Number(m[3]));
}

export function fmtCountdown(c: Countdown): string {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${p(c.hours)}:${p(c.minutes)}:${p(c.seconds)}`;
}

/** Date → "YYYY-MM-DDTHH:mm:ss"（本地时区，后端可解析）。 */
export function toLocalInput(d: Date): string {
  const p = (n: number) => String(n).padStart(2, '0');
  return `${d.getFullYear()}-${p(d.getMonth() + 1)}-${p(d.getDate())}T${p(d.getHours())}:${p(d.getMinutes())}:${p(d.getSeconds())}`;
}

/** datetime-local 值("YYYY-MM-DDTHH:mm") → 后端格式(补秒)。 */
export function datetimeLocalToBackend(v: string): string {
  if (!v) return v;
  return v.length === 16 ? `${v}:00` : v;
}

/** 后端时间 → datetime-local 输入值。 */
export function backendToDatetimeLocal(v: string | null | undefined): string {
  const d = parseTime(v);
  if (!d) return '';
  return toLocalInput(d).slice(0, 16);
}

/** 展示用日期（YYYY-MM-DD HH:mm）。 */
export function fmtDateTime(v: string | null | undefined): string {
  const d = parseTime(v);
  if (!d) return '—';
  return `${d.getFullYear()}-${String(d.getMonth() + 1).padStart(2, '0')}-${String(d.getDate()).padStart(2, '0')} ${String(d.getHours()).padStart(2, '0')}:${String(d.getMinutes()).padStart(2, '0')}`;
}
