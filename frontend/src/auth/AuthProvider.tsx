import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react"
import { useQueryClient } from "@tanstack/react-query"
import { authApi } from "@/lib/api/endpoints"
import { setUnauthorizedHandler, tokenStore } from "@/lib/api/client"
import type { UserDetails } from "@/lib/api/types"
import { AuthContext, matchScope, type Role } from "./context"

const USER_KEY = "neurons.user"

function readStoredUser(): UserDetails | null {
  const raw = localStorage.getItem(USER_KEY)
  if (!raw) return null
  try {
    return JSON.parse(raw) as UserDetails
  } catch {
    // A corrupted entry should log the user out, not white-screen the app.
    localStorage.removeItem(USER_KEY)
    return null
  }
}

/**
 * The access token is the authority on what an account may do — it is what the
 * API itself checks. The login response body carries a `scopes` array that is
 * not always populated, so reading the token avoids a UI that disagrees with
 * the server about the user's own role.
 */
function scopesFromToken(token: string | null): string[] {
  if (!token) return []
  try {
    const payload = token.split(".")[1]
    const json = atob(payload.replace(/-/g, "+").replace(/_/g, "/"))
    const claims = JSON.parse(json) as { scopes?: string[] }
    return claims.scopes ?? []
  } catch {
    return []
  }
}

export function AuthProvider({ children }: { children: ReactNode }) {
  const [user, setUser] = useState<UserDetails | null>(readStoredUser)
  const [scopes, setScopes] = useState<string[]>(() => scopesFromToken(tokenStore.access))
  const queryClient = useQueryClient()

  const logout = useCallback(() => {
    tokenStore.clear()
    localStorage.removeItem(USER_KEY)
    setUser(null)
    setScopes([])
    queryClient.clear()
  }, [queryClient])

  // Any 401 anywhere in the app drops the session, so a revoked or expired
  // token cannot leave the UI in a half-logged-in state.
  useEffect(() => {
    setUnauthorizedHandler(() => {
      localStorage.removeItem(USER_KEY)
      setUser(null)
      setScopes([])
    })
  }, [])

  const login = useCallback(async (email: string, code: string, tenantId: string) => {
    const res = await authApi.verifyLogin(email, code, tenantId)
    tokenStore.set(res.access_token, res.refresh_token)
    localStorage.setItem(USER_KEY, JSON.stringify(res.user))
    setUser(res.user)
    setScopes(scopesFromToken(res.access_token))
  }, [])

  const loginWithTokens = useCallback(async (accessToken: string, refreshToken: string) => {
    tokenStore.set(accessToken, refreshToken)
    const res = await authApi.me()
    localStorage.setItem(USER_KEY, JSON.stringify(res.user))
    setUser(res.user)
    setScopes(scopesFromToken(accessToken))
  }, [])

  const value = useMemo(() => {
    const hasScope = (scope: string) => matchScope(scopes, scope)
    const role: Role = hasScope("neurons:grant") ? "teacher" : "student"

    return {
      user,
      isAuthenticated: Boolean(user && tokenStore.access),
      role,
      hasScope,
      login,
      loginWithTokens,
      logout,
    }
  }, [user, scopes, login, loginWithTokens, logout])

  return <AuthContext.Provider value={value}>{children}</AuthContext.Provider>
}
