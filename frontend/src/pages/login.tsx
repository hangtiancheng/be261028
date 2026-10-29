import { createRef, customElement, state, type Ref } from '@yukino.js/lit-jsx';
import { animate, stagger } from 'motion';
import { genProductArt } from '../lib/art';
import { setUsername } from '../lib/auth';
import { navigate } from '../lib/router';
import { toast } from '../lib/toast';
import { UiElement } from '../components/base';

const DEMO_USERS = ['xfg01', 'xfg02', 'xfg03'];

/** 桌面端左侧宣传位特性列表。 */
const FEATURES = ['直降优惠 · 开团立享拼团价', '实时倒计时 · 成团进度一目了然', '极速退单 · 名额即时释放'];

/** 宣传缩略图：canvas 代码生成，不使用图片文件。 */
const HERO_THUMBS = [0, 1, 2].map((i) => genProductArt(`login-hero-${i}`));

/** 登录页：玻璃拟态卡片 + 渐变光斑动效；纯前端登录（写 cookie）。 */
@customElement('x-login-page')
export class LoginPage extends UiElement {
  @state() private username = '';
  @state() private password = '';
  @state() private busy = false;

  private cardRef: Ref<HTMLDivElement> = createRef();

  override connectedCallback(): void {
    super.connectedCallback();
    requestAnimationFrame(() => {
      const card = this.cardRef.value;
      if (card) {
        animate(
          card,
          { opacity: [0, 1], transform: ['translateY(28px) scale(.97)', 'translateY(0) scale(1)'] },
          { duration: 0.6, ease: [0.22, 1, 0.36, 1] },
        );
      }
      this.querySelectorAll<HTMLElement>('[data-blob]').forEach((el, i) => {
        animate(
          el,
          {
            transform: [
              'translate(0, 0) scale(1)',
              `translate(${i % 2 ? 34 : -30}px, ${i % 2 ? -26 : 22}px) scale(1.12)`,
            ],
          },
          { duration: 6.5 + i * 1.7, repeat: Infinity, repeatType: 'mirror', ease: 'easeInOut' },
        );
      });
      const hero = this.querySelector<HTMLElement>('[data-hero]');
      if (hero) {
        animate(hero, { opacity: [0, 1], transform: ['translateX(-24px)', 'translateX(0)'] }, {
          duration: 0.6,
          ease: [0.22, 1, 0.36, 1],
        });
      }
      const feats = this.querySelectorAll<HTMLElement>('[data-feature]');
      if (feats.length) {
        animate(feats, { opacity: [0, 1], transform: ['translateY(12px)', 'translateY(0)'] }, {
          duration: 0.45,
          delay: stagger(0.1, { startDelay: 0.25 }),
          ease: [0.22, 1, 0.36, 1],
        });
      }
    });
  }

  private submit(e: Event): void {
    e.preventDefault();
    if (this.busy) return;
    const name = this.username.trim();
    if (!name) {
      toast.error('请输入用户名');
      return;
    }
    if (!this.password) {
      toast.error('请输入密码');
      return;
    }
    this.busy = true;
    setUsername(name);
    window.setTimeout(() => {
      toast.success(`欢迎回来，${name}`);
      navigate('/', { replace: true });
    }, 380);
  }

