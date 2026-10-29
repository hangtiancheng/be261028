import { createRef, customElement, state, type Ref } from '@yukino.js/lit-jsx';
import { unsafeHTML } from 'lit/directives/unsafe-html.js';
import { animate, stagger } from 'motion';
import {
  api,
  deriveDiscountExpr,
  friendlyError,
  MARKET_PLAN_LABELS,
  type MarketConfig,
  type Team,
} from '../lib/api';
import { genAvatarArt, genProductArt, genQrArt } from '../lib/art';
import { clearUsername, getUsername } from '../lib/auth';
import { ICONS, svgIcon } from '../lib/icons';
import { navigate, onRouteChange } from '../lib/router';
import { toast } from '../lib/toast';
import { fmtPrice, newOutTradeNo, parseTime } from '../lib/utils';
import { UiElement } from '../components/base';
import { Carousel } from '../components/carousel';
import { CountDown } from '../components/countdown';

const DEFAULT_GOODS_ID = '9890001';

/** Demo goods with seeded campaigns, offered by the goods switcher pills. */
const DEMO_GOODS = ['9890001', '9890002'];

const PRODUCT_TITLES: Record<string, string> = {
  '9890001': 'Handwritten MyBatis: Progressive Source-Code Practice (Full-Color Edition)',
  '9890002': 'Two-Person Group-Buy Demo Goods — Marketing Showcase Kit',
};

/**
 * Background refresh cadence. The market-config endpoint is rate limited at
 * 1 req/s per user, so we never poll faster than every 3 seconds.
 */
const POLL_INTERVAL_MS = 3000;

/**
 * Supports ?goodsId=<id> (defaults to 9890001). The hash query wins over the
 * search query so the goods switcher can navigate within the SPA without a
 * stale location.search overriding the selection.
 */
function resolveGoodsId(): string {
  let q: string | null = null;
  if (location.hash.includes('?')) {
    q = new URLSearchParams(location.hash.slice(location.hash.indexOf('?'))).get('goodsId');
  }
  if (!q) q = new URLSearchParams(location.search).get('goodsId');
  return q && q.trim() ? q.trim() : DEFAULT_GOODS_ID;
}

/** Home page: goods detail + group-buy teams + the lock → settle → refund trade flow. */
@customElement('x-home-page')
export class HomePage extends UiElement {
  @state() private loading = true;
  @state() private data: MarketConfig | null = null;
  @state() private errorMsg = '';
  @state() private payOpen = false;
  @state() private paying = false;
  @state() private locking = false;
  /** Solo ("buy alone") payment: simulated list-price payment, no group order. */
  @state() private paySolo = false;
  @state() private payAmount = '0.00';
  @state() private payOutTradeNo = '';
  @state() private pendingOutTradeNo = '';
  @state() private settledOutTradeNo = '';
  @state() private goodsId: string = resolveGoodsId();
  /** Carousel artwork: generated on canvas from a goodsId-derived seed; no image files. */
  @state() private carouselImages: string[] = [0, 1, 2].map((i) => genProductArt(`${resolveGoodsId()}-${i}`));

  private overlayRef: Ref<HTMLDivElement> = createRef();
  private panelRef: Ref<HTMLDivElement> = createRef();
  private animatedCards = new WeakSet<Element>();
  private me: string | null = getUsername();
  private routeUnsub?: () => void;

  private pollTimer?: number;
  private inFlight = false;
  private lastQueryAt = 0;

  /** localStorage keys: the pending (unpaid) trade number survives refreshes;
   *  the settled trade number keeps the refund action reachable for joiners. */
  private get pendingKey(): string {
    return `be261028:pending:${this.me}:${this.goodsId}`;
  }

  private get settledKey(): string {
    return `be261028:settled:${this.me}:${this.goodsId}`;
  }

  private readStorage(key: string): string {
    try {
      return localStorage.getItem(key) ?? '';
    } catch {
      return '';
    }
  }

  private savePending(outTradeNo: string): void {
    try {
      localStorage.setItem(this.pendingKey, outTradeNo);
    } catch {
      /* storage errors are non-fatal */
    }
    this.pendingOutTradeNo = outTradeNo;
  }

  private clearPending(): void {
    try {
      localStorage.removeItem(this.pendingKey);
    } catch {
      /* storage errors are non-fatal */
    }
    this.pendingOutTradeNo = '';
  }

  private saveSettled(outTradeNo: string): void {
    try {
      localStorage.setItem(this.settledKey, outTradeNo);
    } catch {
      /* storage errors are non-fatal */
    }
    this.settledOutTradeNo = outTradeNo;
  }

