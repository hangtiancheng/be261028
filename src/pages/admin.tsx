import { customElement, state } from "@yukino.js/lit-jsx";
import { unsafeHTML } from "lit/directives/unsafe-html.js";
import { animate, stagger } from "motion";
import { api, type ActivityUpsertBody, type AdminActivity } from "../lib/api";
import { ICONS, svgIcon } from "../lib/icons";
import { navigate } from "../lib/router";
import { toast } from "../lib/toast";
import {
  backendToDatetimeLocal,
  datetimeLocalToBackend,
  fmtDateTime,
} from "../lib/utils";
import { UiElement } from "../components/base";

type Tab = "activity" | "dcc";

type DccKey = "downgradeSwitch" | "cacheSwitch" | "cutRange" | "scBlacklist";

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
  // Discount (submitted on create only)
  discountName: string;
  discountDesc: string;
  discountType: string;
  marketPlan: string;
  marketExpr: string;
  tagId: string;
  // SKU (submitted on create only)
  goodsId: string;
  goodsName: string;
  skuSource: string;
  skuChannel: string;
  originalPrice: string;
}

const emptyForm = (): ActivityForm => ({
  activityId: "",
  activityName: "",
  discountId: "",
  groupType: "0",
  takeLimitCount: "1",
  target: "1",
  validTime: "15",
  status: "1",
  startTime: "",
  endTime: "",
  discountName: "",
  discountDesc: "",
  discountType: "0",
  marketPlan: "ZJ",
  marketExpr: "",
  tagId: "",
  goodsId: "",
  goodsName: "",
  skuSource: "s01",
  skuChannel: "c01",
  originalPrice: "",
});

const STATUS_META: Record<number, { label: string; cls: string }> = {
  0: { label: "Created", cls: "bg-zinc-100 text-zinc-500" },
  1: { label: "Effective", cls: "bg-emerald-50 text-emerald-600" },
  2: { label: "Expired", cls: "bg-amber-50 text-amber-600" },
  3: { label: "Discarded", cls: "bg-rose-50 text-rose-500" },
};

const GROUP_TYPE_LABELS: Record<number, string> = {
  0: "Auto-complete",
  1: "Target-based",
};

const EXPR_HINTS: Record<string, string> = {
  ZJ: "Deduction amount, e.g. 20 (¥20 off)",
  MJ: "Threshold,amount — e.g. 100,20 (¥20 off orders over ¥100)",
  N: "Final price, e.g. 80 (buy at ¥80)",
  ZK: "Discount rate, e.g. 0.8 (pay 80%)",
};

/** Admin console: activity management (list / create / edit / enable / disable / delete) + DCC dynamic-config panel. */
@customElement("x-admin-page")
export class AdminPage extends UiElement {
  @state() private tab: Tab = "activity";
  @state() private activities: AdminActivity[] = [];
  @state() private loading = true;
  @state() private formOpen = false;
  @state() private editing = false;
  @state() private form: ActivityForm = emptyForm();
  @state() private confirmDeleteId: number | null = null;
  @state() private saving = false;
  @state() private detail: AdminActivity | null = null;
  @state() private dccSaving: DccKey | null = null;

  private listAnimated = false;
  private animatedPanels = new WeakSet<Element>();

  // DCC has no read endpoint; these are the documented defaults, saving overwrites.
  @state() private dcc: Record<DccKey, string> = {
    downgradeSwitch: "0",
    cacheSwitch: "0",
    cutRange: "100",
    scBlacklist: "s02c02",
  };

  override connectedCallback(): void {
    super.connectedCallback();
    void this.loadActivities();
    requestAnimationFrame(() => {
      animate(
        this,
        { opacity: [0, 1], transform: ["translateY(10px)", "translateY(0)"] },
        {
          duration: 0.4,
          ease: [0.22, 1, 0.36, 1],
        },
      );
    });
  }

  private async loadActivities(): Promise<void> {
    this.loading = true;
    try {
      const res = await api.adminList();
      if (res.code === "SUCCESS" && Array.isArray(res.data)) {
        this.activities = res.data;
        this.listAnimated = false;
      } else {
        toast.error(res.info || "Failed to load the activity list");
      }
    } catch {
      toast.error("Network error — failed to load the activity list");
    }
    this.loading = false;
  }

  /** Activity details: GET /api/v1/admin/activity/detail/:id */
  private async loadDetail(id: number): Promise<void> {
    try {
      const res = await api.adminDetail(id);
      if (res.code === "SUCCESS" && res.data) {
        this.detail = res.data;
      } else {
        toast.error(res.info || "Failed to load the activity details");
      }
    } catch {
      toast.error("Network error — failed to load the activity details");
    }
  }