  render() {
    return (
      <div className="relative min-h-dvh overflow-hidden bg-gradient-to-br from-brand-600 via-rose-500 to-orange-400">
        {/* 漂浮光斑 */}
        <div data-blob className="absolute -left-24 -top-24 h-80 w-80 rounded-full bg-white/25 blur-3xl" />
        <div data-blob className="absolute -right-16 top-1/3 h-72 w-72 rounded-full bg-orange-200/40 blur-3xl" />
        <div data-blob className="absolute -bottom-28 left-1/4 h-96 w-96 rounded-full bg-brand-300/30 blur-3xl" />

        <div className="relative z-10 mx-auto flex min-h-dvh w-full max-w-md flex-col justify-center px-6 lg:max-w-5xl lg:flex-row lg:items-center lg:gap-16 lg:px-8 xl:max-w-6xl xl:gap-24">
          {/* 桌面端左侧宣传位（移动端隐藏） */}
          <div data-hero className="hidden flex-1 lg:block">
            <div className="inline-flex items-center gap-2 rounded-full border border-white/25 bg-white/10 px-3.5 py-1.5 text-xs font-medium text-white/85 backdrop-blur">
              🛒 be261028 · 拼团营销演示
            </div>
            <h2 className="mt-6 text-4xl font-black leading-tight tracking-wide text-white xl:text-5xl">
              越拼越便宜
              <br />
              人满即成团
            </h2>
            <p className="mt-4 max-w-sm text-sm leading-relaxed text-white/70">
              锁单立减 · 到时自动成团，支持直减 / 满减 / N元购 / 折扣多种营销计划，交易结算与退单全链路模拟。
            </p>
            <ul className="mt-7 flex flex-col gap-3">
              {FEATURES.map((f) => (
                <li data-feature className="flex items-center gap-3 text-sm font-medium text-white/90">
                  <span className="grid h-5 w-5 shrink-0 place-items-center rounded-full bg-white/25 text-[10px] text-white">
                    ✓
                  </span>
                  {f}
                </li>
              ))}
            </ul>
            <div data-feature className="mt-9 flex gap-3">
              {HERO_THUMBS.map((src) => (
                <img
                  src={src}
                  alt="商品海报（代码生成）"
                  className="h-20 w-20 rounded-2xl border border-white/25 object-cover shadow-lg shadow-brand-900/25 xl:h-24 xl:w-24"
                />
              ))}
            </div>
          </div>

          {/* 登录卡片（移动端 / 桌面端通用） */}
          <div className="w-full lg:max-w-md lg:shrink-0">
          <div className="mb-8 text-center">
            <div className="mx-auto grid h-16 w-16 place-items-center rounded-3xl bg-white/20 text-3xl shadow-lg backdrop-blur">
              🛒
            </div>
            <h1 className="mt-4 text-2xl font-black tracking-wide text-white">be261028 拼团</h1>
            <p className="mt-1.5 text-sm text-white/70">越拼越便宜 · 人满即成团</p>
          </div>

          <div
            ref={this.cardRef}
            className="rounded-3xl border border-white/25 bg-white/15 p-7 shadow-2xl shadow-brand-900/20 backdrop-blur-xl"
          >
            <h2 className="mb-4 text-center text-base font-black text-white">欢迎登录</h2>
            <form onSubmit={(e) => this.submit(e)} className="flex flex-col gap-4">
              <label className="flex flex-col gap-1.5">
                <span className="text-xs font-medium text-white/80">用户名</span>
                <input
                  type="text"
                  value={this.username}
                  onInput={(e) => {
                    this.username = (e.target as HTMLInputElement).value;
                  }}
                  placeholder="输入任意用户名即可体验"
                  autoComplete="username"
                  className="w-full rounded-xl border border-white/25 bg-white/15 px-4 py-3 text-sm text-white outline-none transition placeholder:text-white/40 focus:border-white/70 focus:bg-white/25"
                />
              </label>
              <label className="flex flex-col gap-1.5">
                <span className="text-xs font-medium text-white/80">密码</span>
                <input
                  type="password"
                  value={this.password}
                  onInput={(e) => {
                    this.password = (e.target as HTMLInputElement).value;
                  }}
                  placeholder="演示环境，任意密码"
                  autoComplete="current-password"
                  className="w-full rounded-xl border border-white/25 bg-white/15 px-4 py-3 text-sm text-white outline-none transition placeholder:text-white/40 focus:border-white/70 focus:bg-white/25"
                />
              </label>
              <button
                type="submit"
                disabled={this.busy}
                className="mt-1 w-full rounded-xl bg-white py-3 text-sm font-bold text-brand-600 shadow-lg shadow-brand-900/20 transition active:scale-[.98] disabled:opacity-70"
              >
                {this.busy ? '登录中…' : '登 录'}
              </button>
            </form>

            <div className="mt-5 border-t border-white/15 pt-4">
              <p className="text-center text-[11px] text-white/55">演示账号（点击填充）</p>
              <div className="mt-2 flex justify-center gap-2">
                {DEMO_USERS.map((u) => (
                  <button
                    onClick={() => {
                      this.username = u;
                      this.password = 'demo';
                    }}
                    className="rounded-full border border-white/25 bg-white/10 px-3 py-1 text-xs text-white/85 transition hover:bg-white/25"
                  >
                    {u}
                  </button>
                ))}
              </div>
            </div>
          </div>

          <p className="mt-6 text-center text-[11px] text-white/45">
            纯前端登录（与旧版一致，仅写入本地 cookie）
          </p>
          </div>
        </div>
      </div>
    );
  }
}