  private clearSettled(): void {
    try {
      localStorage.removeItem(this.settledKey);
    } catch {
      /* storage errors are non-fatal */
    }
    this.settledOutTradeNo = '';
  }

  override connectedCallback(): void {
    super.connectedCallback();
    if (!this.me) {
      navigate('/login', { replace: true });
      return;
    }
    // In-app goods switching: the switcher navigates the hash query, and the
    // route listener swaps the goods without a full page reload.
    this.routeUnsub = onRouteChange(() => {
      const id = resolveGoodsId();
      if (id !== this.goodsId) this.applyGoods(id);
    });
    void this.load(false);
    // Silent background refresh so team seats and countdowns stay current.
    this.pollTimer = window.setInterval(() => {
      if (document.hidden || this.inFlight || !this.data) return;
      if (Date.now() - this.lastQueryAt < POLL_INTERVAL_MS) return;
      void this.load(true);
    }, POLL_INTERVAL_MS);
    requestAnimationFrame(() => {
      animate(this, { opacity: [0, 1], transform: ['translateY(10px)', 'translateY(0)'] }, {
        duration: 0.4,
        ease: [0.22, 1, 0.36, 1],
      });
    });
  }

  override disconnectedCallback(): void {
    super.disconnectedCallback();
    if (this.pollTimer !== undefined) window.clearInterval(this.pollTimer);
    this.routeUnsub?.();
  }

  /** Swaps the active goods in place (hash-query driven; no page reload). */
  private applyGoods(id: string): void {
    this.goodsId = id;
    this.carouselImages = [0, 1, 2].map((i) => genProductArt(`${id}-${i}`));
    this.data = null;
    this.errorMsg = '';
    this.animatedCards = new WeakSet<Element>();
    void this.load(false);
  }

  private async load(silent: boolean, retryOnRateLimit = true): Promise<void> {
    if (this.inFlight) return;
    this.inFlight = true;
    if (!silent) this.loading = true;
    this.pendingOutTradeNo = this.readStorage(this.pendingKey);
    this.settledOutTradeNo = this.readStorage(this.settledKey);
    this.lastQueryAt = Date.now();
    try {
      const res = await api.queryMarketConfig(this.me!, this.goodsId);
      if (res.code === '0000' && res.data) {
        this.data = res.data;
        this.errorMsg = '';
      } else if (res.code === '0006' && retryOnRateLimit) {
        // Per-user rate limit (1 req/s): back off briefly and retry once instead
        // of rendering the limiter as an error page.
        window.setTimeout(() => void this.load(silent, false), 1100);
      } else if (!silent) {
        this.errorMsg = friendlyError(res.code, res.info);
      }
    } catch {
      if (!silent) this.errorMsg = 'Network error — please make sure the gateway is running.';
    }
    this.loading = false;
    this.inFlight = false;
  }

  protected override updated(): void {
    // Team-card entrance: only newly added cards animate (staggered).
    const fresh: HTMLElement[] = [];
    this.querySelectorAll<HTMLElement>('.team-card').forEach((el) => {
      if (!this.animatedCards.has(el)) {
        this.animatedCards.add(el);
        fresh.push(el);
      }
    });
    if (fresh.length) {
      animate(
        fresh,
        { opacity: [0, 1], transform: ['translateY(18px)', 'translateY(0)'] },
        { duration: 0.45, delay: stagger(0.08), ease: [0.22, 1, 0.36, 1] },
      );
    }
  }

  private get ownTeam(): Team | null {
    return this.data?.teamList?.find((t) => t.userId === this.me) ?? null;
  }

  /**
   * Hide teams that are full or expired; my own team always stays visible so
   * the refund action remains reachable.
   */
  private isTeamVisible(t: Team): boolean {
    if (t.userId === this.me) return true;
    const full = t.lockCount >= t.targetCount;
    const end = parseTime(t.validEndTime);
    const expired = end ? end.getTime() <= Date.now() : t.validTimeCountdown === 'Ended';
    return !full && !expired;
  }

  private switchGoods(id: string): void {
    if (id === this.goodsId) return;
    // Always carry the goodsId in the hash query so it wins over any stale
    // ?goodsId= in location.search (see resolveGoodsId).
    navigate(`/?goodsId=${id}`);
  }

