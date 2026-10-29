# be261028 前端

拼团营销系统前端，由旧版静态 HTML/JS/CSS 全面重建为**独立工程**（旧 `web/` 已删除，原件仍在 legacy 仓库 `docs/dev-ops/nginx/html/`）。

## 技术栈

| 技术 | 用途 |
| ---- | ---- |
| [Vite](https://vite.dev) 7 | 构建/开发服务器（`/api` 代理到网关 8091） |
| [Lit](https://lit.dev) 3 | Web Components 框架 |
| [@yukino.js/lit-jsx](https://www.npmjs.com/package/@yukino.js/lit-jsx) | 用 **JSX** 编写 Lit 组件（automatic runtime，`jsxImportSource: @yukino.js/lit-jsx`） |
| [TailwindCSS](https://tailwindcss.com) v4 | 原子化样式（`@tailwindcss/vite` 插件，**零手写 CSS**，仅 `styles.css` 中 `@import` + `@theme` 品牌令牌） |
| [motion](https://motion.dev) 14 | 动画（入场/列表 stagger/弹窗弹簧/轮播滑动/倒计时跳动/光斑漂浮） |

> 说明：npm 上的 `@yukino.js/lit-jsx@0.0.1` 未导出 `community` 入口（router/icon），
> 因此按技能文档的 hash 路由语义在 `src/lib/router.ts` 实现了最小路由；
> 图标使用内联 SVG 字符串 + lit 的 `unsafeHTML` 指令。

## 目录

```
src/
├── main.tsx            # 入口：createRoot 挂载 <x-app>
├── app.tsx             # 根组件：hash 路由分发 + 全局 toast 宿主
├── styles.css          # 仅 Tailwind @import 与 @theme 品牌色令牌
├── lib/
│   ├── api.ts          # 后端 API 客户端 + 错误码友好文案 + 折扣表达式推导
│   ├── art.ts          # canvas 代码生成视觉（商品海报/模拟二维码），全站零图片文件
│   ├── auth.ts         # cookie 登录态（与旧版一致，纯前端）
│   ├── router.ts       # 最小 hash 路由
│   ├── toast.ts        # 全局 toast store
│   └── utils.ts        # 随机单号/金额/时间/倒计时工具
├── components/
│   ├── base.ts         # UiElement 基类（light DOM，使 Tailwind 全局生效）
│   ├── carousel.tsx    # 商品轮播（motion 弹簧滑动 + 自动播放）
│   ├── countdown.tsx   # 实时倒计时（<60s 变红跳动）
│   └── toast-host.tsx  # toast 渲染宿主（motion 进出场）
└── pages/
    ├── login.tsx       # 登录页（玻璃拟态 + 漂浮光斑 + 演示账号 + 桌面端宣传位）
    ├── home.tsx        # 首页：商品详情/拼团队伍/支付弹窗/锁单→结算/退单/待支付恢复
    └── admin.tsx       # 管理后台：活动 CRUD/详情/启停 + DCC 动态配置面板
```

> **零图片资源（硬性要求）**：不使用任何图片文件；轮播海报与支付二维码
> 均由 `src/lib/art.ts` 在 canvas 上按 seed（goodsId/交易单号）确定性生成
> （fnv1a → xorShift32 → toDataURL），favicon 为 index.html 内联 data URI。

## 运行

```bash
pnpm install

# 开发（推荐）：http://127.0.0.1:5173 ，/api 自动代理到网关 8091
pnpm dev

# 生产构建 → dist/（由 Go 网关静态服务，webDir: frontend/dist）
pnpm build

# 本地预览构建产物
pnpm preview
```

生产模式下由 be261028 网关（`:8091`）直接服务 `dist/`，前端与 API 同源，
无需代理；修改 `configs/gateway.yaml` 的 `webDir` 后重启网关生效。

## 页面与流程

- **登录** `#/login`：任意用户名/密码（写 cookie `username`，1 天有效期），演示账号一键填充。
- **首页** `#/`：查询营销配置 → 轮播/价格/促销文案/拼团队伍（实时倒计时）；
  开团/参团 → 锁单 → 支付弹窗 → 结算；我的团支持退单；
  锁单后单号写入 localStorage，刷新页面出现**待支付恢复栏**可继续支付（旧版刷新即丢失）。
- **管理后台** `#/admin`：活动列表（状态徽标/启停/编辑/两步删除）、创建活动
  （折扣计划 ZJ/MJ/N/ZK + 表达式提示 + SKU）、DCC 动态配置（降级/缓存/切量/黑名单）。

## 与后端约定

- 统一响应 `{code, info, data}`；`0000`=成功（admin 为 `SUCCESS`/`ERROR`）。
- 金额为两位小数 number（后端 `Money` 类型），前端 `fmtPrice` 统一展示。
- 时间传 `YYYY-MM-DDTHH:mm:ss`（本地）或 ISO；结算 `outTradeTime` 用 `toISOString()`。
- 锁单回调地址 `notifyUrl` 指向 `${origin}/api/v1/test/group_buy_notify`（开发经代理、生产同源）；
  请求体用后端契约的 `notifyConfigVO: { notifyType: 'HTTP', notifyUrl }`。
- 首页营销查询按 userId 限流 1 req/s（对齐旧 `RateLimiterAccessInterceptor`）：
  命中 `0006` 时首页自动延后 1.1s 重试一次，不把限流渲染成错误页。
- 首页支持 `?goodsId=<id>` 覆盖默认商品 `9890001`；种子数据中 `9890002`
  对应双人团活动 `300001`，用于展示拼团队伍列表/参与拼团流程。
