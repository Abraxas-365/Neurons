import axios, {
  AxiosError,
  type AxiosInstance,
  type AxiosRequestConfig,
} from "axios"
import type { ApiErrorBody } from "./types"

/**
 * A normalized error. The backend speaks errx codes (e.g. LEDGER_INSUFFICIENT_VAULT)
 * and the UI keys off `code` rather than message text, so copy changes on the
 * server never silently break a branch in the client.
 */
export class ApiError extends Error {
  readonly code: string
  readonly status: number
  readonly type?: string
  readonly requestId?: string
  readonly details?: Record<string, unknown>

  constructor(body: ApiErrorBody) {
    super(body.error)
    this.name = "ApiError"
    this.code = body.code
    this.status = body.status
    this.type = body.type
    this.requestId = body.request_id
    this.details = body.details
  }

  /** The teacher granted an unusually large amount and must confirm (§11.9). */
  get needsConfirmation() {
    return this.code === "LEDGER_CONFIRMATION_REQUIRED"
  }

  get isUnauthorized() {
    return this.status === 401
  }

  get isForbidden() {
    return this.status === 403
  }
}

const ACCESS_TOKEN_KEY = "neurons.access_token"
const REFRESH_TOKEN_KEY = "neurons.refresh_token"

export const tokenStore = {
  get access() {
    return localStorage.getItem(ACCESS_TOKEN_KEY)
  },
  get refresh() {
    return localStorage.getItem(REFRESH_TOKEN_KEY)
  },
  set(access: string, refresh: string) {
    localStorage.setItem(ACCESS_TOKEN_KEY, access)
    localStorage.setItem(REFRESH_TOKEN_KEY, refresh)
  },
  clear() {
    localStorage.removeItem(ACCESS_TOKEN_KEY)
    localStorage.removeItem(REFRESH_TOKEN_KEY)
  },
}

/** Fires when a request comes back 401 so the app can bounce to login. */
type UnauthorizedHandler = () => void
let onUnauthorized: UnauthorizedHandler = () => {}
export function setUnauthorizedHandler(fn: UnauthorizedHandler) {
  onUnauthorized = fn
}

const client: AxiosInstance = axios.create({
  baseURL: import.meta.env.VITE_API_URL ?? "",
  headers: { "Content-Type": "application/json" },
})

client.interceptors.request.use((config) => {
  const token = tokenStore.access
  if (token) {
    config.headers.Authorization = `Bearer ${token}`
  }
  return config
})

client.interceptors.response.use(
  (response) => response,
  (error: AxiosError<ApiErrorBody>) => {
    if (error.response) {
      const { status, data } = error.response

      if (status === 401) {
        tokenStore.clear()
        onUnauthorized()
      }

      // Some IAM handlers return a bare {error: "..."} without the full
      // envelope, so fill in the gaps rather than crashing on undefined.
      const body: ApiErrorBody = {
        error: data?.error ?? error.message ?? "Request failed",
        code: data?.code ?? `HTTP_${status}`,
        type: data?.type,
        status: data?.status ?? status,
        request_id: data?.request_id,
        details: data?.details,
      }
      return Promise.reject(new ApiError(body))
    }

    return Promise.reject(
      new ApiError({
        error: "Cannot reach the server. Check your connection.",
        code: "NETWORK_ERROR",
        status: 0,
      }),
    )
  },
)

async function request<T>(config: AxiosRequestConfig): Promise<T> {
  const response = await client.request<T>(config)
  return response.data
}

export const http = {
  get: <T>(url: string, params?: Record<string, unknown>) =>
    request<T>({ method: "GET", url, params }),
  post: <T>(url: string, data?: unknown) =>
    request<T>({ method: "POST", url, data }),
  put: <T>(url: string, data?: unknown) =>
    request<T>({ method: "PUT", url, data }),
  patch: <T>(url: string, data?: unknown) =>
    request<T>({ method: "PATCH", url, data }),
  delete: <T>(url: string, data?: unknown) =>
    request<T>({ method: "DELETE", url, data }),
}

export const API = "/api/v1"
