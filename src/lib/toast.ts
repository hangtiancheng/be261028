/** Global toasts: module-level store rendered by <x-toast-host>; motion handles the enter/leave animations. */

export type ToastKind = "success" | "error" | "info";

export interface ToastItem {
  id: number;
  kind: ToastKind;
  message: string;
  leaving?: boolean;
}

type ToastListener = (items: ToastItem[]) => void;

let items: ToastItem[] = [];
let seq = 0;
const listeners = new Set<ToastListener>();

function emit(): void {
  listeners.forEach((fn) => fn([...items]));
}

export function subscribeToasts(fn: ToastListener): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

export function markLeaving(id: number): void {
  items = items.map((t) => (t.id === id ? { ...t, leaving: true } : t));
  emit();
}

export function removeToast(id: number): void {
  items = items.filter((t) => t.id !== id);
  emit();
}

function push(kind: ToastKind, message: string, duration = 2600): number {
  const id = ++seq;
  items = [...items, { id, kind, message }].slice(-4);
  emit();
  window.setTimeout(() => {
    markLeaving(id);
    window.setTimeout(() => removeToast(id), 300);
  }, duration);
  return id;
}

export const toast = {
  success: (msg: string) => push("success", msg),
  error: (msg: string) => push("error", msg, 3200),
  info: (msg: string) => push("info", msg),
};
