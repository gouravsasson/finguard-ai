import { ReactNode, useEffect, useState } from "react";
import {
  Link,
  Navigate,
  NavLink,
  Route,
  Routes,
  useNavigate,
  useLocation,
} from "react-router-dom";
import { api, User } from "./api";
import { CaseWorkspace } from "./pages/CaseWorkspace";
import { Dashboard } from "./pages/Dashboard";
import { Brand, Icon } from "./components/Icon";
import { Landing } from "./pages/Landing";
import { Login } from "./pages/Login";

export function App() {
  const [user, setUser] = useState<User | null>(null);
  const [ready, setReady] = useState(false);
  const navigate = useNavigate();
  const location = useLocation();

  useEffect(() => {
    api<{ user: User }>("/api/v1/auth/me")
      .then((res) => setUser(res.user))
      .catch(() => setUser(null))
      .finally(() => setReady(true));
  }, []);

  async function logout() {
    await api("/api/v1/auth/logout", { method: "POST" });
    setUser(null);
    navigate("/login");
  }

  function protect(node: ReactNode) {
    if (!ready)
      return (
        <main id="content" className="main" role="status">
          Loading workspace…
        </main>
      );
    return user ? (
      <Shell user={user} onLogout={logout}>
        {node}
      </Shell>
    ) : (
      <Navigate to="/login" replace state={{ from: location.pathname }} />
    );
  }

  return (
    <>
      <a className="skip" href="#content">
        Skip to content
      </a>
      <Routes>
        <Route
          path="/login"
          element={
            !ready ? (
              <main id="content" className="main" role="status">
                Loading session…
              </main>
            ) : user ? (
              <Navigate to="/workspace" replace />
            ) : (
              <Login onLogin={setUser} />
            )
          }
        />
        <Route path="/" element={<Landing signedIn={!!user} />} />
        <Route path="/workspace" element={protect(<Dashboard />)} />
        <Route
          path="/cases/:id"
          element={protect(<CaseWorkspace role={user?.role || "viewer"} />)}
        />
        <Route
          path="*"
          element={protect(
            <p>
              That page is not on the desk.{" "}
              <Link to="/workspace">Return to the queue.</Link>
            </p>,
          )}
        />
      </Routes>
    </>
  );
}

function Shell({
  user,
  onLogout,
  children,
}: {
  user: User;
  onLogout: () => Promise<void>;
  children: ReactNode;
}) {
  const location = useLocation();
  const [logoutError, setLogoutError] = useState("");
  const [signingOut, setSigningOut] = useState(false);
  return (
    <div className="shell">
      <aside className="rail">
        <Link to="/" className="brand-link" aria-label="FinGuard home">
          <Brand />
        </Link>
        <div className="workspace-label">
          <span className="workspace-avatar">FG</span>
          <div>
            Investigation workspace<small>Synthetic demo environment</small>
          </div>
        </div>
        <div className="nav-label">WORKSPACE</div>
        <nav aria-label="Primary">
          <NavLink to="/workspace" end>
            <Icon name="grid" /> Case queue <span className="nav-dot" />
          </NavLink>
        </nav>
        <div className="rail-bottom">
          <div className="who">
            <span className="avatar">
              {user.name
                .split(" ")
                .map((n) => n[0])
                .slice(0, 2)
                .join("")}
            </span>
            <div>
              <strong>{user.name}</strong>
              <small>{user.role.replaceAll("_", " ")}</small>
            </div>
            <button
              className="logout"
              aria-label="Log out"
              disabled={signingOut}
              onClick={async () => {
                setSigningOut(true);
                try {
                  await onLogout();
                } catch {
                  setLogoutError("Could not log out. Please try again.");
                  setSigningOut(false);
                }
              }}
            >
              <Icon name="logout" size={18} />
            </button>
          </div>
          {logoutError && (
            <p className="error" role="alert">
              {logoutError}
            </p>
          )}
        </div>
      </aside>
      <div className="workspace">
        <header className="topbar">
          <div className="breadcrumb">
            Workspace <span>/</span>{" "}
            <strong>
              {location.pathname.startsWith("/cases/")
                ? "Case investigation"
                : "Case queue"}
            </strong>
          </div>
          <span className="environment">
            <span /> Demo environment
          </span>
        </header>
        <main id="content" className="main">
          {children}
        </main>
        <footer className="workspace-footer">
          <span>FinGuard · synthetic tenant</span>
          <span>No account freeze · no money movement</span>
        </footer>
      </div>
    </div>
  );
}