  protected override updated(): void {
    // List entrance (staggered) for both the mobile cards and the desktop rows.
    if (!this.listAnimated && !this.loading && this.activities.length) {
      this.listAnimated = true;
      const cards = this.querySelectorAll<HTMLElement>(
        ".admin-card, .admin-row",
      );
      if (cards.length) {
        animate(
          cards,
          { opacity: [0, 1], transform: ["translateY(16px)", "translateY(0)"] },
          { duration: 0.4, delay: stagger(0.06), ease: [0.22, 1, 0.36, 1] },
        );
      }
    }
    // Modal panel entrance (each freshly rendered panel animates once).
    this.querySelectorAll<HTMLElement>("[data-modal-panel]").forEach((el) => {
      if (this.animatedPanels.has(el)) return;
      this.animatedPanels.add(el);
      animate(
        el,
        {
          opacity: [0, 1],
          transform: ["translateY(36px) scale(.97)", "translateY(0) scale(1)"],
        },
        { type: "spring", stiffness: 260, damping: 26 },
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
      if (s.trim() === "" || !Number.isFinite(n)) {
        toast.error(`${label} must be a number`);
        return null;
      }
      return n;
    };
    const activityId = num(f.activityId, "Activity ID");
    const takeLimitCount = num(f.takeLimitCount, "Participation limit");
    const target = num(f.target, "Target size");
    const validTime = num(f.validTime, "Team window");
    if (
      activityId === null ||
      takeLimitCount === null ||
      target === null ||
      validTime === null
    )
      return;
    if (!f.activityName.trim())
      return void toast.error("Please enter an activity name");
    if (!f.discountId.trim())
      return void toast.error("Please enter the discount ID");
    if (!f.startTime || !f.endTime)
      return void toast.error("Please set the start and end times");
    if (
      datetimeLocalToBackend(f.startTime) > datetimeLocalToBackend(f.endTime)
    ) {
      return void toast.error("Start time must not be later than end time");
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
      // On create, the discount and SKU sections are submitted as well. Editing
      // only touches the activity fields: the backend updates rows by
      // existence, so omitting them avoids accidental changes to shared data.
      const originalPrice = num(f.originalPrice, "Goods price");
      if (originalPrice === null) return;
      if (originalPrice <= 0)
        return void toast.error("Goods price must be greater than 0");
      if (!f.discountName.trim() || !f.marketExpr.trim()) {
        return void toast.error(
          "Please fill in the discount name and expression",
        );
      }
      if (!f.goodsId.trim() || !f.goodsName.trim()) {
        return void toast.error("Please fill in the goods ID and name");
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
        source: f.skuSource.trim() || "s01",
        channel: f.skuChannel.trim() || "c01",
        originalPrice,
      };
    }

    this.saving = true;
    try {
      const res = this.editing
        ? await api.adminUpdate(body)
        : await api.adminCreate(body);
      if (res.code === "SUCCESS") {
        toast.success(this.editing ? "Activity updated" : "Activity created");
        this.formOpen = false;
        void this.loadActivities();
      } else {
        toast.error(res.data?.errorMessage || res.info || "Operation failed");
      }
    } catch {
      toast.error("Network error — operation failed");
    }
    this.saving = false;
  }

  private async toggleStatus(a: AdminActivity): Promise<void> {
    try {
      const res =
        a.status === 1
          ? await api.adminDisable(a.activityId)
          : await api.adminEnable(a.activityId);
      if (res.code === "SUCCESS") {
        toast.success(
          a.status === 1 ? "Activity disabled" : "Activity enabled",
        );
        void this.loadActivities();
      } else {
        toast.error(res.data?.errorMessage || res.info || "Operation failed");
      }
    } catch {
      toast.error("Network error — operation failed");
    }
  }

  /** First click arms the delete button; the armed state expires after 3 s. */
  private requestDelete(a: AdminActivity): void {
    this.confirmDeleteId = a.activityId;
    window.setTimeout(() => {
      if (this.confirmDeleteId === a.activityId) this.confirmDeleteId = null;
    }, 3000);
  }

  private async doDelete(id: number): Promise<void> {
    this.confirmDeleteId = null;
    try {
      const res = await api.adminDelete(id);
      if (res.code === "SUCCESS") {
        toast.success("Activity deleted");
        void this.loadActivities();
      } else {
        toast.error(res.data?.errorMessage || res.info || "Delete failed");
      }
    } catch {
      toast.error("Network error — delete failed");
    }
  }

  private async setDcc(key: DccKey): Promise<void> {
    const value = this.dcc[key].trim();
    if (
      (key === "downgradeSwitch" || key === "cacheSwitch") &&
      value !== "0" &&
      value !== "1"
    ) {
      toast.error(`"${key}" must be "0" or "1"`);
      return;
    }
    if (key === "cutRange") {
      const n = Number(value);
      if (!Number.isInteger(n) || n < 0 || n > 100) {
        toast.error('"cutRange" must be an integer between 0 and 100');
        return;
      }
    }
    this.dccSaving = key;
    try {
      const res = await api.dccUpdate(key, value);
      if (res.code === "0000") {
        toast.success(`Configuration "${key}" updated to "${value}"`);
      } else {
        toast.error(res.info || "Failed to update the configuration");
      }
    } catch {
      toast.error("Network error — failed to update the configuration");
    }
    this.dccSaving = null;
  }

  render() {
    return (
      <div className="min-h-dvh bg-zinc-100">
        <div className="mx-auto min-h-dvh w-full max-w-3xl bg-zinc-50 px-4 pb-16 shadow-2xl shadow-zinc-900/10 sm:px-6 lg:max-w-6xl lg:px-8 xl:max-w-7xl">
          {/* Top bar */}
          <header className="sticky top-0 z-20 -mx-4 border-b border-zinc-200/60 bg-zinc-50/85 px-4 py-3 backdrop-blur sm:-mx-6 sm:px-6 lg:-mx-8 lg:px-8">
            <div className="flex items-center gap-3">
              <button
                onClick={() => navigate("/")}
                aria-label="Back to the storefront"
                className="grid h-9 w-9 place-items-center rounded-xl border border-zinc-200 bg-white text-zinc-500 transition hover:border-brand-300 hover:text-brand-600"
              >
                {unsafeHTML(svgIcon(ICONS.arrowLeft, "h-4 w-4"))}
              </button>
              <div>
                <h1 className="text-base font-black text-zinc-900">
                  Admin console
                </h1>
                <p className="text-[11px] text-zinc-400">
                  be261028 · Activities & dynamic configuration
                </p>
              </div>
              <div className="ml-auto flex rounded-full border border-zinc-200 bg-white p-0.5">
                {(
                  [
                    ["activity", "Activities"],
                    ["dcc", "Dynamic config"],
                  ] as [Tab, string][]
                ).map(([key, label]) => (
                  <button
                    onClick={() => {
                      this.tab = key;
                    }}
                    className={`rounded-full px-3.5 py-1.5 text-xs font-semibold transition ${
                      this.tab === key
                        ? "bg-gradient-to-r from-brand-500 to-orange-400 text-white shadow"
                        : "text-zinc-500 hover:text-zinc-800"
                    }`}
                  >
                    {label}
                  </button>
                ))}
              </div>
            </div>
          </header>

          {this.tab === "activity" ? this.renderActivities() : this.renderDcc()}
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
      ["Activity ID", String(a.activityId)],
      ["Activity name", a.activityName],
      ["Discount ID", a.discountId],
      ["Group type", GROUP_TYPE_LABELS[a.groupType] ?? String(a.groupType)],
      ["Participation limit", `${a.takeLimitCount}×`],
      ["Target size", `${a.target} members`],
      ["Team window", `${a.validTime} minutes`],
      ["Starts", fmtDateTime(a.startTime)],
      ["Ends", fmtDateTime(a.endTime)],
      ["Created", fmtDateTime(a.createTime)],
      ["Updated", fmtDateTime(a.updateTime)],
    ];
    return (
      <div
        className="fixed inset-0 z-50 flex items-end justify-center bg-zinc-950/45 backdrop-blur-sm lg:items-center lg:p-6"
        onClick={() => {
          this.detail = null;
        }}
      >
        <div
          data-modal-panel
          className="max-h-[92dvh] w-full max-w-lg overflow-y-auto rounded-t-3xl bg-white p-5 shadow-2xl lg:rounded-3xl lg:p-7"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="mx-auto h-1 w-10 rounded-full bg-zinc-200 lg:hidden" />
          <div className="mt-3 flex items-center gap-2 lg:mt-0">
            <h3 className="truncate text-base font-black text-zinc-900">
              Activity details
            </h3>
            <span
              className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold ${st.cls}`}
            >
              {st.label}
            </span>
            <button
              onClick={() => {
                this.detail = null;
              }}
              className="ml-auto grid h-8 w-8 shrink-0 place-items-center rounded-full text-zinc-400 transition hover:bg-zinc-100 hover:text-zinc-600"
              aria-label="Close"
            >
              {unsafeHTML(svgIcon(ICONS.close, "h-4 w-4"))}
            </button>
          </div>
          <dl className="mt-4 grid grid-cols-1 gap-x-6 gap-y-0 sm:grid-cols-2">
            {rows.map(([label, value]) => (
              <div className="flex items-baseline justify-between gap-3 border-b border-zinc-100 py-2.5">
                <dt className="shrink-0 text-xs text-zinc-400">{label}</dt>
                <dd className="truncate text-right text-xs font-semibold text-zinc-700">
                  {value}
                </dd>
              </div>
            ))}
          </dl>
          {a.errorMessage && (
            <p className="mt-3 rounded-xl bg-rose-50 px-3 py-2 text-[11px] text-rose-500">
              {a.errorMessage}
            </p>
          )}
          <button
            onClick={() => {
              this.detail = null;
            }}
            className="mt-5 w-full rounded-xl border border-zinc-200 py-2.5 text-sm font-semibold text-zinc-500 transition active:scale-[.98]"
          >
            Close
          </button>
        </div>
      </div>
    );
  }

  private renderActivities() {
    return (
      <main className="mt-4">
        <div className="mb-3 flex items-center justify-between">
          <p className="text-xs text-zinc-400">
            {this.activities.length}{" "}
            {this.activities.length === 1 ? "activity" : "activities"}
          </p>
          <button
            onClick={() => this.openCreate()}
            className="flex items-center gap-1.5 rounded-full bg-gradient-to-r from-brand-500 to-orange-400 px-4 py-2 text-xs font-bold text-white shadow-lg shadow-brand-500/30 transition active:scale-95"
          >
            {unsafeHTML(svgIcon(ICONS.plus, "h-3.5 w-3.5 [stroke-width:2.6]"))}
            Create activity
          </button>
        </div>

        {this.loading ? (
          <div className="flex animate-pulse flex-col gap-3">
            <div className="h-40 rounded-3xl bg-zinc-200" />
            <div className="h-40 rounded-3xl bg-zinc-200" />
          </div>
        ) : this.activities.length === 0 ? (
          <div className="rounded-3xl border-2 border-dashed border-zinc-200 bg-white/60 px-6 py-12 text-center">
            <div className="mx-auto grid h-12 w-12 place-items-center rounded-2xl bg-zinc-100 text-zinc-400">
              {unsafeHTML(svgIcon(ICONS.tag, "h-6 w-6"))}
            </div>
            <p className="mt-3 text-sm font-semibold text-zinc-600">
              No activities yet
            </p>
            <p className="mt-1 text-xs text-zinc-400">
              Create the first group-buy activity to get started.
            </p>
          </div>
        ) : (
          <>
            {/* Desktop: wide table with every activity field */}
            <div className="hidden overflow-hidden rounded-3xl bg-white shadow-sm lg:block">
              <div className="overflow-x-auto">
                <table className="w-full min-w-[1100px] text-left text-xs">
                  <thead>
                    <tr className="border-b border-zinc-100 text-[10px] uppercase tracking-wider text-zinc-400">
                      <th className="px-4 py-3 font-bold">ID</th>
                      <th className="px-4 py-3 font-bold">Activity</th>
                      <th className="px-4 py-3 font-bold">Discount</th>
                      <th className="px-4 py-3 font-bold">Type</th>
                      <th className="px-4 py-3 font-bold">Limit</th>
                      <th className="px-4 py-3 font-bold">Target</th>
                      <th className="px-4 py-3 font-bold">Window</th>
                      <th className="px-4 py-3 font-bold">Status</th>
                      <th className="px-4 py-3 font-bold">Validity</th>
                      <th className="px-4 py-3 font-bold">Updated</th>
                      <th className="px-4 py-3 text-right font-bold">
                        Actions
                      </th>
                    </tr>
                  </thead>
                  <tbody>
                    {this.activities.map((a) => {
                      const st = STATUS_META[a.status] ?? STATUS_META[0];
                      return (
                        <tr
                          key={String(a.activityId)}
                          className="admin-row border-b border-zinc-50 align-middle transition last:border-0 hover:bg-zinc-50/70"
                        >
                          <td className="px-4 py-3 font-mono text-zinc-400">
                            #{a.activityId}
                          </td>
                          <td className="max-w-[220px] truncate px-4 py-3 font-bold text-zinc-800">
                            {a.activityName}
                          </td>
                          <td className="px-4 py-3 font-mono text-zinc-500">
                            {a.discountId}
                          </td>
                          <td className="px-4 py-3 text-zinc-500">
                            {GROUP_TYPE_LABELS[a.groupType] ?? a.groupType}
                          </td>
                          <td className="px-4 py-3 tabular-nums text-zinc-500">
                            {a.takeLimitCount}×
                          </td>
                          <td className="px-4 py-3 tabular-nums text-zinc-500">
                            {a.target}
                          </td>
                          <td className="px-4 py-3 tabular-nums text-zinc-500">
                            {a.validTime} min
                          </td>
                          <td className="px-4 py-3">
                            <span
                              className={`rounded-full px-2 py-0.5 text-[10px] font-bold ${st.cls}`}
                            >
                              {st.label}
                            </span>
                          </td>
                          <td className="px-4 py-3 tabular-nums text-zinc-500">
                            <div>{fmtDateTime(a.startTime)}</div>
                            <div className="text-zinc-300">
                              → {fmtDateTime(a.endTime)}
                            </div>
                          </td>
                          <td className="px-4 py-3 tabular-nums text-zinc-400">
                            {fmtDateTime(a.updateTime)}
                          </td>
                          <td className="px-4 py-3">
                            <div className="flex items-center justify-end gap-1.5">
                              {this.renderTableActions(a)}
                            </div>
                          </td>
                        </tr>
                      );
                    })}
                  </tbody>
                </table>
              </div>
            </div>

            {/* Mobile: stacked cards */}
            <div className="flex flex-col gap-3 lg:hidden">
              {this.activities.map((a) => {
                const st = STATUS_META[a.status] ?? STATUS_META[0];
                return (
                  <div
                    key={String(a.activityId)}
                    className="admin-card rounded-3xl bg-white p-4 shadow-sm"
                  >
                    <div className="flex items-center gap-2">
                      <h3 className="truncate text-sm font-black text-zinc-900">
                        {a.activityName}
                      </h3>
                      <span
                        className={`shrink-0 rounded-full px-2 py-0.5 text-[10px] font-bold ${st.cls}`}
                      >
                        {st.label}
                      </span>
                      <span className="ml-auto shrink-0 font-mono text-[11px] text-zinc-300">
                        #{a.activityId}
                      </span>
                    </div>
                    <div className="mt-2.5 grid grid-cols-2 gap-x-4 gap-y-1.5 text-[11px] text-zinc-500 sm:grid-cols-3">
                      <span>
                        Target <b className="text-zinc-700">{a.target}</b>{" "}
                        members
                      </span>
                      <span>
                        Limit{" "}
                        <b className="text-zinc-700">{a.takeLimitCount}</b>×
                      </span>
                      <span>
                        Window <b className="text-zinc-700">{a.validTime}</b>{" "}
                        min
                      </span>
                      <span>
                        Discount <b className="text-zinc-700">{a.discountId}</b>
                      </span>
                      <span>
                        Type{" "}
                        <b className="text-zinc-700">
                          {GROUP_TYPE_LABELS[a.groupType] ?? a.groupType}
                        </b>
                      </span>
                      <span className="col-span-2 sm:col-span-1">
                        {fmtDateTime(a.startTime)} ~ {fmtDateTime(a.endTime)}
                      </span>
                    </div>
                    <div className="mt-3 flex flex-wrap gap-2 border-t border-zinc-100 pt-3">
                      {this.renderCardActions(a)}
                    </div>
                  </div>
                );
              })}
            </div>
          </>
        )}
      </main>
    );
  }

  private renderTableActions(a: AdminActivity) {
    const iconBtn =
      "grid h-8 w-8 place-items-center rounded-full border border-zinc-200 bg-white text-zinc-500 transition hover:border-brand-300 hover:text-brand-600 active:scale-95";
    return (
      <>
        <button
          onClick={() => void this.toggleStatus(a)}
          title={a.status === 1 ? "Disable" : "Enable"}
          aria-label={a.status === 1 ? "Disable activity" : "Enable activity"}
          className={
            a.status === 1
              ? "grid h-8 w-8 place-items-center rounded-full border border-amber-200 bg-amber-50 text-amber-600 transition hover:bg-amber-100 active:scale-95"
              : "grid h-8 w-8 place-items-center rounded-full border border-emerald-200 bg-emerald-50 text-emerald-600 transition hover:bg-emerald-100 active:scale-95"
          }
        >
          {unsafeHTML(svgIcon(ICONS.bolt, "h-3.5 w-3.5"))}
        </button>
        <button
          onClick={() => void this.loadDetail(a.activityId)}
          title="Details"
          aria-label="View details"
          className={iconBtn}
        >
          {unsafeHTML(svgIcon(ICONS.info, "h-3.5 w-3.5"))}
        </button>
        <button
          onClick={() => this.openEdit(a)}
          title="Edit"
          aria-label="Edit activity"
          className={iconBtn}
        >
          {unsafeHTML(svgIcon(ICONS.pencil, "h-3.5 w-3.5"))}
        </button>
        {this.confirmDeleteId === a.activityId ? (
          <button
            onClick={() => void this.doDelete(a.activityId)}
            title="Confirm delete?"
            aria-label="Confirm delete"
            className="grid h-8 place-items-center gap-1 rounded-full bg-rose-500 px-3 text-[10px] font-bold text-white shadow shadow-rose-500/30 transition active:scale-95"
          >
            Confirm?
          </button>
        ) : (
          <button
            onClick={() => this.requestDelete(a)}
            title="Delete"
            aria-label="Delete activity"
            className="grid h-8 w-8 place-items-center rounded-full border border-rose-100 bg-rose-50/60 text-rose-400 transition hover:bg-rose-50 hover:text-rose-500 active:scale-95"
          >
            {unsafeHTML(svgIcon(ICONS.trash, "h-3.5 w-3.5"))}
          </button>
        )}
      </>
    );
  }

  private renderCardActions(a: AdminActivity) {
    const btn =
      "rounded-full border border-zinc-200 bg-white px-3 py-1.5 text-xs font-semibold text-zinc-600 transition hover:border-brand-300 hover:text-brand-600 active:scale-95";
    return (
      <>
        <button
          onClick={() => void this.toggleStatus(a)}
          className={`rounded-full px-3 py-1.5 text-xs font-semibold transition active:scale-95 ${
            a.status === 1
              ? "border border-amber-200 bg-amber-50 text-amber-600 hover:bg-amber-100"
              : "border border-emerald-200 bg-emerald-50 text-emerald-600 hover:bg-emerald-100"
          }`}
        >
          {a.status === 1 ? "Disable" : "Enable"}
        </button>
        <button
          onClick={() => void this.loadDetail(a.activityId)}
          className={btn}
        >
          Details
        </button>
        <button onClick={() => this.openEdit(a)} className={btn}>
          Edit
        </button>
        {this.confirmDeleteId === a.activityId ? (
          <button
            onClick={() => void this.doDelete(a.activityId)}
            className="rounded-full bg-rose-500 px-3 py-1.5 text-xs font-bold text-white shadow shadow-rose-500/30 transition active:scale-95"
          >
            Confirm delete?
          </button>
        ) : (
          <button
            onClick={() => this.requestDelete(a)}
            className="rounded-full border border-rose-100 bg-rose-50/60 px-3 py-1.5 text-xs font-semibold text-rose-400 transition hover:bg-rose-50 hover:text-rose-500 active:scale-95"
          >
            Delete
          </button>
        )}
      </>
    );
  }

  private renderDcc() {
    const d = this.dcc;
    const rows: {
      key: DccKey;
      title: string;
      desc: string;
      quick?: string[];
      mono?: boolean;
    }[] = [
      {
        key: "downgradeSwitch",
        title: "Downgrade switch",
        desc: 'When set to "1", every group-buy pricing request is intercepted (E0003).',
        quick: ["0", "1"],
      },
      {
        key: "cacheSwitch",
        title: "Cache switch",
        desc: '"0" = cache enabled; "1" = bypass the cache and read straight from the database.',
        quick: ["0", "1"],
      },
      {
        key: "cutRange",
        title: "Traffic cut range",
        desc: "Percentage of users allowed through, based on the userId hash (0–100).",
        quick: ["0", "50", "100"],
      },
      {
        key: "scBlacklist",
        title: "SC channel blacklist",
        desc: "Blocked source+channel combinations, comma-separated, e.g. s02c02,s03c03.",
        mono: true,
      },
    ];
    return (
      <main className="mt-4 flex flex-col gap-3">
        <p className="text-[11px] leading-relaxed text-zinc-400">
          Dynamic configuration is published over Redis pub/sub and applied live
          by the activity and trade services. The backend exposes no read
          endpoint — the values below are the documented defaults, and “Update”
          overwrites the current value.
        </p>

        {rows.map((r) => (
          <div className="rounded-3xl bg-white p-4 shadow-sm lg:p-5">
            <div className="flex flex-col gap-3 lg:flex-row lg:items-center">
              <div className="min-w-0 flex-1">
                <h3 className="text-sm font-bold text-zinc-800">
                  {r.title}
                  <span className="ml-2 font-mono text-[10px] font-medium text-zinc-300">
                    {r.key}
                  </span>
                </h3>
                <p className="mt-0.5 text-[11px] leading-relaxed text-zinc-400">
                  {r.desc}
                </p>
                {r.quick && (
                  <div className="mt-2 flex gap-1.5">
                    {r.quick.map((v) => (
                      <button
                        onClick={() => {
                          this.dcc = { ...this.dcc, [r.key]: v };
                        }}
                        className={`rounded-full px-2.5 py-0.5 font-mono text-[10px] font-semibold transition ${
                          d[r.key] === v
                            ? "bg-brand-500 text-white shadow shadow-brand-500/30"
                            : "bg-zinc-100 text-zinc-400 hover:bg-zinc-200"
                        }`}
                      >
                        {v}
                      </button>
                    ))}
                  </div>
                )}
              </div>
              <div className="flex shrink-0 items-center gap-2">
                <input
                  type="text"
                  value={d[r.key]}
                  onInput={(e) => {
                    this.dcc = {
                      ...this.dcc,
                      [r.key]: (e.target as HTMLInputElement).value,
                    };
                  }}
                  aria-label={`${r.key} value`}
                  className={`w-40 rounded-xl border border-zinc-200 bg-zinc-50 px-3 py-2 text-sm outline-none transition focus:border-brand-400 focus:bg-white ${
                    r.mono ? "font-mono" : ""
                  }`}
                />
                <button
                  onClick={() => void this.setDcc(r.key)}
                  disabled={this.dccSaving === r.key}
                  className="flex items-center gap-1.5 rounded-xl bg-gradient-to-r from-brand-500 to-orange-400 px-4 py-2 text-xs font-bold text-white shadow shadow-brand-500/30 transition active:scale-95 disabled:opacity-60"
                >
                  {unsafeHTML(svgIcon(ICONS.refresh, "h-3.5 w-3.5"))}
                  {this.dccSaving === r.key ? "Saving…" : "Update"}
                </button>
              </div>
            </div>
          </div>
        ))}
      </main>
    );
  }

  private renderFormModal() {
    const f = this.form;
    const inputCls =
      "w-full rounded-xl border border-zinc-200 bg-zinc-50 px-3 py-2 text-sm outline-none transition focus:border-brand-400 focus:bg-white";
    const labelCls = "flex flex-col gap-1 text-xs font-semibold text-zinc-500";
    return (
      <div
        className="fixed inset-0 z-40 flex items-end justify-center bg-zinc-950/45 backdrop-blur-sm lg:items-center lg:p-6"
        onClick={() => {
          this.formOpen = false;
        }}
      >
        <div
          data-modal-panel
          className="max-h-[92dvh] w-full max-w-lg overflow-y-auto rounded-t-3xl bg-white p-5 shadow-2xl lg:rounded-3xl lg:p-7 xl:max-w-2xl"
          onClick={(e) => e.stopPropagation()}
        >
          <div className="mx-auto h-1 w-10 rounded-full bg-zinc-200 lg:hidden" />
          <h3 className="mt-3 text-base font-black text-zinc-900 lg:mt-0">
            {this.editing
              ? `Edit activity #${f.activityId}`
              : "Create group-buy activity"}
          </h3>

          <div className="mt-4 grid grid-cols-2 gap-3">
            <label className={labelCls}>
              Activity ID *
              <input
                type="number"
                className={inputCls}
                value={f.activityId}
                disabled={this.editing}
                onInput={(e) =>
                  this.setField(
                    "activityId",
                    (e.target as HTMLInputElement).value,
                  )
                }
              />
            </label>
            <label className={labelCls}>
              Activity name *
              <input
                type="text"
                className={inputCls}
                value={f.activityName}
                onInput={(e) =>
                  this.setField(
                    "activityName",
                    (e.target as HTMLInputElement).value,
                  )
                }
              />
            </label>
            <label className={labelCls}>
              Discount ID *
              <input
                type="text"
                className={inputCls}
                value={f.discountId}
                onInput={(e) =>
                  this.setField(
                    "discountId",
                    (e.target as HTMLInputElement).value,
                  )
                }
              />
            </label>
            <label className={labelCls}>
              Group type
              <select
                className={inputCls}
                value={f.groupType}
                onChange={(e) =>
                  this.setField(
                    "groupType",
                    (e.target as HTMLSelectElement).value,
                  )
                }
              >
                <option value="0">Auto-complete</option>
                <option value="1">Target-based</option>
              </select>
            </label>
            <label className={labelCls}>
              Participation limit
              <input
                type="number"
                className={inputCls}
                value={f.takeLimitCount}
                onInput={(e) =>
                  this.setField(
                    "takeLimitCount",
                    (e.target as HTMLInputElement).value,
                  )
                }
              />
            </label>
            <label className={labelCls}>
              Target size (members)
              <input
                type="number"
                className={inputCls}
                value={f.target}
                onInput={(e) =>
                  this.setField("target", (e.target as HTMLInputElement).value)
                }
              />
            </label>
            <label className={labelCls}>
              Team window (minutes)
              <input
                type="number"
                className={inputCls}
                value={f.validTime}
                onInput={(e) =>
                  this.setField(
                    "validTime",
                    (e.target as HTMLInputElement).value,
                  )
                }
              />
            </label>
            <label className={labelCls}>
              Status
              <select
                className={inputCls}
                value={f.status}
                onChange={(e) =>
                  this.setField("status", (e.target as HTMLSelectElement).value)
                }
              >
                <option value="0">Created</option>
                <option value="1">Effective</option>
              </select>
            </label>
            <label className={labelCls}>
              Start time *
              <input
                type="datetime-local"
                className={inputCls}
                value={f.startTime}
                onInput={(e) =>
                  this.setField(
                    "startTime",
                    (e.target as HTMLInputElement).value,
                  )
                }
              />
            </label>
            <label className={labelCls}>
              End time *
              <input
                type="datetime-local"
                className={inputCls}
                value={f.endTime}
                onInput={(e) =>
                  this.setField("endTime", (e.target as HTMLInputElement).value)
                }
              />
            </label>
          </div>

          {!this.editing && (
            <>
              <h4 className="mt-5 text-xs font-black uppercase tracking-wider text-zinc-400">
                Discount configuration
              </h4>
              <div className="mt-2 grid grid-cols-2 gap-3">
                <label className={labelCls}>
                  Discount name *
                  <input
                    type="text"
                    className={inputCls}
                    value={f.discountName}
                    onInput={(e) =>
                      this.setField(
                        "discountName",
                        (e.target as HTMLInputElement).value,
                      )
                    }
                  />
                </label>
                <label className={labelCls}>
                  Discount type
                  <select
                    className={inputCls}
                    value={f.discountType}
                    onChange={(e) =>
                      this.setField(
                        "discountType",
                        (e.target as HTMLSelectElement).value,
                      )
                    }
                  >
                    <option value="0">Base discount</option>
                    <option value="1">Crowd tag</option>
                  </select>
                </label>
                <label className={labelCls}>
                  Marketing plan
                  <select
                    className={inputCls}
                    value={f.marketPlan}
                    onChange={(e) =>
                      this.setField(
                        "marketPlan",
                        (e.target as HTMLSelectElement).value,
                      )
                    }
                  >
                    <option value="ZJ">ZJ · Direct deduction</option>
                    <option value="MJ">MJ · Threshold discount</option>
                    <option value="N">N · Fixed-price buy</option>
                    <option value="ZK">ZK · Percentage off</option>
                  </select>
                </label>
                <label className={labelCls}>
                  Expression *
                  <input
                    type="text"
                    className={inputCls}
                    value={f.marketExpr}
                    placeholder={EXPR_HINTS[f.marketPlan]}
                    onInput={(e) =>
                      this.setField(
                        "marketExpr",
                        (e.target as HTMLInputElement).value,
                      )
                    }
                  />
                </label>
                <p className="col-span-2 -mt-1 text-[10px] text-zinc-400">
                  {EXPR_HINTS[f.marketPlan]}
                </p>
                <label className={labelCls}>
                  Description
                  <input
                    type="text"
                    className={inputCls}
                    value={f.discountDesc}
                    onInput={(e) =>
                      this.setField(
                        "discountDesc",
                        (e.target as HTMLInputElement).value,
                      )
                    }
                  />
                </label>
                <label className={labelCls}>
                  Crowd tag ID (optional)
                  <input
                    type="text"
                    className={inputCls}
                    value={f.tagId}
                    onInput={(e) =>
                      this.setField(
                        "tagId",
                        (e.target as HTMLInputElement).value,
                      )
                    }
                  />
                </label>
              </div>

              <h4 className="mt-5 text-xs font-black uppercase tracking-wider text-zinc-400">
                Goods (SKU)
              </h4>
              <div className="mt-2 grid grid-cols-2 gap-3">
                <label className={labelCls}>
                  Goods ID *
                  <input
                    type="text"
                    className={inputCls}
                    value={f.goodsId}
                    onInput={(e) =>
                      this.setField(
                        "goodsId",
                        (e.target as HTMLInputElement).value,
                      )
                    }
                  />
                </label>
                <label className={labelCls}>
                  Goods name *
                  <input
                    type="text"
                    className={inputCls}
                    value={f.goodsName}
                    onInput={(e) =>
                      this.setField(
                        "goodsName",
                        (e.target as HTMLInputElement).value,
                      )
                    }
                  />
                </label>
                <label className={labelCls}>
                  Source
                  <input
                    type="text"
                    className={inputCls}
                    value={f.skuSource}
                    onInput={(e) =>
                      this.setField(
                        "skuSource",
                        (e.target as HTMLInputElement).value,
                      )
                    }
                  />
                </label>
                <label className={labelCls}>
                  Channel
                  <input
                    type="text"
                    className={inputCls}
                    value={f.skuChannel}
                    onInput={(e) =>
                      this.setField(
                        "skuChannel",
                        (e.target as HTMLInputElement).value,
                      )
                    }
                  />
                </label>
                <label className={labelCls}>
                  Price *
                  <input
                    type="number"
                    step="0.01"
                    className={inputCls}
                    value={f.originalPrice}
                    onInput={(e) =>
                      this.setField(
                        "originalPrice",
                        (e.target as HTMLInputElement).value,
                      )
                    }
                  />
                </label>
              </div>
            </>
          )}

          {this.editing && (
            <p className="mt-4 rounded-xl bg-zinc-50 px-3 py-2 text-[11px] leading-relaxed text-zinc-400">
              Editing updates the activity fields only; the discount and goods
              configuration stay untouched (the backend updates rows by
              existence, which avoids accidental changes to shared data).
            </p>
          )}

          <div className="sticky bottom-0 mt-6 flex gap-3 bg-white pt-2 pb-1">
            <button
              onClick={() => {
                this.formOpen = false;
              }}
              className="flex-1 rounded-xl border border-zinc-200 py-2.5 text-sm font-semibold text-zinc-500 transition active:scale-[.98]"
            >
              Cancel
            </button>
            <button
              onClick={() => void this.submitForm()}
              disabled={this.saving}
              className="flex-1 rounded-xl bg-gradient-to-r from-brand-500 to-orange-400 py-2.5 text-sm font-black text-white shadow-lg shadow-brand-500/30 transition active:scale-[.98] disabled:opacity-60"
            >
              {this.saving
                ? "Saving…"
                : this.editing
                  ? "Save changes"
                  : "Create activity"}
            </button>
          </div>
        </div>
      </div>
    );
  }
}
