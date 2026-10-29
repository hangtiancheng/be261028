import { customElement, state } from '@yukino.js/lit-jsx';
import { animate, stagger } from 'motion';
import { api, type ActivityUpsertBody, type AdminActivity } from '../lib/api';
import { navigate } from '../lib/router';
import { toast } from '../lib/toast';
import { backendToDatetimeLocal, datetimeLocalToBackend, fmtDateTime } from '../lib/utils';
import { UiElement } from '../components/base';

type Tab = 'activity' | 'dcc';

interface ActivityForm {
  activityId: string;
  activityName: string;
  discountId: string;
  groupType: string;
  takeLimitCount: string;
  target: string;
  validTime: string;
  status: string;
  startTime: string;
  endTime: string;
  // 折扣（仅创建时提交）
  discountName: string;
  discountDesc: string;
  discountType: string;
  marketPlan: string;
  marketExpr: string;
  tagId: string;
  // SKU（仅创建时提交）
  goodsId: string;
  goodsName: string;
  skuSource: string;
  skuChannel: string;
  originalPrice: string;
}

const emptyForm = (): ActivityForm => ({
  activityId: '',
  activityName: '',
  discountId: '',
  groupType: '0',
  takeLimitCount: '1',
  target: '1',
  validTime: '15',
  status: '1',
  startTime: '',
  endTime: '',
  discountName: '',
  discountDesc: '',
  discountType: '0',
  marketPlan: 'ZJ',
  marketExpr: '',
  tagId: '',
  goodsId: '',
  goodsName: '',
  skuSource: 's01',
  skuChannel: 'c01',
  originalPrice: '',
});

const STATUS_META: Record<number, { label: string; cls: string }> = {
  0: { label: '创建', cls: 'bg-zinc-100 text-zinc-500' },
  1: { label: '生效', cls: 'bg-emerald-50 text-emerald-600' },
  2: { label: '过期', cls: 'bg-amber-50 text-amber-600' },
  3: { label: '废弃', cls: 'bg-rose-50 text-rose-500' },
};

const EXPR_HINTS: Record<string, string> = {
  ZJ: '直减金额，如 20（减 20 元）',
  MJ: '门槛,减额，如 100,20（满 100 减 20）',
  N: '优惠后价格，如 80（80 元购）',
  ZK: '折扣率，如 0.8（8 折）',
};

/** 管理后台：活动管理（列表/创建/编辑/启停/删除）+ DCC 动态配置面板。 */
@customElement('x-admin-page')
export class AdminPage extends UiElement {
  @state() private tab: Tab = 'activity';
  @state() private activities: AdminActivity[] = [];
  @state() private loading = true;
  @state() private formOpen = false;
  @state() private editing = false;
  @state() private form: ActivityForm = emptyForm();
  @state() private confirmDeleteId: number | null = null;
  @state() private saving = false;
  @state() private detail: AdminActivity | null = null;

  private listAnimated = false;
  private animatedPanels = new WeakSet<Element>();

  // DCC（无读取接口，展示默认值，保存即覆盖）
  @state() private dcc = {
    downgradeSwitch: '0',
    cutRange: '100',
    scBlacklist: 's02c02',
    cacheSwitch: '0',
  };

  override connectedCallback(): void {
    super.connectedCallback();
    void this.loadActivities();
    requestAnimationFrame(() => {
      animate(this, { opacity: [0, 1], transform: ['translateY(10px)', 'translateY(0)'] }, {
        duration: 0.4,
        ease: [0.22, 1, 0.36, 1],
      });
    });
  }

  private async loadActivities(): Promise<void> {
    this.loading = true;
    try {
      const res = await api.adminList();
      if (res.code === 'SUCCESS' && Array.isArray(res.data)) {
        this.activities = res.data;
        this.listAnimated = false;
      } else {
        toast.error(res.info || '加载活动列表失败');
      }
    } catch {
      toast.error('网络异常，加载活动列表失败');
    }
    this.loading = false;
  }

