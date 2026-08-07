import { useState } from "react"
import { Link, useParams } from "react-router-dom"
import { QRCodeSVG } from "qrcode.react"
import { formatDistanceToNow } from "date-fns"
import { ArrowLeft, Gift, Loader2, QrCode, RefreshCw } from "lucide-react"
import { useAuth } from "@/auth/context"
import {
  useMyBenefits,
  useMyEnrollment,
  useMyHistory,
  useMyMedals,
  useMyQRToken,
} from "@/hooks/queries"
import { signedAmount, txPresentation } from "@/lib/tx-presentation"
import { cn } from "@/lib/utils"
import { NeuronAmount, NeuronIcon } from "@/components/neuron-amount"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog"
import { Progress } from "@/components/ui/progress"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"

export function MyWalletPage() {
  const { classroomId = "" } = useParams()
  const { user } = useAuth()
  const { data: me, isLoading } = useMyEnrollment(classroomId)
  const { data: history } = useMyHistory(classroomId, { page_size: 30 })
  const { data: medals } = useMyMedals(classroomId)
  const { data: benefits } = useMyBenefits(classroomId)
  const [qrOpen, setQrOpen] = useState(false)

  if (isLoading || !me) {
    return (
      <div className="space-y-4">
        <Skeleton className="h-48" />
        <Skeleton className="h-64" />
      </div>
    )
  }

  return (
    <div className="space-y-5">
      <Button asChild variant="ghost" size="sm" className="-ml-2">
        <Link to="/">
          <ArrowLeft className="size-4" />
          All courses
        </Link>
      </Button>

      <Card className="overflow-hidden border-primary/25">
        <CardContent className="relative space-y-4 py-8 text-center">
          {/* A soft radial glow makes the balance feel like a prize, not a metric. */}
          <div
            aria-hidden
            className="pointer-events-none absolute inset-0 opacity-70"
            style={{
              background:
                "radial-gradient(60% 60% at 50% 0%, color-mix(in oklch, var(--primary) 18%, transparent), transparent)",
            }}
          />
          <div className="relative">
            <div className="text-sm text-muted-foreground">{me.classroom_name}</div>
            <div className="mt-2 flex items-center justify-center">
              <NeuronAmount value={me.balance} size="hero" />
            </div>
            <div className="mt-1 text-xs text-muted-foreground">
              {me.total_received} earned · {me.total_returned} returned
            </div>

            {me.team_name && (
              <Badge variant="secondary" className="mt-3">
                {me.team_name}
              </Badge>
            )}
            {!me.classroom_open && (
              <Badge variant="secondary" className="mt-3 ml-2">
                Course closed
              </Badge>
            )}
          </div>

          <Button className="relative h-12 w-full max-w-xs" onClick={() => setQrOpen(true)}>
            <QrCode className="size-5" />
            Show my code
          </Button>
        </CardContent>
      </Card>

      <Tabs defaultValue="activity">
        <TabsList className="mb-4">
          <TabsTrigger value="activity">Activity</TabsTrigger>
          <TabsTrigger value="benefits">Benefits</TabsTrigger>
          <TabsTrigger value="medals">
            Medals {medals?.length ? `(${medals.length})` : ""}
          </TabsTrigger>
        </TabsList>

        <TabsContent value="activity">
          <Card>
            <CardContent>
              {!history?.items.length ? (
                <p className="py-10 text-center text-sm text-muted-foreground">
                  Nothing yet. Your teacher will start granting soon.
                </p>
              ) : (
                <ul className="divide-y">
                  {history.items.map((tx) => {
                    const p = txPresentation[tx.type]
                    const Icon = p.icon
                    return (
                      <li key={tx.id} className="flex items-center gap-3 py-3">
                        <span
                          className={cn(
                            "flex size-9 shrink-0 items-center justify-center rounded-full",
                            p.chip,
                          )}
                        >
                          <Icon className="size-4" />
                        </span>
                        <div className="min-w-0 flex-1">
                          <div className="truncate text-sm font-medium">
                            {tx.reason_text ?? tx.benefit_text ?? p.label}
                          </div>
                          <div className="text-xs text-muted-foreground">
                            {formatDistanceToNow(new Date(tx.created_at), { addSuffix: true })}
                          </div>
                        </div>
                        <span className={cn("text-base font-semibold tabular", p.tone)}>
                          {signedAmount(tx.type, tx.amount)}
                        </span>
                      </li>
                    )
                  })}
                </ul>
              )}
            </CardContent>
          </Card>
        </TabsContent>

        <TabsContent value="benefits">
          {!benefits?.length ? (
            <Card>
              <CardContent className="py-10 text-center text-sm text-muted-foreground">
                Your teacher has not published any benefits yet.
              </CardContent>
            </Card>
          ) : (
            <div className="space-y-3">
              {benefits.map((b) => {
                const cost = b.cost
                const pct = cost ? Math.min(100, (me.balance / cost) * 100) : 100
                const affordable = cost == null || me.balance >= cost
                return (
                  <Card key={b.id} className={affordable ? "border-grant/40" : undefined}>
                    <CardContent className="space-y-3">
                      <div className="flex items-start gap-3">
                        <span className="text-2xl">{b.icon ?? "🎁"}</span>
                        <div className="min-w-0 flex-1">
                          <div className="font-medium">{b.name}</div>
                          {b.description && (
                            <p className="text-xs text-muted-foreground">{b.description}</p>
                          )}
                        </div>
                        {cost == null ? (
                          <Badge variant="outline">Any amount</Badge>
                        ) : (
                          <NeuronAmount value={cost} />
                        )}
                      </div>

                      {cost != null && (
                        <>
                          <Progress value={pct} />
                          <p className="text-xs text-muted-foreground">
                            {affordable
                              ? "You can claim this — ask your teacher."
                              : `${cost - me.balance} more neurons to go.`}
                          </p>
                        </>
                      )}
                    </CardContent>
                  </Card>
                )
              })}
            </div>
          )}
        </TabsContent>

        <TabsContent value="medals">
          {!medals?.length ? (
            <Card>
              <CardContent className="flex flex-col items-center gap-2 py-12 text-center">
                <Gift className="size-8 text-muted-foreground" />
                <p className="text-sm text-muted-foreground">No medals yet.</p>
              </CardContent>
            </Card>
          ) : (
            <div className="grid grid-cols-2 gap-3 sm:grid-cols-3">
              {medals.map((m) => (
                <Card key={m.id} className="text-center">
                  <CardContent className="space-y-1 py-6">
                    <div className="mx-auto flex size-16 items-center justify-center rounded-full bg-medal-muted text-3xl">
                      {m.medal_icon ?? "🏅"}
                    </div>
                    <div className="text-sm font-medium">{m.medal_name}</div>
                    <div className="text-xs text-muted-foreground">
                      {formatDistanceToNow(new Date(m.awarded_at), { addSuffix: true })}
                    </div>
                    {m.team_name && (
                      <Badge variant="secondary" className="mt-1">
                        {m.team_name}
                      </Badge>
                    )}
                  </CardContent>
                </Card>
              ))}
            </div>
          )}
        </TabsContent>
      </Tabs>

      <MyQRDialog
        classroomId={classroomId}
        studentName={user?.name ?? ""}
        teamName={me.team_name}
        classroomName={me.classroom_name}
        open={qrOpen}
        onOpenChange={setQrOpen}
      />
    </div>
  )
}

