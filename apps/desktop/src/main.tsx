import ReactDOM from "react-dom/client";
import { HashRouter } from "react-router-dom";
import App from "./App";
import "./index.css";
import "./styles/forge-base.css";
import "./styles/forge-shell.css";
import "./styles/forge-chat.css";
import "./styles/forge-ops.css";
import "./styles/forge-os-shell.css";
import "./styles/forge-os-start-menu.css";
import "./styles/forge-os-window-login.css";

/**
 * HashRouter keeps SPA routes working when the shell is served as static assets
 * (Tauri production `https://tauri.localhost`, file protocol, or any host without
 * a server rewrite to index.html). Paths look like `/#/chat` instead of `/chat`.
 */
ReactDOM.createRoot(document.getElementById("root")!).render(
  <HashRouter>
    <App />
  </HashRouter>,
);
