import { Suspense, lazy } from "react"
import { Navigate, Route, Routes } from "react-router-dom"
import { Loader2 } from "lucide-react"
import { Toaster } from "@/components/ui/sonner"
import { TooltipProvider } from "@/components/ui/tooltip"
import { useAuth } from "@/auth/context"
import { ClassroomLayout } from "@/layouts/ClassroomLayout"
import { StudentLayout } from "@/layouts/StudentLayout"
import { LoginPage } from "@/pages/LoginPage"
import { CoursesPage } from "@/pages/teacher/CoursesPage"
import { GrantPage } from "@/pages/teacher/GrantPage"
import { StudentsPage } from "@/pages/teacher/StudentsPage"
import { TeamsPage } from "@/pages/teacher/TeamsPage"
import { CatalogPage } from "@/pages/teacher/CatalogPage"
import { MedalsPage } from "@/pages/teacher/MedalsPage"
import { LedgerPage } from "@/pages/teacher/LedgerPage"
import { RedeemPage } from "@/pages/teacher/RedeemPage"
import { MyCoursesPage } from "@/pages/student/MyCoursesPage"
import { MyWalletPage } from "@/pages/student/MyWalletPage"

// The charting and camera libraries are large and each is used on exactly one
// screen, so they load only when a teacher actually opens that screen.
const DashboardPage = lazy(() =>
  import("@/pages/teacher/DashboardPage").then((m) => ({ default: m.DashboardPage })),
)
const ScanPage = lazy(() =>
  import("@/pages/teacher/ScanPage").then((m) => ({ default: m.ScanPage })),
)

export default function App() {
  const { isAuthenticated, role } = useAuth()

  if (!isAuthenticated) {
    return (
      <TooltipProvider delayDuration={200}>
        <Routes>
          <Route path="/login" element={<LoginPage />} />
          <Route path="*" element={<Navigate to="/login" replace />} />
        </Routes>
        <Toaster position="top-center" richColors />
      </TooltipProvider>
    )
  }

  return (
    <TooltipProvider delayDuration={200}>
      <Suspense fallback={<PageLoading />}>
        {role === "teacher" ? <TeacherRoutes /> : <StudentRoutes />}
      </Suspense>
      <Toaster position="top-center" richColors />
    </TooltipProvider>
  )
}

function PageLoading() {
  return (
    <div className="flex min-h-svh items-center justify-center">
      <Loader2 className="size-6 animate-spin text-muted-foreground" />
    </div>
  )
}

function TeacherRoutes() {
  return (
    <Routes>
      <Route path="/courses" element={<CoursesPage />} />

      <Route path="/courses/:classroomId" element={<ClassroomLayout />}>
        <Route index element={<DashboardPage />} />
        <Route path="grant" element={<GrantPage />} />
        <Route path="scan" element={<ScanPage />} />
        <Route path="students" element={<StudentsPage />} />
        <Route path="teams" element={<TeamsPage />} />
        <Route path="catalog" element={<CatalogPage />} />
        <Route path="medals" element={<MedalsPage />} />
        <Route path="ledger" element={<LedgerPage />} />
        <Route path="redeem" element={<RedeemPage />} />
      </Route>

      <Route path="/login" element={<Navigate to="/courses" replace />} />
      <Route path="*" element={<Navigate to="/courses" replace />} />
    </Routes>
  )
}

function StudentRoutes() {
  return (
    <Routes>
      <Route path="/" element={<StudentLayout />}>
        <Route index element={<MyCoursesPage />} />
        <Route path="wallet/:classroomId" element={<MyWalletPage />} />
      </Route>

      <Route path="/login" element={<Navigate to="/" replace />} />
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  )
}
