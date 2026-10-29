import { createRef, customElement, state, type Ref } from '@yukino.js/lit-jsx';
import { animate, stagger } from 'motion';
import {
  api,
  deriveDiscountExpr,
  friendlyError,
  MARKET_PLAN_LABELS,
  type MarketConfig,
  type Team,
} from '../lib/api';
import { genProductArt, genQrArt } from '../lib/art';
import { clearUsername, getUsername } from '../lib/auth';
import { navigate } from '../lib/router';
import { toast } from '../lib/toast';
import { fmtPrice, randomNumeric } from '../lib/utils';
import { UiElement } from '../components/base';
import { Carousel } from '../components/carousel';
import { CountDown } from '../components/countdown';

const DEFAULT_GOODS_ID = '9890001';

/** 支持 ?goodsId=xxx 指定商品（默认 9890001），便于演示不同活动配置。 */
function resolveGoodsId(): string {
  const q = new URLSearchParams(location.search).get('goodsId');
  return q && q.trim() ? q.trim() : DEFAULT_GOODS_ID;
}
const PRODUCT_TITLE = '手写MyBatis：渐进式源码实践（全彩）';

/** 首页：商品详情 + 拼团队伍 + 锁单/结算交易流程（复刻并美化旧版 index.html）。 */
@customElement('x-home-page')
export class HomePage extends UiElement {
  @state() private loading = true;
  @state() private data: MarketConfig | null = null;
  @state() private errorMsg = '';
  @state() private payOpen = false;
  @state() private paying = false;
  @state() private payAmount = '0.00';
  @state() private payOutTradeNo = '';
  @state() private pendingOutTradeNo = '';

  private overlayRef: Ref<HTMLDivElement> = createRef();
  private panelRef: Ref<HTMLDivElement> = createRef();
  private listAnimated = false;
  private me: string | null = getUsername();
  private readonly goodsId: string = resolveGoodsId();
  /** 轮播图：canvas 代码生成（goodsId 派生 seed），全站不使用图片文件。 */
  private readonly carouselImages: string[] = [0, 1, 2].map((i) => genProductArt(`${this.goodsId}-${i}`));

  /** 待支付单号持久化 key：刷新后可恢复支付（旧版刷新即丢失，此处为前端改进）。 */
  private get pendingKey(): string {
    return `be261028:pending:${this.me}`;
  }

  private savePending(outTradeNo: string): void {
    try {
      localStorage.setItem(this.pendingKey, outTradeNo);
      this.pendingOutTradeNo = outTradeNo;
    } catch {
      /* 忽略存储异常 */
    }
  }

  private clearPending(): void {
    try {
      localStorage.removeItem(this.pendingKey);
    } catch {
      /* 忽略存储异常 */
    }
    this.pendingOutTradeNo = '';
  }

  override connectedCallback(): void {
    super.connectedCallback();
    if (!this.me) {
      navigate('/login', { replace: true });
      return;
    }
    void this.load();
    requestAnimationFrame(() => {
      animate(this, { opacity: [0, 1], transform: ['translateY(10px)', 'translateY(0)'] }, {
        duration: 0.4,
        ease: [0.22, 1, 0.36, 1],
      });
    });
  }

  private async load(retryOnRateLimit = true): Promise<void> {
    this.loading = true;
    try {
      this.pendingOutTradeNo = localStorage.getItem(this.pendingKey) ?? '';
    } catch {
      this.pendingOutTradeNo = '';
    }
    try {
      const res = await api.queryMarketConfig(this.me!, this.goodsId);
      if (res.code === '0000' && res.data) {
        this.data = res.data;
        this.errorMsg = '';
        this.listAnimated = false;
      } else if (res.code === '0006' && retryOnRateLimit) {
        // 网关限流（每用户 1 req/s）：稍后自动重试，避免刷新结果被渲染成错误页。
        setTimeout(() => void this.load(false), 1100);
        return;
      } else {
        this.errorMsg = friendlyError(res.code, res.info);
      }
    } catch {
      this.errorMsg = '网络异常，请确认网关服务已启动';
    }
    this.loading = false;
  }

