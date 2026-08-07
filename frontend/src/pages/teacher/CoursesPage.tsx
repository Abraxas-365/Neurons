import { useState } from "react"
import { Link } from "react-router-dom"
import { BookOpen, Loader2, Plus } from "lucide-react"
import { useClassrooms, useCreateClassroom } from "@/hooks/queries"
import { NeuronAmount, NeuronIcon } from "@/components/neuron-amount"
import { UserMenu } from "@/layouts/UserMenu"
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
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"
import { Switch } from "@/components/ui/switch"
import { Textarea } from "@/components/ui/textarea"
import type { ClassroomStatus } from "@/lib/api/types"

const statusTone: Record<ClassroomStatus, string> = {
  DRAFT: "bg-muted text-muted-foreground",
  ACTIVE: "bg-grant-muted text-grant",
  CLOSED: "bg-redeem-muted text-redeem-foreground",
  ARCHIVED: "bg-muted text-muted-foreground",
}

export function CoursesPage() {
  const { data: classrooms, isLoading } = useClassrooms()

  return (
    <div className="min-h-svh bg-background">
      <header className="flex h-16 items-center justify-between border-b px-4 md:px-8">
        <div className="flex items-center gap-2">
          <NeuronIcon className="size-5 text-primary" />
          <span className="font-semibold tracking-tight">NEURONS</span>
        </div>
        <UserMenu />
      </header>

      <div className="mx-auto max-w-6xl p-4 md:p-8">
        <div className="mb-6 flex items-end justify-between gap-4">
          <div>
            <h1 className="text-2xl font-semibold tracking-tight">Your courses</h1>
            <p className="text-sm text-muted-foreground">
              Each course keeps its own vault and its own neurons.
            </p>
          </div>
          <NewCourseDialog />
        </div>

        {isLoading ? (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {[0, 1, 2].map((i) => (
              <Skeleton key={i} className="h-40" />
            ))}
          </div>
        ) : !classrooms?.length ? (
          <EmptyState />
        ) : (
          <div className="grid gap-4 sm:grid-cols-2 lg:grid-cols-3">
            {classrooms.map((c) => (
              <Link key={c.id} to={`/courses/${c.id}`}>
                <Card className="h-full transition-all hover:border-primary/40 hover:shadow-md">
                  <CardContent className="flex h-full flex-col gap-4">
                    <div className="flex items-start justify-between gap-2">
                      <div className="min-w-0">
                        <div className="truncate font-medium">{c.name}</div>
                        <div className="truncate text-xs text-muted-foreground">
                          {[c.section, c.term].filter(Boolean).join(" · ") || "—"}
                        </div>
                      </div>
                      <Badge className={statusTone[c.status]} variant="secondary">
                        {c.status}
                      </Badge>
                    </div>

                    <div className="mt-auto grid grid-cols-2 gap-3 border-t pt-4">
                      <div>
                        <div className="text-[11px] uppercase tracking-wide text-muted-foreground">
                          Vault
                        </div>
                        {c.unlimited_issuance ? (
                          <div className="flex items-center gap-1 text-lg font-semibold text-primary">
                            <NeuronIcon className="size-4" />∞
                          </div>
                        ) : (
                          <NeuronAmount value={c.vault_balance} />
                        )}
                      </div>
                      <div>
                        <div className="text-[11px] uppercase tracking-wide text-muted-foreground">
                          With students
                        </div>
                        <NeuronAmount value={c.distributed} />
                      </div>
                    </div>
                  </CardContent>
                </Card>
              </Link>
            ))}
          </div>
        )}
      </div>
    </div>
  )
}

function EmptyState() {
  return (
    <Card className="border-dashed">
      <CardContent className="flex flex-col items-center gap-3 py-16 text-center">
        <div className="flex size-12 items-center justify-center rounded-full bg-accent">
          <BookOpen className="size-6 text-primary" />
        </div>
        <div>
          <div className="font-medium">No courses yet</div>
          <p className="text-sm text-muted-foreground">
            Create one to start rewarding your students.
          </p>
        </div>
        <NewCourseDialog />
      </CardContent>
    </Card>
  )
}

function NewCourseDialog() {
  const [open, setOpen] = useState(false)
  const [unlimited, setUnlimited] = useState(false)
  const create = useCreateClassroom()

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const form = new FormData(e.currentTarget)

    const created = await create.mutateAsync({
      name: String(form.get("name")),
      section: (form.get("section") as string) || null,
      term: (form.get("term") as string) || null,
      description: (form.get("description") as string) || null,
      unlimited_issuance: unlimited,
      initial_neurons: unlimited ? 0 : Number(form.get("initial_neurons") || 0),
      // A course is usable the moment it is created; nobody wants a
      // second "publish" step before their first class.
      status: "ACTIVE",
    })

    if (created) setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus className="size-4" />
          New course
        </Button>
      </DialogTrigger>

      <DialogContent>
        <form onSubmit={onSubmit}>
          <DialogHeader>
            <DialogTitle>New course</DialogTitle>
            <DialogDescription>
              Students will join with the invite code generated here.
            </DialogDescription>
          </DialogHeader>

          <div className="grid gap-4 py-4">
            <div className="space-y-2">
              <Label htmlFor="name">Course name</Label>
              <Input id="name" name="name" required autoFocus placeholder="Calculus I" />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="section">Section</Label>
                <Input id="section" name="section" placeholder="A" />
              </div>
              <div className="space-y-2">
                <Label htmlFor="term">Term</Label>
                <Input id="term" name="term" placeholder="2026-1" />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea id="description" name="description" rows={2} />
            </div>

            <div className="flex items-center justify-between rounded-lg border p-3">
              <div className="pr-4">
                <Label htmlFor="unlimited" className="text-sm">
                  Unlimited issuance
                </Label>
                <p className="text-xs text-muted-foreground">
                  Grant without drawing down a vault.
                </p>
              </div>
              <Switch id="unlimited" checked={unlimited} onCheckedChange={setUnlimited} />
            </div>

            {!unlimited && (
              <div className="space-y-2">
                <Label htmlFor="initial_neurons">Starting vault</Label>
                <Input
                  id="initial_neurons"
                  name="initial_neurons"
                  type="number"
                  min={0}
                  defaultValue={1000}
                />
                <p className="text-xs text-muted-foreground">
                  You can top this up at any time.
                </p>
              </div>
            )}
          </div>

          <DialogFooter>
            <Button type="submit" disabled={create.isPending}>
              {create.isPending && <Loader2 className="size-4 animate-spin" />}
              Create course
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
