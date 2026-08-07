import { useState } from "react"
import { useParams } from "react-router-dom"
import { Check, Loader2, Search, Undo2 } from "lucide-react"
import { toast } from "sonner"
import { useBenefits, useRedeem, useRoster } from "@/hooks/queries"
import { ApiError } from "@/lib/api/client"
import { cn } from "@/lib/utils"
import { NeuronAmount } from "@/components/neuron-amount"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { ScrollArea } from "@/components/ui/scroll-area"
import { Skeleton } from "@/components/ui/skeleton"
import { Textarea } from "@/components/ui/textarea"

/**
 * Flow C: the student hands neurons back for something. The teacher drives it
 * so the balance check and the ledger entry happen in one place.
 */
export function RedeemPage() {
  const { classroomId = "" } = useParams()
  const [search, setSearch] = useState("")
  const [studentId, setStudentId] = useState<string | null>(null)
  const [benefitId, setBenefitId] = useState<string | null>(null)
  const [amount, setAmount] = useState(0)
  const [note, setNote] = useState("")

  const { data: roster, isLoading } = useRoster(classroomId, {
    search: search || undefined,
    status: "ACTIVE",
    page_size: 200,
  })
  const { data: benefits } = useBenefits(classroomId, true)
  const redeem = useRedeem(classroomId)

  const students = roster?.items ?? []
  const student = students.find((s) => s.id === studentId)
  const benefit = benefits?.find((b) => b.id === benefitId)

  // A benefit with no fixed price lets the student decide what to spend (HU-063).
  const openPrice = benefit != null && benefit.cost == null
  const cost = openPrice ? amount : (benefit?.cost ?? amount)
  const affordable = student != null && cost > 0 && cost <= student.balance

  async function submit() {
    if (!studentId || cost <= 0) return
    if (!benefitId && !note.trim()) {
      toast.error("Say what the neurons are being returned for.")
      return
    }

    try {
      await redeem.mutateAsync({
        enrollment_id: studentId,
        amount: cost,
        benefit_id: benefitId,
        benefit_text: benefitId ? null : note.trim(),
      })
      toast.success(`${cost} neurons returned.`)
      setBenefitId(null)
      setAmount(0)
      setNote("")
    } catch (err) {
      toast.error(err instanceof ApiError ? err.message : "Redemption failed.")
    }
  }

  return (
    <div className="mx-auto max-w-4xl space-y-5">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Redeem neurons</h1>
        <p className="text-sm text-muted-foreground">
          Neurons a student returns go back into the classroom vault.
        </p>
      </div>

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

            <ScrollArea className="h-64 rounded-md border">
              {isLoading ? (
                <div className="space-y-2 p-3">
                  {[0, 1, 2].map((i) => (
                    <Skeleton key={i} className="h-12" />
                  ))}
                </div>
              ) : (
                <div className="divide-y">
                  {students.map((s) => (
                    <button
                      key={s.id}
                      type="button"
                      onClick={() => setStudentId(s.id)}
                      className={cn(
                        "flex w-full items-center justify-between gap-3 px-3 py-2.5 text-left transition-colors",
                        studentId === s.id ? "bg-accent" : "hover:bg-muted/50",
                      )}
                    >
                      <div className="min-w-0">
                        <div className="truncate text-sm font-medium">{s.name}</div>
                        {s.team_name && (
                          <div className="truncate text-xs text-muted-foreground">
                            {s.team_name}
                          </div>
                        )}
                      </div>
                      <NeuronAmount value={s.balance} size="sm" />
                    </button>
                  ))}
                </div>
              )}
            </ScrollArea>

            <div>
              <Label className="mb-2 block">What are they getting?</Label>
              {!benefits?.length ? (
                <p className="text-sm text-muted-foreground">
                  No benefits published — describe it below instead.
                </p>
              ) : (
                <div className="grid gap-2 sm:grid-cols-2">
                  {benefits.map((b) => {
                    const on = benefitId === b.id
                    const tooExpensive =
                      student != null && b.cost != null && b.cost > student.balance
                    return (
                      <button
                        key={b.id}
                        type="button"
                        disabled={tooExpensive}
                        onClick={() => {
                          setBenefitId(on ? null : b.id)
                          setAmount(0)
                          setNote("")
                        }}
                        className={cn(
                          "flex items-center gap-3 rounded-lg border p-3 text-left transition-colors",
                          on ? "border-primary bg-accent" : "hover:bg-muted/50",
                          tooExpensive && "cursor-not-allowed opacity-45",
                        )}
                      >
                        <span className="text-xl">{b.icon ?? "🎁"}</span>
                        <span className="min-w-0 flex-1">
                          <span className="block truncate text-sm font-medium">{b.name}</span>
                          {b.cost == null ? (
                            <Badge variant="outline" className="mt-0.5">
                              Choose amount
                            </Badge>
                          ) : (
                            <NeuronAmount value={b.cost} size="sm" />
                          )}
                        </span>
                      </button>
                    )
                  })}
                </div>
              )}

              {(!benefitId || openPrice) && (
                <div className="mt-3 space-y-3">
                  {(openPrice || !benefitId) && (
                    <div className="space-y-2">
                      <Label htmlFor="amount">Neurons to return</Label>
                      <Input
                        id="amount"
                        type="number"
                        min={1}
                        max={student?.balance}
                        value={amount || ""}
                        onChange={(e) => setAmount(Math.max(0, Number(e.target.value) || 0))}
                        className="tabular"
                      />
                    </div>
                  )}
                  {!benefitId && (
                    <Textarea
                      value={note}
                      onChange={(e) => setNote(e.target.value)}
                      rows={2}
                      placeholder="What is this redemption for?"
                    />
                  )}
                </div>
              )}
            </div>
          </CardContent>
        </Card>

        <Card className="lg:sticky lg:top-6 lg:self-start">
          <CardContent className="space-y-4">
            {!student ? (
              <p className="py-8 text-center text-sm text-muted-foreground">
                Pick a student to start.
              </p>
            ) : (
              <>
                <div>
                  <div className="text-sm font-medium">{student.name}</div>
                  <div className="text-xs text-muted-foreground">Current balance</div>
                  <NeuronAmount value={student.balance} size="lg" />
                </div>

                <div className="space-y-1 border-t pt-3 text-sm">
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Returning</span>
                    <span className="font-semibold tabular text-redeem-foreground">
                      −{cost || 0}
                    </span>
                  </div>
                  <div className="flex justify-between">
                    <span className="text-muted-foreground">Left after</span>
                    <span className={cn("font-semibold tabular", !affordable && "text-reverse")}>
                      {student.balance - (cost || 0)}
                    </span>
                  </div>
                </div>

                {cost > 0 && !affordable && (
                  <p className="rounded-md bg-reverse-muted/50 p-2 text-xs text-reverse">
                    Not enough neurons — a balance can never go negative.
                  </p>
                )}

                <Button
                  className="h-12 w-full text-base"
                  disabled={redeem.isPending || !affordable}
                  onClick={submit}
                >
                  {redeem.isPending ? (
                    <Loader2 className="size-5 animate-spin" />
                  ) : affordable ? (
                    <Check className="size-5" />
                  ) : (
                    <Undo2 className="size-5" />
                  )}
                  Confirm redemption
                </Button>
              </>
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  )
}
