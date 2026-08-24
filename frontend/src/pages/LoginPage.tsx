import { useState } from "react"
import { Loader2 } from "lucide-react"
import { toast } from "sonner"
import { authApi } from "@/lib/api/endpoints"
import { ApiError } from "@/lib/api/client"
import { NeuronIcon } from "@/components/neuron-amount"
import { Button } from "@/components/ui/button"
import {
  Card,
  CardContent,
  CardDescription,
  CardHeader,
  CardTitle,
} from "@/components/ui/card"

/**
 * Accounts are provisioned out-of-band (an admin creates the user/invitation
 * directly), so the only self-serve action here is starting the Google OAuth
 * dance. The backend redirects back to /auth/callback with tokens once done.
 */
export function LoginPage() {
  const [busy, setBusy] = useState(false)

  async function signInWithGoogle() {
    setBusy(true)
    try {
      const authUrl = await authApi.googleLogin()
      window.location.assign(authUrl)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Could not reach the server.")
      setBusy(false)
    }
  }

  return (
    <div className="flex min-h-svh items-center justify-center bg-gradient-to-b from-accent/40 to-background p-4">
      <div className="w-full max-w-sm">
        <div className="mb-8 flex flex-col items-center gap-3 text-center">
          <div className="flex size-14 items-center justify-center rounded-2xl bg-primary text-primary-foreground shadow-lg shadow-primary/25">
            <NeuronIcon className="size-8" />
          </div>
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">NEURONS</h1>
            <p className="text-sm text-muted-foreground">
              Reward what you want to see more of.
            </p>
          </div>
        </div>

        <Card>
          <CardHeader>
            <CardTitle>Sign in</CardTitle>
            <CardDescription>Use your Google account to continue.</CardDescription>
          </CardHeader>

          <CardContent>
            <Button
              type="button"
              className="w-full"
              variant="outline"
              disabled={busy}
              onClick={signInWithGoogle}
            >
              {busy ? (
                <Loader2 className="size-4 animate-spin" />
              ) : (
                <GoogleIcon className="size-4" />
              )}
              Continue with Google
            </Button>
          </CardContent>
        </Card>
      </div>
    </div>
  )
}

function GoogleIcon({ className }: { className?: string }) {
  return (
    <svg viewBox="0 0 24 24" className={className} aria-hidden="true">
      <path
        fill="#4285F4"
        d="M23.52 12.27c0-.85-.08-1.67-.22-2.45H12v4.64h6.48a5.54 5.54 0 0 1-2.4 3.63v3h3.88c2.27-2.09 3.56-5.17 3.56-8.82Z"
      />
      <path
        fill="#34A853"
        d="M12 24c3.24 0 5.96-1.07 7.95-2.91l-3.88-3c-1.08.72-2.46 1.15-4.07 1.15-3.13 0-5.78-2.11-6.73-4.96H1.26v3.09A11.998 11.998 0 0 0 12 24Z"
      />
      <path
        fill="#FBBC05"
        d="M5.27 14.28A7.2 7.2 0 0 1 4.89 12c0-.79.14-1.56.38-2.28V6.63H1.26A11.998 11.998 0 0 0 0 12c0 1.94.46 3.77 1.26 5.37l4.01-3.09Z"
      />
      <path
        fill="#EA4335"
        d="M12 4.77c1.76 0 3.34.6 4.59 1.79l3.44-3.44C17.95 1.19 15.24 0 12 0 7.31 0 3.26 2.69 1.26 6.63l4.01 3.09C6.22 6.88 8.87 4.77 12 4.77Z"
      />
    </svg>
  )
}
