import { LitElement } from '@yukino.js/lit-jsx';

/**
 * 组件基类：关闭 Shadow DOM、渲染到 light DOM，
 * 使全局 Tailwind 工具类对所有组件生效（本项目不手写 CSS）。
 */
export class UiElement extends LitElement {
  protected override createRenderRoot(): HTMLElement {
    return this;
  }
}