  /** 活动详情：GET /api/v1/admin/activity/detail/:id */
  private async loadDetail(id: number): Promise<void> {
    try {
      const res = await api.adminDetail(id);
      if (res.code === 'SUCCESS' && res.data) {
        this.detail = res.data;
      } else {
        toast.error(res.info || '加载活动详情失败');
      }
    } catch {
      toast.error('网络异常，加载活动详情失败');
    }
  }

  protected override updated(): void {
    // 列表卡片进场（stagger）
    if (!this.listAnimated && !this.loading && this.activities.length) {
      this.listAnimated = true;
      const cards = this.querySelectorAll<HTMLElement>('.admin-card');
      if (cards.length) {
        animate(
          cards,
          { opacity: [0, 1], transform: ['translateY(16px)', 'translateY(0)'] },
          { duration: 0.4, delay: stagger(0.06), ease: [0.22, 1, 0.36, 1] },
        );
      }
    }
    // 弹窗面板进场（条件渲染出的新面板各触发一次）
    this.querySelectorAll<HTMLElement>('[data-modal-panel]').forEach((el) => {
      if (this.animatedPanels.has(el)) return;
      this.animatedPanels.add(el);
      animate(
        el,
        { opacity: [0, 1], transform: ['translateY(36px) scale(.97)', 'translateY(0) scale(1)'] },
        { type: 'spring', stiffness: 260, damping: 26 },
      );
    });
  }

  private setField(key: keyof ActivityForm, value: string): void {
    this.form = { ...this.form, [key]: value };
  }

  private openCreate(): void {
    this.form = emptyForm();
    this.editing = false;
    this.formOpen = true;
  }

  private openEdit(a: AdminActivity): void {
    this.form = {
      ...emptyForm(),
      activityId: String(a.activityId),
      activityName: a.activityName,
      discountId: a.discountId,
      groupType: String(a.groupType),
      takeLimitCount: String(a.takeLimitCount),
      target: String(a.target),
      validTime: String(a.validTime),
      status: String(a.status),
      startTime: backendToDatetimeLocal(a.startTime),
      endTime: backendToDatetimeLocal(a.endTime),
    };
    this.editing = true;
    this.formOpen = true;
  }

  private async submitForm(): Promise<void> {
    if (this.saving) return;
    const f = this.form;
    const num = (s: string, label: string): number | null => {
      const n = Number(s);
      if (s.trim() === '' || !Number.isFinite(n)) {
        toast.error(`${label}必须是数字`);
        return null;
      }
      return n;
    };
    const activityId = num(f.activityId, '活动ID');
    const takeLimitCount = num(f.takeLimitCount, '参与次数限制');
    const target = num(f.target, '拼团目标');
    const validTime = num(f.validTime, '拼团时长');
    if (activityId === null || takeLimitCount === null || target === null || validTime === null) return;
    if (!f.activityName.trim()) return void toast.error('请填写活动名称');
    if (!f.discountId.trim()) return void toast.error('请填写折扣ID');
    if (!f.startTime || !f.endTime) return void toast.error('请填写活动起止时间');
    if (datetimeLocalToBackend(f.startTime) > datetimeLocalToBackend(f.endTime)) {
      return void toast.error('开始时间不能晚于结束时间');
    }

    const body: ActivityUpsertBody = {
      activityId,
      activityName: f.activityName.trim(),
      discountId: f.discountId.trim(),
      groupType: Number(f.groupType),
      takeLimitCount,
      target,
      validTime,
      status: Number(f.status),
      startTime: datetimeLocalToBackend(f.startTime),
      endTime: datetimeLocalToBackend(f.endTime),
    };

    if (!this.editing) {
      // 创建时同时提交折扣与 SKU（后端要求相关行存在才可更新，故编辑只改活动字段）
      const originalPrice = num(f.originalPrice, '商品价格');
      if (originalPrice === null) return;
      if (originalPrice <= 0) return void toast.error('商品价格必须大于 0');
      if (!f.discountName.trim() || !f.marketExpr.trim()) {
        return void toast.error('请填写折扣名称与优惠表达式');
      }
      if (!f.goodsId.trim() || !f.goodsName.trim()) {
        return void toast.error('请填写商品ID与名称');
      }
      body.discountInfo = {
        discountId: f.discountId.trim(),
        discountName: f.discountName.trim(),
        discountDesc: f.discountDesc.trim() || f.discountName.trim(),
        discountType: Number(f.discountType),
        marketPlan: f.marketPlan,
        marketExpr: f.marketExpr.trim(),
        tagId: f.tagId.trim() || undefined,
      };
      body.skuInfo = {
        goodsId: f.goodsId.trim(),
        goodsName: f.goodsName.trim(),
        source: f.skuSource.trim() || 's01',
        channel: f.skuChannel.trim() || 'c01',
        originalPrice,
      };
    }

    this.saving = true;
    try {
      const res = this.editing ? await api.adminUpdate(body) : await api.adminCreate(body);
      if (res.code === 'SUCCESS') {
        toast.success(this.editing ? '活动已更新' : '活动已创建');
        this.formOpen = false;
        void this.loadActivities();
      } else {
        toast.error(res.data?.errorMessage || res.info || '操作失败');
      }
    } catch {
      toast.error('网络异常，操作失败');
    }
    this.saving = false;
  }