  /** Start a team / join one: if I am already in a team, resume my own order (idempotent, parity with legacy). */
  private async startOrJoin(teamId?: string): Promise<void> {
    if (!this.data || this.locking) return;
    const own = this.ownTeam;
    if (own) {
      // Owner team entries carry outTradeNo; random-sampled entries of other
      // users do not. Fall back to the locally persisted pending trade number.
      const resumeNo = own.outTradeNo || this.pendingOutTradeNo;
      if (resumeNo && resumeNo === this.settledOutTradeNo) {
        // The seat is already paid — reopening the cashier would only produce
        // a duplicate-settlement error, so explain instead.
        toast.info('You already paid for your seat in this campaign');
      } else if (resumeNo) {
        this.savePending(resumeNo);
        this.openPay(fmtPrice(this.data.goods.payPrice), resumeNo);
      } else {
        toast.info('You already have a seat in this campaign');
      }
      return;
    }
    this.locking = true;
    const outTradeNo = newOutTradeNo();
    try {
      const res = await api.lockOrder({
        userId: this.me!,
        teamId,
        activityId: Number(this.data.goods.activityId),
        goodsId: this.goodsId,
        outTradeNo,
        notifyUrl: `${location.origin}/api/v1/test/group_buy_notify`,
      });
      if (res.code === '0000') {
        this.savePending(outTradeNo);
        this.openPay(fmtPrice(res.data?.payPrice ?? this.data.goods.payPrice), outTradeNo);
        // Refresh the team list once the new lock is visible upstream.
        window.setTimeout(() => void this.load(true), 1200);
      } else {
        toast.error(friendlyError(res.code, res.info));
        void this.load(true);
      }
    } catch {
      toast.error('Network error — lock order failed');
    }
    this.locking = false;
  }

  /**
   * "Buy alone" (legacy parity): shows the cashier at the list price without
   * creating a group order — the backend has no solo-purchase API, so the
   * payment is simulated locally (the legacy page behaved the same way: modal
   * only, no lock_market_pay_order call).
   */
  private buyAlone(): void {
    if (!this.data || this.locking) return;
    this.paySolo = true;
    this.openPay(fmtPrice(this.data.goods.originalPrice), '');
  }

  private openPay(amount: string, outTradeNo: string): void {
    this.payAmount = amount;
    this.payOutTradeNo = outTradeNo;
    this.payOpen = true;
    requestAnimationFrame(() => {
      const overlay = this.overlayRef.value;
      const panel = this.panelRef.value;
      if (overlay) animate(overlay, { opacity: [0, 1] }, { duration: 0.25 });
      if (panel) {
        animate(
          panel,
          { opacity: [0, 1], transform: ['translateY(44px) scale(.96)', 'translateY(0) scale(1)'] },
          { type: 'spring', stiffness: 260, damping: 24 },
        );
      }
    });
  }

  private closePay(): void {
    const overlay = this.overlayRef.value;
    const panel = this.panelRef.value;
    const done = () => {
      this.payOpen = false;
      this.paySolo = false;
    };
    if (!overlay || !panel) {
      done();
      return;
    }
    Promise.all([
      animate(overlay, { opacity: [1, 0] }, { duration: 0.22 }).finished,
      animate(
        panel,
        { opacity: [1, 0], transform: ['translateY(0) scale(1)', 'translateY(30px) scale(.97)'] },
        { duration: 0.22 },
      ).finished,
    ])
      .then(done)
      .catch(done);
  }

  private async confirmPay(): Promise<void> {
    if (this.paying) return;
    if (this.paySolo) {
      // Simulated list-price purchase: no group order exists to settle.
      toast.success('Solo purchase completed (simulated)');
      this.closePay();
      return;
    }
    this.paying = true;
    try {
      const res = await api.settle(this.me!, this.payOutTradeNo);
      if (res.code === '0000') {
        toast.success('Payment settled — team progress +1');
        this.saveSettled(this.payOutTradeNo);
        this.clearPending();
        this.closePay();
        void this.load(true);
      } else {
        toast.error(friendlyError(res.code, res.info));
        // A business failure (e.g. order already settled/refunded) means the
        // pending trade number is no longer usable — drop it to stop nagging.
        this.clearPending();
      }
    } catch {
      toast.error('Network error — settlement failed');
    }
    this.paying = false;
  }

