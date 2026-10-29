# be261028 Frontend

Frontend of the group-buy marketing system, rebuilt from the legacy static
HTML/JS/CSS pages into a **standalone project** (the old `web/` directory is
gone; the originals remain in the legacy repository under
`docs/dev-ops/nginx/html/`).

## Tech stack

| Technology | Purpose |
| ---------- | ------- |
| [Vite](https://vite.dev) 7 | Build tool / dev server (`/api` proxied to the gateway on 8091) |
| [Lit](https://lit.dev) 3 | Web Components framework |
| [@yukino.js/lit-jsx](https://www.npmjs.com/package/@yukino.js/lit-jsx) | Lit components written in **JSX** (automatic runtime, `jsxImportSource: @yukino.js/lit-jsx`) |
| [TailwindCSS](https://tailwindcss.com) v4 | Atomic styling via the `@tailwindcss/vite` plugin — **zero hand-written CSS**; `styles.css` contains only the `@import` plus `@theme` brand tokens |
| [motion](https://motion.dev) 14 | Animations (page entrance / list stagger / modal springs / carousel sliding / countdown pulse / floating blobs) |

> Note: `@yukino.js/lit-jsx@0.0.1` on npm does not export the `community`
> entry (router/icon), so the hash-router semantics from the skill docs are
> implemented directly in `src/lib/router.ts`, and icons are inline SVG strings
> rendered through Lit's `unsafeHTML` directive.

## Layout

```
src/
├── main.tsx            # Entry: createRoot mounts <x-app>
├── app.tsx             # Root: hash-router dispatch + global toast host
├── styles.css          # Tailwind @import + @theme brand color tokens only
├── lib/
│   ├── api.ts          # Backend API client + friendly error-code copy + discount expression derivation
│   ├── art.ts          # Canvas-generated visuals (product posters / avatars / simulated QR); zero image files
│   ├── auth.ts         # Cookie-based login state (front-end only, parity with legacy)
│   ├── icons.ts        # Inline SVG icon set + class-injection helper
│   ├── router.ts       # Minimal hash router
│   ├── toast.ts        # Global toast store
│   └── utils.ts        # Trade-number / money / time / countdown helpers
├── components/
│   ├── base.ts         # UiElement base class (light DOM so global Tailwind applies)
│   ├── carousel.tsx    # Product carousel (motion spring sliding + autoplay)
│   ├── countdown.tsx   # Live countdown (turns red and pulses under 60 s)
│   └── toast-host.tsx  # Toast renderer (motion enter/leave)
└── pages/
    ├── login.tsx       # Login (glassmorphism + floating blobs + demo accounts + desktop hero column)
    ├── home.tsx        # Home: goods detail / teams / payment modal / lock → settle / refund / pending-order resume
    └── admin.tsx       # Admin console: activity CRUD / details / enable-disable + DCC dynamic-config panel
```

> **Zero image assets (hard requirement)**: no image files are used anywhere.
> Carousel posters, avatars and the payment QR are generated deterministically
> on a canvas by `src/lib/art.ts` from a seed (goodsId / userId / trade number)
> via fnv1a → xorShift32 → toDataURL; the favicon is an inline SVG data URI in
> `index.html`.

## Running

```bash
pnpm install

# Development (recommended): http://127.0.0.1:5173, /api proxied to the gateway on 8091
pnpm dev

# Production build → dist/ (served statically by the Go gateway, webDir: frontend/dist)
pnpm build

# Preview the production build locally
pnpm preview
```

In production the be261028 gateway (`:8091`) serves `dist/` directly, so the
frontend and the API share one origin and no proxy is needed. After changing
`webDir` in `configs/gateway.yaml`, restart the gateway.

## Pages and flows

- **Login** `#/login`: any username/password (writes the `username` cookie for
  one day); demo accounts fill the form with one click.
- **Home** `#/`: query the market config → carousel / prices / promotion copy /
  teams with live countdowns; start or join a team → lock → payment modal →
  settle; refund is available for your own seat (leaders via the team card,
  joiners via the "paid seat" bar); the locked trade number is persisted in
  localStorage so a **pending-payment bar** offers resume/cancel after a
  refresh (the legacy page lost it).
- **Admin** `#/admin`: activity list (status badges, enable/disable, edit,
  two-step delete; wide table on desktop, cards on mobile), create activities
  (discount plans ZJ/MJ/N/ZK with expression hints + SKU), and DCC dynamic
  configuration (downgrade / cache / cut range / blacklist) with an input and
  an Update button per key.

## Backend conventions

- Uniform envelope `{code, info, data}`; `0000` = success for gbm endpoints
  (admin endpoints use `SUCCESS` / `ERROR`).
- Money values are two-decimal JSON numbers (the backend `Money` type); the
  frontend formats them through `fmtPrice`.
- Times are exchanged as `YYYY-MM-DDTHH:mm:ss` (naive local time, matching the
  backend `JSONTime` type) — settlement sends `outTradeTime` in that format.
- The lock-order callback `notifyUrl` points at
  `${origin}/api/v1/test/group_buy_notify` (proxied in dev, same-origin in
  production); the request body uses the contracted
  `notifyConfigVO: { notifyType: 'HTTP', notifyUrl }`.
- The market-config query is rate limited per user (1 req/s, mirroring the
  legacy `RateLimiterAccessInterceptor`): on code `0006` the home page backs
  off ~1.1 s and retries once instead of rendering the limiter as an error,
  and background polling never runs faster than every 3 s.
- The home page supports `?goodsId=<id>` to override the default goods
  `9890001`; in the seed data `9890002` maps to the two-person campaign
  `300001`, which exercises the team list / join flow.
- External trade numbers are generated per attempt as
  `OTN-<timestamp>-<random digits>`.
