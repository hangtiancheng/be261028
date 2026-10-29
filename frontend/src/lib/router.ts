/**
 * 轻量 hash 路由。
 * 说明：npm 上的 @yukino.js/lit-jsx@0.0.1 未导出 community 入口（router/icon），
 * 因此按技能文档中 community router 的 hash 模式语义实现一个最小路由：
 * hash 变化 → 订阅者收到规范化路径；navigate() 修改 location.hash。
 * 选用 hash 模式还因为生产环境由 Go 网关静态服务 dist，无 SPA fallback。
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

/** 启动 hashchange 监听（幂等）。 */
export function startRouter(): void {
  if (started) return;
  started = true;
  window.addEventListener('hashchange', () => emit());
}
