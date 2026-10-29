import { customElement, state } from '@yukino.js/lit-jsx';
import { UiElement } from './components/base';
import { ToastHost } from './components/toast-host';
import { AdminPage } from './pages/admin';
import { HomePage } from './pages/home';
import { LoginPage } from './pages/login';
import { currentPath, onRouteChange, startRouter } from './lib/router';

/**
 * 应用根组件：轻量 hash 路由分发（登录 / 首页 / 管理后台）+ 全局 toast 宿主。
 * 说明：@yukino.js/lit-jsx@0.0.1 未导出 community router，按技能文档的 hash
 * 模式语义用 lib/router.ts 实现最小路由。
 */
@customElement('x-app')
export class App extends UiElement {
  @state() private path = currentPath();
  private unsub?: () => void;

  override connectedCallback(): void {
    super.connectedCallback();
    startRouter();
    this.unsub = onRouteChange((p) => {
      this.path = p;
    });
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    this.unsub?.();
  }

  render() {
    let page: unknown;
    if (this.path.startsWith('/login')) {
      page = <LoginPage />;
    } else if (this.path.startsWith('/admin')) {
      page = <AdminPage />;
    } else {
      page = <HomePage />;
    }
    return (
      <div className="font-sans text-zinc-900 antialiased">
        {page}
        <ToastHost />
      </div>
    );
  }
}
