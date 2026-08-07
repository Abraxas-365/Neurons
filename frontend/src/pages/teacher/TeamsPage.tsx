import { useState } from "react"
import { useParams } from "react-router-dom"
import { Loader2, Plus, Shuffle, Trash2, Users } from "lucide-react"
import { toast } from "sonner"
import {
  useCreateTeam,
  useDeleteTeam,
  useRandomizeTeams,
  useRoster,
  useSetStudentTeam,
  useTeams,
} from "@/hooks/queries"
import { NeuronAmount } from "@/components/neuron-amount"
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
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { Textarea } from "@/components/ui/textarea"
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"

const TEAM_COLORS = [
  "#7c5cff",
  "#22a06b",
  "#e2a33c",
  "#e5484d",
  "#3b82f6",
  "#ec4899",
  "#14b8a6",
  "#f97316",
]

export function TeamsPage() {
  const { classroomId = "" } = useParams()
  const { data: teams, isLoading } = useTeams(classroomId)
  const { data: roster } = useRoster(classroomId, { status: "ACTIVE", page_size: 200 })
  const deleteTeam = useDeleteTeam(classroomId)
  const setTeam = useSetStudentTeam(classroomId)

  const students = roster?.items ?? []
  const unassigned = students.filter((s) => !s.team_id)

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Teams</h1>
          <p className="text-sm text-muted-foreground">
            A team grant pays every member the full amount.
          </p>
        </div>
        <div className="flex gap-2">
          <RandomizeDialog classroomId={classroomId} studentCount={students.length} />
          <NewTeamDialog classroomId={classroomId} />
        </div>
      </div>

      {isLoading ? (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {[0, 1, 2].map((i) => (
            <Skeleton key={i} className="h-48" />
          ))}
        </div>
      ) : (
        <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
          {teams?.map((t) => {
            const members = students.filter((s) => s.team_id === t.id)
            return (
              <Card key={t.id}>
                <CardHeader className="flex-row items-center gap-3">
                  <div
                    className="flex size-10 shrink-0 items-center justify-center rounded-lg text-sm font-semibold text-white"
                    style={{ backgroundColor: t.color ?? "var(--primary)" }}
                  >
                    {t.icon ?? t.name.charAt(0).toUpperCase()}
                  </div>
                  <div className="min-w-0 flex-1">
                    <CardTitle className="truncate text-base">{t.name}</CardTitle>
                    <p className="text-xs text-muted-foreground">
                      {t.member_count} member{t.member_count === 1 ? "" : "s"}
                    </p>
                  </div>
                  <AlertDialog>
                    <AlertDialogTrigger asChild>
                      <Button variant="ghost" size="icon" className="size-8">
                        <Trash2 className="size-4" />
                      </Button>
                    </AlertDialogTrigger>
                    <AlertDialogContent>
                      <AlertDialogHeader>
                        <AlertDialogTitle>Delete {t.name}?</AlertDialogTitle>
                        <AlertDialogDescription>
                          Members keep their neurons and history — only the grouping goes away.
                        </AlertDialogDescription>
                      </AlertDialogHeader>
                      <AlertDialogFooter>
                        <AlertDialogCancel>Cancel</AlertDialogCancel>
                        <AlertDialogAction
                          onClick={() =>
                            deleteTeam.mutateAsync(t.id).then(() => toast.success("Team deleted."))
                          }
                        >
                          Delete
                        </AlertDialogAction>
                      </AlertDialogFooter>
                    </AlertDialogContent>
                  </AlertDialog>
                </CardHeader>
                <CardContent>
                  {members.length === 0 ? (
                    <p className="py-4 text-center text-sm text-muted-foreground">
                      No members yet.
                    </p>
                  ) : (
                    <ul className="space-y-1">
                      {members.map((m) => (
                        <li key={m.id} className="flex items-center justify-between gap-2 text-sm">
                          <span className="truncate">{m.name}</span>
                          <NeuronAmount value={m.balance} size="sm" />
                        </li>
                      ))}
                    </ul>
                  )}
                </CardContent>
              </Card>
            )
          })}
        </div>
      )}

      {unassigned.length > 0 && (
        <Card className="border-dashed">
          <CardHeader>
            <CardTitle className="flex items-center gap-2 text-base">
              <Users className="size-4" />
              Not in a team ({unassigned.length})
            </CardTitle>
          </CardHeader>
          <CardContent>
            <ul className="divide-y">
              {unassigned.map((s) => (
                <li key={s.id} className="flex items-center justify-between gap-3 py-2">
                  <span className="min-w-0 truncate text-sm">{s.name}</span>
                  <Select
                    onValueChange={(teamId) => setTeam.mutate({ enrollmentId: s.id, teamId })}
                  >
                    <SelectTrigger size="sm" className="w-40">
                      <SelectValue placeholder="Assign to…" />
                    </SelectTrigger>
                    <SelectContent>
                      {teams?.map((t) => (
                        <SelectItem key={t.id} value={t.id}>
                          {t.name}
                        </SelectItem>
                      ))}
                    </SelectContent>
                  </Select>
                </li>
              ))}
            </ul>
          </CardContent>
        </Card>
      )}
    </div>
  )
}

