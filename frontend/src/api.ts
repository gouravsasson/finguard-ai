/**
 * Compatibility surface. The real client lives in src/lib/api.
 * New code should import from "@/lib/api/hooks" or "@/lib/api/endpoints".
 */
export { ApiError, request } from "./lib/api/client";
export { endpoints } from "./lib/api/endpoints";
export { qk } from "./lib/api/keys";
export type * from "./lib/api/types";

import { request } from "./lib/api/client";

/** @deprecated Use `request` or the endpoint helpers. */
export function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  return request<T>(path, init as Parameters<typeof request>[1]);
}
