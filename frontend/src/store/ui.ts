import { create } from "zustand";

/**
 * Client-only state. Nothing here is fetched, so it stays out of React Query:
 * queue filters, the active case tab, and the transient notice strip.
 */
export type QueueFilter = {
  status: string;
  search: string;
  priority: string;
};

export type Notice = { tone: "info" | "error"; text: string } | null;

type UiState = {
  queue: QueueFilter;
  setQueueStatus: (status: string) => void;
  setQueueSearch: (search: string) => void;
  setQueuePriority: (priority: string) => void;
  clearQueueFilters: () => void;

  caseTab: string;
  setCaseTab: (tab: string) => void;

  notice: Notice;
  notify: (tone: "info" | "error", text: string) => void;
  clearNotice: () => void;
};

const emptyQueue: QueueFilter = { status: "all", search: "", priority: "all" };

export const useUiStore = create<UiState>((set) => ({
  queue: emptyQueue,
  setQueueStatus: (status) =>
    set((s) => ({ queue: { ...s.queue, status } })),
  setQueueSearch: (search) =>
    set((s) => ({ queue: { ...s.queue, search } })),
  setQueuePriority: (priority) =>
    set((s) => ({ queue: { ...s.queue, priority } })),
  clearQueueFilters: () => set({ queue: emptyQueue }),

  caseTab: "signals",
  setCaseTab: (caseTab) => set({ caseTab }),

  notice: null,
  notify: (tone, text) => set({ notice: { tone, text } }),
  clearNotice: () => set({ notice: null }),
}));
