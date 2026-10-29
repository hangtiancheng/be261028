/**
 * 后端 API 客户端。
 * 开发环境经 Vite 代理、生产环境由 Go 网关同源服务，均使用相对路径 /api。
 * 统一响应: { code, info, data }；code "0000"=成功（admin 接口为 SUCCESS/ERROR）。
 */

export interface ApiResponse<T = unknown> {
  code: string;
  info: string;
  data?: T;
}

export interface Goods {
  goodsId: string;
  activityId: string;
  originalPrice: number;
  deductionPrice: number;
  payPrice: number;
}

export interface Team {
  userId: string;
  teamId: string;
  activityId: number;
  targetCount: number;
  completeCount: number;
  lockCount: number;
  validStartTime: string | null;
  validEndTime: string | null;
  validTimeCountdown: string;
  outTradeNo: string;
}

export interface TeamStatistic {
  allTeamCount: number;
  allTeamCompleteCount: number;
  allTeamUserCount: number;
}

/** 拼团优惠（折扣）明细 —— index 接口当前未下发，存在时直接渲染。 */
export interface GroupBuyDiscount {
  discountName?: string;
  discountDesc?: string;
  discountType?: number;
  marketPlan?: string;
  marketExpr?: string;
}

export interface MarketConfig {
  goods: Goods;
  teamList: Team[] | null;
  teamStatistic: TeamStatistic;
  /** 与旧版 Java GoodsMarketResponseDTO 对齐的可选字段；网关当前省略。 */
  groupBuyActivityDiscountVO?: { groupBuyDiscount?: GroupBuyDiscount } | null;
}

/** 营销计划 code → 中文名（ZJ/MJ/N元购/ZK）。 */
export const MARKET_PLAN_LABELS: Record<string, string> = {
  ZJ: '直减',
  MJ: '满减',
  N: 'N元购',
  ZK: '折扣',
};

/**
 * 优惠表达式兜底渲染：网关 index 响应不含折扣明细（DTO 有意省略），
 * 由试算结果（原价/直降/到手价）反推出等效 ZJ / ZK 表达式。
 */
export function deriveDiscountExpr(goods: Goods): { plan: string; expr: string; note: string } {
  const original = Number(goods.originalPrice) || 0;
  const pay = Number(goods.payPrice) || 0;
  const deduction = Math.max(0, original - pay);
  if (original > 0 && pay > 0 && pay < original) {
    const zhe = ((pay / original) * 10).toFixed(1).replace(/\.0$/, '');
    return { plan: 'ZJ', expr: `直减 ${deduction.toFixed(2)}`, note: `等效 ${zhe} 折` };
  }
  return { plan: 'ZJ', expr: '直减 0.00', note: '无额外优惠' };
}

export interface LockOrderResult {
  orderId: string;
  originalPrice: number;
  deductionPrice: number;
  payPrice: number;
  tradeOrderStatus: number;
}

export interface SettlementResult {
  userId: string;
  teamId: string;
  activityId: number;
  outTradeNo: string;
}

export interface RefundResult {
  userId: string;
  orderId: string;
  teamId: string;
  code: string;
  info: string;
}

export interface AdminActivity {
  activityId: number;
  activityName: string;
  discountId: string;
  groupType: number;
  takeLimitCount: number;
  target: number;
  validTime: number;
  status: number;
  startTime: string | null;
  endTime: string | null;
  createTime: string | null;
  updateTime: string | null;
  success?: boolean;
  errorMessage?: string;
}

export interface ActivityUpsertBody {
  activityId: number;
  activityName: string;
  discountId: string;
  groupType: number;
  takeLimitCount: number;
  target: number;
  validTime: number;
  status: number;
  startTime: string;
  endTime: string;
  discountInfo?: {
    discountId: string;
    discountName: string;
    discountDesc: string;
    discountType: number;
    marketPlan: string;
    marketExpr: string;
    tagId?: string;
  };
  skuInfo?: {
    goodsId: string;
    goodsName: string;
    source: string;
    channel: string;
    originalPrice: number;
  };
}

