import { useState } from "react"
import { Link } from "react-router-dom"
import { ArrowRight, Loader2, Plus } from "lucide-react"
import { useJoinClassroom, useMyClassrooms } from "@/hooks/queries"
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
import { Input } from "@/components/ui/input"
import { Label } from "@/components/ui/label"
import { Skeleton } from "@/components/ui/skeleton"

export function MyCoursesPage() {
  const { data: courses, isLoading } = useMyClassrooms()

  return (
    <div className="space-y-5">
      <div className="flex items-end justify-between gap-4">
        <div>
          <h1 className="text-xl font-semibold tracking-tight">My courses</h1>
          <p className="text-sm text-muted-foreground">
            Neurons belong to one course and cannot move between them.
          </p>
        </div>
        <JoinDialog />
      </div>

      {isLoading ? (
        <div className="space-y-3">
          {[0, 1].map((i) => (
            <Skeleton key={i} className="h-28" />
          ))}
        </div>
      ) : !courses?.length ? (
        <Card className="border-dashed">
          <CardContent className="flex flex-col items-center gap-3 py-16 text-center">
            <div className="font-medium">You have not joined a course yet</div>
            <p className="max-w-xs text-sm text-muted-foreground">
              Ask your teacher for the invite code.
            </p>
            <JoinDialog />
          </CardContent>
        </Card>
      ) : (
        <div className="space-y-3">
          {courses.map((c) => (
            <Link key={c.id} to={`/wallet/${c.classroom_id}`}>
              <Card className="transition-all hover:border-primary/40 hover:shadow-md">
                <CardContent className="flex items-center gap-4">
                  <div className="flex size-12 shrink-0 items-center justify-center rounded-xl bg-accent text-2xl">
                    {c.icon ?? "📘"}
                  </div>

                  <div className="min-w-0 flex-1">
                    <div className="truncate font-medium">{c.classroom_name}</div>
                    <div className="truncate text-xs text-muted-foreground">
                      {[c.section, c.term, c.team_name].filter(Boolean).join(" · ") || "—"}
                    </div>
                    {!c.classroom_open && (
                      <Badge variant="secondary" className="mt-1">
                        Closed
                      </Badge>
                    )}
                  </div>

                  <div className="text-right">
                    <NeuronAmount value={c.balance} size="lg" />
                    <div className="text-[11px] text-muted-foreground">
                      {c.total_received} earned · {c.total_returned} spent
                    </div>
                  </div>

                  <ArrowRight className="size-4 shrink-0 text-muted-foreground" />
                </CardContent>
              </Card>
            </Link>
          ))}
        </div>
      )}
    </div>
  )
}

function JoinDialog() {
  const [open, setOpen] = useState(false)
  const join = useJoinClassroom()

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const code = String(new FormData(e.currentTarget).get("code") ?? "").trim()
    if (!code) return

    await join.mutateAsync(code.toUpperCase())
    setOpen(false)
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>
        <Button>
          <Plus className="size-4" />
          Join a course
        </Button>
      </DialogTrigger>
      <DialogContent className="sm:max-w-sm">
        <form onSubmit={onSubmit}>
          <DialogHeader>
            <DialogTitle>Join a course</DialogTitle>
            <DialogDescription>Enter the code your teacher gave you.</DialogDescription>
          </DialogHeader>
          <div className="space-y-2 py-4">
            <Label htmlFor="code">Invite code</Label>
            <Input
              id="code"
              name="code"
              autoFocus
              autoCapitalize="characters"
              placeholder="ABC123"
              className="text-center font-mono text-lg tracking-[0.3em] uppercase"
            />
          </div>
          <DialogFooter>
            <Button type="submit" disabled={join.isPending}>
              {join.isPending && <Loader2 className="size-4 animate-spin" />}
              Join
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}
