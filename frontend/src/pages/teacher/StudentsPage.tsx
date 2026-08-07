import { useState } from "react"
import { useParams } from "react-router-dom"
import { Loader2, Mail, MoreHorizontal, Search, UserPlus } from "lucide-react"
import { toast } from "sonner"
import {
  useApproveEnrollment,
  useInviteStudents,
  useRoster,
  useSetStudentTeam,
  useTeams,
  useWithdrawEnrollment,
} from "@/hooks/queries"
import { NeuronAmount } from "@/components/neuron-amount"
import { Badge } from "@/components/ui/badge"
import { Button } from "@/components/ui/button"
import { Card, CardContent } from "@/components/ui/card"
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog"
import {
  DropdownMenu,
  DropdownMenuContent,
  DropdownMenuItem,
  DropdownMenuLabel,
  DropdownMenuSeparator,
  DropdownMenuTrigger,
} from "@/components/ui/dropdown-menu"
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from "@/components/ui/table"
import { Textarea } from "@/components/ui/textarea"
import type { EnrollmentStatus, InviteResult } from "@/lib/api/types"

const statusTone: Record<EnrollmentStatus, string> = {
  PENDING: "bg-redeem-muted text-redeem-foreground",
  ACTIVE: "bg-grant-muted text-grant",
  WITHDRAWN: "bg-muted text-muted-foreground",
}

