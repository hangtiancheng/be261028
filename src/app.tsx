import { customElement } from "@yukino.js/lit-jsx";
import { UiElement } from "./components/base";
import { ToastHost } from "./components/toast-host";

@customElement("x-app")
export class App extends UiElement {
  render() {
    return (
      <div className="font-sans text-zinc-900 antialiased">
        <route-view />
        <ToastHost />
      </div>
    );
  }
}