  /** Cancel an unpaid pending order: refund releases the locked seat upstream. */
  private async cancelPending(): Promise<void> {
    const outTradeNo = this.pendingOutTradeNo;
    this.clearPending();
    if (!outTradeNo) return;
    try {
      const res = await api.refund(this.me!, outTradeNo);
      if (res.code === '0000' && res.data?.code === 'repeat') {
        toast.info('That order was already refunded');
      } else if (res.code === '0000') {
        toast.info('Pending order cancelled — your seat has been released');
      } else {
        toast.info('Pending order discarded');
      }
    } catch {
      toast.info('Pending order discarded');
    }
    void this.load(true);
  }

  private async refundOrder(outTradeNo: string): Promise<void> {
    try {
      const res = await api.refund(this.me!, outTradeNo);
      if (res.code === '0000') {
        if (res.data?.code === 'success') {
          toast.success('Refund successful — your seat has been released');
        } else if (res.data?.code === 'repeat') {
          toast.info('This order was already refunded');
        } else {
          toast.info(res.data?.info ?? 'Refund processed');
        }
        this.clearSettled();
        this.clearPending();
        void this.load(true);
      } else {
        toast.error(friendlyError(res.code, res.info));
      }
    } catch {
      toast.error('Network error — refund failed');
    }
  }

  private logout(): void {
    clearUsername();
    navigate('/login', { replace: true });
  }

