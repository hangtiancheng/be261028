import { LitElement } from "@yukino.js/lit-jsx";

/**
 * Component base class: disables the Shadow DOM and renders into the light DOM
 * so global Tailwind utility classes apply to every component
 * (this project contains no hand-written CSS).
 */
export class UiElement extends LitElement {
  protected override createRenderRoot(): HTMLElement {
    return this;
  }
}
