import { create } from "zustand";
import type { User } from "@/lib/api/types";

/**
 * Who is signed in, plus the role checks the desk reads on every render.
 *
 * React Query owns fetching the principal; this store holds the resolved
 * value so components can read it synchronously without threading props.
 * The permission helpers mirror auth.Can in internal/auth/auth.go. The
 * backend is still the authority: these only decide what to show.
 */
type SessionState = {
  user: User | null;
  setUser: (user: User | null) => void;
  can: (action: Permission) => boolean;
};

export type Permission =
  | "read"
  | "ingest"
  | "note"
  | "review"
  | "report"
  | "ask"
  | "escalate"
  | "resolve"
  | "close"
  | "replay_job"
  | "retention"
  | "metrics";

const WRITE_ROLES = ["investigator", "senior_investigator", "admin"];
const SENIOR_ROLES = ["senior_investigator", "admin"];

export function can(role: string | undefined, action: Permission): boolean {
  if (!role) return false;
  switch (action) {
    case "read":
      return ["viewer", ...WRITE_ROLES].includes(role);
    case "ingest":
    case "note":
    case "review":
    case "report":
    case "ask":
      return WRITE_ROLES.includes(role);
    case "escalate":
    case "resolve":
    case "close":
    case "replay_job":
      return SENIOR_ROLES.includes(role);
    case "retention":
    case "metrics":
      return role === "admin";
    default:
      return false;
  }
}

export const useSessionStore = create<SessionState>((set, get) => ({
  user: null,
  setUser: (user) => set({ user }),
  can: (action) => can(get().user?.role, action),
}));
