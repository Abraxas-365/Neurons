import { useMemo, useState } from "react"
import { useParams } from "react-router-dom"
import { Check, Loader2, Search, Sparkles, TriangleAlert, Users } from "lucide-react"
import { toast } from "sonner"
import {
  useClassroom,
  useGrant,
  useReasons,
  useRoster,
  useTeamGrant,
  useTeams,
} from "@/hooks/queries"
import { ApiError } from "@/lib/api/client"
import { cn } from "@/lib/utils"
import { NeuronAmount, NeuronIcon } from "@/components/neuron-amount"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog"

const QUICK_AMOUNTS = [1, 2, 3, 5, 10]

/**
 * The screen a teacher uses while standing in front of a class. Everything is
 * one tap: pick people, pick a reason (which pre-fills the amount), confirm.
 * Nothing here opens a second page, because the interaction has to finish
 * inside the few seconds between one student answering and the next.
 */
export function GrantPage() {
  const { classroomId = "" } = useParams()
  const { data: classroom } = useClassroom(classroomId)

  return (
    <div className="mx-auto max-w-4xl">
      <div className="mb-5">
        <h1 className="text-xl font-semibold tracking-tight">Grant neurons</h1>
        <p className="text-sm text-muted-foreground">
          Every recipient receives the full amount — team grants are not split.
        </p>
      </div>

      {classroom && classroom.status !== "ACTIVE" && (
        <Card className="mb-5 border-redeem bg-redeem-muted/40">
          <CardContent className="flex items-center gap-3 py-4">
            <TriangleAlert className="size-5 shrink-0 text-redeem-foreground" />
            <div className="text-sm">
              <div className="font-medium">This course is {classroom.status.toLowerCase()}.</div>
              <div className="text-muted-foreground">
                Neurons can no longer move until it is reopened.
              </div>
            </div>
          </CardContent>
        </Card>
      )}

      <Tabs defaultValue="students">
        <TabsList className="mb-4">
          <TabsTrigger value="students">
            <Users className="size-4" />
            Students
          </TabsTrigger>
          <TabsTrigger value="teams">Teams</TabsTrigger>
        </TabsList>

        <TabsContent value="students">
          <StudentGrant classroomId={classroomId} />
        </TabsContent>
        <TabsContent value="teams">
          <TeamGrant classroomId={classroomId} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

/** Shared reason picker + amount stepper used by both grant modes. */
function AmountAndReason({
  classroomId,
  amount,
  setAmount,
  reasonId,
  setReasonId,
  reasonText,
  setReasonText,
  scope,
}: {
  classroomId: string
  amount: number
  setAmount: (n: number) => void
  reasonId: string | null
  setReasonId: (id: string | null) => void
  reasonText: string
  setReasonText: (s: string) => void
  scope: "INDIVIDUAL" | "TEAM"
}) {
  const { data: reasons, isLoading } = useReasons(classroomId, true)

  const applicable = useMemo(
    () => (reasons ?? []).filter((r) => r.scope === scope || r.scope === "BOTH"),
    [reasons, scope],
  )

  return (
    <div className="space-y-5">
      <div>
        <Label className="mb-2 block">Amount</Label>
        <div className="flex flex-wrap items-center gap-2">
          {QUICK_AMOUNTS.map((n) => (
            <Button
              key={n}
              type="button"
              variant={amount === n ? "default" : "outline"}
              className="h-11 min-w-14 text-base font-semibold tabular"
              onClick={() => setAmount(n)}
            >
              {n}
            </Button>
          ))}
          {/* Separated from the presets so the free-entry box does not read as
              a sixth quick amount. */}
          <div className="ml-1 flex items-center gap-2 border-l pl-3">
            <span className="text-xs text-muted-foreground">Other</span>
            <Input
              type="number"
              min={1}
              value={amount}
              onChange={(e) => setAmount(Math.max(1, Number(e.target.value) || 1))}
              className="h-11 w-20 text-center text-base font-semibold tabular"
              aria-label="Custom amount"
            />
          </div>
        </div>
      </div>

      <div>
        <Label className="mb-2 block">
          Reason <span className="text-muted-foreground">(required)</span>
        </Label>

        {isLoading ? (
          <div className="flex gap-2">
            <Skeleton className="h-9 w-24" />
            <Skeleton className="h-9 w-28" />
          </div>
        ) : applicable.length > 0 ? (
          <div className="flex flex-wrap gap-2">
            {applicable.map((r) => (
              <button
                key={r.id}
                type="button"
                onClick={() => {
                  setReasonId(r.id === reasonId ? null : r.id)
                  setReasonText("")
                  // A reason carries the amount the teacher normally gives for
                  // it, so the common case becomes a single tap.
                  if (r.id !== reasonId && r.suggested_amount) {
                    setAmount(r.suggested_amount)
                  }
                }}
                className={cn(
                  "flex items-center gap-2 rounded-full border px-3.5 py-2 text-sm transition-colors",
                  reasonId === r.id
                    ? "border-primary bg-primary text-primary-foreground"
                    : "hover:bg-accent",
                )}
              >
                {r.icon && <span>{r.icon}</span>}
                {r.name}
                {r.suggested_amount != null && (
                  <span className="tabular opacity-70">·{r.suggested_amount}</span>
                )}
              </button>
            ))}
          </div>
        ) : (
          <p className="text-sm text-muted-foreground">
            No saved reasons yet — type one below, or add reusable ones in the Catalog.
          </p>
        )}

        {!reasonId && (
          <Textarea
            value={reasonText}
            onChange={(e) => setReasonText(e.target.value)}
            placeholder="Why are you granting these neurons?"
            rows={2}
            className="mt-3"
          />
        )}
      </div>
    </div>
  )
}

function StudentGrant({ classroomId }: { classroomId: string }) {
  const [search, setSearch] = useState("")
  const [selected, setSelected] = useState<Set<string>>(new Set())
  const [amount, setAmount] = useState(1)
  const [reasonId, setReasonId] = useState<string | null>(null)
  const [reasonText, setReasonText] = useState("")
  const [pendingConfirm, setPendingConfirm] = useState(false)

  const { data: roster, isLoading } = useRoster(classroomId, {
    search: search || undefined,
    status: "ACTIVE",
    page_size: 200,
  })
  const grant = useGrant(classroomId)

  const students = roster?.items ?? []
  const total = amount * selected.size

  function toggle(id: string) {
    setSelected((prev) => {
      const next = new Set(prev)
      next.has(id) ? next.delete(id) : next.add(id)
      return next
    })
  }

  async function submit(confirmed = false) {
    if (selected.size === 0) {
      toast.error("Pick at least one student.")
      return
    }
    if (!reasonId && !reasonText.trim()) {
      toast.error("A reason is required for every grant.")
      return
    }

    try {
      const res = await grant.mutateAsync({
        enrollment_ids: [...selected],
        amount,
        reason_id: reasonId,
        reason_text: reasonId ? null : reasonText.trim(),
        channel: "MANUAL",
        confirmed,
      })

      toast.success(
        `${res.amount_each} neurons to ${res.recipients} student${res.recipients === 1 ? "" : "s"}.`,
      )
      setSelected(new Set())
      setPendingConfirm(false)
    } catch (err) {
      if (err instanceof ApiError && err.needsConfirmation) {
        setPendingConfirm(true)
        return
      }
      toast.error(err instanceof ApiError ? err.message : "Grant failed.")
    }
  }

  return (
    <div className="grid gap-5 lg:grid-cols-[1fr_20rem]">
      <Card>
        <CardContent className="space-y-4">
          <div className="relative">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search students"
              className="pl-9"
            />
          </div>

          <div className="flex items-center justify-between text-sm">
            <span className="text-muted-foreground">
              {selected.size} of {students.length} selected
            </span>
            <div className="flex gap-2">
              <Button
                variant="ghost"
                size="sm"
                onClick={() => setSelected(new Set(students.map((s) => s.id)))}
              >
                Select all
              </Button>
              {selected.size > 0 && (
                <Button variant="ghost" size="sm" onClick={() => setSelected(new Set())}>
                  Clear
                </Button>
              )}
            </div>
          </div>

          <ScrollArea className="h-[26rem] rounded-md border">
            {isLoading ? (
              <div className="space-y-2 p-3">
                {[0, 1, 2, 3, 4].map((i) => (
                  <Skeleton key={i} className="h-12" />
                ))}
              </div>
            ) : students.length === 0 ? (
              <p className="p-8 text-center text-sm text-muted-foreground">
                No students found.
              </p>
            ) : (
              <div className="divide-y">
                {students.map((s) => {
                  const on = selected.has(s.id)
                  return (
                    <button
                      key={s.id}
                      type="button"
                      onClick={() => toggle(s.id)}
                      className={cn(
                        "flex w-full items-center gap-3 px-3 py-2.5 text-left transition-colors",
                        on ? "bg-accent" : "hover:bg-muted/50",
                      )}
                    >
                      <div
                        className={cn(
                          "flex size-5 shrink-0 items-center justify-center rounded border-2 transition-colors",
                          on ? "border-primary bg-primary" : "border-input",
                        )}
                      >
                        {on && <Check className="size-3.5 text-primary-foreground" />}
                      </div>

                      <div className="min-w-0 flex-1">
                        <div className="truncate text-sm font-medium">{s.name}</div>
                        {s.team_name && (
                          <div className="truncate text-xs text-muted-foreground">
                            {s.team_name}
                          </div>
                        )}
                      </div>

                      <NeuronAmount value={s.balance} size="sm" className="text-muted-foreground" />
                    </button>
                  )
                })}
              </div>
            )}
          </ScrollArea>
        </CardContent>
      </Card>

      <div className="space-y-4 lg:sticky lg:top-6 lg:self-start">
        <Card>
          <CardContent>
            <AmountAndReason
              classroomId={classroomId}
              amount={amount}
              setAmount={setAmount}
              reasonId={reasonId}
              setReasonId={setReasonId}
              reasonText={reasonText}
              setReasonText={setReasonText}
              scope="INDIVIDUAL"
            />
          </CardContent>
        </Card>

        <Card className="border-primary/30 bg-accent/40">
          <CardContent className="space-y-3">
            <div className="flex items-baseline justify-between">
              <span className="text-sm text-muted-foreground">Total to give</span>
              <NeuronAmount value={total} size="lg" />
            </div>
            <Button
              className="h-12 w-full text-base"
              disabled={grant.isPending || selected.size === 0}
              onClick={() => submit(false)}
            >
              {grant.isPending ? (
                <Loader2 className="size-5 animate-spin" />
              ) : (
                <Sparkles className="size-5" />
              )}
              Grant to {selected.size || "…"}
            </Button>
          </CardContent>
        </Card>
      </div>

      <ConfirmLargeGrant
        open={pendingConfirm}
        onOpenChange={setPendingConfirm}
        amount={amount}
        recipients={selected.size}
        onConfirm={() => submit(true)}
      />
    </div>
  )
}

function TeamGrant({ classroomId }: { classroomId: string }) {
  const [teamId, setTeamId] = useState<string | null>(null)
  const [amount, setAmount] = useState(1)
  const [reasonId, setReasonId] = useState<string | null>(null)
  const [reasonText, setReasonText] = useState("")
  const [pendingConfirm, setPendingConfirm] = useState(false)

  const { data: teams, isLoading } = useTeams(classroomId)
  const teamGrant = useTeamGrant(classroomId)

  const team = teams?.find((t) => t.id === teamId)

  async function submit(confirmed = false) {
    if (!teamId) {
      toast.error("Pick a team.")
      return
    }
    if (!reasonId && !reasonText.trim()) {
      toast.error("A reason is required for every grant.")
      return
    }

    try {
      const res = await teamGrant.mutateAsync({
        team_id: teamId,
        amount,
        reason_id: reasonId,
        reason_text: reasonId ? null : reasonText.trim(),
        confirmed,
      })
      toast.success(
        `${res.amount_each} neurons each to ${res.recipients} member${res.recipients === 1 ? "" : "s"}.`,
      )
      setPendingConfirm(false)
    } catch (err) {
      if (err instanceof ApiError && err.needsConfirmation) {
        setPendingConfirm(true)
        return
      }
      toast.error(err instanceof ApiError ? err.message : "Grant failed.")
    }
  }

  return (
    <div className="grid gap-5 lg:grid-cols-[1fr_20rem]">
      <Card>
        <CardContent>
          {isLoading ? (
            <div className="space-y-2">
              {[0, 1, 2].map((i) => (
                <Skeleton key={i} className="h-16" />
              ))}
            </div>
          ) : !teams?.length ? (
            <p className="py-10 text-center text-sm text-muted-foreground">
              No teams yet. Create them from the Teams tab.
            </p>
          ) : (
            <div className="grid gap-2 sm:grid-cols-2">
              {teams.map((t) => (
                <button
                  key={t.id}
                  type="button"
                  onClick={() => setTeamId(t.id === teamId ? null : t.id)}
                  className={cn(
                    "flex items-center gap-3 rounded-lg border p-3 text-left transition-colors",
                    teamId === t.id
                      ? "border-primary bg-accent"
                      : "hover:bg-muted/50",
                  )}
                >
                  <div
                    className="flex size-9 shrink-0 items-center justify-center rounded-md text-sm font-semibold"
                    style={{
                      backgroundColor: t.color ?? "var(--accent)",
                      color: t.color ? "#fff" : "var(--accent-foreground)",
                    }}
                  >
                    {t.icon ?? t.name.charAt(0).toUpperCase()}
                  </div>
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm font-medium">{t.name}</div>
                    <div className="text-xs text-muted-foreground">
                      {t.member_count} member{t.member_count === 1 ? "" : "s"}
                    </div>
                  </div>
                </button>
              ))}
            </div>
          )}
        </CardContent>
      </Card>

      <div className="space-y-4 lg:sticky lg:top-6 lg:self-start">
        <Card>
          <CardContent>
            <AmountAndReason
              classroomId={classroomId}
              amount={amount}
              setAmount={setAmount}
              reasonId={reasonId}
              setReasonId={setReasonId}
              reasonText={reasonText}
              setReasonText={setReasonText}
              scope="TEAM"
            />
          </CardContent>
        </Card>

        <Card className="border-primary/30 bg-accent/40">
          <CardContent className="space-y-3">
            {team && (
              <div className="rounded-md bg-background/60 p-3 text-xs text-muted-foreground">
                Each of the <strong className="text-foreground">{team.member_count}</strong>{" "}
                members receives{" "}
                <strong className="text-foreground">{amount}</strong> — the amount is not
                divided.
              </div>
            )}
            <div className="flex items-baseline justify-between">
              <span className="text-sm text-muted-foreground">Total to give</span>
              <NeuronAmount value={amount * (team?.member_count ?? 0)} size="lg" />
            </div>
            <Button
              className="h-12 w-full text-base"
              disabled={teamGrant.isPending || !teamId}
              onClick={() => submit(false)}
            >
              {teamGrant.isPending ? (
                <Loader2 className="size-5 animate-spin" />
              ) : (
                <Sparkles className="size-5" />
              )}
              Grant to team
            </Button>
          </CardContent>
        </Card>
      </div>

      <ConfirmLargeGrant
        open={pendingConfirm}
        onOpenChange={setPendingConfirm}
        amount={amount}
        recipients={team?.member_count ?? 0}
        onConfirm={() => submit(true)}
      />
    </div>
  )
}

/** §11.9: an unusually large amount is a prompt, not an error. */
function ConfirmLargeGrant({
  open,
  onOpenChange,
  amount,
  recipients,
  onConfirm,
}: {
  open: boolean
  onOpenChange: (v: boolean) => void
  amount: number
  recipients: number
  onConfirm: () => void
}) {
  return (
    <AlertDialog open={open} onOpenChange={onOpenChange}>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>That is a large grant</AlertDialogTitle>
          <AlertDialogDescription asChild>
            <div className="space-y-3">
              <p>
                You are about to give <strong>{amount}</strong> neurons to{" "}
                <strong>{recipients}</strong> recipient{recipients === 1 ? "" : "s"}.
              </p>
              <div className="flex items-center justify-center gap-2 rounded-lg bg-accent py-4">
                <NeuronIcon className="size-6 text-primary" />
                <span className="text-3xl font-semibold tabular">
                  {(amount * recipients).toLocaleString()}
                </span>
              </div>
            </div>
          </AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Go back</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm}>Yes, grant it</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}
