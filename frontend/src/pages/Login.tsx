import { FormEvent, useEffect, useState } from "react";
import { Link, useLocation, useNavigate } from "react-router-dom";
import { useLogin } from "@/lib/api/hooks";
import "../landing.css";

export function Login() {
  const navigate = useNavigate();
  const location = useLocation();
  const login = useLogin();
  const [email, setEmail] = useState("investigator@northwind.demo");
  const [password, setPassword] = useState("synthetic-demo-pass");
  const [error, setError] = useState("");
  const busy = login.isPending;
  const [shot, setShot] = useState(0);
  const shots = [
    {
      src: "/images/workspace-preview.png",
      alt: "FinGuard case queue with priorities, review status, and account search",
      caption: "The case queue. Synthetic records only.",
    },
    {
      src: "/images/case-preview.png",
      alt: "FinGuard case investigation with source evidence, detected signals, and transaction details",
      caption: "The case file, with each signal next to its record.",
    },
    {
      src: "/images/relationships-preview.png",
      alt: "FinGuard observed relationship graph connecting accounts, transactions, merchants, devices, and locations",
      caption: "The observed graph. Recorded links, not conclusions.",
    },
  ];
  const frame = shots[shot];

  useEffect(() => {
    const motion = window.matchMedia("(prefers-reduced-motion: reduce)");
    if (motion.matches) return;
    const timer = window.setInterval(() => {
      setShot((index) => (index + 1) % shots.length);
    }, 4500);
    return () => window.clearInterval(timer);
  }, [shots.length]);

  async function submit(e: FormEvent) {
    e.preventDefault();
    setError("");
    try {
      await login.mutateAsync({ email, password });
      const from = (location.state as { from?: string } | null)?.from;
      navigate(
        typeof from === "string" && from.startsWith("/cases/")
          ? from
          : "/workspace",
        { replace: true },
      );
    } catch (err) {
      setError(err instanceof Error ? err.message : "Login failed");
    }
  }

  return (
    <main id="content" className="login-layout">
      <header className="site-header">
        <Link to="/" className="brand-link" aria-label="FinGuard home">
          <span className="brand">
            <span className="logo-mark" aria-hidden="true">
              <svg viewBox="0 0 24 24" width="22" height="22">
                <path
                  d="M12 3 4 6v6c0 5 8 9 8 9s8-4 8-9V6l-8-3Z M8.5 12l2.5 2.5 4.5-5"
                  fill="none"
                  stroke="currentColor"
                  strokeWidth="1.6"
                  strokeLinecap="round"
                  strokeLinejoin="round"
                />
              </svg>
            </span>
            FinGuard
          </span>
        </Link>
      </header>
      <div className="login-split">
        <section className="login-story">
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
          <figure className="login-shot">
            <img
              key={frame.src}
              src={frame.src}
              alt={frame.alt}
              width="1440"
              height="1000"
            />
            <figcaption>{frame.caption}</figcaption>
          </figure>
        </section>
        <section className="login-panel">
          <span className="environment">Demo environment</span>
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
      </div>
    </main>
  );
}
