/**
 * Minimal hash router.
 * Note: @yukino.js/lit-jsx@0.0.1 on npm does not export the community entry
 * (router/icon), so this implements the community router's hash-mode semantics
 * directly: a hash change emits the normalized path to subscribers, and
 * navigate() rewrites location.hash. Hash mode is also required because the Go
 * gateway serves dist/ statically in production with no SPA fallback.
 */

export type RouteChangeListener = (path: string) => void;

const listeners = new Set<RouteChangeListener>();

export function currentPath(): string {
  const raw = location.hash.replace(/^#/, '');
  const path = raw || '/';
  return path.startsWith('/') ? path : `/${path}`;
}

export function navigate(to: string, opts: { replace?: boolean } = {}): void {
  const target = `#${to.startsWith('/') ? to : `/${to}`}`;
  if (location.hash === target) {
    emit();
    return;
  }
  if (opts.replace) {
    history.replaceState(null, '', target);
    emit();
  } else {
    location.hash = target;
  }
}

export function onRouteChange(fn: RouteChangeListener): () => void {
  listeners.add(fn);
  return () => listeners.delete(fn);
}

function emit(): void {
  const path = currentPath();
  listeners.forEach((fn) => fn(path));
}

let started = false;

/** Starts the hashchange listener (idempotent). */
export function startRouter(): void {
  if (started) return;
  started = true;
  window.addEventListener('hashchange', () => emit());
}
