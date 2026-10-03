import { FormEvent, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { Brand } from "../components/Icon";
import { api, User } from "../api";

export function Login({ onLogin }: { onLogin: (user: User) => void }) {
  const navigate = useNavigate();
  const location = useLocation();
  const [email, setEmail] = useState("investigator@northwind.demo");
  const [password, setPassword] = useState("synthetic-demo-pass");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    try {
      const res = await api<{ user: User }>("/api/v1/auth/login", {
        method: "POST",
        body: JSON.stringify({ email, password }),
      });
      onLogin(res.user);
      const from = location.state?.from;
      navigate(
        typeof from === "string" && from.startsWith("/cases/")
          ? from
          : "/workspace",
        { replace: true },
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    } finally {
      setBusy(false);
    }
  }

  return (
    <main id="content" className="login-layout">
      <section className="login-story">
        <Link to="/" className="brand-link">
          <Brand />
        </Link>
        <div className="story-content">
          <h1>Northwind investigation desk</h1>
          <p>
            Sign in with a demo role. Cases use synthetic transactions. Nothing
            on this desk freezes an account or moves money.
          </p>
          <ul>
            <li>Investigator reads and drafts.</li>
            <li>Senior investigator can escalate and resolve.</li>
            <li>Viewer can read and cannot write.</li>
          </ul>
        </div>
      </section>
      <section className="login-panel">
        <span className="environment">
          <span /> Demo environment
        </span>
        <div className="login-form">
          <h2>Sign in</h2>
          <p className="muted">Use a Northwind demo account.</p>
          <form onSubmit={submit} className="grid">
            <div>
              <label htmlFor="email">Email</label>
              <input
                type="email"
                required
                id="email"
                autoComplete="username"
                value={email}
                onChange={(e) => setEmail(e.target.value)}
              />
            </div>
            <div>
              <label htmlFor="password">Password</label>
              <input
                required
                id="password"
                type="password"
                autoComplete="current-password"
                value={password}
                onChange={(e) => setPassword(e.target.value)}
              />
            </div>
            {error && (
              <p className="error" role="alert">
                {error}
              </p>
            )}
            <button type="submit" disabled={busy}>
              {busy ? "Signing in…" : "Sign in to workspace"}
            </button>
          </form>
          <div className="demo-accounts">
            <strong>Explore the demo</strong>
            <p>Choose a role to preview its workspace.</p>
            <div className="role-options">
              {["investigator", "senior", "viewer", "admin"].map((role) => (
                <button
                  type="button"
                  key={role}
                  aria-pressed={email === `${role}@northwind.demo`}
                  className={
                    email === `${role}@northwind.demo` ? "selected" : ""
                  }
                  onClick={() => {
                    setEmail(`${role}@northwind.demo`);
                    setPassword("synthetic-demo-pass");
                  }}
                >
                  {role === "senior" ? "Senior investigator" : role}
                </button>
              ))}
            </div>
            <p>
              Demo password: <code>synthetic-demo-pass</code>
            </p>
          </div>
          <p className="login-disclaimer">
            No accounts are frozen and no money is moved.
          </p>
        </div>
        <footer className="login-footer">Northwind desk · synthetic records</footer>
      </section>
    </main>
  );
}
