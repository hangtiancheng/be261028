/**
 * Backend API client.
 * In development requests go through the Vite proxy; in production the Go
 * gateway serves the app and the API from the same origin — both use the
 * relative /api base. Every response shares the envelope { code, info, data };
 * code "0000" means success for gbm endpoints (admin endpoints use
 * SUCCESS/ERROR instead).
 */

import { toLocalInput } from "./utils";

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

/** Group-buy discount details — currently omitted by the index endpoint; rendered when present. */
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
  /** Optional field aligned with the legacy Java GoodsMarketResponseDTO; the gateway omits it today. */
  groupBuyActivityDiscountVO?: { groupBuyDiscount?: GroupBuyDiscount } | null;
}

/** Marketing plan code → English label (ZJ / MJ / fixed-price buy / ZK). */
export const MARKET_PLAN_LABELS: Record<string, string> = {
  ZJ: "Direct deduction",
  MJ: "Threshold discount",
  N: "Fixed-price buy",
  ZK: "Percentage off",
};

/**
 * Discount expression fallback renderer: the gateway index response carries no
 * discount details (the DTO omits them on purpose), so an equivalent ZJ / ZK
 * expression is derived from the trial prices (original / deduction / payable).
 */
export function deriveDiscountExpr(goods: Goods): {
  plan: string;
  expr: string;
  note: string;
} {
  const original = Number(goods.originalPrice) || 0;
  const pay = Number(goods.payPrice) || 0;
  const deduction = Math.max(0, original - pay);
  if (original > 0 && pay > 0 && pay < original) {
    const pct = Math.round((pay / original) * 100);
    return {
      plan: "ZJ",
      expr: `Direct deduction ¥${deduction.toFixed(2)}`,
      note: `You pay ${pct}% of the list price`,
    };
  }
  return {
    plan: "ZJ",
    expr: "Direct deduction ¥0.00",
    note: "No extra discount",
  };
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

async function request<T>(
  url: string,
  init?: RequestInit,
): Promise<ApiResponse<T>> {
  const resp = await fetch(url, {
    headers: { "Content-Type": "application/json" },
    ...init,
  });
  if (!resp.ok) {
    return {
      code: "HTTP_ERROR",
      info: `HTTP ${resp.status} ${resp.statusText}`,
    };
  }
  return (await resp.json()) as ApiResponse<T>;
}

const post = <T>(url: string, body: unknown) =>
  request<T>(url, { method: "POST", body: JSON.stringify(body) });

const get = <T>(url: string) => request<T>(url, { method: "GET" });

/** Business error code → user-friendly English message. */
export const ERROR_MESSAGES: Record<string, string> = {
  "0001": "Something went wrong on our side. Please try again later.",
  "0002": "Invalid or incomplete parameters. Please review and retry.",
  "0003": "Duplicate submission rejected.",
  "0004": "Nothing changed — no matching records were updated.",
  "0005": "Upstream service call failed. Please retry.",
  "0006": "Too many requests. Please wait a moment and try again.",
  E0001: "No discount plan is available right now.",
  E0002: "This item has no active group-buy campaign.",
  E0003: "Group-buy pricing is temporarily paused for maintenance.",
  E0004: "You are outside the rollout range for this campaign.",
  E0005: "Too late — this team just filled up.",
  E0006: "This team is full. Start a new one instead.",
  E0007: "This campaign is restricted to specific user groups.",
  E0008: "Team stock is exhausted. Please start a new team later.",
  E0101: "The campaign is not effective yet.",
  E0102: "Outside the campaign validity window.",
  E0103: "You have reached the participation limit for this campaign.",
  E0104: "Order not found, or it has already been refunded.",
  E0105: "This source/channel combination is blocked.",
  E0106: "Payment time is outside the team validity window.",
  E0107: "This order was already refunded.",
  UN_ERROR: "Unknown error. Please try again later.",
};

export function friendlyError(code: string, info: string): string {
  return ERROR_MESSAGES[code] ?? `${info || "Unknown error"} (${code})`;
}

export const api = {
  queryMarketConfig: (userId: string, goodsId: string) =>
    post<MarketConfig>("/api/v1/gbm/index/query_group_buy_market_config", {
      userId,
      source: "s01",
      channel: "c01",
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
    post<LockOrderResult>("/api/v1/gbm/trade/lock_market_pay_order", {
      userId: body.userId,
      teamId: body.teamId ?? "",
      activityId: body.activityId,
      goodsId: body.goodsId,
      source: "s01",
      channel: "c01",
      outTradeNo: body.outTradeNo,
      notifyConfigVO: { notifyType: "HTTP", notifyUrl: body.notifyUrl },
    }),

  settle: (userId: string, outTradeNo: string) =>
    post<SettlementResult>("/api/v1/gbm/trade/settlement_market_pay_order", {
      source: "s01",
      channel: "c01",
      userId,
      outTradeNo,
      // The backend expects naive local "YYYY-MM-DDTHH:mm:ss" (JSONTime),
      // which must fall before the team's validEndTime.
      outTradeTime: toLocalInput(new Date()),
    }),

  refund: (userId: string, outTradeNo: string) =>
    post<RefundResult>("/api/v1/gbm/trade/refund_market_pay_order", {
      userId,
      outTradeNo,
      source: "s01",
      channel: "c01",
    }),

  adminList: () => get<AdminActivity[]>("/api/v1/admin/activity/list"),
  adminDetail: (id: number) =>
    get<AdminActivity>(`/api/v1/admin/activity/detail/${id}`),
  adminCreate: (body: ActivityUpsertBody) =>
    post<AdminActivity>("/api/v1/admin/activity/create", body),
  adminUpdate: (body: ActivityUpsertBody) =>
    post<AdminActivity>("/api/v1/admin/activity/update", body),
  adminDelete: (id: number) =>
    post<AdminActivity>(`/api/v1/admin/activity/delete/${id}`, {}),
  adminEnable: (id: number) =>
    post<AdminActivity>(`/api/v1/admin/activity/enable/${id}`, {}),
  adminDisable: (id: number) =>
    post<AdminActivity>(`/api/v1/admin/activity/disable/${id}`, {}),

  dccUpdate: (key: string, value: string) =>
    get<boolean>(
      `/api/v1/gbm/dcc/update_config?key=${encodeURIComponent(key)}&value=${encodeURIComponent(value)}`,
    ),
};
