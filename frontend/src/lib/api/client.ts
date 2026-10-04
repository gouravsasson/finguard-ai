export class ApiError extends Error {
  status: number;
  code: string;
  requestId: string;

  constructor(status: number, message: string, code = "", requestId = "") {
    super(message);
    this.name = "ApiError";
    this.status = status;
    this.code = code;
    this.requestId = requestId;
  }

  /** The case changed under us. The caller should refetch before retrying. */
  get isVersionConflict() {
    return this.status === 409 && this.code === "version_conflict";
  }

  get isUnauthenticated() {
    return this.status === 401;
  }

  get isForbidden() {
    return this.status === 403;
  }
}

type RequestOptions = Omit<RequestInit, "body"> & {
  /** Serialized as JSON unless it is already a string. */
  body?: unknown;
  query?: Record<string, string | number | undefined | null>;
};

function withQuery(
  path: string,
  query?: RequestOptions["query"],
): string {
  if (!query) return path;
  const params = new URLSearchParams();
  for (const [key, value] of Object.entries(query)) {
    if (value === undefined || value === null || value === "") continue;
    params.set(key, String(value));
  }
  const qs = params.toString();
  return qs ? `${path}?${qs}` : path;
}

/**
 * Single entry point for the FinGuard API.
 *
 * Every request carries the X-FinGuard-Request header because the backend
 * rejects POSTs without it (CSRF), and credentials are included so the
 * httpOnly finguard_session cookie travels with the call.
 */
export async function request<T>(
  path: string,
  { body, query, ...init }: RequestOptions = {},
): Promise<T> {
  const headers = new Headers(init.headers);
  headers.set("X-FinGuard-Request", "1");

  let payload: BodyInit | undefined;
  if (body !== undefined) {
    payload = typeof body === "string" ? body : JSON.stringify(body);
    if (!headers.has("Content-Type")) {
      headers.set("Content-Type", "application/json");
    }
  }

  const res = await fetch(withQuery(path, query), {
    ...init,
    headers,
    body: payload,
    credentials: "include",
  });

  const text = await res.text();
  let data: unknown = {};
  if (text) {
    try {
      data = JSON.parse(text);
    } catch {
      if (!res.ok) throw new ApiError(res.status, res.statusText);
      return text as T;
    }
  }

  if (!res.ok) {
    const err = (data as { error?: { code?: string; message?: string; request_id?: string } })
      ?.error;
    throw new ApiError(
      res.status,
      err?.message || res.statusText || "Request failed",
      err?.code,
      err?.request_id,
    );
  }

  return data as T;
}