  render() {
    if (!this.me) return null;
    const d = this.data;
    const teams = (d?.teamList ?? []).filter((t) => this.isTeamVisible(t));
    const stat = d?.teamStatistic;

    return (
      <div className="min-h-dvh bg-zinc-200/70">
        <div className="relative mx-auto flex min-h-dvh w-full max-w-md flex-col bg-zinc-50 shadow-2xl shadow-zinc-900/10 lg:max-w-6xl xl:max-w-7xl">
          {/* Top bar */}
          <header className="sticky top-0 z-20 flex items-center justify-between gap-3 border-b border-zinc-200/60 bg-zinc-50/85 px-4 py-3 backdrop-blur lg:px-8">
            <div className="flex items-center gap-2.5">
              <span className="grid h-9 w-9 place-items-center rounded-2xl bg-gradient-to-br from-brand-500 to-orange-400 text-white shadow-md shadow-brand-500/30">
                {unsafeHTML(svgIcon(ICONS.cart, 'h-5 w-5'))}
              </span>
              <div>
                <div className="text-sm font-black leading-none text-zinc-900">be261028 Group Buy</div>
                <div className="mt-1 text-[10px] text-zinc-400">Team up and pay less</div>
              </div>
            </div>
            <div className="flex items-center gap-1.5">
              {/* Demo goods switcher (drives ?goodsId=) */}
              <div className="hidden items-center gap-0.5 rounded-full border border-zinc-200 bg-white p-0.5 md:flex">
                {DEMO_GOODS.map((id) => (
                  <button
                    onClick={() => this.switchGoods(id)}
                    className={`rounded-full px-2.5 py-1 font-mono text-[10px] font-semibold transition ${
                      id === this.goodsId
                        ? 'bg-gradient-to-r from-brand-500 to-orange-400 text-white shadow'
                        : 'text-zinc-400 hover:text-zinc-700'
                    }`}
                  >
                    {id}
                  </button>
                ))}
              </div>
              <button
                onClick={() => navigate('/admin')}
                className="rounded-full border border-zinc-200 bg-white px-3 py-1.5 text-xs font-medium text-zinc-500 transition hover:border-brand-300 hover:text-brand-600"
              >
                Admin console
              </button>
              <button
                onClick={() => this.logout()}
                className="flex items-center gap-1 rounded-full px-2 py-1.5 text-xs text-zinc-400 transition hover:text-brand-600"
              >
                {this.me}
                {unsafeHTML(svgIcon(ICONS.logout, 'h-3.5 w-3.5'))}
              </button>
            </div>
          </header>

          <main className="flex-1 px-4 pb-32 pt-4 lg:grid lg:grid-cols-12 lg:items-start lg:gap-8 lg:px-8 lg:pb-12 lg:pt-8">
            {this.loading ? this.renderSkeleton() : this.errorMsg ? this.renderError() : d && (
              <>
                {/* Left column: generated gallery (sticky on desktop) */}
                <div className="lg:sticky lg:top-24 lg:col-span-5 xl:col-span-4">
                  <Carousel images={this.carouselImages} />
                </div>

                {/* Right column: goods info, statistics, teams */}
                <div className="lg:col-span-7 lg:min-w-0 xl:col-span-8">
                  <section className="mt-4 rounded-3xl bg-white p-4 shadow-sm lg:mt-0 lg:p-6">
                    <div className="flex items-end gap-2">
                      <div className="flex items-baseline text-brand-600">
                        <span className="text-sm font-bold">¥</span>
                        <span className="bg-gradient-to-r from-brand-600 to-orange-500 bg-clip-text text-3xl font-black tracking-tight text-transparent">
                          {fmtPrice(d.goods.payPrice)}
                        </span>
                      </div>
                      <span className="pb-1 text-xs text-zinc-400 line-through">
                        ¥{fmtPrice(d.goods.originalPrice)}
                      </span>
                      <span className="mb-1 ml-auto rounded-full bg-gradient-to-r from-brand-500 to-orange-400 px-2.5 py-1 text-[10px] font-bold text-white shadow shadow-brand-500/30">
                        Group-buy deal
                      </span>
                    </div>
                    <p className="mt-2 text-xs font-medium text-brand-600">
                      Direct deduction ¥{fmtPrice(d.goods.deductionPrice)}
                    </p>
                    {this.renderDiscount(d)}
                    <h1 className="mt-2 text-[15px] font-bold leading-snug text-zinc-900">
                      {PRODUCT_TITLES[this.goodsId] ?? `Demo goods ${this.goodsId}`}
                    </h1>
                    <div className="mt-2 flex flex-wrap gap-1.5">
                      <span className="rounded-md bg-zinc-100 px-2 py-0.5 text-[10px] text-zinc-500">Genuine goods guarantee</span>
                      <span className="rounded-md bg-zinc-100 px-2 py-0.5 text-[10px] text-zinc-500">Instant refunds</span>
                      <span className="rounded-md bg-zinc-100 px-2 py-0.5 text-[10px] text-zinc-500">Auto team completion</span>
                    </div>

                    {/* Desktop purchase actions (mobile uses the fixed bottom bar) */}
                    <div className="mt-5 hidden gap-3 lg:flex">
                      <button
                        onClick={() => this.buyAlone()}
                        disabled={this.locking}
                        className="flex-1 rounded-2xl border-2 border-zinc-200 bg-white py-3 text-sm font-bold text-zinc-600 transition hover:border-brand-200 active:scale-[.98] disabled:opacity-60"
                      >
                        Buy solo
                        <span className="ml-1 text-xs font-medium text-zinc-400">¥{fmtPrice(d.goods.originalPrice)}</span>
                      </button>
                      <button
                        onClick={() => void this.startOrJoin()}
                        disabled={this.locking}
                        className="flex-1 rounded-2xl bg-gradient-to-r from-brand-500 to-orange-400 py-3 text-sm font-black text-white shadow-lg shadow-brand-500/35 transition active:scale-[.98] disabled:opacity-60"
                      >
                        Start a team
                        <span className="ml-1 text-xs font-semibold text-white/85">¥{fmtPrice(d.goods.payPrice)}</span>
                      </button>
                    </div>
                  </section>

                  {/* Team statistics row */}
                  {stat && (
                    <section className="mt-4 grid grid-cols-3 gap-2.5 lg:mt-5">
                      {this.renderStat(ICONS.users, 'Teams created', stat.allTeamCount)}
                      {this.renderStat(ICONS.check, 'Teams completed', stat.allTeamCompleteCount)}
                      {this.renderStat(ICONS.bolt, 'Participants', stat.allTeamUserCount)}
                    </section>
                  )}

                  {/* Pending (unpaid) order resume bar — survives page refreshes */}
                  {this.pendingOutTradeNo && !this.payOpen && (
                    <div className="mt-4 flex items-center justify-between gap-3 rounded-2xl border border-brand-200/70 bg-gradient-to-r from-brand-50 to-orange-50 px-4 py-3 shadow-sm lg:mt-5">
                      <div className="min-w-0">
                        <p className="text-xs font-bold text-brand-700">You have an unpaid group-buy order</p>
                        <p className="mt-0.5 truncate font-mono text-[10px] text-brand-400">{this.pendingOutTradeNo}</p>
                      </div>
                      <div className="flex shrink-0 items-center gap-2">
                        <button
                          onClick={() => void this.cancelPending()}
                          className="rounded-full px-2 py-1 text-[11px] text-zinc-400 transition hover:text-zinc-600"
                        >
                          Cancel
                        </button>
                        <button
                          onClick={() => d && this.openPay(fmtPrice(d.goods.payPrice), this.pendingOutTradeNo)}
                          className="rounded-full bg-gradient-to-r from-brand-500 to-orange-400 px-4 py-1.5 text-xs font-bold text-white shadow shadow-brand-500/30 transition active:scale-95"
                        >
                          Resume payment
                        </button>
                      </div>
                    </div>
                  )}

                  {/* Paid seat (joiner) — refund stays reachable even when I do not lead a team */}
                  {this.settledOutTradeNo && !this.ownTeam && (
                    <div className="mt-4 flex items-center justify-between gap-3 rounded-2xl border border-emerald-200/70 bg-emerald-50/70 px-4 py-3 shadow-sm lg:mt-5">
                      <div className="min-w-0">
                        <p className="text-xs font-bold text-emerald-700">You have a paid seat in this campaign</p>
                        <p className="mt-0.5 truncate font-mono text-[10px] text-emerald-500">{this.settledOutTradeNo}</p>
                      </div>
                      <button
                        onClick={() => void this.refundOrder(this.settledOutTradeNo)}
                        className="shrink-0 rounded-full border border-emerald-200 bg-white px-4 py-1.5 text-xs font-semibold text-emerald-600 transition hover:bg-emerald-100 active:scale-95"
                      >
                        Refund
                      </button>
                    </div>
                  )}

                  {/* Team list */}
                  <section className="mt-5 lg:mt-6">
                    <div className="mb-2.5 flex items-center justify-between px-1">
                      <h2 className="text-sm font-black text-zinc-900">
                        Active teams
                        <span className="ml-1.5 text-xs font-medium text-zinc-400">Join one directly</span>
                      </h2>
                      <button
                        onClick={() => void this.load(true)}
                        disabled={this.inFlight}
                        className="flex items-center gap-1 text-xs text-zinc-400 transition hover:text-brand-600 disabled:opacity-50"
                      >
                        {unsafeHTML(svgIcon(ICONS.refresh, 'h-3.5 w-3.5'))}
                        Refresh
                      </button>
                    </div>

                    {teams.length === 0 ? (
                      <div className="rounded-3xl border-2 border-dashed border-zinc-200 bg-white/60 px-6 py-10 text-center">
                        <div className="mx-auto grid h-12 w-12 place-items-center rounded-2xl bg-gradient-to-br from-brand-100 to-orange-100 text-brand-500">
                          {unsafeHTML(svgIcon(ICONS.sparkles, 'h-6 w-6'))}
                        </div>
                        <p className="mt-3 text-sm font-semibold text-zinc-600">
                          No active teams right now — start the first one!
                        </p>
                        <p className="mt-1 text-xs text-zinc-400">Lock in the group price and let your team fill up.</p>
                      </div>
                    ) : (
                      <div className="flex flex-col gap-2.5">
                        {teams.map((t) => this.renderTeam(t))}
                      </div>
                    )}
                  </section>
                </div>
              </>
            )}
          </main>

          {/* Mobile bottom action bar */}
          {!this.loading && d && (
            <div className="fixed inset-x-0 bottom-0 z-20 mx-auto w-full max-w-md border-t border-zinc-200/60 bg-white/90 px-4 pb-[calc(env(safe-area-inset-bottom,0px)+0.75rem)] pt-3 backdrop-blur lg:hidden">
              <div className="mx-auto flex w-full gap-3">
                <button
                  onClick={() => this.buyAlone()}
                  disabled={this.locking}
                  className="flex-1 rounded-2xl border-2 border-zinc-200 bg-white py-3 text-sm font-bold text-zinc-600 transition active:scale-[.98] disabled:opacity-60"
                >
                  Buy solo
                  <span className="ml-1 text-xs font-medium text-zinc-400">¥{fmtPrice(d.goods.originalPrice)}</span>
                </button>
                <button
                  onClick={() => void this.startOrJoin()}
                  disabled={this.locking}
                  className="flex-1 rounded-2xl bg-gradient-to-r from-brand-500 to-orange-400 py-3 text-sm font-black text-white shadow-lg shadow-brand-500/35 transition active:scale-[.98] disabled:opacity-60"
                >
                  Start a team
                  <span className="ml-1 text-xs font-semibold text-white/85">¥{fmtPrice(d.goods.payPrice)}</span>
                </button>
              </div>
            </div>
          )}

          {/* Payment modal */}
          {this.payOpen && (
            <div
              ref={this.overlayRef}
              onClick={() => this.closePay()}
              className="fixed inset-0 z-40 bg-zinc-950/45 backdrop-blur-sm"
            >
              <div className="absolute inset-0 mx-auto flex w-full max-w-md items-end px-4 pb-6 lg:items-center lg:justify-center lg:pb-0">
                <div ref={this.panelRef} onClick={(e) => e.stopPropagation()} className="w-full rounded-3xl bg-white p-6 shadow-2xl">
                  <div className="mx-auto h-1 w-10 rounded-full bg-zinc-200 lg:hidden" />
                  <h3 className="mt-3 text-center text-base font-black text-zinc-900 lg:mt-0">Complete payment</h3>
                  <p className="mt-1 text-center text-[11px] text-zinc-400">Simulated QR — no real charge is made</p>
                  <div className="mt-2 text-center">
                    <span className="text-sm font-bold text-brand-600">¥</span>
                    <span className="text-3xl font-black text-brand-600">{this.payAmount}</span>
                  </div>
                  {/* Simulated payment QR: canvas-generated from the trade number; not an image file. */}
                  <img
                    src={genQrArt(
                      this.paySolo ? `solo-${this.goodsId}-${this.me}` : this.payOutTradeNo || 'be261028-pay',
                    )}
                    alt="Simulated payment QR code"
                    className="mx-auto mt-4 h-44 w-44 rounded-2xl border border-zinc-100 object-contain"
                  />
                  <p className="mt-3 text-center font-mono text-[11px] text-zinc-400">
                    {this.paySolo ? 'Solo purchase · no group order' : `Trade no. ${this.payOutTradeNo}`}
                  </p>
                  <div className="mt-5 flex gap-3">
                    <button
                      onClick={() => this.closePay()}
                      className="flex-1 rounded-xl border border-zinc-200 py-2.5 text-sm font-medium text-zinc-500 transition active:scale-[.98]"
                    >
                      Cancel
                    </button>
                    <button
                      onClick={() => void this.confirmPay()}
                      disabled={this.paying}
                      className="flex-1 rounded-xl bg-gradient-to-r from-brand-500 to-orange-400 py-2.5 text-sm font-black text-white shadow-lg shadow-brand-500/30 transition active:scale-[.98] disabled:opacity-60"
                    >
                      {this.paying ? 'Processing…' : 'Confirm payment'}
                    </button>
                  </div>
                </div>
              </div>
            </div>
          )}
        </div>
      </div>
    );
  }

