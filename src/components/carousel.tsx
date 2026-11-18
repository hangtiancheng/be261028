import {
  createRef,
  customElement,
  property,
  state,
  type Ref,
} from "@yukino.js/lit-jsx";
import type { PropertyValues } from "@yukino.js/lit-jsx";
import { unsafeHTML } from "lit/directives/unsafe-html.js";
import { animate } from "motion";
import { ICONS } from "../lib/icons";
import { icon } from "@yukino.js/lit-jsx/community";
import { UiElement } from "./base";

/** Product artwork carousel: autoplay + arrows + indicator dots; pauses on hover. */
@customElement("x-carousel")
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

  protected override updated(changed: PropertyValues<this>): void {
    if (changed.has("images")) {
      if (this.index >= this.images.length) this.index = 0;
      const el = this.trackRef.value;
      if (el) {
        animate(
          el,
          { x: `${-this.index * 100}%` },
          { type: "spring", stiffness: 220, damping: 28 },
        );
      }
    }
  }

  private go(i: number): void {
    this.index = i;
    const el = this.trackRef.value;
    if (el) {
      animate(
        el,
        { x: `${-i * 100}%` },
        { type: "spring", stiffness: 220, damping: 28 },
      );
    }
  }

  render() {
    const arrow =
      "absolute top-1/2 hidden h-9 w-9 -translate-y-1/2 place-items-center rounded-full bg-white/90 text-zinc-700 shadow-lg shadow-zinc-900/10 opacity-0 backdrop-blur transition hover:bg-white group-hover:opacity-100 pressable lg:grid";
    return (
      <div
        className="group relative overflow-hidden rounded-card bg-zinc-100 shadow-raised ring-1 ring-zinc-900/5"
        onMouseenter={() => {
          this.paused = true;
        }}
        onMouseleave={() => {
          this.paused = false;
        }}
      >
        <div ref={this.trackRef} className="flex will-change-transform">
          {this.images.map((src, i) => (
            <div key={src} className="w-full shrink-0">
              <img
                src={src}
                alt=""
                loading={i === 0 ? "eager" : "lazy"}
                className="aspect-square w-full object-cover"
              />
            </div>
          ))}
        </div>

        <div className="pointer-events-none absolute inset-x-0 bottom-0 h-24 bg-gradient-to-t from-zinc-950/30 to-transparent" />

        <button
          type="button"
          onClick={() =>
            this.go((this.index - 1 + this.images.length) % this.images.length)
          }
          aria-label="Previous slide"
          className={`${arrow} left-3`}
        >
          {unsafeHTML(icon(ICONS.arrowLeft, "h-4 w-4"))}
        </button>
        <button
          type="button"
          onClick={() => this.go((this.index + 1) % this.images.length)}
          aria-label="Next slide"
          className={`${arrow} right-3`}
        >
          {unsafeHTML(icon(ICONS.arrowLeft, "h-4 w-4 rotate-180"))}
        </button>

        <div className="absolute bottom-3.5 left-1/2 flex -translate-x-1/2 gap-1.5">
          {this.images.map((src, i) => (
            <button
              key={src}
              type="button"
              onClick={() => this.go(i)}
              aria-label={`Go to slide ${i + 1}`}
              className={`h-1.5 rounded-full shadow-sm transition-all duration-300 ${
                i === this.index
                  ? "w-6 bg-white"
                  : "w-1.5 bg-white/60 hover:bg-white"
              }`}
            />
          ))}
        </div>
      </div>
    );
  }
}