export function StudentsPage() {
  const { classroomId = "" } = useParams()
  const [search, setSearch] = useState("")

  const { data, isLoading } = useRoster(classroomId, {
    search: search || undefined,
    page_size: 200,
  })
  const { data: teams } = useTeams(classroomId)
  const approve = useApproveEnrollment(classroomId)
  const withdraw = useWithdrawEnrollment(classroomId)
  const setTeam = useSetStudentTeam(classroomId)

  const students = data?.items ?? []
  const pending = students.filter((s) => s.status === "PENDING").length

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Students</h1>
          <p className="text-sm text-muted-foreground">
            {students.length} enrolled
            {pending > 0 && ` · ${pending} waiting for approval`}
          </p>
        </div>
        <InviteDialog classroomId={classroomId} />
      </div>

      <Card>
        <CardContent className="space-y-4">
          <div className="relative max-w-sm">
            <Search className="absolute left-3 top-1/2 size-4 -translate-y-1/2 text-muted-foreground" />
            <Input
              value={search}
              onChange={(e) => setSearch(e.target.value)}
              placeholder="Search by name or email"
              className="pl-9"
            />
          </div>

          {isLoading ? (
            <div className="space-y-2">
              {[0, 1, 2, 3, 4].map((i) => (
                <Skeleton key={i} className="h-12" />
              ))}
            </div>
          ) : students.length === 0 ? (
            <p className="py-12 text-center text-sm text-muted-foreground">
              No students yet. Share the invite code from the dashboard.
            </p>
          ) : (
            <Table>
              <TableHeader>
                <TableRow>
                  <TableHead>Student</TableHead>
                  <TableHead>Team</TableHead>
                  <TableHead className="text-right">Balance</TableHead>
                  <TableHead className="text-right">Received</TableHead>
                  <TableHead>Status</TableHead>
                  <TableHead className="w-10" />
                </TableRow>
              </TableHeader>
              <TableBody>
                {students.map((s) => (
                  <TableRow key={s.id}>
                    <TableCell>
                      <div className="font-medium">{s.name}</div>
                      <div className="text-xs text-muted-foreground">{s.email}</div>
                    </TableCell>
                    <TableCell className="text-sm text-muted-foreground">
                      {s.team_name ?? "—"}
                    </TableCell>
                    <TableCell className="text-right">
                      <NeuronAmount value={s.balance} size="sm" />
                    </TableCell>
                    <TableCell className="text-right text-sm tabular text-muted-foreground">
                      {s.total_received}
                    </TableCell>
                    <TableCell>
                      <Badge variant="secondary" className={statusTone[s.status]}>
                        {s.status}
                      </Badge>
                    </TableCell>
                    <TableCell>
                      <DropdownMenu>
                        <DropdownMenuTrigger asChild>
                          <Button variant="ghost" size="icon" className="size-8">
                            <MoreHorizontal className="size-4" />
                          </Button>
                        </DropdownMenuTrigger>
                        <DropdownMenuContent align="end">
                          {s.status === "PENDING" && (
                            <>
                              <DropdownMenuItem
                                onClick={() =>
                                  approve.mutateAsync(s.id).then(() => toast.success("Approved."))
                                }
                              >
                                Approve
                              </DropdownMenuItem>
                              <DropdownMenuSeparator />
                            </>
                          )}

                          <DropdownMenuLabel className="text-xs font-normal text-muted-foreground">
                            Move to team
                          </DropdownMenuLabel>
                          <DropdownMenuItem
                            onClick={() => setTeam.mutate({ enrollmentId: s.id, teamId: null })}
                          >
                            No team
                          </DropdownMenuItem>
                          {teams?.map((t) => (
                            <DropdownMenuItem
                              key={t.id}
                              onClick={() => setTeam.mutate({ enrollmentId: s.id, teamId: t.id })}
                            >
                              {t.name}
                            </DropdownMenuItem>
                          ))}

                          <DropdownMenuSeparator />
                          <DropdownMenuItem
                            variant="destructive"
                            onClick={() =>
                              withdraw
                                .mutateAsync(s.id)
                                .then(() => toast.success("Student withdrawn."))
                            }
                          >
                            Withdraw
                          </DropdownMenuItem>
                        </DropdownMenuContent>
                      </DropdownMenu>
                    </TableCell>
                  </TableRow>
                ))}
              </TableBody>
            </Table>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function InviteDialog({ classroomId }: { classroomId: string }) {
  const [open, setOpen] = useState(false)
  const [failed, setFailed] = useState<InviteResult[]>([])
  const invite = useInviteStudents(classroomId)

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const raw = String(new FormData(e.currentTarget).get("emails") ?? "")
    const emails = raw
      .split(/[\s,;]+/)
      .map((s) => s.trim())
      .filter(Boolean)

    if (emails.length === 0) return

    const results = await invite.mutateAsync(emails.map((email) => ({ email })))
    setFailed(results.filter((r) => r.status === "ERROR"))
    if (!results.some((r) => r.status === "ERROR")) setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <UserPlus className="size-4" />
          Add students
        </Button>
      </DialogTrigger>
      <DialogContent>
        <form onSubmit={onSubmit}>
          <DialogHeader>
            <DialogTitle>Add students</DialogTitle>
            <DialogDescription>
              Paste emails separated by commas, spaces or new lines.
            </DialogDescription>
          </DialogHeader>
          <div className="space-y-2 py-4">
            <Label htmlFor="emails" className="flex items-center gap-2">
              <Mail className="size-4" />
              Emails
            </Label>
            <Textarea id="emails" name="emails" rows={6} autoFocus />
            <p className="text-xs text-muted-foreground">
              Only students who already have an account can be enrolled. Anyone else
              should register first, then be added here.
            </p>

            {failed.length > 0 && (
              <div className="rounded-md border border-reverse/40 bg-reverse-muted/40 p-3">
                <div className="mb-1 text-sm font-medium">
                  {failed.length} could not be added
                </div>
                <ul className="space-y-0.5 text-xs text-muted-foreground">
                  {failed.map((f) => (
                    <li key={f.email}>
                      <span className="font-medium text-foreground">{f.email}</span> — {f.detail}
                    </li>
                  ))}
                </ul>
              </div>
            )}
          </div>
          <DialogFooter>
            <Button type="submit" disabled={invite.isPending}>
              {invite.isPending && <Loader2 className="size-4 animate-spin" />}
              Add
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
