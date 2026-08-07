import { useEffect, useRef, useState } from "react"
import { useParams } from "react-router-dom"
import { Html5Qrcode } from "html5-qrcode"
import { CameraOff, Check, Loader2, ScanLine, Sparkles } from "lucide-react"
import { toast } from "sonner"
import { useGrant, useReasons, useScanQR } from "@/hooks/queries"
import { ApiError } from "@/lib/api/client"
import { cn } from "@/lib/utils"
import { NeuronAmount } from "@/components/neuron-amount"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Textarea } from "@/components/ui/textarea"
import type { ScannedStudent } from "@/lib/api/types"

const READER_ID = "neurons-qr-reader"
const QUICK_AMOUNTS = [1, 2, 3, 5, 10]

/**
 * Flow A: the teacher scans a student's rotating QR, the API resolves it to an
 * enrollment, and the grant that follows carries the grant_key the scan handed
 * back — so a double tap on "confirm" pays exactly once (§11.3).
 */
export function ScanPage() {
  const { classroomId = "" } = useParams()
  const [scanned, setScanned] = useState<ScannedStudent | null>(null)
  const scan = useScanQR(classroomId)

  async function handleCode(code: string) {
    if (scan.isPending) return
    try {
      setScanned(await scan.mutateAsync(code))
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "That code is not valid.")
    }
  }

  return (
    <div className="mx-auto max-w-2xl">
      <div className="mb-5">
        <h1 className="text-xl font-semibold tracking-tight">Scan to grant</h1>
        <p className="text-sm text-muted-foreground">
          Point the camera at the student's code. Codes are single-use and expire fast.
        </p>
      </div>

      {scanned ? (
        <GrantToScanned
          classroomId={classroomId}
          student={scanned}
          onDone={() => setScanned(null)}
        />
      ) : (
        <Scanner onCode={handleCode} busy={scan.isPending} />
      )}
    </div>
  )
}

function Scanner({ onCode, busy }: { onCode: (code: string) => void; busy: boolean }) {
  const [error, setError] = useState<string | null>(null)
  const [manual, setManual] = useState("")
  const onCodeRef = useRef(onCode)
  onCodeRef.current = onCode

  useEffect(() => {
    const qr = new Html5Qrcode(READER_ID)
    let started = false

    qr.start(
      { facingMode: "environment" },
      { fps: 10, qrbox: { width: 250, height: 250 } },
      (text) => onCodeRef.current(text),
      // Per-frame decode failures are the normal case while aiming; ignore them.
      () => {},
    )
      .then(() => {
        started = true
      })
      .catch(() => setError("No camera available. Type the code instead."))

    return () => {
      // Stopping a scanner that never started throws synchronously, so the
      // camera is only torn down once it is actually running. Machines without
      // a camera would otherwise crash the page on unmount.
      if (!started) return
      qr.stop()
        .then(() => qr.clear())
        .catch(() => {})
    }
  }, [])

  return (
    <div className="space-y-4">
      <Card className="overflow-hidden">
        <CardContent className="p-0">
          {/* Without a camera there is nothing to show, so the viewfinder
              collapses to a short notice instead of a full-height black box
              that would push manual entry below the fold. */}
          <div className={cn("relative w-full bg-black", error ? "h-40" : "aspect-square")}>
            <div id={READER_ID} className="size-full [&_video]:size-full [&_video]:object-cover" />

            {!error && (
              <div className="pointer-events-none absolute inset-0 flex items-center justify-center">
                <div className="relative size-56 rounded-2xl border-2 border-white/70">
                  <ScanLine className="absolute inset-x-0 top-1/2 mx-auto size-10 -translate-y-1/2 text-primary opacity-80" />
                </div>
              </div>
            )}

            {error && (
              <div className="absolute inset-0 flex flex-col items-center justify-center gap-2 text-center text-white/80">
                <CameraOff className="size-8" />
                <p className="max-w-xs text-sm">{error}</p>
              </div>
            )}

            {busy && (
              <div className="absolute inset-0 flex items-center justify-center bg-black/60">
                <Loader2 className="size-8 animate-spin text-white" />
              </div>
            )}
          </div>
        </CardContent>
      </Card>

      <Card>
        <CardContent className="space-y-2">
          <Label htmlFor="manual">Or type the code</Label>
          <div className="flex gap-2">
            <Input
              id="manual"
              value={manual}
              // Codes are base64url and case-sensitive, so the text is passed
              // through untouched — upper-casing it would invalidate every code.
              onChange={(e) => setManual(e.target.value.trim())}
              placeholder="Paste the student's code"
              className="font-mono"
            />
            <Button
              disabled={!manual.trim() || busy}
              onClick={() => {
                onCode(manual.trim())
                setManual("")
              }}
            >
              Look up
            </Button>
          </div>
        </CardContent>
      </Card>
    </div>
  )
}

