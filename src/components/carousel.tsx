import {
  createRef,
  customElement,
  property,
  state,
  type Ref,
} from "@yukino.js/lit-jsx";
import type { PropertyValues } from "lit";
import { animate } from "motion";
import { UiElement } from "./base";

/** Product artwork carousel: autoplay + indicator dots + motion spring sliding; pauses on hover. */
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
    // Keep the slide index (and the track offset) valid when the images swap,
    // e.g. after the home page switches goods in place.
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
              {/* Generated on canvas (data URI) — never an image file. */}
              <img
                src={src}
                alt="Generated product artwork"
                loading="lazy"
                className="aspect-square w-full object-cover"
              />
            </div>
          ))}
        </div>
        <div className="absolute bottom-3 left-1/2 flex -translate-x-1/2 gap-1.5">
          {this.images.map((_, i) => (
            <button
              onClick={() => this.go(i)}
              aria-label={`Go to slide ${i + 1}`}
              className={`h-1.5 rounded-full transition-all duration-300 ${
                i === this.index ? "w-5 bg-white" : "w-1.5 bg-white/60"
              }`}
            />
          ))}
        </div>
      </div>
    );
  }
}
