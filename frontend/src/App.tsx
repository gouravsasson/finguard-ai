import { ReactNode, useState } from "react";
import {
  Link,
  Navigate,
  NavLink,
  Route,
  Routes,
  useNavigate,
  useLocation,
} from "react-router-dom";
import { Brand, Icon } from "./components/Icon";
import { Landing } from "./pages/Landing";
import { Login } from "./pages/Login";
import { Dashboard } from "./pages/Dashboard";
import { CaseLayout } from "./pages/case/CaseLayout";
import { SignalsPage } from "./pages/case/SignalsPage";
import { EvidencePage } from "./pages/case/EvidencePage";
import { KnockoutsPage } from "./pages/case/KnockoutsPage";
import { TransactionsPage } from "./pages/case/TransactionsPage";
import { ReportPage } from "./pages/case/ReportPage";
import { GraphPage } from "./pages/case/GraphPage";
import { AskPage } from "./pages/case/AskPage";
import { DecisionPage } from "./pages/case/DecisionPage";
import { NotesPage } from "./pages/case/NotesPage";
import { AuditPage } from "./pages/case/AuditPage";
import { useLogout, useSession } from "@/lib/api/hooks";
import { useSessionStore } from "@/store/session";

export function App() {
  const session = useSession();
  const user = useSessionStore((s) => s.user) ?? session.data ?? null;
  const ready = !session.isPending;
  const navigate = useNavigate();
  const location = useLocation();
  const logout = useLogout();

  async function signOut() {
    await logout.mutateAsync();
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
      <Shell user={user} onLogout={signOut}>
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
              <Login />
            )
          }
        />
        <Route path="/" element={<Landing signedIn={!!user} />} />
        <Route path="/workspace" element={protect(<Dashboard />)} />
        <Route path="/cases/:id" element={protect(<CaseLayout />)}>
          <Route index element={<SignalsPage />} />
          <Route path="evidence" element={<EvidencePage />} />
          <Route path="knockouts" element={<KnockoutsPage />} />
          <Route path="transactions" element={<TransactionsPage />} />
          <Route path="report" element={<ReportPage />} />
          <Route path="graph" element={<GraphPage />} />
          <Route path="ask" element={<AskPage />} />
          <Route path="decision" element={<DecisionPage />} />
          <Route path="notes" element={<NotesPage />} />
          <Route path="audit" element={<AuditPage />} />
        </Route>
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

const caseSections: Record<string, string> = {
  evidence: "Evidence",
  knockouts: "Knockout desk",
  transactions: "Transactions",
  report: "Draft report",
  graph: "Observed graph",
  ask: "Ask",
  decision: "Decision",
  notes: "Notes",
  audit: "Audit",
};

function crumb(pathname: string) {
  if (!pathname.startsWith("/cases/")) return "Case queue";
  const section = pathname.split("/")[3];
  return section ? caseSections[section] || "Case" : "Signals";
}

function Shell({
  user,
  onLogout,
  children,
}: {
  user: { name: string; role: string };
  onLogout: () => Promise<void>;
  children: ReactNode;
}) {
  const location = useLocation();
  const [logoutError, setLogoutError] = useState("");
  const [signingOut, setSigningOut] = useState(false);
  return (
    <div className="shell desk">
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
            <strong>{crumb(location.pathname)}</strong>
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