/**
 * The student's identity code. It is deliberately short-lived and single-use
 * (RN-13), so the dialog fetches a fresh one every time it opens and offers an
 * explicit regenerate rather than silently polling.
 */
function MyQRDialog({
  classroomId,
  studentName,
  teamName,
  classroomName,
  open,
  onOpenChange,
}: {
  classroomId: string
  studentName: string
  teamName: string | null
  classroomName: string
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const { data: token, isFetching, refetch } = useMyQRToken(classroomId, open)

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Show this to your teacher</DialogTitle>
          <DialogDescription>
            The code expires quickly and works once.
          </DialogDescription>
        </DialogHeader>

        <div className="flex flex-col items-center gap-4 py-4">
          <div className="text-center">
            <p className="text-lg font-semibold">{studentName}</p>
            <p className="text-sm text-muted-foreground">{classroomName}</p>
            {teamName && (
              <Badge variant="secondary" className="mt-1">{teamName}</Badge>
            )}
          </div>

          <div className="rounded-2xl bg-white p-4">
            {token ? (
              <QRCodeSVG value={token.code} size={220} level="M" />
            ) : (
              <Skeleton className="size-[220px]" />
            )}
          </div>

          {token && (
            <>
              <div className="font-mono text-xl tracking-[0.3em]">{token.code}</div>
              <div className="flex items-center gap-2 text-xs text-muted-foreground">
                <NeuronIcon className="size-3.5" />
                Expires in {token.ttl_seconds}s
              </div>
            </>
          )}

          <Button variant="outline" onClick={() => refetch()} disabled={isFetching}>
            {isFetching ? (
              <Loader2 className="size-4 animate-spin" />
            ) : (
              <RefreshCw className="size-4" />
            )}
            New code
          </Button>
        </div>
      </DialogContent>
    </Dialog>
  )
}
