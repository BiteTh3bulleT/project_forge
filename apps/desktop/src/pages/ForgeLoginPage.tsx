import { FormEvent, useMemo, useState } from "react";

type ForgeLoginPageProps = {
  onUnlock: () => void;
};

const LOGIN_USER = (import.meta.env.VITE_FORGE_LOGIN_USER || "operator").trim();
const LOGIN_PASSWORD =
  import.meta.env.VITE_FORGE_LOGIN_PASSWORD || "forge";

export function ForgeLoginPage({ onUnlock }: ForgeLoginPageProps) {
  const [user, setUser] = useState(LOGIN_USER);
  const [password, setPassword] = useState("");
  const [error, setError] = useState("");
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
    <section className="forge-login-screen" aria-label="FORGE login">
      <div className="forge-login-screen__brand">
        <span className="forge-login-screen__mark" aria-hidden="true" />
        <div>
          <div className="forge-login-screen__product">FORGE-OS</div>
          <div className="forge-login-screen__subtitle">Operator Runtime</div>
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
        {error ? <div className="forge-login-panel__error">{error}</div> : null}
        <button className="forge-login-panel__submit" type="submit">
          Sign in
        </button>
      </form>
    </section>
  );
}
