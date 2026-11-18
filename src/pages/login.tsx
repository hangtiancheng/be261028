import { createRef, customElement, state, type Ref } from "@yukino.js/lit-jsx";
import { unsafeHTML } from "lit/directives/unsafe-html.js";
import { animate, stagger } from "motion";
import { genProductArt } from "../lib/art";
import { setUsername } from "../lib/auth";
import { ICONS } from "../lib/icons";
import { icon } from "@yukino.js/lit-jsx/community";
import { navigate } from "@yukino.js/lit-jsx/community";
import { toast } from "../lib/toast";
import { UiElement } from "../components/base";

const DEMO_USERS = ["yukino01", "yukino02", "yukino03"];

const HERO_THUMBS = [0, 1, 2].map((i) => genProductArt(`login-hero-${i}`));

/** Login page: glass card + floating gradient blobs. */
@customElement("x-login-page")
export class LoginPage extends UiElement {
  @state() private username = "";
  @state() private password = "";
  @state() private busy = false;

  private cardRef: Ref<HTMLDivElement> = createRef();

  override connectedCallback(): void {
    super.connectedCallback();
    requestAnimationFrame(() => {
      const card = this.cardRef.value;
      if (card) {
        animate(
          card,
          {
            opacity: [0, 1],
            transform: [
              "translateY(28px) scale(.97)",
              "translateY(0) scale(1)",
            ],
          },
          { duration: 0.6, ease: [0.22, 1, 0.36, 1] },
        );
      }
      this.querySelectorAll<HTMLElement>("[data-blob]").forEach((el, i) => {
        animate(
          el,
          {
            transform: [
              "translate(0, 0) scale(1)",
              `translate(${i % 2 ? 34 : -30}px, ${i % 2 ? -26 : 22}px) scale(1.12)`,
            ],
          },
          {
            duration: 6.5 + i * 1.7,
            repeat: Infinity,
            repeatType: "mirror",
            ease: "easeInOut",
          },
        );
      });
      const hero = this.querySelector<HTMLElement>("[data-hero]");
      if (hero) {
        animate(
          hero,
          {
            opacity: [0, 1],
            transform: ["translateX(-24px)", "translateX(0)"],
          },
          { duration: 0.6, ease: [0.22, 1, 0.36, 1] },
        );
      }
      const items = this.querySelectorAll<HTMLElement>("[data-hero-item]");
      if (items.length) {
        animate(
          items,
          { opacity: [0, 1], transform: ["translateY(14px)", "translateY(0)"] },
          {
            duration: 0.5,
            delay: stagger(0.09, { startDelay: 0.2 }),
            ease: [0.22, 1, 0.36, 1],
          },
        );
      }
    });
  }

  private submit(e: Event): void {
    e.preventDefault();
    if (this.busy) return;
    const name = this.username.trim();
    if (!name) {
      toast.error("Please enter a username");
      return;
    }
    if (!this.password) {
      toast.error("Please enter a password");
      return;
    }
    this.busy = true;
    setUsername(name);
    window.setTimeout(() => {
      toast.success(`Welcome back, ${name}`);
      navigate("/", { replace: true });
    }, 380);
  }

