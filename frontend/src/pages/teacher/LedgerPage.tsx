import { useState } from "react"
import { useParams } from "react-router-dom"
import { format } from "date-fns"
import { ChevronLeft, ChevronRight, Undo2 } from "lucide-react"
import { useClassroom, useHistory, useReverse } from "@/hooks/queries"
import { channelLabel, signedAmount, txPresentation } from "@/lib/tx-presentation"
import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
  AlertDialogTrigger,
} from "@/components/ui/alert-dialog"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import type { Transaction, TxType } from "@/lib/api/types"

const TYPE_FILTERS: { value: TxType | "ALL"; label: string }[] = [
  { value: "ALL", label: "All movements" },
  { value: "GRANT", label: "Grants" },
  { value: "REDEMPTION", label: "Redemptions" },
  { value: "VAULT_TOPUP", label: "Vault top-ups" },
  { value: "GRANT_REVERSAL", label: "Reversed grants" },
  { value: "REDEMPTION_REVERSAL", label: "Reversed redemptions" },
  { value: "ADJUSTMENT", label: "Adjustments" },
]

export function LedgerPage() {
  const { classroomId = "" } = useParams()
  const [page, setPage] = useState(1)
  const [type, setType] = useState<TxType | "ALL">("ALL")

  const { data: classroom } = useClassroom(classroomId)
  const { data, isLoading } = useHistory(classroomId, {
    page,
    page_size: 25,
    type: type === "ALL" ? undefined : type,
  })

  const rows = data?.items ?? []
  const pages = data?.pagination.pages ?? 1

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Ledger</h1>
        <p className="text-sm text-muted-foreground">
          Every neuron that moved, in order. Entries are never edited — a mistake is undone
          with a new entry.
        </p>
      </div>

      <Card>
        <CardContent className="space-y-4">
          <Select
            value={type}
            onValueChange={(v) => {
              setType(v as TxType | "ALL")
              setPage(1)
            }}
          >
            <SelectTrigger className="w-56">
              <SelectValue />
            </SelectTrigger>
            <SelectContent>
              {TYPE_FILTERS.map((f) => (
                <SelectItem key={f.value} value={f.value}>
                  {f.label}
                </SelectItem>
              ))}
            </SelectContent>
          </Select>

          {isLoading ? (
            <div className="space-y-2">
              {[0, 1, 2, 3, 4, 5].map((i) => (
                <Skeleton key={i} className="h-12" />
              ))}
            </div>
          ) : rows.length === 0 ? (
            <p className="py-12 text-center text-sm text-muted-foreground">
              No movements recorded.
            </p>
          ) : (
            <>
              <Table>
                <TableHeader>
                  <TableRow>
                    <TableHead>When</TableHead>
                    <TableHead>Movement</TableHead>
                    <TableHead>Who</TableHead>
                    <TableHead>Detail</TableHead>
                    <TableHead className="text-right">Amount</TableHead>
                    <TableHead className="text-right">Balance after</TableHead>
                    <TableHead className="w-10" />
                  </TableRow>
                </TableHeader>
                <TableBody>
                  {rows.map((tx) => (
                    <LedgerRow
                      key={tx.id}
                      tx={tx}
                      classroomId={classroomId}
                      voidWindow={classroom?.void_window_seconds ?? 0}
                    />
                  ))}
                </TableBody>
              </Table>

              <div className="flex items-center justify-between">
                <span className="text-sm text-muted-foreground">
                  Page {page} of {pages} · {data?.pagination.total ?? 0} movements
                </span>
                <div className="flex gap-2">
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page <= 1}
                    onClick={() => setPage((p) => p - 1)}
                  >
                    <ChevronLeft className="size-4" />
                    Previous
                  </Button>
                  <Button
                    variant="outline"
                    size="sm"
                    disabled={page >= pages}
                    onClick={() => setPage((p) => p + 1)}
                  >
                    Next
                    <ChevronRight className="size-4" />
                  </Button>
                </div>
              </div>
            </>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function LedgerRow({
  tx,
  classroomId,
  voidWindow,
}: {
  tx: Transaction
  classroomId: string
  voidWindow: number
}) {
  const reverse = useReverse(classroomId)
  const p = txPresentation[tx.type]
  const Icon = p.icon

  // A reversal can only undo a live entry, and only inside the window the
  // teacher configured (§15.4). Reversal rows are themselves terminal.
  const ageSeconds = (Date.now() - new Date(tx.created_at).getTime()) / 1000
  const canReverse =
    tx.status === "APPLIED" &&
    !tx.reversed_by_transaction_id &&
    (tx.type === "GRANT" || tx.type === "REDEMPTION") &&
    (voidWindow === 0 || ageSeconds <= voidWindow)

  return (
    <TableRow className={tx.status === "REVERSED" ? "opacity-55" : undefined}>
      <TableCell className="whitespace-nowrap text-xs text-muted-foreground">
        {format(new Date(tx.created_at), "d MMM HH:mm")}
      </TableCell>

      <TableCell>
        <span className="flex items-center gap-2">
          <span className={cn("flex size-7 items-center justify-center rounded-full", p.chip)}>
            <Icon className="size-3.5" />
          </span>
          <span className="text-sm">{p.label}</span>
        </span>
      </TableCell>

      <TableCell className="text-sm">
        {tx.student_name ?? tx.team_name ?? <span className="text-muted-foreground">Vault</span>}
      </TableCell>

      <TableCell className="max-w-56">
        <div className="truncate text-sm">{tx.reason_text ?? tx.benefit_text ?? "—"}</div>
        <div className="text-xs text-muted-foreground">
          {channelLabel[tx.channel]}
          {tx.status === "REVERSED" && " · reversed"}
        </div>
      </TableCell>

      <TableCell className={cn("text-right text-sm font-semibold tabular", p.tone)}>
        {signedAmount(tx.type, tx.amount)}
      </TableCell>

      <TableCell className="text-right text-sm tabular text-muted-foreground">
        {tx.student_balance_after ?? tx.vault_balance_after ?? "—"}
      </TableCell>

      <TableCell>
        {canReverse && (
          <AlertDialog>
            <AlertDialogTrigger asChild>
              <Button variant="ghost" size="icon" className="size-8" title="Reverse">
                <Undo2 className="size-4" />
              </Button>
            </AlertDialogTrigger>
            <AlertDialogContent>
              <AlertDialogHeader>
                <AlertDialogTitle>Reverse this movement?</AlertDialogTitle>
                <AlertDialogDescription>
                  This does not erase anything. A new opposite entry is written, and both stay
                  visible in the ledger.
                </AlertDialogDescription>
              </AlertDialogHeader>
              <AlertDialogFooter>
                <AlertDialogCancel>Cancel</AlertDialogCancel>
                <AlertDialogAction
                  onClick={() => reverse.mutate({ id: tx.id, reason: "Teacher reversal" })}
                >
                  Reverse
                </AlertDialogAction>
              </AlertDialogFooter>
            </AlertDialogContent>
          </AlertDialog>
        )}
        {tx.status === "REVERSED" && <Badge variant="secondary">Reversed</Badge>}
      </TableCell>
    </TableRow>
  )
}
