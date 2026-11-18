import "./styles.css";
import { createRoot } from "@yukino.js/lit-jsx";
import { createHashRouter } from "@yukino.js/lit-jsx/community";
import { AdminPage } from "./pages/admin";
import { HomePage } from "./pages/home";
import { LoginPage } from "./pages/login";
import { App } from "./app";

createHashRouter([
  { path: "/login", element: () => <LoginPage /> },
  { path: "/admin", element: () => <AdminPage /> },
  { path: "/", element: () => <HomePage /> },
]);

createRoot(document.getElementById("app")!).render(<App />);