  render() {
    const inputCls =
      "w-full rounded-xl border border-white/25 bg-white/10 py-3 pl-10 pr-4 text-sm text-white outline-none transition placeholder:text-white/35 focus:border-white/70 focus:bg-white/20 focus:ring-2 focus:ring-white/20";
    return (
      <div className="relative flex min-h-dvh items-center overflow-hidden bg-gradient-to-br from-brand-700 via-brand-500 to-orange-400">
        <div className="pointer-events-none absolute inset-0 opacity-[0.16] [background-image:radial-gradient(circle_at_1px_1px,white_1px,transparent_0)] [background-size:28px_28px]" />

        <div
          data-blob
          className="absolute -left-28 -top-28 h-96 w-96 rounded-full bg-white/25 blur-3xl"
        />
        <div
          data-blob
          className="absolute -right-20 top-1/4 h-80 w-80 rounded-full bg-orange-200/40 blur-3xl"
        />
        <div
          data-blob
          className="absolute -bottom-32 left-1/3 h-[28rem] w-[28rem] rounded-full bg-brand-300/30 blur-3xl"
        />

        <div className="relative z-10 mx-auto flex w-full max-w-md flex-col justify-center px-6 py-10 lg:max-w-6xl lg:flex-row lg:items-center lg:gap-24 lg:px-10">
          <div data-hero className="hidden flex-1 lg:block">
            <div
              data-hero-item
              className="inline-flex items-center gap-2 rounded-full border border-white/25 bg-white/10 px-3.5 py-1.5 text-xs font-semibold text-white/90 backdrop-blur"
            >
              {unsafeHTML(icon(ICONS.cart, "h-3.5 w-3.5"))}
              be261028 · Group Buy
            </div>

            <h2
              data-hero-item
              className="mt-7 text-[3.25rem] font-black leading-[1.05] tracking-tight text-white xl:text-6xl"
            >
              Team up.
              <br />
              Pay less.
            </h2>

            <div data-hero-item className="mt-10 flex items-center gap-4">
              {HERO_THUMBS.map((src, i) => (
                <img
                  key={src}
                  src={src}
                  alt=""
                  className={`h-24 w-24 rounded-2xl border border-white/30 object-cover shadow-2xl shadow-brand-900/25 xl:h-28 xl:w-28 ${
                    i === 1 ? "translate-y-3" : i === 2 ? "-translate-y-2" : ""
                  }`}
                />
              ))}
            </div>
          </div>

          <div className="w-full lg:w-[26rem] lg:shrink-0">
            <div
              ref={this.cardRef}
              className="rounded-[1.75rem] border border-white/25 bg-white/15 p-7 shadow-2xl shadow-brand-900/25 backdrop-blur-2xl"
            >
              <div className="mb-6 text-center">
                <div className="mx-auto grid h-14 w-14 place-items-center rounded-2xl border border-white/25 bg-white/20 text-white shadow-lg backdrop-blur">
                  {unsafeHTML(icon(ICONS.cart, "h-7 w-7"))}
                </div>
                <h1 className="mt-4 text-2xl font-black tracking-tight text-white">
                  be261028 Group Buy
                </h1>
              </div>

              <form
                onSubmit={(e) => this.submit(e)}
                className="flex flex-col gap-4"
              >
                <label className="flex flex-col gap-1.5">
                  <span className="text-xs font-semibold text-white/85">
                    Username
                  </span>
                  <span className="relative block">
                    <span className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-white/50">
                      {unsafeHTML(icon(ICONS.users, "h-4 w-4"))}
                    </span>
                    <input
                      type="text"
                      value={this.username}
                      onInput={(e) => {
                        this.username = (e.target as HTMLInputElement).value;
                      }}
                      placeholder="Username"
                      autoComplete="username"
                      className={inputCls}
                    />
                  </span>
                </label>
                <label className="flex flex-col gap-1.5">
                  <span className="text-xs font-semibold text-white/85">
                    Password
                  </span>
                  <span className="relative block">
                    <span className="pointer-events-none absolute left-3.5 top-1/2 -translate-y-1/2 text-white/50">
                      {unsafeHTML(icon(ICONS.shield, "h-4 w-4"))}
                    </span>
                    <input
                      type="password"
                      value={this.password}
                      onInput={(e) => {
                        this.password = (e.target as HTMLInputElement).value;
                      }}
                      placeholder="Password"
                      autoComplete="current-password"
                      className={inputCls}
                    />
                  </span>
                </label>
                <button
                  type="submit"
                  disabled={this.busy}
                  className="pressable mt-1 flex w-full items-center justify-center gap-2 rounded-xl bg-white py-3 text-sm font-black text-brand-600 shadow-lg shadow-brand-900/20 hover:bg-white/95 disabled:opacity-70"
                >
                  {this.busy ? "Signing in…" : "Sign in"}
                  {!this.busy &&
                    unsafeHTML(icon(ICONS.arrowLeft, "h-4 w-4 rotate-180"))}
                </button>
              </form>

              <div className="mt-6 border-t border-white/15 pt-5">
                <div className="flex flex-wrap justify-center gap-2">
                  {DEMO_USERS.map((u) => (
                    <button
                      key={u}
                      type="button"
                      onClick={() => {
                        this.username = u;
                        this.password = "demo";
                      }}
                      className="pressable rounded-full border border-white/25 bg-white/10 px-3.5 py-1.5 font-mono text-xs text-white/90 hover:bg-white/25"
                    >
                      {u}
                    </button>
                  ))}
                </div>
              </div>
            </div>
          </div>
        </div>
      </div>
    );
  }
}
