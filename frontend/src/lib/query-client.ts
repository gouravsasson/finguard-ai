import { QueryClient } from "@tanstack/react-query";
import { ApiError } from "./api/client";

/**
 * Retries are deliberately narrow: an authentication or permission answer is
 * final, and a 409 means the caller has stale state and must refetch rather
 * than hammer the endpoint.
 */
export function createQueryClient() {
  return new QueryClient({
    defaultOptions: {
      queries: {
        staleTime: 30_000,
        refetchOnWindowFocus: false,
        retry: (failureCount, error) => {
          if (error instanceof ApiError) {
            if (error.status >= 400 && error.status < 500) return false;
          }
          return failureCount < 2;
        },
      },
      mutations: { retry: false },
    },
  });
}
