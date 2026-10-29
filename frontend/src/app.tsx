import { customElement, state } from '@yukino.js/lit-jsx';
import { UiElement } from './components/base';
import { ToastHost } from './components/toast-host';
import { AdminPage } from './pages/admin';
import { HomePage } from './pages/home';
import { LoginPage } from './pages/login';
import { currentPath, onRouteChange, startRouter } from './lib/router';

/**
 * Application root: lightweight hash-router dispatch (login / home / admin)
 * plus the global toast host.
 * Note: @yukino.js/lit-jsx@0.0.1 does not export the community router, so the
 * hash-mode semantics described in the skill docs are implemented directly in
 * lib/router.ts as a minimal router.
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
