import { createRef, customElement, property, state, type Ref } from '@yukino.js/lit-jsx';
import { animate } from 'motion';
import { UiElement } from './base';

/** 商品图轮播：自动播放 + 指示点 + motion 弹簧滑动，悬停暂停。 */
@customElement('x-carousel')
export class Carousel extends UiElement {
  @property({ attribute: false }) images: string[] = [];

  @state() private index = 0;
  private trackRef: Ref<HTMLDivElement> = createRef();
  private timer?: number;
  private paused = false;

  override connectedCallback(): void {
    super.connectedCallback();
    this.timer = window.setInterval(() => {
      if (!this.paused && this.images.length > 1) {
        this.go((this.index + 1) % this.images.length);
      }
    }, 3800);
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    if (this.timer !== undefined) window.clearInterval(this.timer);
  }

  private go(i: number): void {
    this.index = i;
    const el = this.trackRef.value;
    if (el) {
      animate(el, { x: `${-i * 100}%` }, { type: 'spring', stiffness: 220, damping: 28 });
    }
  }

  render() {
    return (
      <div
        className="relative overflow-hidden rounded-3xl bg-zinc-200 shadow-sm"
        onMouseenter={() => {
          this.paused = true;
        }}
        onMouseleave={() => {
          this.paused = false;
        }}
      >
        <div ref={this.trackRef} className="flex will-change-transform">
          {this.images.map((src) => (
            <div className="w-full shrink-0">
              <img src={src} alt="商品图" loading="lazy" className="aspect-square w-full object-cover" />
            </div>
          ))}
        </div>
        <div className="absolute bottom-3 left-1/2 flex -translate-x-1/2 gap-1.5">
          {this.images.map((_, i) => (
            <button
              onClick={() => this.go(i)}
              aria-label={`第${i + 1}张`}
              className={`h-1.5 rounded-full transition-all duration-300 ${
                i === this.index ? 'w-5 bg-white' : 'w-1.5 bg-white/60'
              }`}
            />
          ))}
        </div>
      </div>
    );
  }
}
