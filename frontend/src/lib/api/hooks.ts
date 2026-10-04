import {
  useInfiniteQuery,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { ApiError } from "./client";
import { endpoints } from "./endpoints";
import { qk } from "./keys";
import type { CaseAction, IngestEvent, User } from "./types";
import { useSessionStore } from "@/store/session";

/* -------------------------------------------------------------------------- */
/* session                                                                    */
/* -------------------------------------------------------------------------- */

/**
 * Resolves the signed-in principal. A 401 is a valid answer ("nobody is
 * signed in"), so it resolves to null instead of retrying.
 */
export function useSession() {
  const setUser = useSessionStore((s) => s.setUser);
  return useQuery({
    queryKey: qk.session,
    queryFn: async () => {
      try {
        const { user } = await endpoints.me();
        setUser(user);
        return user;
      } catch (err) {
        if (err instanceof ApiError && err.isUnauthenticated) {
          setUser(null);
          return null;
        }
        throw err;
      }
    },
    retry: false,
    staleTime: 60_000,
  });
}

export function useLogin() {
  const client = useQueryClient();
  const setUser = useSessionStore((s) => s.setUser);
  return useMutation({
    mutationFn: ({ email, password }: { email: string; password: string }) =>
      endpoints.login(email, password),
    onSuccess: ({ user }) => {
      setUser(user);
      client.setQueryData<User | null>(qk.session, user);
      client.invalidateQueries({ queryKey: qk.dashboard });
    },
  });
}

export function useLogout() {
  const client = useQueryClient();
  const setUser = useSessionStore((s) => s.setUser);
  return useMutation({
    mutationFn: () => endpoints.logout(),
    onSuccess: () => {
      setUser(null);
      client.clear();
      client.setQueryData<User | null>(qk.session, null);
    },
  });
}

/* -------------------------------------------------------------------------- */
/* product context                                                            */
/* -------------------------------------------------------------------------- */

export function useMeta(enabled = true) {
  return useQuery({
    queryKey: qk.meta,
    queryFn: endpoints.meta,
    enabled,
    staleTime: 5 * 60_000,
  });
}

export function useDashboard() {
  return useQuery({ queryKey: qk.dashboard, queryFn: endpoints.dashboard });
}

/** Admin only. The backend answers 403 for every other role. */
export function useMetrics(enabled = true) {
  return useQuery({
    queryKey: qk.metrics,
    queryFn: endpoints.metrics,
    enabled,
    retry: false,
  });
}

export function useHealth(enabled = true) {
  return useQuery({
    queryKey: qk.health,
    queryFn: endpoints.health,
    enabled,
    retry: false,
  });
}

export function useReadiness(enabled = true) {
  return useQuery({
    queryKey: qk.ready,
    queryFn: endpoints.ready,
    enabled,
    retry: false,
  });
}

export function useTransactions(limit = 50, enabled = true) {
  return useQuery({
    queryKey: qk.transactions(limit),
    queryFn: () => endpoints.transactions(limit).then((r) => r.data),
    enabled,
  });
}

/* -------------------------------------------------------------------------- */
/* case queue                                                                 */
/* -------------------------------------------------------------------------- */

/** Cursor pagination, matching the backend's opaque next_cursor. */
export function useCaseQueue(filters: { status?: string; limit?: number } = {}) {
  return useInfiniteQuery({
    queryKey: qk.cases(filters),
    queryFn: ({ pageParam }) =>
      endpoints.cases({
        status: filters.status,
        limit: filters.limit ?? 25,
        cursor: pageParam || undefined,
      }),
    initialPageParam: "",
    getNextPageParam: (last) => last.page.next_cursor || undefined,
  });
}

/* -------------------------------------------------------------------------- */
/* one case                                                                   */
/* -------------------------------------------------------------------------- */

export function useCaseDetail(id: string | undefined) {
  return useQuery({
    queryKey: qk.caseDetail(id ?? ""),
    queryFn: () => endpoints.caseDetail(id!),
    enabled: !!id,
  });
}

export function useCaseEvidence(id: string | undefined) {
  return useQuery({
    queryKey: qk.caseEvidence(id ?? ""),
    queryFn: () => endpoints.evidence(id!).then((r) => r.data),
    enabled: !!id,
  });
}

export function useCaseReports(id: string | undefined) {
  return useQuery({
    queryKey: qk.caseReports(id ?? ""),
    queryFn: () => endpoints.reports(id!).then((r) => r.data),
    enabled: !!id,
  });
}

export function useCaseGraph(id: string | undefined) {
  return useQuery({
    queryKey: qk.caseGraph(id ?? ""),
    queryFn: () => endpoints.graph(id!),
    enabled: !!id,
  });
}

export function useCaseKnockouts(id: string | undefined) {
  return useQuery({
    queryKey: qk.caseKnockouts(id ?? ""),
    queryFn: () => endpoints.knockouts(id!),
    enabled: !!id,
  });
}

export function useCaseAudit(id: string | undefined) {
  return useQuery({
    queryKey: qk.caseAudit(id ?? ""),
    queryFn: () => endpoints.audit(id!).then((r) => r.data),
    enabled: !!id,
  });
}

export function useCaseAssistant(id: string | undefined) {
  return useQuery({
    queryKey: qk.caseAssistant(id ?? ""),
    queryFn: () => endpoints.assistant(id!).then((r) => r.data),
    enabled: !!id,
  });
}

/* -------------------------------------------------------------------------- */
/* case mutations                                                             */
/* -------------------------------------------------------------------------- */

function useCaseInvalidation(id: string) {
  const client = useQueryClient();
  return () => {
    client.invalidateQueries({ queryKey: qk.case(id) });
    client.invalidateQueries({ queryKey: ["cases"] });
    client.invalidateQueries({ queryKey: qk.dashboard });
  };
}

export function useAddNote(id: string) {
  const refresh = useCaseInvalidation(id);
  return useMutation({
    mutationFn: (body: string) => endpoints.addNote(id, body),
    onSuccess: refresh,
  });
}

/**
 * Case transitions carry the version the client last saw. A 409 means
 * someone else moved first, so the panels are refetched either way.
 */
export function useCaseAction(id: string) {
  const refresh = useCaseInvalidation(id);
  return useMutation({
    mutationFn: (action: CaseAction) => endpoints.act(id, action),
    onSettled: refresh,
  });
}

export function useCreateReport(id: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: () => endpoints.createReport(id),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: qk.caseReports(id) });
      client.invalidateQueries({ queryKey: qk.caseAudit(id) });
    },
  });
}