  private async toggleStatus(a: AdminActivity): Promise<void> {
    const res = a.status === 1 ? await api.adminDisable(a.activityId) : await api.adminEnable(a.activityId);
    if (res.code === 'SUCCESS') {
      toast.success(a.status === 1 ? '已禁用' : '已启用');
      void this.loadActivities();
    } else {
      toast.error(res.info || '操作失败');
    }
  }

  private async doDelete(id: number): Promise<void> {
    const res = await api.adminDelete(id);
    this.confirmDeleteId = null;
    if (res.code === 'SUCCESS') {
      toast.success('活动已删除');
      void this.loadActivities();
    } else {
      toast.error(res.info || '删除失败');
    }
  }

  private async setDcc(key: keyof AdminPage['dcc'], value: string): Promise<void> {
    this.dcc = { ...this.dcc, [key]: value };
    const res = await api.dccUpdate(key, value);
    if (res.code === '0000') toast.success(`配置 ${key} 已更新为 ${value}`);
    else toast.error(res.info || '配置更新失败');
  }

  render() {
    return (
      <div className="min-h-dvh bg-zinc-100">
        <div className="mx-auto min-h-dvh w-full max-w-3xl bg-zinc-50 px-4 pb-16 shadow-2xl shadow-zinc-900/10 sm:px-6 lg:max-w-6xl lg:px-8 xl:max-w-7xl">
          {/* 顶栏 */}
          <header className="sticky top-0 z-20 -mx-4 border-b border-zinc-200/60 bg-zinc-50/85 px-4 py-3 backdrop-blur sm:-mx-6 sm:px-6">
            <div className="flex items-center gap-3">
              <button
                onClick={() => navigate('/')}
                className="grid h-9 w-9 place-items-center rounded-xl border border-zinc-200 bg-white text-zinc-500 transition hover:border-brand-300 hover:text-brand-600"
              >
                ←
              </button>
              <div>
                <h1 className="text-base font-black text-zinc-900">管理后台</h1>
                <p className="text-[11px] text-zinc-400">be261028 · 活动与动态配置</p>
              </div>
              <div className="ml-auto flex rounded-full border border-zinc-200 bg-white p-0.5">
                {(
                  [
                    ['activity', '活动管理'],
                    ['dcc', '动态配置'],
                  ] as [Tab, string][]
                ).map(([key, label]) => (
                  <button
                    onClick={() => {
                      this.tab = key;
                    }}
                    className={`rounded-full px-3.5 py-1.5 text-xs font-semibold transition ${
                      this.tab === key
                        ? 'bg-gradient-to-r from-brand-500 to-orange-400 text-white shadow'
                        : 'text-zinc-500 hover:text-zinc-800'
                    }`}
                  >
                    {label}
                  </button>
                ))}
              </div>
            </div>
          </header>

          {this.tab === 'activity' ? this.renderActivities() : this.renderDcc()}
        </div>

        {this.formOpen && this.renderFormModal()}
        {this.detail && this.renderDetailModal()}
      </div>
    );
  }