  protected override updated(): void {
    if (!this.listAnimated && this.data && !this.loading) {
      this.listAnimated = true;
      const cards = this.querySelectorAll<HTMLElement>('.team-card');
      if (cards.length) {
        animate(
          cards,
          { opacity: [0, 1], transform: ['translateY(18px)', 'translateY(0)'] },
          { duration: 0.45, delay: stagger(0.08), ease: [0.22, 1, 0.36, 1] },
        );
      }
    }
  }

  private get ownTeam(): Team | null {
    return this.data?.teamList?.find((t) => t.userId === this.me) ?? null;
  }

  /** 开新团 / 参团：若本人已在团中则直接拉起支付（幂等，与旧版一致）。 */
  private async startOrJoin(teamId?: string): Promise<void> {
    if (!this.data) return;
    const own = this.ownTeam;
    if (own) {
      this.savePending(own.outTradeNo);
      this.openPay(fmtPrice(this.data.goods.payPrice), own.outTradeNo);
      return;
    }
    const outTradeNo = randomNumeric(12);
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
      } else {
        toast.error(friendlyError(res.code, res.info));
        void this.load();
      }
    } catch {
      toast.error('网络异常，锁单失败');
    }
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
    this.paying = true;
    try {
      const res = await api.settle(this.me!, this.payOutTradeNo);
      if (res.code === '0000') {
        toast.success('支付 & 结算成功，拼团进度 +1');
        this.clearPending();
        this.closePay();
        void this.load();
      } else {
        toast.error(friendlyError(res.code, res.info));
        // 业务失败（如订单已结算/已退）说明待支付单已失效，清除避免反复提示
        this.clearPending();
      }
    } catch {
      toast.error('网络异常，结算失败');
    }
    this.paying = false;
  }

  private async refundOwn(team: Team): Promise<void> {
    try {
      const res = await api.refund(this.me!, team.outTradeNo);
      if (res.code === '0000') {
        if (res.data?.code === 'success') {
          toast.success('退单成功，名额已释放');
        } else if (res.data?.code === 'repeat') {
          toast.info('已经退过单啦');
        } else {
          toast.info(res.data?.info ?? '退单完成');
        }
        this.clearPending();
        void this.load();
      } else {
        toast.error(friendlyError(res.code, res.info));
      }
    } catch {
      toast.error('网络异常，退单失败');
    }
  }

  private logout(): void {
    clearUsername();
    navigate('/login', { replace: true });
  }

  render() {
    if (!this.me) return null;
    const d = this.data;
    const teams = d?.teamList ?? [];
    const stat = d?.teamStatistic;

    return (
      <div className="min-h-dvh bg-zinc-200/70">
        <div className="relative mx-auto flex min-h-dvh w-full max-w-md flex-col bg-zinc-50 shadow-2xl shadow-zinc-900/10 lg:max-w-5xl xl:max-w-6xl">
          {/* 顶栏 */}
          <header className="sticky top-0 z-20 flex items-center justify-between border-b border-zinc-200/60 bg-zinc-50/85 px-4 py-3 backdrop-blur">
            <div className="flex items-center gap-2.5">
              <span className="grid h-9 w-9 place-items-center rounded-2xl bg-gradient-to-br from-brand-500 to-orange-400 text-lg shadow-md shadow-brand-500/30">
                🛒
              </span>
              <div>
                <div className="text-sm font-black leading-none text-zinc-900">be261028 拼团</div>
                <div className="mt-1 text-[10px] text-zinc-400">越拼越便宜 · 人满即成团</div>
              </div>
            </div>
            <div className="flex items-center gap-1.5">
              <button
                onClick={() => navigate('/admin')}
                className="rounded-full border border-zinc-200 bg-white px-3 py-1.5 text-xs font-medium text-zinc-500 transition hover:border-brand-300 hover:text-brand-600"
              >
                管理后台
              </button>
              <button
                onClick={() => this.logout()}
                className="rounded-full px-2 py-1.5 text-xs text-zinc-400 transition hover:text-brand-600"
              >
                {this.me} · 退出
              </button>
            </div>
          </header>

          <main className="flex-1 px-4 pb-32 pt-4 lg:grid lg:grid-cols-12 lg:items-start lg:gap-8 lg:px-8 lg:pb-28 lg:pt-8">
            {this.loading ? this.renderSkeleton() : this.errorMsg ? this.renderError() : d && (
              <>
                <div className="lg:sticky lg:top-24 lg:col-span-5 xl:col-span-4">
                  <Carousel images={this.carouselImages} />
                </div>

                <div className="lg:col-span-7 lg:min-w-0 xl:col-span-8">
                  {/* 商品信息 */}
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
                        大促拼团
                      </span>
                    </div>
                    <p className="mt-2 text-xs font-medium text-brand-600">
                      直降 ¥{fmtPrice(d.goods.deductionPrice)}
                      {stat ? ` · ${stat.allTeamUserCount} 人已抢 · 成团 ${stat.allTeamCompleteCount}/${stat.allTeamCount} 队` : ''}
                    </p>
                    {this.renderDiscount(d)}
                    <h1 className="mt-2 text-[15px] font-bold leading-snug text-zinc-900">{PRODUCT_TITLE}</h1>
                    <div className="mt-2 flex gap-1.5">
                      <span className="rounded-md bg-zinc-100 px-2 py-0.5 text-[10px] text-zinc-500">正品保障</span>
                      <span className="rounded-md bg-zinc-100 px-2 py-0.5 text-[10px] text-zinc-500">极速退款</span>
                      <span className="rounded-md bg-zinc-100 px-2 py-0.5 text-[10px] text-zinc-500">人满自动成团</span>
                    </div>
                  </section>

                  {/* 待支付订单恢复栏（刷新后可继续支付，旧版刷新即丢失） */}
                  {this.pendingOutTradeNo && !this.payOpen && (
                    <div className="pending-bar mt-4 flex items-center justify-between rounded-2xl border border-brand-200/70 bg-gradient-to-r from-brand-50 to-orange-50 px-4 py-3 shadow-sm">
                      <div className="min-w-0">
                        <p className="text-xs font-bold text-brand-700">你有一笔待支付的拼团订单</p>
                        <p className="mt-0.5 truncate font-mono text-[10px] text-brand-400">{this.pendingOutTradeNo}</p>
                      </div>
                      <div className="flex shrink-0 items-center gap-2">
                        <button
                          onClick={() => this.clearPending()}
                          className="rounded-full px-2 py-1 text-[11px] text-zinc-400 transition hover:text-zinc-600"
                        >
                          忽略
                        </button>
                        <button
                          onClick={() => d && this.openPay(fmtPrice(d.goods.payPrice), this.pendingOutTradeNo)}
                          className="rounded-full bg-gradient-to-r from-brand-500 to-orange-400 px-4 py-1.5 text-xs font-bold text-white shadow shadow-brand-500/30 transition active:scale-95"
                        >
                          继续支付
                        </button>
                      </div>
                    </div>
                  )}

                  {/* 拼团队伍 */}
                  <section className="mt-5 lg:mt-6">
                    <div className="mb-2.5 flex items-center justify-between px-1">
                      <h2 className="text-sm font-black text-zinc-900">
                        正在拼团
                        <span className="ml-1.5 text-xs font-medium text-zinc-400">可直接参与</span>
                      </h2>
                      <button
                        onClick={() => void this.load()}
                        className="text-xs text-zinc-400 transition hover:text-brand-600"
                      >
                        刷新 ↻
                      </button>
                    </div>

                    {teams.length === 0 ? (
                      <div className="rounded-3xl border-2 border-dashed border-zinc-200 bg-white/60 px-6 py-10 text-center">
                        <div className="text-4xl">🎉</div>
                        <p className="mt-3 text-sm font-semibold text-zinc-600">
                          小伙伴，赶紧去开团吧，做村里最靓的仔！
                        </p>
                        <p className="mt-1 text-xs text-zinc-400">当前还没有进行中的队伍</p>
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

          {/* 底部操作栏 */}
          {!this.loading && d && (
            <div className="fixed inset-x-0 bottom-0 z-20 mx-auto w-full max-w-md border-t border-zinc-200/60 bg-white/90 px-4 pb-[calc(env(safe-area-inset-bottom,0px)+0.75rem)] pt-3 backdrop-blur lg:max-w-5xl lg:px-8 xl:max-w-6xl">
              <div className="mx-auto flex w-full gap-3 lg:max-w-lg">
                <button
                  onClick={() => this.startOrJoin()}
                  className="flex-1 rounded-2xl border-2 border-zinc-200 bg-white py-3 text-sm font-bold text-zinc-600 transition active:scale-[.98]"
                >
                  单独购买
                  <span className="ml-1 text-xs font-medium text-zinc-400">¥{fmtPrice(d.goods.originalPrice)}</span>
                </button>
                <button
                  onClick={() => this.startOrJoin()}
                  className="flex-1 rounded-2xl bg-gradient-to-r from-brand-500 to-orange-400 py-3 text-sm font-black text-white shadow-lg shadow-brand-500/35 transition active:scale-[.98]"
                >
                  开团购买
                  <span className="ml-1 text-xs font-semibold text-white/85">¥{fmtPrice(d.goods.payPrice)}</span>
                </button>
              </div>
            </div>
          )}

          {/* 支付弹窗 */}
          {this.payOpen && (
            <div
              ref={this.overlayRef}
              onClick={() => this.closePay()}
              className="fixed inset-0 z-40 bg-zinc-950/45 backdrop-blur-sm"
            >
              <div className="absolute inset-0 mx-auto flex w-full max-w-md items-end px-4 pb-6 lg:items-center lg:justify-center lg:pb-0">
                <div ref={this.panelRef} onClick={(e) => e.stopPropagation()} className="w-full rounded-3xl bg-white p-6 shadow-2xl">
                  <div className="mx-auto h-1 w-10 rounded-full bg-zinc-200" />
                  <h3 className="mt-3 text-center text-base font-black text-zinc-900">请扫码支付</h3>
                  <div className="mt-1.5 text-center">
                    <span className="text-sm font-bold text-brand-600">¥</span>
                    <span className="text-3xl font-black text-brand-600">{this.payAmount}</span>
                  </div>
                  {/* 模拟支付二维码：按交易单号 canvas 生成，不使用图片文件 */}
                  <img
                    src={genQrArt(this.payOutTradeNo || 'be261028-pay')}
                    alt="支付二维码（模拟）"
                    className="mx-auto mt-4 h-44 w-44 rounded-2xl border border-zinc-100 object-contain"
                  />
                  <p className="mt-3 text-center font-mono text-[11px] text-zinc-400">
                    交易单号 {this.payOutTradeNo}
                  </p>
                  <div className="mt-5 flex gap-3">
                    <button
                      onClick={() => this.closePay()}
                      className="flex-1 rounded-xl border border-zinc-200 py-2.5 text-sm font-medium text-zinc-500 transition active:scale-[.98]"
                    >
                      取消支付
                    </button>
                    <button
                      onClick={() => void this.confirmPay()}
                      disabled={this.paying}
                      className="flex-1 rounded-xl bg-gradient-to-r from-brand-500 to-orange-400 py-2.5 text-sm font-black text-white shadow-lg shadow-brand-500/30 transition active:scale-[.98] disabled:opacity-60"
                    >
                      {this.paying ? '结算中…' : '支付完成'}
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

  private renderTeam(t: Team) {
    const remaining = Math.max(0, t.targetCount - t.lockCount);
    const pct = Math.min(100, Math.round((t.completeCount / Math.max(1, t.targetCount)) * 100));
    const mine = t.userId === this.me;
    return (
      <div className="team-card flex items-center justify-between rounded-2xl bg-white p-3.5 shadow-sm">
        <div className="flex min-w-0 items-center gap-3">
          <div className="grid h-10 w-10 shrink-0 place-items-center rounded-full bg-gradient-to-br from-brand-100 to-orange-100 text-sm font-black text-brand-600">
            {t.userId.slice(0, 1).toUpperCase()}
          </div>
          <div className="min-w-0">
            <div className="flex items-center gap-1.5 text-sm font-bold text-zinc-800">
              <span className="truncate">{t.userId}</span>
              {mine && (
                <span className="shrink-0 rounded-full bg-brand-50 px-1.5 py-0.5 text-[10px] font-semibold text-brand-600">
                  我的团
                </span>
              )}
            </div>
            {/* 成团进度：completeCount / targetCount */}
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
                仅剩 {remaining} 个名额
              </span>
              <span className="text-zinc-200">|</span>
              <CountDown endTime={t.validEndTime} initial={t.validTimeCountdown} />
            </div>
          </div>
        </div>
        <div className="flex shrink-0 items-center gap-2">
          {mine && t.outTradeNo && (
            <button
              onClick={() => void this.refundOwn(t)}
              className="rounded-full border border-zinc-200 px-3 py-1.5 text-xs text-zinc-400 transition hover:border-brand-200 hover:text-brand-600"
            >
              退单
            </button>
          )}
          <button
            onClick={() => this.startOrJoin(t.teamId)}
            className="rounded-full bg-gradient-to-r from-brand-500 to-orange-400 px-4 py-1.5 text-xs font-bold text-white shadow shadow-brand-500/30 transition active:scale-95"
          >
            参与拼团
          </button>
        </div>
      </div>
    );
  }

  /** 优惠（折扣）信息：后端返回 VO 则原样渲染；否则按试算价格推导等效表达式。 */
  private renderDiscount(d: MarketConfig) {
    const disc = d.groupBuyActivityDiscountVO?.groupBuyDiscount;
    if (disc && (disc.discountName || disc.discountDesc || disc.marketExpr)) {
      const plan = disc.marketPlan ?? '';
      return (
        <div className="mt-2.5 flex flex-wrap items-center gap-1.5">
          <span className="rounded-md bg-brand-50 px-2 py-1 text-[10px] font-bold text-brand-600">
            {plan ? `${plan} · ${MARKET_PLAN_LABELS[plan] ?? '优惠'}` : '优惠'}
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
      <div className="flex animate-pulse flex-col gap-4">
        <div className="aspect-square w-full rounded-3xl bg-zinc-200" />
        <div className="h-28 rounded-3xl bg-zinc-200" />
        <div className="h-16 rounded-2xl bg-zinc-200" />
        <div className="h-16 rounded-2xl bg-zinc-200" />
      </div>
    );
  }

  private renderError() {
    return (
      <div className="mt-16 flex flex-col items-center px-6 text-center">
        <div className="text-5xl">🫧</div>
        <p className="mt-4 text-sm font-semibold text-zinc-600">{this.errorMsg}</p>
        <button
          onClick={() => void this.load()}
          className="mt-5 rounded-full bg-gradient-to-r from-brand-500 to-orange-400 px-6 py-2.5 text-sm font-bold text-white shadow-lg shadow-brand-500/30 transition active:scale-95"
        >
          重新加载
        </button>
      </div>
    );
  }
}
