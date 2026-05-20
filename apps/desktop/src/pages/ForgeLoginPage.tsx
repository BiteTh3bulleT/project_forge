import { FormEvent, useMemo, useState } from "react";

type ForgeLoginPageProps = {
  onUnlock: () => void;
  mode?: "login" | "lock";
};

const LOGIN_USER = (import.meta.env.VITE_FORGE_LOGIN_USER || "operator").trim();
const LOGIN_PASSWORD =
  import.meta.env.VITE_FORGE_LOGIN_PASSWORD || "forge";

export function ForgeLoginPage({
  mode = "login",
  onUnlock,
}: ForgeLoginPageProps) {
  const [user, setUser] = useState(LOGIN_USER);
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
  const locked = mode === "lock";
  const currentTime = useMemo(
    () =>
      new Intl.DateTimeFormat(undefined, {
        hour: "numeric",
        minute: "2-digit",
      }).format(new Date()),
    [],
  );

  const handleSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    const normalizedUser = user.trim().toLowerCase();
    if (
      normalizedUser === LOGIN_USER.toLowerCase() &&
      password === LOGIN_PASSWORD
    ) {
      setError("");
      onUnlock();
      return;
    }
    setError("Invalid local operator login.");
    setPassword("");
  };

  return (
    <section
      className="forge-login-screen"
      aria-label={locked ? "FORGE lock screen" : "FORGE login"}
    >
      <div className="forge-login-screen__brand">
        <img
          className="forge-login-screen__mark"
          src="/brand/forge-start-button.png"
          alt=""
          draggable={false}
        />
        <div>
          <div className="forge-login-screen__product">
            {locked ? "Session Locked" : "FORGE-OS"}
          </div>
          <div className="forge-login-screen__subtitle">
            {locked ? "Operator Re-auth Required" : "Operator Runtime"}
          </div>
        </div>
      </div>

      <form className="forge-login-panel" onSubmit={handleSubmit}>
        <div className="forge-login-panel__time">{currentTime}</div>
        <label className="forge-login-field">
          <span>Operator</span>
          <input
            autoComplete="username"
            autoFocus
            value={user}
            onChange={(event) => setUser(event.target.value)}
          />
        </label>
        <label className="forge-login-field">
          <span>Password</span>
          <input
            autoComplete="current-password"
            type="password"
            value={password}
            onChange={(event) => setPassword(event.target.value)}
          />
        </label>
        {error ? (
          <div
            className="forge-login-panel__error"
            role="alert"
            aria-live="assertive"
          >
            {error}
          </div>
        ) : null}
        <button className="forge-login-panel__submit" type="submit">
          {locked ? "Unlock" : "Sign in"}
        </button>
      </form>
    </section>
  );
}
