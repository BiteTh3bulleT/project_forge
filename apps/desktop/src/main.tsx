import ReactDOM from "react-dom/client";
import { HashRouter } from "react-router-dom";
import App from "./App";
import "./index.css";

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
