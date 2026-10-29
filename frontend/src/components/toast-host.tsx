import { customElement, state } from '@yukino.js/lit-jsx';
import { unsafeHTML } from 'lit/directives/unsafe-html.js';
import { animate } from 'motion';
import { subscribeToasts, type ToastItem } from '../lib/toast';
import { UiElement } from './base';

const ICONS: Record<string, string> = {
  success:
    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" stroke-linejoin="round" class="h-4 w-4"><path d="M20 6 9 17l-5-5"/></svg>',
  error:
    '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" class="h-4 w-4"><path d="M18 6 6 18M6 6l12 12"/></svg>',
  info: '<svg viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2.4" stroke-linecap="round" class="h-4 w-4"><path d="M12 16v-5M12 8h.01"/></svg>',
};

const KIND_STYLE: Record<string, string> = {
  success: 'border-emerald-200/70 bg-emerald-50/95 text-emerald-700',
  error: 'border-rose-200/70 bg-rose-50/95 text-rose-600',
  info: 'border-zinc-200/70 bg-white/95 text-zinc-600',
};

/** 全局 toast 宿主：订阅 lib/toast 的 store，motion 负责进出场。 */
@customElement('x-toast-host')
export class ToastHost extends UiElement {
  @state() private items: ToastItem[] = [];
  private unsub?: () => void;
  private entered = new Set<number>();

  override connectedCallback(): void {
    super.connectedCallback();
    this.unsub = subscribeToasts((items) => {
      this.items = items;
    });
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    this.unsub?.();
  }

  protected override updated(): void {
    const live = new Set(this.items.map((t) => t.id));
    this.entered.forEach((id) => {
      if (!live.has(id)) this.entered.delete(id);
    });
    this.querySelectorAll<HTMLElement>('[data-toast-id]').forEach((el) => {
      const id = Number(el.dataset.toastId);
      const item = this.items.find((t) => t.id === id);
      if (!item) return;
      if (!this.entered.has(id)) {
        this.entered.add(id);
        animate(
          el,
          { opacity: [0, 1], transform: ['translateY(-16px) scale(.96)', 'translateY(0) scale(1)'] },
          { duration: 0.32, ease: [0.22, 1, 0.36, 1] },
        );
      } else if (item.leaving) {
        animate(
          el,
          { opacity: [1, 0], transform: ['translateY(0) scale(1)', 'translateY(-12px) scale(.96)'] },
          { duration: 0.26, ease: 'easeIn' },
        );
      }
    });
  }

  render() {
    return (
      <div className="pointer-events-none fixed inset-x-0 top-16 z-[100] flex flex-col items-center gap-2 px-4">
        {this.items.map((t) => (
          <div
            data-toast-id={String(t.id)}
            className={`pointer-events-auto flex max-w-sm items-center gap-2 rounded-full border px-4 py-2 text-sm font-medium shadow-lg shadow-zinc-900/5 backdrop-blur ${KIND_STYLE[t.kind]}`}
          >
            {unsafeHTML(ICONS[t.kind])}
            <span>{t.message}</span>
          </div>
        ))}
      </div>
    );
  }
}
