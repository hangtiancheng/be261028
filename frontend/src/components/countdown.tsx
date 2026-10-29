import { customElement, property, state } from '@yukino.js/lit-jsx';
import { animate } from 'motion';
import { fromSeconds, fmtCountdown, parseCountdown, parseTime, type Countdown } from '../lib/utils';
import { UiElement } from './base';

/**
 * 实时倒计时：优先按 endTime（绝对时间）计算；
 * 无 endTime 时以 initial("HH:MM:SS") 为基准随时间递减。
 * 剩余 60s 内变红并以 motion 做每秒轻微跳动。
 */
@customElement('x-countdown')
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
      animate(this, { scale: [1.06, 1] }, { duration: 0.24, ease: 'easeOut' });
    }
  }

  render() {
    if (this.cd.ended) {
      return <span className="text-[11px] font-medium text-zinc-300">已结束</span>;
    }
    const urgent = this.cd.total < 60;
    return (
      <span
        className={`font-mono text-[11px] font-semibold tabular-nums ${
          urgent ? 'animate-pulse text-brand-600' : 'text-zinc-400'
        }`}
      >
        {fmtCountdown(this.cd)}
      </span>
    );
  }
}
