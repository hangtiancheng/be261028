import { customElement, property, state } from "@yukino.js/lit-jsx";
import type { PropertyValues } from "@yukino.js/lit-jsx";
import { animate } from "motion";
import {
  fromSeconds,
  fmtCountdown,
  parseCountdown,
  parseTime,
  type Countdown,
} from "../lib/utils";
import { UiElement } from "./base";

/**
 * Live countdown: prefers the absolute endTime when available; otherwise
 * decrements from the initial "HH:MM:SS" string as wall time passes.
 * Turns red within the last 60 seconds with a subtle per-tick motion pulse.
 */
@customElement("x-countdown")
export class CountDown extends UiElement {
  @property({ attribute: false }) endTime: string | null = null;
  @property({ attribute: false }) initial: string | null = null;

  @state() private cd: Countdown = fromSeconds(0);
  private timer?: number;
  private baseTotal = -1;
  private baseAt = 0;

  override connectedCallback(): void {
    super.connectedCallback();
    this.recalc();
    this.timer = window.setInterval(() => this.recalc(), 1000);
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    if (this.timer !== undefined) window.clearInterval(this.timer);
  }

  protected override updated(changed: PropertyValues<this>): void {
    // Re-anchor the wall-clock baseline whenever a refreshed team list hands
    // us a new endTime / countdown string, so silent polls can correct drift.
    if (changed.has("endTime") || changed.has("initial")) {
      this.baseTotal = -1;
      this.recalc();
    }
  }

  private recalc(): void {
    const prev = this.cd.total;
    const end = parseTime(this.endTime);
    if (end) {
      this.cd = fromSeconds((end.getTime() - Date.now()) / 1000);
    } else {
      if (this.baseTotal < 0) {
        const p = parseCountdown(this.initial);
        this.baseTotal = p ? p.total : 0;
        this.baseAt = Date.now();
      }
      this.cd = fromSeconds(this.baseTotal - (Date.now() - this.baseAt) / 1000);
    }
    if (!this.cd.ended && this.cd.total < 60 && this.cd.total !== prev) {
      animate(this, { scale: [1.06, 1] }, { duration: 0.24, ease: "easeOut" });
    }
  }

  render() {
    if (this.cd.ended) {
      return (
        <span className="rounded-md bg-zinc-100 px-1.5 py-0.5 text-[11px] font-medium text-zinc-400">
          Ended
        </span>
      );
    }
    const urgent = this.cd.total < 60;
    return (
      <span
        className={`rounded-md px-1.5 py-0.5 font-mono text-[11px] font-semibold tabular-nums ${
          urgent
            ? "animate-pulse bg-brand-50 text-brand-600"
            : "bg-zinc-100 text-zinc-500"
        }`}
      >
        {fmtCountdown(this.cd)}
      </span>
    );
  }
}
