import { Link, Outlet, useLocation } from "react-router-dom"
import { NeuronIcon } from "@/components/neuron-amount"
import { UserMenu } from "@/layouts/UserMenu"
import { cn } from "@/lib/utils"

export function StudentLayout() {
  const { pathname } = useLocation()

  return (
    <div className="min-h-svh bg-background">
      <header className="sticky top-0 z-20 border-b bg-background/80 backdrop-blur">
        <div className="mx-auto flex h-16 max-w-3xl items-center justify-between px-4">
          <Link to="/" className="flex items-center gap-2">
            <NeuronIcon className="size-5 text-primary" />
            <span className="font-semibold tracking-tight">NEURONS</span>
          </Link>
          <UserMenu />
        </div>
      </header>

      <main className="mx-auto max-w-3xl px-4 py-6">
        <Outlet key={pathname} />
      </main>
    </div>
  )
}

export function StudentTab({ to, label }: { to: string; label: string }) {
  const { pathname } = useLocation()
  return (
    <Link
      to={to}
      className={cn(
        "rounded-full px-4 py-2 text-sm transition-colors",
        pathname === to ? "bg-primary text-primary-foreground" : "hover:bg-accent",
      )}
    >
      {label}
    </Link>
  )
}
