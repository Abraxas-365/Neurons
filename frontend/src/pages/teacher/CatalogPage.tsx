import { useState } from "react"
import { useParams } from "react-router-dom"
import { Gift, Loader2, Pencil, Plus, Sparkles, Trash2 } from "lucide-react"
import {
  useBenefits,
  useCreateBenefit,
  useCreateReason,
  useDeleteBenefit,
  useDeleteReason,
  useReasons,
  useUpdateBenefit,
  useUpdateReason,
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
import { Tabs, TabsContent, TabsList, TabsTrigger } from "@/components/ui/tabs"
import { Textarea } from "@/components/ui/textarea"
import type { Benefit, ReasonScope, Reason } from "@/lib/api/types"

export function CatalogPage() {
  const { classroomId = "" } = useParams()

  return (
    <div className="space-y-5">
      <div>
        <h1 className="text-xl font-semibold tracking-tight">Catalog</h1>
        <p className="text-sm text-muted-foreground">
          Reasons are why neurons are given. Benefits are what they buy.
        </p>
      </div>

      <Tabs defaultValue="reasons">
        <TabsList className="mb-4">
          <TabsTrigger value="reasons">
            <Sparkles className="size-4" />
            Reasons
          </TabsTrigger>
          <TabsTrigger value="benefits">
            <Gift className="size-4" />
            Benefits
          </TabsTrigger>
        </TabsList>
        <TabsContent value="reasons">
          <ReasonsTab classroomId={classroomId} />
        </TabsContent>
        <TabsContent value="benefits">
          <BenefitsTab classroomId={classroomId} />
        </TabsContent>
      </Tabs>
    </div>
  )
}

function ReasonsTab({ classroomId }: { classroomId: string }) {
  const { data: reasons, isLoading } = useReasons(classroomId)
  const [editing, setEditing] = useState<Reason | null>(null)
  const [open, setOpen] = useState(false)
  const remove = useDeleteReason(classroomId)

  return (
    <div className="space-y-4">
      <Button
        onClick={() => {
          setEditing(null)
          setOpen(true)
        }}
      >
        <Plus className="size-4" />
        New reason
      </Button>

      {isLoading ? (
        <Skeleton className="h-40" />
      ) : !reasons?.length ? (
        <EmptyCard text="No reasons yet. Reasons make granting a one-tap action." />
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {reasons.map((r) => (
            <Card key={r.id} className={r.is_active ? undefined : "opacity-60"}>
              <CardContent className="space-y-3">
                <div className="flex items-start gap-3">
                  <span className="text-2xl leading-none">{r.icon ?? "✨"}</span>
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-medium">{r.name}</div>
                    {r.description && (
                      <p className="line-clamp-2 text-xs text-muted-foreground">
                        {r.description}
                      </p>
                    )}
                  </div>
                </div>

                <div className="flex flex-wrap items-center gap-2">
                  {r.suggested_amount != null && <NeuronAmount value={r.suggested_amount} size="sm" />}
                  <Badge variant="outline">{scopeLabel(r.scope)}</Badge>
                  {!r.is_active && <Badge variant="secondary">Hidden</Badge>}
                </div>

                <div className="flex justify-end gap-1">
                  <Button
                    variant="ghost"
                    size="icon"
                    className="size-8"
                    onClick={() => {
                      setEditing(r)
                      setOpen(true)
                    }}
                  >
                    <Pencil className="size-4" />
                  </Button>
                  <DeleteButton
                    title={`Delete "${r.name}"?`}
                    description="Past grants keep their reason text — only future use is removed."
                    onConfirm={() => remove.mutate(r.id)}
                  />
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <ReasonDialog
        classroomId={classroomId}
        reason={editing}
        open={open}
        onOpenChange={setOpen}
      />
    </div>
  )
}

function ReasonDialog({
  classroomId,
  reason,
  open,
  onOpenChange,
}: {
  classroomId: string
  reason: Reason | null
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const create = useCreateReason(classroomId)
  const update = useUpdateReason(classroomId)
  const [scope, setScope] = useState<ReasonScope>("BOTH")
  const [active, setActive] = useState(true)

  // Remounting on the edited row resets the uncontrolled inputs for free.
  const key = reason?.id ?? "new"

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = new FormData(e.currentTarget)
    const amountRaw = String(f.get("suggested_amount") ?? "").trim()

    const input = {
      name: String(f.get("name")),
      description: (f.get("description") as string) || null,
      icon: (f.get("icon") as string) || null,
      suggested_amount: amountRaw ? Number(amountRaw) : null,
      scope,
      is_active: active,
    }

    if (reason) await update.mutateAsync({ id: reason.id, input })
    else await create.mutateAsync(input)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <form key={key} onSubmit={onSubmit}>
          <DialogHeader>
            <DialogTitle>{reason ? "Edit reason" : "New reason"}</DialogTitle>
            <DialogDescription>
              The suggested amount pre-fills the grant screen.
            </DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="grid grid-cols-[4rem_1fr] gap-3">
              <div className="space-y-2">
                <Label htmlFor="icon">Icon</Label>
                <Input
                  id="icon"
                  name="icon"
                  maxLength={4}
                  defaultValue={reason?.icon ?? ""}
                  className="text-center text-lg"
                  placeholder="🎯"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="name">Name</Label>
                <Input
                  id="name"
                  name="name"
                  required
                  defaultValue={reason?.name}
                  placeholder="Great answer in class"
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="description">Description</Label>
              <Textarea id="description" name="description" rows={2} defaultValue={reason?.description ?? ""} />
            </div>

            <div className="grid grid-cols-2 gap-3">
              <div className="space-y-2">
                <Label htmlFor="suggested_amount">Suggested amount</Label>
                <Input
                  id="suggested_amount"
                  name="suggested_amount"
                  type="number"
                  min={1}
                  defaultValue={reason?.suggested_amount ?? ""}
                  className="tabular"
                />
              </div>
              <div className="space-y-2">
                <Label>Applies to</Label>
                <ScopeSelect value={scope} onChange={setScope} />
              </div>
            </div>

            <div className="flex items-center justify-between rounded-lg border p-3">
              <Label htmlFor="active">Visible when granting</Label>
              <Switch id="active" checked={active} onCheckedChange={setActive} />
            </div>
          </div>

          <DialogFooter>
            <Button type="submit" disabled={create.isPending || update.isPending}>
              {(create.isPending || update.isPending) && (
                <Loader2 className="size-4 animate-spin" />
              )}
              Save
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function BenefitsTab({ classroomId }: { classroomId: string }) {
  const { data: benefits, isLoading } = useBenefits(classroomId)
  const [editing, setEditing] = useState<Benefit | null>(null)
  const [open, setOpen] = useState(false)
  const remove = useDeleteBenefit(classroomId)

  return (
    <div className="space-y-4">
      <Button
        onClick={() => {
          setEditing(null)
          setOpen(true)
        }}
      >
        <Plus className="size-4" />
        New benefit
      </Button>

      {isLoading ? (
        <Skeleton className="h-40" />
      ) : !benefits?.length ? (
        <EmptyCard text="No benefits yet. Students need something worth saving for." />
      ) : (
        <div className="grid gap-3 sm:grid-cols-2 lg:grid-cols-3">
          {benefits.map((b) => (
            <Card key={b.id} className={b.is_active ? undefined : "opacity-60"}>
              <CardContent className="space-y-3">
                <div className="flex items-start gap-3">
                  <span className="text-2xl leading-none">{b.icon ?? "🎁"}</span>
                  <div className="min-w-0 flex-1">
                    <div className="truncate font-medium">{b.name}</div>
                    {b.description && (
                      <p className="line-clamp-2 text-xs text-muted-foreground">
                        {b.description}
                      </p>
                    )}
                  </div>
                </div>

                <div className="flex flex-wrap items-center gap-2">
                  {b.cost == null ? (
                    <Badge variant="outline">Student chooses amount</Badge>
                  ) : (
                    <NeuronAmount value={b.cost} size="sm" />
                  )}
                  {b.requires_approval && <Badge variant="secondary">Needs approval</Badge>}
                  {b.max_uses_per_student != null && (
                    <Badge variant="outline">Max {b.max_uses_per_student}/student</Badge>
                  )}
                  {!b.is_active && <Badge variant="secondary">Hidden</Badge>}
                </div>

                <div className="flex items-center justify-between">
                  <span className="text-xs text-muted-foreground">
                    Redeemed {b.uses_count}×
                  </span>
                  <div className="flex gap-1">
                    <Button
                      variant="ghost"
                      size="icon"
                      className="size-8"
                      onClick={() => {
                        setEditing(b)
                        setOpen(true)
                      }}
                    >
                      <Pencil className="size-4" />
                    </Button>
                    <DeleteButton
                      title={`Delete "${b.name}"?`}
                      description="Past redemptions stay in the ledger."
                      onConfirm={() => remove.mutate(b.id)}
                    />
                  </div>
                </div>
              </CardContent>
            </Card>
          ))}
        </div>
      )}

      <BenefitDialog
        classroomId={classroomId}
        benefit={editing}
        open={open}
        onOpenChange={setOpen}
      />
    </div>
  )
}

function BenefitDialog({
  classroomId,
  benefit,
  open,
  onOpenChange,
}: {
  classroomId: string
  benefit: Benefit | null
  open: boolean
  onOpenChange: (v: boolean) => void
}) {
  const create = useCreateBenefit(classroomId)
  const update = useUpdateBenefit(classroomId)
  const [freeAmount, setFreeAmount] = useState(false)
  const [approval, setApproval] = useState(false)
  const [active, setActive] = useState(true)

  const key = benefit?.id ?? "new"

  async function onSubmit(e: React.FormEvent<HTMLFormElement>) {
    e.preventDefault()
    const f = new FormData(e.currentTarget)
    const maxUses = String(f.get("max_uses_per_student") ?? "").trim()

    const input = {
      name: String(f.get("name")),
      description: (f.get("description") as string) || null,
      icon: (f.get("icon") as string) || null,
      // A null cost means the student decides how many neurons to spend
      // (HU-063) — used for things like "buy back a late submission".
      cost: freeAmount ? null : Number(f.get("cost") || 0),
      max_uses_per_student: maxUses ? Number(maxUses) : null,
      requires_approval: approval,
      is_active: active,
    }

    if (benefit) await update.mutateAsync({ id: benefit.id, input })
    else await create.mutateAsync(input)
    onOpenChange(false)
  }

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="sm:max-w-md">
        <form key={key} onSubmit={onSubmit}>
          <DialogHeader>
            <DialogTitle>{benefit ? "Edit benefit" : "New benefit"}</DialogTitle>
            <DialogDescription>What students can trade their neurons for.</DialogDescription>
          </DialogHeader>

          <div className="space-y-4 py-4">
            <div className="grid grid-cols-[4rem_1fr] gap-3">
              <div className="space-y-2">
                <Label htmlFor="b-icon">Icon</Label>
                <Input
                  id="b-icon"
                  name="icon"
                  maxLength={4}
                  defaultValue={benefit?.icon ?? ""}
                  className="text-center text-lg"
                  placeholder="🎁"
                />
              </div>
              <div className="space-y-2">
                <Label htmlFor="b-name">Name</Label>
                <Input
                  id="b-name"
                  name="name"
                  required
                  defaultValue={benefit?.name}
                  placeholder="+1 point on the next quiz"
                />
              </div>
            </div>

            <div className="space-y-2">
              <Label htmlFor="b-desc">Description</Label>
              <Textarea id="b-desc" name="description" rows={2} defaultValue={benefit?.description ?? ""} />
            </div>

            <div className="flex items-center justify-between rounded-lg border p-3">
              <div className="pr-4">
                <Label htmlFor="free">Student chooses the amount</Label>
                <p className="text-xs text-muted-foreground">
                  For open-ended perks with no fixed price.
                </p>
              </div>
              <Switch id="free" checked={freeAmount} onCheckedChange={setFreeAmount} />
            </div>

            <div className="grid grid-cols-2 gap-3">
              {!freeAmount && (
                <div className="space-y-2">
                  <Label htmlFor="cost">Cost</Label>
                  <Input
                    id="cost"
                    name="cost"
                    type="number"
                    min={1}
                    required
                    defaultValue={benefit?.cost ?? 10}
                    className="tabular"
                  />
                </div>
              )}
              <div className="space-y-2">
                <Label htmlFor="max_uses_per_student">Max per student</Label>
                <Input
                  id="max_uses_per_student"
                  name="max_uses_per_student"
                  type="number"
                  min={1}
                  placeholder="Unlimited"
                  defaultValue={benefit?.max_uses_per_student ?? ""}
                  className="tabular"
                />
              </div>
            </div>

            <div className="flex items-center justify-between rounded-lg border p-3">
              <Label htmlFor="approval">Requires my approval</Label>
              <Switch id="approval" checked={approval} onCheckedChange={setApproval} />
            </div>

            <div className="flex items-center justify-between rounded-lg border p-3">
              <Label htmlFor="b-active">Visible to students</Label>
              <Switch id="b-active" checked={active} onCheckedChange={setActive} />
            </div>
          </div>

          <DialogFooter>
            <Button type="submit" disabled={create.isPending || update.isPending}>
              {(create.isPending || update.isPending) && (
                <Loader2 className="size-4 animate-spin" />
              )}
              Save
            </Button>
          </DialogFooter>
        </form>
      </DialogContent>
    </Dialog>
  )
}

function ScopeSelect({
  value,
  onChange,
}: {
  value: ReasonScope
  onChange: (v: ReasonScope) => void
}) {
  return (
    <Select value={value} onValueChange={(v) => onChange(v as ReasonScope)}>
      <SelectTrigger>
        <SelectValue />
      </SelectTrigger>
      <SelectContent>
        <SelectItem value="BOTH">Students & teams</SelectItem>
        <SelectItem value="INDIVIDUAL">Students only</SelectItem>
        <SelectItem value="TEAM">Teams only</SelectItem>
      </SelectContent>
    </Select>
  )
}

function scopeLabel(scope: ReasonScope) {
  return scope === "BOTH" ? "Students & teams" : scope === "TEAM" ? "Teams" : "Students"
}

function DeleteButton({
  title,
  description,
  onConfirm,
}: {
  title: string
  description: string
  onConfirm: () => void
}) {
  return (
    <AlertDialog>
      <AlertDialogTrigger asChild>
        <Button variant="ghost" size="icon" className="size-8">
          <Trash2 className="size-4" />
        </Button>
      </AlertDialogTrigger>
      <AlertDialogContent>
        <AlertDialogHeader>
          <AlertDialogTitle>{title}</AlertDialogTitle>
          <AlertDialogDescription>{description}</AlertDialogDescription>
        </AlertDialogHeader>
        <AlertDialogFooter>
          <AlertDialogCancel>Cancel</AlertDialogCancel>
          <AlertDialogAction onClick={onConfirm}>Delete</AlertDialogAction>
        </AlertDialogFooter>
      </AlertDialogContent>
    </AlertDialog>
  )
}

function EmptyCard({ text }: { text: string }) {
  return (
    <Card className="border-dashed">
      <CardContent className="py-14 text-center text-sm text-muted-foreground">{text}</CardContent>
    </Card>
  )
}
