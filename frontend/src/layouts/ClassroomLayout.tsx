import { NavLink, Outlet, useParams } from "react-router-dom"
import {
  ArrowLeft,
  BarChart3,
  Gift,
  LayoutDashboard,
  Medal,
  ScanLine,
  Sparkles,
  Tags,
  Users,
  UsersRound,
} from "lucide-react"
import { useClassroom } from "@/hooks/queries"
import { cn } from "@/lib/utils"
import { NeuronAmount, NeuronIcon } from "@/components/neuron-amount"
import { Badge } from "@/components/ui/badge"
import { Skeleton } from "@/components/ui/skeleton"
import { UserMenu } from "./UserMenu"

const nav = [
  { to: "", label: "Dashboard", icon: LayoutDashboard, end: true },
  { to: "grant", label: "Grant", icon: Sparkles },
  { to: "scan", label: "Scan", icon: ScanLine },
  { to: "students", label: "Students", icon: Users },
  { to: "teams", label: "Teams", icon: UsersRound },
  { to: "catalog", label: "Catalog", icon: Tags },
  { to: "medals", label: "Medals", icon: Medal },
  { to: "ledger", label: "Ledger", icon: BarChart3 },
  { to: "redeem", label: "Redeem", icon: Gift },
]

export function ClassroomLayout() {
  const { classroomId } = useParams<{ classroomId: string }>()
  const { data: classroom, isLoading } = useClassroom(classroomId)

  return (
    <div className="flex min-h-svh bg-background">
      <aside className="hidden w-64 shrink-0 flex-col border-r bg-sidebar md:flex">
        <div className="flex h-16 items-center gap-2 border-b px-5">
          <NeuronIcon className="size-5 text-primary" />
          <span className="font-semibold tracking-tight">NEURONS</span>
        </div>

        <div className="border-b px-5 py-4">
          <NavLink
            to="/courses"
            className="mb-3 inline-flex items-center gap-1.5 text-xs text-muted-foreground hover:text-foreground"
          >
            <ArrowLeft className="size-3" />
            All courses
          </NavLink>

          {isLoading || !classroom ? (
            <Skeleton className="h-10 w-full" />
          ) : (
            <>
              <div className="truncate text-sm font-medium">{classroom.name}</div>
              <div className="mt-0.5 flex items-center gap-2">
                <span className="truncate text-xs text-muted-foreground">
                  {[classroom.section, classroom.term].filter(Boolean).join(" · ") ||
                    "No section"}
                </span>
                {classroom.status !== "ACTIVE" && (
                  <Badge variant="outline" className="h-4 px-1 text-[10px]">
                    {classroom.status}
                  </Badge>
                )}
              </div>
            </>
          )}
        </div>

        <nav className="flex-1 space-y-0.5 overflow-y-auto p-3">
          {nav.map(({ to, label, icon: Icon, end }) => (
            <NavLink
              key={to || "dashboard"}
              to={to}
              end={end}
              className={({ isActive }) =>
                cn(
                  "flex items-center gap-3 rounded-md px-3 py-2 text-sm font-medium transition-colors",
                  isActive
                    ? "bg-sidebar-accent text-sidebar-accent-foreground"
                    : "text-muted-foreground hover:bg-sidebar-accent/50 hover:text-foreground",
                )
              }
            >
              <Icon className="size-4" />
              {label}
            </NavLink>
          ))}
        </nav>

        {/* The vault is always visible: a teacher must never grant without
            knowing what is left to give (RN-05). */}
        <div className="border-t p-4">
          <div className="rounded-lg bg-accent/60 p-3">
            <div className="text-xs font-medium text-muted-foreground">Vault</div>
            {isLoading || !classroom ? (
              <Skeleton className="mt-1 h-7 w-24" />
            ) : classroom.unlimited_issuance ? (
              <div className="mt-0.5 flex items-center gap-1.5 text-lg font-semibold text-primary">
                <NeuronIcon className="size-4" />∞
              </div>
            ) : (
              <NeuronAmount value={classroom.vault_balance} size="lg" className="mt-0.5" />
            )}
          </div>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-16 shrink-0 items-center justify-between gap-4 border-b px-4 md:px-6">
          <MobileClassroomTitle name={classroom?.name} />
          <UserMenu />
        </header>

        <main className="flex-1 overflow-y-auto p-4 md:p-6">
          <Outlet />
        </main>

        <MobileNav />
      </div>
    </div>
  )
}

function MobileClassroomTitle({ name }: { name?: string }) {
  return (
    <div className="flex items-center gap-2 md:hidden">
      <NeuronIcon className="size-5 text-primary" />
      <span className="truncate text-sm font-medium">{name ?? "NEURONS"}</span>
    </div>
  )
}

/** On a phone a teacher only needs the three things they do mid-class. */
function MobileNav() {
  const items = nav.filter((n) =>
    ["", "grant", "scan", "students"].includes(n.to),
  )

  return (
    <nav className="flex shrink-0 border-t bg-card md:hidden">
      {items.map(({ to, label, icon: Icon, end }) => (
        <NavLink
          key={to || "dashboard"}
          to={to}
          end={end}
          className={({ isActive }) =>
            cn(
              "flex flex-1 flex-col items-center gap-1 py-2.5 text-[11px] font-medium transition-colors",
              isActive ? "text-primary" : "text-muted-foreground",
            )
          }
        >
          <Icon className="size-5" />
          {label}
        </NavLink>
      ))}
    </nav>
  )
}
