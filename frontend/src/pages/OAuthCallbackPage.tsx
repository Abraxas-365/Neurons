import { useEffect, useRef, useState } from "react"
import { Navigate, useNavigate, useSearchParams } from "react-router-dom"
import { Loader2 } from "lucide-react"
import { useAuth } from "@/auth/context"
import { ApiError } from "@/lib/api/client"

/**
 * Landing spot for the backend OAuth redirect (GET /auth/callback/google
 * -> 302 here with access_token/refresh_token in the query string). Adopts
 * the tokens into the client and bounces into the app.
 */
export function OAuthCallbackPage() {
  const [params] = useSearchParams()
  const navigate = useNavigate()
  const { loginWithTokens } = useAuth()
  const [error, setError] = useState<string | null>(null)
  const ran = useRef(false)

  const accessToken = params.get("access_token")
  const refreshToken = params.get("refresh_token")

  useEffect(() => {
    if (ran.current) return
    ran.current = true

    if (!accessToken || !refreshToken) {
      setError("Missing tokens from the sign-in redirect.")
      return
    }

    loginWithTokens(accessToken, refreshToken)
      .then(() => navigate("/", { replace: true }))
      .catch((err) => {
        setError(err instanceof ApiError ? err.message : "Could not complete sign-in.")
      })
  }, [accessToken, refreshToken, loginWithTokens, navigate])

  if (error) {
    return <Navigate to="/login" replace />
  }

  return (
    <div className="flex min-h-svh items-center justify-center bg-gradient-to-b from-accent/40 to-background p-4">
      <Loader2 className="size-8 animate-spin text-muted-foreground" />
    </div>
  )
}