  private renderStat(icon: string, label: string, value: number) {
    return (
      <div className="rounded-2xl bg-white px-2 py-3 text-center shadow-sm">
        <div className="mx-auto grid h-7 w-7 place-items-center rounded-full bg-brand-50 text-brand-600">
          {unsafeHTML(svgIcon(icon, 'h-3.5 w-3.5'))}
        </div>
        <div className="mt-1.5 text-lg font-black tabular-nums text-zinc-900">{value}</div>
        <div className="text-[10px] font-medium text-zinc-400">{label}</div>
      </div>
    );
  }

  private renderTeam(t: Team) {
    const remaining = Math.max(0, t.targetCount - t.lockCount);
    const pct = Math.min(100, Math.round((t.completeCount / Math.max(1, t.targetCount)) * 100));
    const mine = t.userId === this.me;
    // Owner entries carry outTradeNo (the refund handle); random-sampled
    // entries of other users omit it. The locally persisted settled trade
    // number is the fallback for teams that already completed and left the
    // in-progress list (e.g. the target-1 demo campaign).
    const refundNo = mine ? (t.outTradeNo || this.settledOutTradeNo) : '';
    return (
      <div className="team-card flex items-center justify-between gap-3 rounded-2xl bg-white p-3.5 shadow-sm">
        <div className="flex min-w-0 items-center gap-3">
          {/* Avatar: deterministic canvas identicon, never an image file. */}
          <img
            src={genAvatarArt(t.userId)}
            alt={`Avatar of ${t.userId}`}
            className="h-10 w-10 shrink-0 rounded-full shadow ring-2 ring-white"
          />
          <div className="min-w-0">
            <div className="flex items-center gap-1.5 text-sm font-bold text-zinc-800">
              <span className="truncate">{t.userId}</span>
              {mine && (
                <span className="shrink-0 rounded-full bg-brand-50 px-1.5 py-0.5 text-[10px] font-semibold text-brand-600">
                  My team
                </span>
              )}
            </div>
            {/* Team progress: completeCount (paid seats) / targetCount */}
            <div className="mt-1 flex items-center gap-2">
              <span className="h-1.5 w-20 overflow-hidden rounded-full bg-zinc-100">
                <span
                  className="block h-full rounded-full bg-gradient-to-r from-brand-500 to-orange-400 transition-all duration-500"
                  style={{ width: `${pct}%` }}
                />
              </span>
              <span className="font-mono text-[10px] tabular-nums text-zinc-400">
                {t.completeCount}/{t.targetCount}
              </span>
            </div>
            <div className="mt-1 flex items-center gap-2 text-xs text-zinc-400">
              <span className={remaining <= 1 ? 'font-semibold text-brand-500' : ''}>
                {remaining === 1 ? 'Only 1 seat left' : `Only ${remaining} seats left`}
              </span>
              <span className="text-zinc-200">|</span>
              <CountDown endTime={t.validEndTime} initial={t.validTimeCountdown} />
            </div>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {refundNo && (
            <button
              onClick={() => void this.refundOrder(refundNo)}
              className="rounded-full border border-zinc-200 px-3 py-1.5 text-xs text-zinc-400 transition hover:border-brand-200 hover:text-brand-600"
            >
              Refund
            </button>
          )}
          <button
            onClick={() => void this.startOrJoin(t.teamId)}
            disabled={this.locking}
            className="rounded-full bg-gradient-to-r from-brand-500 to-orange-400 px-4 py-1.5 text-xs font-bold text-white shadow shadow-brand-500/30 transition active:scale-95 disabled:opacity-60"
          >
            Join team
          </button>
        </div>
      </div>
    );
  }