function NewTeamDialog({ classroomId }: { classroomId: string }) {
  const [open, setOpen] = useState(false)
  const [color, setColor] = useState(TEAM_COLORS[0])
  const create = useCreateTeam(classroomId)

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)
    await create.mutateAsync({
      name: String(form.get("name")),
      description: (form.get("description") as string) || null,
      color,
    })
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus className="size-4" />
          New team
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-sm">
        <form onSubmit={onSubmit}>
          <DialogHeader>
            <DialogTitle>New team</DialogTitle>
          </DialogHeader>
          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="name">Name</Label>
              <Input id="name" name="name" required autoFocus placeholder="Team Alpha" />
            </div>
            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea id="description" name="description" rows={2} />
            </div>
            <div className="space-y-2">
              <Label>Color</Label>
              <div className="flex flex-wrap gap-2">
                {TEAM_COLORS.map((c) => (
                  <button
                    key={c}
                    type="button"
                    onClick={() => setColor(c)}
                    aria-label={`Color ${c}`}
                    className="size-8 rounded-full ring-offset-2 ring-offset-background transition-all data-[on=true]:ring-2"
                    data-on={color === c}
                    style={{ backgroundColor: c, boxShadow: color === c ? `0 0 0 2px ${c}` : undefined }}
                  />
                ))}
              </div>
            </div>
          </div>
          <DialogFooter>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending && <Loader2 className="size-4 animate-spin" />}
              Create
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function RandomizeDialog({
  classroomId,
  studentCount,
}: {
  classroomId: string
  studentCount: number
}) {
  const [open, setOpen] = useState(false)
  const [teamCount, setTeamCount] = useState(4)
  const randomize = useRandomizeTeams(classroomId)

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">
          <Shuffle className="size-4" />
          Randomize
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-sm">
        <DialogHeader>
          <DialogTitle>Randomize teams</DialogTitle>
          <DialogDescription>
            Distributes the {studentCount} active students evenly. Existing teams are replaced.
          </DialogDescription>
        </DialogHeader>
        <div className="space-y-2 py-4">
          <Label htmlFor="teams">Number of teams</Label>
          <Input
            id="teams"
            type="number"
            min={2}
            max={Math.max(2, studentCount)}
            value={teamCount}
            onChange={(e) => setTeamCount(Number(e.target.value))}
            className="tabular"
          />
          <p className="text-xs text-muted-foreground">
            About {Math.ceil(studentCount / Math.max(1, teamCount))} students per team.
          </p>
        </div>
        <DialogFooter>
          <Button
            disabled={randomize.isPending || studentCount < 2}
            onClick={() =>
              randomize
                .mutateAsync({ team_count: teamCount })
                .then((r) => {
                  toast.success(`${r.teams.length} teams created.`)
                  setOpen(false)
                })
                .catch(() => {})
            }
          >
            {randomize.isPending && <Loader2 className="size-4 animate-spin" />}
            Shuffle
          </Button>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  )
}
