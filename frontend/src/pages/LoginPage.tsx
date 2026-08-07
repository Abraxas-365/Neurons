import { useState } from "react"
import { useNavigate } from "react-router-dom"
import { Loader2 } from "lucide-react"
import { toast } from "sonner"
import { useAuth } from "@/auth/context"
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
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

type Step = "email" | "tenant" | "code"

interface TenantOption {
  tenant_id: string
  company_name: string
}

/**
 * Passwordless OTP login. The backend deliberately does not reveal whether an
 * email exists, so the UI always advances to the code step and lets the
 * verification fail — never leak account existence here.
 */
export function LoginPage() {
  const navigate = useNavigate()
  const { login } = useAuth()

  const [step, setStep] = useState<Step>("email")
  const [email, setEmail] = useState("")
  const [code, setCode] = useState("")
  const [tenants, setTenants] = useState<TenantOption[]>([])
  const [tenantId, setTenantId] = useState("")
  const [busy, setBusy] = useState(false)

  async function submitEmail(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    try {
      const list = await authApi.tenants(email)

      if (list.length === 0) {
        toast.error("No organization found for that email.")
        return
      }

      setTenants(list)

      if (list.length === 1) {
        setTenantId(list[0].tenant_id)
        await sendCode(list[0].tenant_id)
      } else {
        setStep("tenant")
      }
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Could not reach the server.")
    } finally {
      setBusy(false)
    }
  }

  async function sendCode(tid: string) {
    const res = await authApi.initiateLogin(email, tid)
    toast.success(res.message)
    setStep("code")
  }

  async function submitTenant(e: React.FormEvent) {
    e.preventDefault()
    if (!tenantId) return
    setBusy(true)
    try {
      await sendCode(tenantId)
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Could not send the code.")
    } finally {
      setBusy(false)
    }
  }

  async function submitCode(e: React.FormEvent) {
    e.preventDefault()
    setBusy(true)
    try {
      await login(email, code, tenantId)
      navigate("/", { replace: true })
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Invalid or expired code.")
    } finally {
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
            <CardTitle>
              {step === "email" && "Sign in"}
              {step === "tenant" && "Choose your organization"}
              {step === "code" && "Enter your code"}
            </CardTitle>
            <CardDescription>
              {step === "email" && "We'll email you a one-time code."}
              {step === "tenant" && "This email belongs to more than one organization."}
              {step === "code" && `Sent to ${email}`}
            </CardDescription>
          </CardHeader>

          <CardContent>
            {step === "email" && (
              <form onSubmit={submitEmail} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="email">Email</Label>
                  <Input
                    id="email"
                    type="email"
                    autoComplete="email"
                    autoFocus
                    required
                    value={email}
                    onChange={(e) => setEmail(e.target.value)}
                    placeholder="you@university.edu"
                  />
                </div>
                <Button type="submit" className="w-full" disabled={busy}>
                  {busy && <Loader2 className="size-4 animate-spin" />}
                  Continue
                </Button>
              </form>
            )}

            {step === "tenant" && (
              <form onSubmit={submitTenant} className="space-y-4">
                <div className="space-y-2">
                  <Label>Organization</Label>
                  <Select value={tenantId} onValueChange={setTenantId}>
                    <SelectTrigger>
                      <SelectValue placeholder="Select one" />
                    </SelectTrigger>
                    <SelectContent>
                      {tenants.map((t) => (
                        <SelectItem key={t.tenant_id} value={t.tenant_id}>
                          {t.company_name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </div>
                <Button type="submit" className="w-full" disabled={busy || !tenantId}>
                  {busy && <Loader2 className="size-4 animate-spin" />}
                  Send code
                </Button>
              </form>
            )}

            {step === "code" && (
              <form onSubmit={submitCode} className="space-y-4">
                <div className="space-y-2">
                  <Label htmlFor="code">6-digit code</Label>
                  <Input
                    id="code"
                    inputMode="numeric"
                    autoComplete="one-time-code"
                    autoFocus
                    required
                    maxLength={6}
                    value={code}
                    onChange={(e) => setCode(e.target.value.replace(/\D/g, ""))}
                    className="text-center text-2xl tracking-[0.4em] tabular"
                    placeholder="······"
                  />
                </div>
                <Button
                  type="submit"
                  className="w-full"
                  disabled={busy || code.length < 4}
                >
                  {busy && <Loader2 className="size-4 animate-spin" />}
                  Sign in
                </Button>
                <div className="flex justify-between text-xs">
                  <button
                    type="button"
                    className="text-muted-foreground hover:text-foreground"
                    onClick={() => {
                      setStep("email")
                      setCode("")
                    }}
                  >
                    Use another email
                  </button>
                  <button
                    type="button"
                    className="text-primary hover:underline"
                    onClick={async () => {
                      try {
                        await authApi.resendOtp(email, tenantId, "login")
                        toast.success("New code sent.")
                      } catch {
                        toast.error("Could not resend the code.")
                      }
                    }}
                  >
                    Resend
                  </button>
                </div>
              </form>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