  private renderDetailModal() {
    const a = this.detail!;
    const st = STATUS_META[a.status] ?? STATUS_META[0];
    const rows: [string, string][] = [
      ['活动ID', String(a.activityId)],
      ['活动名称', a.activityName],
      ['折扣ID', a.discountId],
      ['拼团方式', a.groupType === 0 ? '自动成团' : '达成目标拼团'],
      ['参与次数限制', `${a.takeLimitCount} 次`],
      ['拼团目标人数', `${a.target} 人`],
      ['拼团时长', `${a.validTime} 分钟`],
      ['开始时间', fmtDateTime(a.startTime)],
      ['结束时间', fmtDateTime(a.endTime)],
      ['创建时间', fmtDateTime(a.createTime)],
      ['更新时间', fmtDateTime(a.updateTime)],
    ];
    return (
      <div
        className="fixed inset-0 z-50 flex items-end justify-center bg-zinc-950/45 backdrop-blur-sm lg:items-center lg:p-6"
        onClick={() => { this.detail = null; }}
      >
        <div
          data-modal-panel
          className="max-h-[92dvh] w-full max-w-lg overflow-y-auto rounded-t-3xl bg-white p-5 shadow-2xl lg:rounded-3xl lg:p-7"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="mx-auto h-1 w-10 rounded-full bg-zinc-200 lg:hidden" />
          <div className="mt-3 flex items-center gap-2 lg:mt-0">
            <h3 className="truncate text-base font-black text-zinc-900">活动详情</h3>
            <span className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold ${st.cls}`}>{st.label}</span>
            <button
              onClick={() => { this.detail = null; }}
              className="ml-auto grid h-8 w-8 shrink-0 place-items-center rounded-full text-zinc-400 transition hover:bg-zinc-100 hover:text-zinc-600"
              aria-label="关闭"
            >
              ✕
            </button>
          </div>
          <dl className="mt-4 grid grid-cols-1 gap-x-6 gap-y-0 sm:grid-cols-2">
            {rows.map(([label, value]) => (
              <div className="flex items-baseline justify-between gap-3 border-b border-zinc-100 py-2.5">
                <dt className="shrink-0 text-xs text-zinc-400">{label}</dt>
                <dd className="truncate text-right text-xs font-semibold text-zinc-700">{value}</dd>
              </div>
            ))}
          </dl>
          {a.errorMessage && (
            <p className="mt-3 rounded-xl bg-rose-50 px-3 py-2 text-[11px] text-rose-500">{a.errorMessage}</p>
          )}
          <button
            onClick={() => { this.detail = null; }}
            className="mt-5 w-full rounded-xl border border-zinc-200 py-2.5 text-sm font-semibold text-zinc-500 transition active:scale-[.98]"
          >
            关闭
          </button>
        </div>
      </div>
    );
  }

  private renderActivities() {
    return (
      <main className="mt-4">
        <div className="mb-3 flex items-center justify-between">
          <p className="text-xs text-zinc-400">共 {this.activities.length} 个活动</p>
          <button
            onClick={() => this.openCreate()}
            className="rounded-full bg-gradient-to-r from-brand-500 to-orange-400 px-4 py-2 text-xs font-bold text-white shadow-lg shadow-brand-500/30 transition active:scale-95"
          >
            + 创建活动
          </button>
        </div>

        {this.loading ? (
          <div className="flex animate-pulse flex-col gap-3">
            <div className="h-32 rounded-3xl bg-zinc-200" />
            <div className="h-32 rounded-3xl bg-zinc-200" />
          </div>
        ) : (
          <div className="flex flex-col gap-3 lg:grid lg:grid-cols-2 lg:items-start xl:grid-cols-3">
            {this.activities.map((a) => {
              const st = STATUS_META[a.status] ?? STATUS_META[0];
              return (
                <div key={String(a.activityId)} className="admin-card rounded-3xl bg-white p-4 shadow-sm">
                  <div className="flex items-center gap-2">
                    <h3 className="truncate text-sm font-black text-zinc-900">{a.activityName}</h3>
                    <span className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold ${st.cls}`}>
                      {st.label}
                    </span>
                    <span className="ml-auto shrink-0 font-mono text-[11px] text-zinc-300">#{a.activityId}</span>
                  </div>
                  <div className="mt-2.5 grid grid-cols-2 gap-x-4 gap-y-1.5 text-[11px] text-zinc-500 sm:grid-cols-3">
                    <span>目标 <b className="text-zinc-700">{a.target}</b> 人</span>
                    <span>限参与 <b className="text-zinc-700">{a.takeLimitCount}</b> 次</span>
                    <span>时长 <b className="text-zinc-700">{a.validTime}</b> 分钟</span>
                    <span>折扣 <b className="text-zinc-700">{a.discountId}</b></span>
                    <span>方式 <b className="text-zinc-700">{a.groupType === 0 ? '自动成团' : '达成目标'}</b></span>
                    <span className="col-span-2 sm:col-span-1">
                      {fmtDateTime(a.startTime)} ~ {fmtDateTime(a.endTime)}
                    </span>
                  </div>
                  <div className="mt-3 flex flex-wrap gap-2 border-t border-zinc-100 pt-3">
                    <button
                      onClick={() => void this.toggleStatus(a)}
                      className={`rounded-full px-3 py-1.5 text-xs font-semibold transition active:scale-95 ${
                        a.status === 1
                          ? 'border border-amber-200 bg-amber-50 text-amber-600 hover:bg-amber-100'
                          : 'border border-emerald-200 bg-emerald-50 text-emerald-600 hover:bg-emerald-100'
                      }`}
                    >
                      {a.status === 1 ? '禁用' : '启用'}
                    </button>
                    <button
                      onClick={() => void this.loadDetail(a.activityId)}
                      className="rounded-full border border-zinc-200 bg-white px-3 py-1.5 text-xs font-semibold text-zinc-600 transition hover:border-brand-300 hover:text-brand-600 active:scale-95"
                    >
                      详情
                    </button>
                    <button
                      onClick={() => this.openEdit(a)}
                      className="rounded-full border border-zinc-200 bg-white px-3 py-1.5 text-xs font-semibold text-zinc-600 transition hover:border-brand-300 hover:text-brand-600 active:scale-95"
                    >
                      编辑
                    </button>
                    {this.confirmDeleteId === a.activityId ? (
                      <button
                        onClick={() => void this.doDelete(a.activityId)}
                        className="rounded-full bg-rose-500 px-3 py-1.5 text-xs font-bold text-white shadow shadow-rose-500/30 transition active:scale-95"
                      >
                        确认删除？
                      </button>
                    ) : (
                      <button
                        onClick={() => {
                          this.confirmDeleteId = a.activityId;
                          window.setTimeout(() => {
                            if (this.confirmDeleteId === a.activityId) this.confirmDeleteId = null;
                          }, 3000);
                        }}
                        className="rounded-full border border-rose-100 bg-rose-50/60 px-3 py-1.5 text-xs font-semibold text-rose-400 transition hover:bg-rose-50 hover:text-rose-500 active:scale-95"
                      >
                        删除
                      </button>
                    )}
                  </div>
                </div>
              );
            })}
          </div>
        )}
      </main>
    );
  }

  private renderDcc() {
    const d = this.dcc;
    return (
      <main className="mt-4 flex flex-col gap-3">
        <p className="text-[11px] leading-relaxed text-zinc-400">
          动态配置经 Redis 发布订阅实时下发到 activity / trade 服务。后端不提供读取接口，
          下方为默认值，保存即覆盖。
        </p>

        {/* 降级开关 */}
        <div className="flex items-center justify-between rounded-3xl bg-white p-4 shadow-sm">
          <div>
            <h3 className="text-sm font-bold text-zinc-800">降级开关</h3>
            <p className="mt-0.5 text-[11px] text-zinc-400">开启后拼团试算全部拦截（E0003）</p>
          </div>
          {this.renderToggle(d.downgradeSwitch === '1', (v) => void this.setDcc('downgradeSwitch', v ? '1' : '0'))}
        </div>

        {/* 缓存开关 */}
        <div className="flex items-center justify-between rounded-3xl bg-white p-4 shadow-sm">
          <div>
            <h3 className="text-sm font-bold text-zinc-800">缓存开关</h3>
            <p className="mt-0.5 text-[11px] text-zinc-400">0=开启缓存、1=降级直查数据库</p>
          </div>
          {this.renderToggle(d.cacheSwitch === '0', (v) => void this.setDcc('cacheSwitch', v ? '0' : '1'))}
        </div>

        {/* 切量 */}
        <div className="rounded-3xl bg-white p-4 shadow-sm">
          <div className="flex items-center justify-between">
            <div>
              <h3 className="text-sm font-bold text-zinc-800">流量切量</h3>
              <p className="mt-0.5 text-[11px] text-zinc-400">按用户哈希尾号放量（0-100）</p>
            </div>
            <span className="rounded-full bg-brand-50 px-3 py-1 font-mono text-sm font-black text-brand-600">
              {d.cutRange}%
            </span>
          </div>
          <input
            type="range"
            min="0"
            max="100"
            value={d.cutRange}
            onInput={(e) => {
              this.dcc = { ...this.dcc, cutRange: (e.target as HTMLInputElement).value };
            }}
            onChange={(e) => void this.setDcc('cutRange', (e.target as HTMLInputElement).value)}
            className="mt-3 w-full accent-brand-500"
          />
        </div>

        {/* 渠道黑名单 */}
        <div className="rounded-3xl bg-white p-4 shadow-sm">
          <h3 className="text-sm font-bold text-zinc-800">SC 渠道黑名单</h3>
          <p className="mt-0.5 text-[11px] text-zinc-400">source+channel 组合，逗号分隔，如 s02c02,s03c03</p>
          <div className="mt-3 flex gap-2">
            <input
              type="text"
              value={d.scBlacklist}
              onInput={(e) => {
                this.dcc = { ...this.dcc, scBlacklist: (e.target as HTMLInputElement).value };
              }}
              className="flex-1 rounded-xl border border-zinc-200 bg-zinc-50 px-3 py-2 text-sm outline-none transition focus:border-brand-400 focus:bg-white"
            />
            <button
              onClick={() => void this.setDcc('scBlacklist', d.scBlacklist)}
              className="rounded-xl bg-gradient-to-r from-brand-500 to-orange-400 px-4 py-2 text-xs font-bold text-white shadow shadow-brand-500/30 transition active:scale-95"
            >
              保存
            </button>
          </div>
        </div>
      </main>
    );
  }

  private renderToggle(checked: boolean, onChange: (v: boolean) => void) {
    return (
      <label className="relative inline-flex cursor-pointer items-center">
        <input
          type="checkbox"
          checked={checked}
          onChange={(e) => onChange((e.target as HTMLInputElement).checked)}
          className="peer sr-only"
        />
        <div className="h-6 w-11 rounded-full bg-zinc-200 transition-colors after:absolute after:left-0.5 after:top-0.5 after:h-5 after:w-5 after:rounded-full after:bg-white after:shadow after:transition-transform after:content-[''] peer-checked:bg-brand-500 peer-checked:after:translate-x-5" />
      </label>
    );
  }

  private renderFormModal() {
    const f = this.form;
    const inputCls =
      'w-full rounded-xl border border-zinc-200 bg-zinc-50 px-3 py-2 text-sm outline-none transition focus:border-brand-400 focus:bg-white';
    const labelCls = 'flex flex-col gap-1 text-xs font-semibold text-zinc-500';
    return (
      <div
        className="fixed inset-0 z-40 flex items-end justify-center bg-zinc-950/45 backdrop-blur-sm lg:items-center lg:p-6"
        onClick={() => { this.formOpen = false; }}
      >
        <div
          data-modal-panel
          className="max-h-[92dvh] w-full max-w-lg overflow-y-auto rounded-t-3xl bg-white p-5 shadow-2xl lg:rounded-3xl lg:p-7"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="mx-auto h-1 w-10 rounded-full bg-zinc-200" />
          <h3 className="mt-3 text-base font-black text-zinc-900">
            {this.editing ? `编辑活动 #${f.activityId}` : '创建拼团活动'}
          </h3>

          <div className="mt-4 grid grid-cols-2 gap-3">
            <label className={labelCls}>
              活动ID *
              <input type="number" className={inputCls} value={f.activityId} disabled={this.editing}
                onInput={(e) => this.setField('activityId', (e.target as HTMLInputElement).value)} />
            </label>
            <label className={labelCls}>
              活动名称 *
              <input type="text" className={inputCls} value={f.activityName}
                onInput={(e) => this.setField('activityName', (e.target as HTMLInputElement).value)} />
            </label>
            <label className={labelCls}>
              折扣ID *
              <input type="text" className={inputCls} value={f.discountId}
                onInput={(e) => this.setField('discountId', (e.target as HTMLInputElement).value)} />
            </label>
            <label className={labelCls}>
              拼团方式
              <select className={inputCls} value={f.groupType}
                onChange={(e) => this.setField('groupType', (e.target as HTMLSelectElement).value)}>
                <option value="0">自动成团</option>
                <option value="1">达成目标拼团</option>
              </select>
            </label>
            <label className={labelCls}>
              参与次数限制
              <input type="number" className={inputCls} value={f.takeLimitCount}
                onInput={(e) => this.setField('takeLimitCount', (e.target as HTMLInputElement).value)} />
            </label>
            <label className={labelCls}>
              拼团目标人数
              <input type="number" className={inputCls} value={f.target}
                onInput={(e) => this.setField('target', (e.target as HTMLInputElement).value)} />
            </label>
            <label className={labelCls}>
              拼团时长（分钟）
              <input type="number" className={inputCls} value={f.validTime}
                onInput={(e) => this.setField('validTime', (e.target as HTMLInputElement).value)} />
            </label>
            <label className={labelCls}>
              状态
              <select className={inputCls} value={f.status}
                onChange={(e) => this.setField('status', (e.target as HTMLSelectElement).value)}>
                <option value="0">创建</option>
                <option value="1">生效</option>
              </select>
            </label>
            <label className={labelCls}>
              开始时间 *
              <input type="datetime-local" className={inputCls} value={f.startTime}
                onInput={(e) => this.setField('startTime', (e.target as HTMLInputElement).value)} />
            </label>
            <label className={labelCls}>
              结束时间 *
              <input type="datetime-local" className={inputCls} value={f.endTime}
                onInput={(e) => this.setField('endTime', (e.target as HTMLInputElement).value)} />
            </label>
          </div>

          {!this.editing && (
            <>
              <h4 className="mt-5 text-xs font-black uppercase tracking-wider text-zinc-400">折扣配置</h4>
              <div className="mt-2 grid grid-cols-2 gap-3">
                <label className={labelCls}>
                  折扣名称 *
                  <input type="text" className={inputCls} value={f.discountName}
                    onInput={(e) => this.setField('discountName', (e.target as HTMLInputElement).value)} />
                </label>
                <label className={labelCls}>
                  折扣类型
                  <select className={inputCls} value={f.discountType}
                    onChange={(e) => this.setField('discountType', (e.target as HTMLSelectElement).value)}>
                    <option value="0">基础优惠</option>
                    <option value="1">人群标签</option>
                  </select>
                </label>
                <label className={labelCls}>
                  优惠计划
                  <select className={inputCls} value={f.marketPlan}
                    onChange={(e) => this.setField('marketPlan', (e.target as HTMLSelectElement).value)}>
                    <option value="ZJ">ZJ · 直减</option>
                    <option value="MJ">MJ · 满减</option>
                    <option value="N">N · N元购</option>
                    <option value="ZK">ZK · 折扣</option>
                  </select>
                </label>
                <label className={labelCls}>
                  优惠表达式 *
                  <input type="text" className={inputCls} value={f.marketExpr} placeholder={EXPR_HINTS[f.marketPlan]}
                    onInput={(e) => this.setField('marketExpr', (e.target as HTMLInputElement).value)} />
                </label>
                <p className="col-span-2 -mt-1 text-[10px] text-zinc-400">{EXPR_HINTS[f.marketPlan]}</p>
                <label className={labelCls}>
                  折扣描述
                  <input type="text" className={inputCls} value={f.discountDesc}
                    onInput={(e) => this.setField('discountDesc', (e.target as HTMLInputElement).value)} />
                </label>
                <label className={labelCls}>
                  人群标签ID（可选）
                  <input type="text" className={inputCls} value={f.tagId}
                    onInput={(e) => this.setField('tagId', (e.target as HTMLInputElement).value)} />
                </label>
              </div>

              <h4 className="mt-5 text-xs font-black uppercase tracking-wider text-zinc-400">商品（SKU）</h4>
              <div className="mt-2 grid grid-cols-2 gap-3">
                <label className={labelCls}>
                  商品ID *
                  <input type="text" className={inputCls} value={f.goodsId}
                    onInput={(e) => this.setField('goodsId', (e.target as HTMLInputElement).value)} />
                </label>
                <label className={labelCls}>
                  商品名称 *
                  <input type="text" className={inputCls} value={f.goodsName}
                    onInput={(e) => this.setField('goodsName', (e.target as HTMLInputElement).value)} />
                </label>
                <label className={labelCls}>
                  渠道 source
                  <input type="text" className={inputCls} value={f.skuSource}
                    onInput={(e) => this.setField('skuSource', (e.target as HTMLInputElement).value)} />
                </label>
                <label className={labelCls}>
                  来源 channel
                  <input type="text" className={inputCls} value={f.skuChannel}
                    onInput={(e) => this.setField('skuChannel', (e.target as HTMLInputElement).value)} />
                </label>
                <label className={labelCls}>
                  商品价格 *
                  <input type="number" step="0.01" className={inputCls} value={f.originalPrice}
                    onInput={(e) => this.setField('originalPrice', (e.target as HTMLInputElement).value)} />
                </label>
              </div>
            </>
          )}

          {this.editing && (
            <p className="mt-4 rounded-xl bg-zinc-50 px-3 py-2 text-[11px] leading-relaxed text-zinc-400">
              编辑仅更新活动字段；折扣与商品配置保持不变（后端按存在性更新，避免误改共享数据）。
            </p>
          )}

          <div className="sticky bottom-0 mt-6 flex gap-3 bg-white pt-2 pb-1">
            <button
              onClick={() => { this.formOpen = false; }}
              className="flex-1 rounded-xl border border-zinc-200 py-2.5 text-sm font-semibold text-zinc-500 transition active:scale-[.98]"
            >
              取消
            </button>
            <button
              onClick={() => void this.submitForm()}
              disabled={this.saving}
              className="flex-1 rounded-xl bg-gradient-to-r from-brand-500 to-orange-400 py-2.5 text-sm font-black text-white shadow-lg shadow-brand-500/30 transition active:scale-[.98] disabled:opacity-60"
            >
              {this.saving ? '保存中…' : this.editing ? '保存修改' : '创建活动'}
            </button>
          </div>
        </div>
      </div>
    );
  }
}