  /** Discount info: render the VO when the backend sends one; otherwise derive an equivalent expression from the trial prices. */
  private renderDiscount(d: MarketConfig) {
    const disc = d.groupBuyActivityDiscountVO?.groupBuyDiscount;
    if (disc && (disc.discountName || disc.discountDesc || disc.marketExpr)) {
      const plan = disc.marketPlan ?? '';
      return (
        <div className="mt-2.5 flex flex-wrap items-center gap-1.5">
          <span className="rounded-md bg-brand-50 px-2 py-1 text-[10px] font-bold text-brand-600">
            {plan ? `${plan} · ${MARKET_PLAN_LABELS[plan] ?? 'Discount'}` : 'Discount'}
          </span>
          {disc.discountName && (
            <span className="text-xs font-semibold text-zinc-700">{disc.discountName}</span>
          )}
          {disc.discountDesc && disc.discountDesc !== disc.discountName && (
            <span className="text-[11px] text-zinc-400">{disc.discountDesc}</span>
          )}
          {disc.marketExpr && (
            <span className="rounded-md bg-zinc-100 px-1.5 py-0.5 font-mono text-[10px] text-zinc-500">
              {disc.marketExpr}
            </span>
          )}
        </div>
      );
    }
    const { plan, expr, note } = deriveDiscountExpr(d.goods);
    return (
      <div className="mt-2.5 flex flex-wrap items-center gap-1.5">
        <span className="rounded-md bg-brand-50 px-2 py-1 text-[10px] font-bold text-brand-600">
          {plan} · {MARKET_PLAN_LABELS[plan]}
        </span>
        <span className="text-xs font-semibold text-zinc-700">{expr}</span>
        <span className="rounded-md bg-zinc-100 px-1.5 py-0.5 font-mono text-[10px] text-zinc-500">{note}</span>
      </div>
    );
  }