export function useDissent(id: string) {
  return useMutation({ mutationFn: () => endpoints.dissent(id) });
}

export function useAsk(id: string) {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (question: string) => endpoints.ask(id, question),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: qk.caseAssistant(id) });
    },
  });
}

/* -------------------------------------------------------------------------- */
/* operations                                                                 */
/* -------------------------------------------------------------------------- */

export function useIngestEvent() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (event: IngestEvent) => endpoints.ingestEvent(event),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: qk.dashboard });
      client.invalidateQueries({ queryKey: ["cases"] });
      client.invalidateQueries({ queryKey: ["transactions"] });
    },
  });
}

export function useIngestBatch() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (events: IngestEvent[]) => endpoints.ingestBatch(events),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: qk.dashboard });
      client.invalidateQueries({ queryKey: ["cases"] });
      client.invalidateQueries({ queryKey: ["transactions"] });
    },
  });
}

export function useReplayJob() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: (jobId: string) => endpoints.replayJob(jobId),
    onSuccess: () => client.invalidateQueries({ queryKey: qk.metrics }),
  });
}

export function useRetention() {
  const client = useQueryClient();
  return useMutation({
    mutationFn: () => endpoints.runRetention(),
    onSuccess: () => {
      client.invalidateQueries({ queryKey: qk.metrics });
      client.invalidateQueries({ queryKey: ["case"] });
    },
  });
}
