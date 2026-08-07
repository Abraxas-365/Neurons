import { useState } from "react"
import { useParams } from "react-router-dom"
import { Award, Loader2, Plus, Trash2, Undo2 } from "lucide-react"
import { formatDistanceToNow } from "date-fns"
import {
  useAwardMedal,
  useCreateMedal,
  useDeleteMedal,
  useMedalAwards,
  useMedals,
  useRevokeAward,
  useRoster,
  useTeams,
} from "@/hooks/queries"
import { cn } from "@/lib/utils"
import { Badge } from "@/components/ui/badge"
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import type { MedalType } from "@/lib/api/types"

export function MedalsPage() {
  const { classroomId = "" } = useParams()
  const { data: medals, isLoading } = useMedals(classroomId)
  const { data: awards } = useMedalAwards(classroomId)
  const remove = useDeleteMedal(classroomId)
  const revoke = useRevokeAward(classroomId)

  return (
    <div className="space-y-5">
      <div className="flex flex-wrap items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">Medals</h1>
          <p className="text-sm text-muted-foreground">
            Recognition that costs no neurons. A team medal shows on every member's profile.
          </p>
        </div>
        <div className="flex gap-2">
          <AwardDialog classroomId={classroomId} />
          <NewMedalDialog classroomId={classroomId} />
        </div>
      </div>

      {isLoading ? (
        <Skeleton className="h-40" />
      ) : !medals?.length ? (
        <Card className="border-dashed">
          <CardContent className="py-14 text-center text-sm text-muted-foreground">
            No medals defined yet.
          </CardContent>
        </Card>
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-4">
          {medals.map((m) => (
            <Card key={m.id} className={cn("text-center", !m.is_active && "opacity-60")}>
              <CardContent className="space-y-2">
                <div className="mx-auto flex size-16 items-center justify-center rounded-full bg-medal-muted text-3xl">
                  {m.icon ?? "🏅"}
                </div>
                <div className="font-medium">{m.name}</div>
                {m.description && (
                  <p className="line-clamp-2 text-xs text-muted-foreground">{m.description}</p>
                )}
                <div className="flex items-center justify-center gap-2">
                  <Badge variant="outline">{m.type === "TEAM" ? "Teams" : "Students"}</Badge>
                  {m.repeatable && <Badge variant="secondary">Repeatable</Badge>}
                </div>
                <AlertDialog>
                  <AlertDialogTrigger asChild>
                    <Button variant="ghost" size="sm" className="text-muted-foreground">
                      <Trash2 className="size-4" />
                      Delete
                    </Button>
                  </AlertDialogTrigger>
                  <AlertDialogContent>
                    <AlertDialogHeader>
                      <AlertDialogTitle>Delete "{m.name}"?</AlertDialogTitle>
                      <AlertDialogDescription>
                        Students who already earned it keep it.
                      </AlertDialogDescription>
                    </AlertDialogHeader>
                    <AlertDialogFooter>
                      <AlertDialogCancel>Cancel</AlertDialogCancel>
                      <AlertDialogAction onClick={() => remove.mutate(m.id)}>
                        Delete
                      </AlertDialogAction>
                    </AlertDialogFooter>
                  </AlertDialogContent>
                </AlertDialog>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <Card>
        <CardHeader>
          <CardTitle className="text-base">Recently awarded</CardTitle>
        </CardHeader>
        <CardContent>
          {!awards?.length ? (
            <p className="py-8 text-center text-sm text-muted-foreground">
              No medals awarded yet.
            </p>
          ) : (
            <ul className="divide-y">
              {awards.map((a) => (
                <li key={a.id} className="flex items-center gap-3 py-2.5">
                  <span className="flex size-9 items-center justify-center rounded-full bg-medal-muted text-lg">
                    {a.medal_icon ?? "🏅"}
                  </span>
                  <div className="min-w-0 flex-1">
                    <div className="truncate text-sm">
                      <span className="font-medium">{a.student_name ?? a.team_name}</span>
                      <span className="text-muted-foreground"> earned {a.medal_name}</span>
                    </div>
                    <div className="text-xs text-muted-foreground">
                      {formatDistanceToNow(new Date(a.awarded_at), { addSuffix: true })}
                      {a.note && ` · ${a.note}`}
                    </div>
                  </div>
                  <Button
                    variant="ghost"
                    size="icon"
                    className="size-8"
                    title="Revoke"
                    onClick={() => revoke.mutate(a.id)}
                  >
                    <Undo2 className="size-4" />
                  </Button>
                </li>
              ))}
            </ul>
          )}
        </CardContent>
      </Card>
    </div>
  )
}

function NewMedalDialog({ classroomId }: { classroomId: string }) {
  const [open, setOpen] = useState(false)
  const [type, setType] = useState<MedalType>("INDIVIDUAL")
  const [repeatable, setRepeatable] = useState(false)
  const create = useCreateMedal(classroomId)

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = new FormData(e.currentTarget)
    await create.mutateAsync({
      name: String(f.get("name")),
      description: (f.get("description") as string) || null,
      icon: (f.get("icon") as string) || null,
      type,
      repeatable,
      // RN-14: a team medal is meant to be visible on each member's profile.
      show_on_member_profile: type === "TEAM",
      visible_to_students: true,
    })
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus className="size-4" />
          New medal
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={onSubmit}>
          <DialogHeader>
            <DialogTitle>New medal</DialogTitle>
            <DialogDescription>Medals are symbolic — they cost no neurons.</DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="grid grid-cols-[4rem_1fr] gap-3">
              <div className="space-y-2">
                <Label htmlFor="m-icon">Icon</Label>
                <Input
                  id="m-icon"
                  name="icon"
                  maxLength={4}
                  className="text-center text-lg"
                  placeholder="🏅"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="m-name">Name</Label>
                <Input id="m-name" name="name" required placeholder="Best presentation" />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="m-desc">Description</Label>
              <Textarea id="m-desc" name="description" rows={2} />
            </div>

            <div className="space-y-2">
              <Label>Awarded to</Label>
              <Select value={type} onValueChange={(v) => setType(v as MedalType)}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  <SelectItem value="INDIVIDUAL">A student</SelectItem>
                  <SelectItem value="TEAM">A team</SelectItem>
                </SelectContent>
              </Select>
            </div>

            <div className="flex items-center justify-between rounded-lg border p-3">
              <div className="pr-4">
                <Label htmlFor="repeatable">Can be earned more than once</Label>
                <p className="text-xs text-muted-foreground">
                  Otherwise each student may earn it a single time.
                </p>
              </div>
              <Switch id="repeatable" checked={repeatable} onCheckedChange={setRepeatable} />
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

function AwardDialog({ classroomId }: { classroomId: string }) {
  const [open, setOpen] = useState(false)
  const [medalId, setMedalId] = useState<string>("")
  const [targetId, setTargetId] = useState<string>("")

  const { data: medals } = useMedals(classroomId, true)
  const { data: roster } = useRoster(classroomId, { status: "ACTIVE", page_size: 200 })
  const { data: teams } = useTeams(classroomId)
  const award = useAwardMedal(classroomId)

  const medal = medals?.find((m) => m.id === medalId)
  const isTeam = medal?.type === "TEAM"

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    if (!medalId || !targetId) return
    const note = String(new FormData(e.currentTarget).get("note") ?? "").trim()

    await award.mutateAsync({
      medalId,
      input: isTeam
        ? { team_id: targetId, note: note || null }
        : { enrollment_ids: [targetId], note: note || null },
    })
    setOpen(false)
    setTargetId("")
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button variant="outline">
          <Award className="size-4" />
          Award a medal
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-md">
        <form onSubmit={onSubmit}>
          <DialogHeader>
            <DialogTitle>Award a medal</DialogTitle>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="space-y-2">
              <Label>Medal</Label>
              <Select
                value={medalId}
                onValueChange={(v) => {
                  setMedalId(v)
                  // Switching between an individual and a team medal makes the
                  // previously picked recipient meaningless.
                  setTargetId("")
                }}
              >
                <SelectTrigger>
                  <SelectValue placeholder="Pick a medal" />
                </SelectTrigger>
                <SelectContent>
                  {medals?.map((m) => (
                    <SelectItem key={m.id} value={m.id}>
                      {m.icon} {m.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </div>

            {medal && (
              <div className="space-y-2">
                <Label>{isTeam ? "Team" : "Student"}</Label>
                <Select value={targetId} onValueChange={setTargetId}>
                  <SelectTrigger>
                    <SelectValue placeholder={isTeam ? "Pick a team" : "Pick a student"} />
                  </SelectTrigger>
                  <SelectContent>
                    {(isTeam ? teams : roster?.items)?.map((t) => (
                      <SelectItem key={t.id} value={t.id}>
                        {t.name}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </div>
            )}

            <div className="space-y-2">
              <Label htmlFor="note">Note</Label>
              <Textarea id="note" name="note" rows={2} placeholder="Optional" />
            </div>
          </div>

          <DialogFooter>
            <Button type="submit" disabled={award.isPending || !medalId || !targetId}>
              {award.isPending && <Loader2 className="size-4 animate-spin" />}
              Award
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
