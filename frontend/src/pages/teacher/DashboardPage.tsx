import { useState } from "react"
import { Link, useParams } from "react-router-dom"
import {
  Award,
  Copy,
  Loader2,
  Plus,
  QrCode,
  Sparkles,
  TrendingUp,
  Users,
  Vault,
} from "lucide-react"
import { toast } from "sonner"
import {
  Bar,
  BarChart,
  CartesianGrid,
  ResponsiveContainer,
  Tooltip as RTooltip,
  XAxis,
  YAxis,
} from "recharts"
import {
  useClassroom,
  useHistory,
  useRanking,
  useReasonUsage,
  useStats,
  useTopup,
} from "@/hooks/queries"
import { signedAmount, txPresentation } from "@/lib/tx-presentation"
import { cn } from "@/lib/utils"
import { NeuronAmount, NeuronIcon } from "@/components/neuron-amount"
import { Button } from "@/components/ui/button"
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { formatDistanceToNow } from "date-fns"

export function DashboardPage() {
  const { classroomId = "" } = useParams()
  const { data: classroom } = useClassroom(classroomId)
  const { data: stats, isLoading } = useStats(classroomId)
  const { data: ranking } = useRanking(classroomId, 8)
  const { data: usage } = useReasonUsage(classroomId)
  const { data: history } = useHistory(classroomId, { page_size: 8 })

  const topStudents = ranking ?? []
  const topReasons = usage ?? []

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">
            {classroom?.name ?? <Skeleton className="h-6 w-40" />}
          </h1>
          <p className="text-sm text-muted-foreground">
            {[classroom?.section, classroom?.term].filter(Boolean).join(" · ")}
          </p>
        </div>
        <div className="flex gap-2">
          <Button asChild variant="outline">
            <Link to={`/courses/${classroomId}/scan`}>
              <QrCode className="size-4" />
              Scan
            </Link>
          </Button>
          <Button asChild>
            <Link to={`/courses/${classroomId}/grant`}>
              <Sparkles className="size-4" />
              Grant neurons
            </Link>
          </Button>
        </div>
      </div>

      {classroom && <InviteCard code={classroom.invite_code} />}

      <div className="grid gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard
          label="Vault"
          icon={Vault}
          loading={isLoading}
          value={
            classroom?.unlimited_issuance ? (
              <span className="flex items-center gap-1 text-primary">
                <NeuronIcon className="size-5" />∞
              </span>
            ) : (
              <NeuronAmount value={stats?.vault_balance ?? 0} size="xl" />
            )
          }
          action={!classroom?.unlimited_issuance && <TopupDialog classroomId={classroomId} />}
        />
        <StatCard
          label="In circulation"
          icon={TrendingUp}
          loading={isLoading}
          value={<NeuronAmount value={stats?.in_circulation ?? 0} size="xl" />}
          hint="Held by students right now"
        />
        <StatCard
          label="Granted all-time"
          icon={Sparkles}
          loading={isLoading}
          value={<NeuronAmount value={stats?.total_granted ?? 0} size="xl" />}
        />
        <StatCard
          label="Active students"
          icon={Users}
          loading={isLoading}
          value={
            <span className="text-2xl font-semibold tabular">{stats?.active_students ?? 0}</span>
          }
          hint={`${stats?.transactions ?? 0} movements`}
        />
      </div>

      <div className="grid gap-4 lg:grid-cols-2">
        <Card>
          <CardHeader className="flex-row items-center justify-between">
            <CardTitle className="text-base">Leaderboard — {classroom?.name}</CardTitle>
            <Button asChild variant="ghost" size="sm">
              <Link to={`/courses/${classroomId}/students`}>All students</Link>
            </Button>
          </CardHeader>
          <CardContent>
            {!ranking ? (
              <div className="space-y-2">
                {[0, 1, 2, 3].map((i) => (
                  <Skeleton key={i} className="h-10" />
                ))}
              </div>
            ) : topStudents.length === 0 ? (
              <p className="py-8 text-center text-sm text-muted-foreground">
                No neurons granted yet.
              </p>
            ) : (
              <ol className="space-y-1">
                {topStudents.map((r, i) => (
                  <li
                    key={r.enrollment_id}
                    className={cn(
                      "flex items-center gap-3 rounded-md px-2 py-2",
                      i === 0 && "bg-medal-muted",
                    )}
                  >
                    <span
                      className={cn(
                        "flex size-7 shrink-0 items-center justify-center rounded-full text-xs font-semibold tabular",
                        i === 0
                          ? "bg-medal text-medal-foreground"
                          : "bg-muted text-muted-foreground",
                      )}
                    >
                      {i + 1}
                    </span>
                    <span className="min-w-0 flex-1 truncate text-sm font-medium">
                      {r.student_name}
                    </span>
                    {r.medal_count > 0 && (
                      <span className="flex items-center gap-1 text-xs text-medal-foreground">
                        <Award className="size-3.5" />
                        {r.medal_count}
                      </span>
                    )}
                    <NeuronAmount value={r.total_received} />
                  </li>
                ))}
              </ol>
            )}
          </CardContent>
        </Card>

        <Card>
          <CardHeader>
            <CardTitle className="text-base">What you reward most</CardTitle>
          </CardHeader>
          <CardContent>
            {!topReasons.length ? (
              <p className="py-8 text-center text-sm text-muted-foreground">
                Reason stats appear once you start granting.
              </p>
            ) : (
              <ResponsiveContainer width="100%" height={240}>
                <BarChart data={topReasons.slice(0, 6)} layout="vertical" margin={{ left: 8 }}>
                  <CartesianGrid horizontal={false} stroke="var(--border)" />
                  <XAxis type="number" tick={{ fontSize: 11 }} stroke="var(--muted-foreground)" />
                  <YAxis
                    type="category"
                    dataKey="reason_name"
                    width={110}
                    tick={{ fontSize: 11 }}
                    stroke="var(--muted-foreground)"
                  />
                  <RTooltip
                    cursor={{ fill: "var(--muted)" }}
                    contentStyle={{
                      background: "var(--popover)",
                      border: "1px solid var(--border)",
                      borderRadius: "0.5rem",
                      fontSize: 12,
                    }}
                  />
                  <Bar dataKey="total_amount" fill="var(--primary)" radius={[0, 4, 4, 0]} />
                </BarChart>
              </ResponsiveContainer>
            )}
          </CardContent>
        </Card>
      </div>

      <Card>
        <CardHeader className="flex-row items-center justify-between">
          <CardTitle className="text-base">Recent activity</CardTitle>
          <Button asChild variant="ghost" size="sm">
            <Link to={`/courses/${classroomId}/ledger`}>Full ledger</Link>
          </Button>
        </CardHeader>
        <CardContent>
          {!history?.items.length ? (
            <p className="py-8 text-center text-sm text-muted-foreground">Nothing yet.</p>
          ) : (
            <ul className="divide-y">
              {history.items.map((tx) => {
                const p = txPresentation[tx.type]
                const Icon = p.icon
                return (
                  <li key={tx.id} className="flex items-center gap-3 py-2.5">
                    <span className={cn("flex size-8 items-center justify-center rounded-full", p.chip)}>
                      <Icon className="size-4" />
                    </span>
                    <div className="min-w-0 flex-1">
                      <div className="truncate text-sm">
                        <span className="font-medium">{tx.student_name ?? "Vault"}</span>{" "}
                        <span className="text-muted-foreground">
                          — {tx.reason_text ?? tx.benefit_text ?? p.label}
                        </span>
                      </div>
                      <div className="text-xs text-muted-foreground">
                        {formatDistanceToNow(new Date(tx.created_at), { addSuffix: true })}
                      </div>
                    </div>
                    <span className={cn("text-sm font-semibold tabular", p.tone)}>
                      {signedAmount(tx.type, tx.amount)}
                    </span>
                  </li>
                )
              })}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function StatCard({
  label,
  icon: Icon,
  value,
  hint,
  action,
  loading,
}: {
  label: string
  icon: React.ElementType
  value: React.ReactNode
  hint?: string
  action?: React.ReactNode
  loading?: boolean
}) {
  return (
    <Card>
      <CardContent className="space-y-2">
        <div className="flex items-center justify-between">
          <span className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            {label}
          </span>
          <Icon className="size-4 text-muted-foreground" />
        </div>
        {loading ? <Skeleton className="h-8 w-24" /> : value}
        <div className="flex items-center justify-between">
          {hint && <span className="text-xs text-muted-foreground">{hint}</span>}
          {action}
        </div>
      </CardContent>
    </Card>
  )
}

function InviteCard({ code }: { code: string }) {
  return (
    <Card className="border-primary/25 bg-accent/40">
      <CardContent className="flex flex-wrap items-center justify-between gap-4">
        <div>
          <div className="text-xs font-medium uppercase tracking-wide text-muted-foreground">
            Invite code
          </div>
          <div className="font-mono text-2xl font-semibold tracking-[0.2em]">{code}</div>
        </div>
        <Button
          variant="outline"
          onClick={() => {
            navigator.clipboard.writeText(code)
            toast.success("Invite code copied.")
          }}
        >
          <Copy className="size-4" />
          Copy
        </Button>
      </CardContent>
    </Card>
  )
}

function TopupDialog({ classroomId }: { classroomId: string }) {
  const [open, setOpen] = useState(false)
  const topup = useTopup(classroomId)

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const amount = Number(new FormData(e.currentTarget).get("amount"))
    if (amount <= 0) return

    try {
      await topup.mutateAsync({ amount, notes: "Vault top-up" })
      toast.success(`${amount} neurons added to the vault.`)
      setOpen(false)
    } catch {
      toast.error("Top-up failed.")
    }
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="ghost" size="sm" className="-mr-2 h-7 px-2 text-xs">
          <Plus className="size-3.5" />
          Top up
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-sm">
        <form onSubmit={onSubmit}>
          <DialogHeader>
            <DialogTitle>Top up the vault</DialogTitle>
            <DialogDescription>
              This is recorded in the ledger like any other movement.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2 py-4">
            <Label htmlFor="amount">Neurons to add</Label>
            <Input
              id="amount"
              name="amount"
              type="number"
              min={1}
              defaultValue={500}
              autoFocus
              className="tabular"
            />
          </div>
          <DialogFooter>
            <Button type="submit" disabled={topup.isPending}>
              {topup.isPending && <Loader2 className="size-4 animate-spin" />}
              Add neurons
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