  private renderSkeleton() {
    return (
      <div className="flex animate-pulse flex-col gap-4 lg:col-span-12 lg:grid lg:grid-cols-12 lg:gap-8">
        <div className="aspect-square w-full rounded-3xl bg-zinc-200 lg:col-span-5 xl:col-span-4" />
        <div className="flex flex-col gap-4 lg:col-span-7 xl:col-span-8">
          <div className="h-40 rounded-3xl bg-zinc-200" />
          <div className="h-16 rounded-2xl bg-zinc-200" />
          <div className="h-16 rounded-2xl bg-zinc-200" />
        </div>
      </div>
    );
  }

  private renderError() {
    return (
      <div className="mt-16 flex flex-col items-center px-6 text-center lg:col-span-12">
        <div className="grid h-14 w-14 place-items-center rounded-2xl bg-zinc-100 text-zinc-400">
          {unsafeHTML(svgIcon(ICONS.alert, 'h-7 w-7'))}
        </div>
        <p className="mt-4 text-sm font-semibold text-zinc-600">{this.errorMsg}</p>
        <button
          onClick={() => void this.load(false)}
          className="mt-5 rounded-full bg-gradient-to-r from-brand-500 to-orange-400 px-6 py-2.5 text-sm font-bold text-white shadow-lg shadow-brand-500/30 transition active:scale-95"
        >
          Reload
        </button>
      </div>
    );
  }
}