async function request<T>(url: string, init?: RequestInit): Promise<ApiResponse<T>> {
  const resp = await fetch(url, {
    headers: { 'Content-Type': 'application/json' },
    ...init,
  });
  if (!resp.ok) {
    return { code: 'HTTP_ERROR', info: `HTTP ${resp.status} ${resp.statusText}` };
  }
  return (await resp.json()) as ApiResponse<T>;
}

const post = <T>(url: string, body: unknown) =>
  request<T>(url, { method: 'POST', body: JSON.stringify(body) });

const get = <T>(url: string) => request<T>(url, { method: 'GET' });

/** 业务错误码 → 友好提示 */
export const ERROR_MESSAGES: Record<string, string> = {
  '0001': '服务开小差了，请稍后再试',
  '0002': '参数不完整，请检查后重试',
  '0003': '重复提交被拦截',
  '0006': '操作太频繁啦，歇一秒再试',
  E0001: '优惠计划暂不可用',
  E0002: '该商品暂无拼团活动',
  E0003: '活动火爆维护中（已降级）',
  E0004: '你暂不在本次活动范围内',
  E0005: '手慢了，该团刚好满员',
  E0006: '该团已满员，开个新团吧',
  E0007: '本活动仅限指定人群参与',
  E0008: '该团库存不足，开个新团吧',
  E0101: '活动未生效',
  E0102: '不在活动有效期内',
  E0103: '你参与该活动的次数已达上限',
  E0104: '订单不存在或已退单',
  E0105: '该渠道已被拦截',
  E0106: '支付时间超出拼团有效期',
  E0107: '已退过单，请勿重复退单',
};

export function friendlyError(code: string, info: string): string {
  return ERROR_MESSAGES[code] ?? `${info || '未知错误'} (${code})`;
}

export const api = {
  queryMarketConfig: (userId: string, goodsId: string) =>
    post<MarketConfig>('/api/v1/gbm/index/query_group_buy_market_config', {
      userId,
      source: 's01',
      channel: 'c01',
      goodsId,
    }),

  lockOrder: (body: {
    userId: string;
    teamId?: string;
    activityId: number;
    goodsId: string;
    outTradeNo: string;
    notifyUrl: string;
  }) =>
    post<LockOrderResult>('/api/v1/gbm/trade/lock_market_pay_order', {
      userId: body.userId,
      teamId: body.teamId ?? '',
      activityId: body.activityId,
      goodsId: body.goodsId,
      source: 's01',
      channel: 'c01',
      outTradeNo: body.outTradeNo,
      notifyConfigVO: { notifyType: 'HTTP', notifyUrl: body.notifyUrl },
    }),

  settle: (userId: string, outTradeNo: string) =>
    post<SettlementResult>('/api/v1/gbm/trade/settlement_market_pay_order', {
      source: 's01',
      channel: 'c01',
      userId,
      outTradeNo,
      outTradeTime: new Date().toISOString(),
    }),

  refund: (userId: string, outTradeNo: string) =>
    post<RefundResult>('/api/v1/gbm/trade/refund_market_pay_order', {
      userId,
      outTradeNo,
      source: 's01',
      channel: 'c01',
    }),

  adminList: () => get<AdminActivity[]>('/api/v1/admin/activity/list'),
  adminDetail: (id: number) => get<AdminActivity>(`/api/v1/admin/activity/detail/${id}`),
  adminCreate: (body: ActivityUpsertBody) =>
    post<AdminActivity>('/api/v1/admin/activity/create', body),
  adminUpdate: (body: ActivityUpsertBody) =>
    post<AdminActivity>('/api/v1/admin/activity/update', body),
  adminDelete: (id: number) =>
    post<AdminActivity>(`/api/v1/admin/activity/delete/${id}`, {}),
  adminEnable: (id: number) => post<AdminActivity>(`/api/v1/admin/activity/enable/${id}`, {}),
  adminDisable: (id: number) => post<AdminActivity>(`/api/v1/admin/activity/disable/${id}`, {}),

  dccUpdate: (key: string, value: string) =>
    get<boolean>(
      `/api/v1/gbm/dcc/update_config?key=${encodeURIComponent(key)}&value=${encodeURIComponent(value)}`,
    ),
};
