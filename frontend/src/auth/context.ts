import { createContext, useContext } from "react"
import type { UserDetails } from "@/lib/api/types"

export type Role = "teacher" | "student"

export interface AuthState {
  user: UserDetails | null
  isAuthenticated: boolean
  /**
   * Derived from scopes, not from a stored field: the backend is the single
   * source of truth about what this account may do. Anyone who can grant
   * neurons is driving the teacher experience.
   */
  role: Role
  hasScope: (scope: string) => boolean
  login: (email: string, code: string, tenantId: string) => Promise<void>
  logout: () => void
}

export const AuthContext = createContext<AuthState | null>(null)

export function useAuth() {
  const ctx = useContext(AuthContext)
  if (!ctx) {
    throw new Error("useAuth must be used inside <AuthProvider>")
  }
  return ctx
}

/** Wildcard-aware scope check mirroring kernel.AuthContext.HasScope. */
export function matchScope(granted: string[], required: string): boolean {
  if (granted.includes(required)) return true
  const [resource] = required.split(":")
  return granted.includes(`${resource}:*`) || granted.includes("*")
}