function GrantToScanned({
  classroomId,
  student,
  onDone,
}: {
  classroomId: string
  student: ScannedStudent
  onDone: () => void
}) {
  const [amount, setAmount] = useState(1)
  const [reasonId, setReasonId] = useState<string | null>(null)
  const [reasonText, setReasonText] = useState("")
  const [done, setDone] = useState(false)

  const { data: reasons } = useReasons(classroomId, true)
  const grant = useGrant(classroomId)

  const applicable = (reasons ?? []).filter((r) => r.scope === "INDIVIDUAL" || r.scope === "BOTH")

  async function submit(confirmed = false) {
    if (!reasonId && !reasonText.trim()) {
      toast.error("A reason is required for every grant.")
      return
    }
    try {
      await grant.mutateAsync({
        enrollment_ids: [student.enrollment_id],
        amount,
        reason_id: reasonId,
        reason_text: reasonId ? null : reasonText.trim(),
        channel: "QR",
        // The scan's key makes this grant idempotent (§11.3).
        idempotency_key: student.grant_key,
        confirmed,
      })
      setDone(true)
      setTimeout(onDone, 1400)
    } catch (err) {
      if (err instanceof ApiError && err.needsConfirmation) {
        void submit(true)
        return
      }
      toast.error(err instanceof ApiError ? err.message : "Grant failed.")
    }
  }

  if (done) {
    return (
      <Card className="border-grant">
        <CardContent className="flex flex-col items-center gap-3 py-14 text-center">
          <div className="flex size-16 items-center justify-center rounded-full bg-grant-muted">
            <Check className="size-8 text-grant" />
          </div>
          <div className="text-lg font-medium">Granted</div>
          <NeuronAmount value={amount} size="hero" />
          <p className="text-sm text-muted-foreground">to {student.student_name}</p>
        </CardContent>
      </Card>
    )
  }

  return (
    <Card>
      <CardContent className="space-y-5">
        <div className="flex items-center justify-between rounded-lg bg-accent p-4">
          <div className="min-w-0">
            <div className="truncate font-medium">{student.student_name}</div>
            <div className="truncate text-xs text-muted-foreground">
              {student.team_name ?? student.student_email}
            </div>
          </div>
          <div className="text-right">
            <div className="text-[11px] uppercase tracking-wide text-muted-foreground">
              Balance
            </div>
            <NeuronAmount value={student.balance} />
          </div>
        </div>

        <div>
          <Label className="mb-2 block">Amount</Label>
          <div className="flex flex-wrap gap-2">
            {QUICK_AMOUNTS.map((n) => (
              <Button
                key={n}
                type="button"
                variant={amount === n ? "default" : "outline"}
                className="h-12 min-w-14 text-base font-semibold tabular"
                onClick={() => setAmount(n)}
              >
                {n}
              </Button>
            ))}
          </div>
        </div>

        <div>
          <Label className="mb-2 block">Reason</Label>
          {applicable.length > 0 && (
            <div className="flex flex-wrap gap-2">
              {applicable.map((r) => (
                <button
                  key={r.id}
                  type="button"
                  onClick={() => {
                    setReasonId(r.id === reasonId ? null : r.id)
                    setReasonText("")
                    if (r.id !== reasonId && r.suggested_amount) setAmount(r.suggested_amount)
                  }}
                  className={cn(
                    "rounded-full border px-3.5 py-2 text-sm transition-colors",
                    reasonId === r.id
                      ? "border-primary bg-primary text-primary-foreground"
                      : "hover:bg-accent",
                  )}
                >
                  {r.icon} {r.name}
                </button>
              ))}
            </div>
          )}
          {!reasonId && (
            <Textarea
              value={reasonText}
              onChange={(e) => setReasonText(e.target.value)}
              rows={2}
              placeholder="Why?"
              className="mt-3"
            />
          )}
        </div>

        <div className="flex gap-2">
          <Button variant="outline" className="h-12" onClick={onDone}>
            Cancel
          </Button>
          <Button className="h-12 flex-1 text-base" disabled={grant.isPending} onClick={() => submit()}>
            {grant.isPending ? (
              <Loader2 className="size-5 animate-spin" />
            ) : (
              <Sparkles className="size-5" />
            )}
            Grant {amount}
          </Button>
        </div>
      </CardContent>
    </Card>
  )
}
