import {
  useMutation,
  useQuery,
  useQueryClient,
  type UseQueryOptions,
} from "@tanstack/react-query"
import { toast } from "sonner"
import { ApiError } from "@/lib/api/client"
import {
  benefitApi,
  classroomApi,
  enrollmentApi,
  ledgerApi,
  medalApi,
  meApi,
  qrApi,
  reasonApi,
  teamApi,
} from "@/lib/api/endpoints"
import type {
  AwardMedalInput,
  CreateBenefitInput,
  CreateClassroomInput,
  CreateMedalInput,
  CreateReasonInput,
  CreateTeamInput,
  RandomizeTeamsInput,
  GrantInput,
  HistoryFilter,
  InviteStudentEntry,
  RedeemInput,
  RosterFilter,
  TeamGrantInput,
  UpdateBenefitInput,
  UpdateClassroomInput,
  UpdateReasonInput,
} from "@/lib/api/types"

/**
 * Query keys are namespaced per classroom so invalidating one course never
 * refetches another (RN-01: courses are independent).
 */
export const qk = {
  classrooms: ["classrooms"] as const,
  classroom: (id: string) => ["classroom", id] as const,
  roster: (id: string, filter?: RosterFilter) => ["roster", id, filter] as const,
  teams: (id: string) => ["teams", id] as const,
  team: (id: string) => ["team", id] as const,
  reasons: (id: string) => ["reasons", id] as const,
  benefits: (id: string) => ["benefits", id] as const,
  medals: (id: string) => ["medals", id] as const,
  medalAwards: (id: string) => ["medal-awards", id] as const,
  stats: (id: string) => ["stats", id] as const,
  history: (id: string, filter?: HistoryFilter) => ["history", id, filter] as const,
  ranking: (id: string) => ["ranking", id] as const,
  reasonUsage: (id: string) => ["reason-usage", id] as const,
  myClassrooms: ["me", "classrooms"] as const,
  myEnrollment: (id: string) => ["me", "enrollment", id] as const,
  myHistory: (id: string) => ["me", "history", id] as const,
  myMedals: (id: string) => ["me", "medals", id] as const,
  myBenefits: (id: string) => ["me", "benefits", id] as const,
}

/** Surfaces the backend's message; the errx code carries the real meaning. */
function notifyError(error: unknown) {
  if (error instanceof ApiError) {
    toast.error(error.message)
    return
  }
  toast.error("Something went wrong.")
}

// ---------------------------------------------------------------------------
// Teacher: classrooms
// ---------------------------------------------------------------------------

export function useClassrooms() {
  return useQuery({ queryKey: qk.classrooms, queryFn: classroomApi.list })
}

export function useClassroom(id: string | undefined) {
  return useQuery({
    queryKey: qk.classroom(id!),
    queryFn: () => classroomApi.get(id!),
    enabled: Boolean(id),
  })
}

export function useCreateClassroom() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateClassroomInput) => classroomApi.create(input),
    onSuccess: (created) => {
      qc.invalidateQueries({ queryKey: qk.classrooms })
      toast.success(`"${created.name}" is ready.`)
    },
    onError: notifyError,
  })
}

export function useUpdateClassroom(id: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: UpdateClassroomInput) => classroomApi.update(id, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.classroom(id) })
      qc.invalidateQueries({ queryKey: qk.classrooms })
      toast.success("Course updated.")
    },
    onError: notifyError,
  })
}

// ---------------------------------------------------------------------------
// Teacher: roster
// ---------------------------------------------------------------------------

export function useRoster(classroomId: string | undefined, filter: RosterFilter = {}) {
  return useQuery({
    queryKey: qk.roster(classroomId!, filter),
    queryFn: () => enrollmentApi.roster(classroomId!, filter),
    enabled: Boolean(classroomId),
    placeholderData: (prev) => prev,
  })
}

export function useInviteStudents(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (students: InviteStudentEntry[]) => enrollmentApi.invite(classroomId, students),
    onSuccess: (results) => {
      qc.invalidateQueries({ queryKey: ["roster", classroomId] })
      const enrolled = results.filter((r) => r.status === "ENROLLED").length
      const failed = results.filter((r) => r.status === "ERROR").length
      // A partial success is the common case when a pasted list contains
      // people who never created an account, so report both halves.
      toast.success(
        failed > 0
          ? `${enrolled} enrolled, ${failed} could not be added.`
          : `${enrolled} student(s) enrolled.`,
      )
    },
    onError: notifyError,
  })
}

export function useSetStudentTeam(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ enrollmentId, teamId }: { enrollmentId: string; teamId: string | null }) =>
      enrollmentApi.setTeam(enrollmentId, teamId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["roster", classroomId] })
      qc.invalidateQueries({ queryKey: qk.teams(classroomId) })
    },
    onError: notifyError,
  })
}

export function useApproveEnrollment(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (enrollmentId: string) => enrollmentApi.approve(enrollmentId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["roster", classroomId] })
      toast.success("Student approved.")
    },
    onError: notifyError,
  })
}

export function useWithdrawEnrollment(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (enrollmentId: string) => enrollmentApi.withdraw(enrollmentId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: ["roster", classroomId] })
      toast.success("Student withdrawn.")
    },
    onError: notifyError,
  })
}

// ---------------------------------------------------------------------------
// Teacher: teams
// ---------------------------------------------------------------------------

export function useTeams(classroomId: string | undefined) {
  return useQuery({
    queryKey: qk.teams(classroomId!),
    queryFn: () => teamApi.list(classroomId!),
    enabled: Boolean(classroomId),
  })
}

export function useTeam(teamId: string | undefined) {
  return useQuery({
    queryKey: qk.team(teamId!),
    queryFn: () => teamApi.detail(teamId!),
    enabled: Boolean(teamId),
  })
}

export function useCreateTeam(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateTeamInput) => teamApi.create(classroomId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.teams(classroomId) })
      qc.invalidateQueries({ queryKey: ["roster", classroomId] })
      toast.success("Team created.")
    },
    onError: notifyError,
  })
}

export function useDeleteTeam(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (teamId: string) => teamApi.remove(teamId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.teams(classroomId) })
      qc.invalidateQueries({ queryKey: ["roster", classroomId] })
      toast.success("Team deleted.")
    },
    onError: notifyError,
  })
}

export function useRandomizeTeams(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: RandomizeTeamsInput) => teamApi.randomize(classroomId, input),
    onSuccess: (res) => {
      // A preview is only a proposal; nothing changed server-side yet.
      if (!res.persisted) return
      qc.invalidateQueries({ queryKey: qk.teams(classroomId) })
      qc.invalidateQueries({ queryKey: ["roster", classroomId] })
    },
    onError: notifyError,
  })
}

// ---------------------------------------------------------------------------
// Teacher: catalog
// ---------------------------------------------------------------------------

export function useReasons(classroomId: string | undefined, activeOnly = false) {
  return useQuery({
    queryKey: [...qk.reasons(classroomId!), activeOnly],
    queryFn: () => reasonApi.list(classroomId!, activeOnly),
    enabled: Boolean(classroomId),
  })
}

export function useCreateReason(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateReasonInput) => reasonApi.create(classroomId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.reasons(classroomId) })
      toast.success("Reason added.")
    },
    onError: notifyError,
  })
}

export function useUpdateReason(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateReasonInput }) =>
      reasonApi.update(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.reasons(classroomId) }),
    onError: notifyError,
  })
}

export function useDeleteReason(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => reasonApi.remove(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.reasons(classroomId) })
      toast.success("Reason removed.")
    },
    onError: notifyError,
  })
}

export function useBenefits(classroomId: string | undefined, activeOnly = false) {
  return useQuery({
    queryKey: [...qk.benefits(classroomId!), activeOnly],
    queryFn: () => benefitApi.list(classroomId!, activeOnly),
    enabled: Boolean(classroomId),
  })
}

export function useCreateBenefit(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateBenefitInput) => benefitApi.create(classroomId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.benefits(classroomId) })
      toast.success("Benefit added.")
    },
    onError: notifyError,
  })
}

export function useUpdateBenefit(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, input }: { id: string; input: UpdateBenefitInput }) =>
      benefitApi.update(id, input),
    onSuccess: () => qc.invalidateQueries({ queryKey: qk.benefits(classroomId) }),
    onError: notifyError,
  })
}

export function useDeleteBenefit(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => benefitApi.remove(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.benefits(classroomId) })
      toast.success("Benefit removed.")
    },
    onError: notifyError,
  })
}

export function useMedals(classroomId: string | undefined, activeOnly = false) {
  return useQuery({
    queryKey: [...qk.medals(classroomId!), activeOnly],
    queryFn: () => medalApi.list(classroomId!, activeOnly),
    enabled: Boolean(classroomId),
  })
}

export function useCreateMedal(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: CreateMedalInput) => medalApi.create(classroomId, input),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.medals(classroomId) })
      toast.success("Medal created.")
    },
    onError: notifyError,
  })
}

export function useDeleteMedal(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (id: string) => medalApi.remove(id),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.medals(classroomId) })
      toast.success("Medal removed.")
    },
    onError: notifyError,
  })
}

export function useMedalAwards(classroomId: string | undefined) {
  return useQuery({
    queryKey: qk.medalAwards(classroomId!),
    queryFn: () => medalApi.awards(classroomId!),
    enabled: Boolean(classroomId),
  })
}

export function useAwardMedal(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ medalId, input }: { medalId: string; input: AwardMedalInput }) =>
      medalApi.award(medalId, input),
    onSuccess: (awards) => {
      qc.invalidateQueries({ queryKey: qk.medalAwards(classroomId) })
      qc.invalidateQueries({ queryKey: ["roster", classroomId] })
      toast.success(`Medal awarded to ${awards.length} recipient(s).`)
    },
    onError: notifyError,
  })
}

export function useRevokeAward(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (awardId: string) => medalApi.revoke(awardId),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.medalAwards(classroomId) })
      qc.invalidateQueries({ queryKey: ["roster", classroomId] })
      toast.success("Award revoked.")
    },
    onError: notifyError,
  })
}

// ---------------------------------------------------------------------------
// Teacher: the neuron economy
// ---------------------------------------------------------------------------

export function useStats(classroomId: string | undefined) {
  return useQuery({
    queryKey: qk.stats(classroomId!),
    queryFn: () => ledgerApi.stats(classroomId!),
    enabled: Boolean(classroomId),
  })
}

export function useHistory(classroomId: string | undefined, filter: HistoryFilter = {}) {
  return useQuery({
    queryKey: qk.history(classroomId!, filter),
    queryFn: () => ledgerApi.history(classroomId!, filter),
    enabled: Boolean(classroomId),
    placeholderData: (prev) => prev,
  })
}

export function useRanking(classroomId: string | undefined, limit = 20) {
  return useQuery({
    queryKey: [...qk.ranking(classroomId!), limit],
    queryFn: () => ledgerApi.ranking(classroomId!, limit),
    enabled: Boolean(classroomId),
  })
}

export function useReasonUsage(classroomId: string | undefined) {
  return useQuery({
    queryKey: qk.reasonUsage(classroomId!),
    queryFn: () => ledgerApi.reasonUsage(classroomId!),
    enabled: Boolean(classroomId),
  })
}

/**
 * Every mutation that moves neurons must refresh the vault, the roster and the
 * ledger together — they are three views of one number (RN-09).
 */
function invalidateEconomy(qc: ReturnType<typeof useQueryClient>, classroomId: string) {
  qc.invalidateQueries({ queryKey: qk.stats(classroomId) })
  qc.invalidateQueries({ queryKey: qk.classroom(classroomId) })
  qc.invalidateQueries({ queryKey: ["roster", classroomId] })
  qc.invalidateQueries({ queryKey: ["history", classroomId] })
  qc.invalidateQueries({ queryKey: qk.ranking(classroomId) })
  qc.invalidateQueries({ queryKey: qk.reasonUsage(classroomId) })
}

export function useGrant(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: GrantInput) => ledgerApi.grant(classroomId, input),
    onSuccess: () => invalidateEconomy(qc, classroomId),
    // Deliberately silent: the caller decides what to show, because a
    // CONFIRMATION_REQUIRED rejection is a prompt, not a failure (§11.9).
  })
}

export function useTeamGrant(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: TeamGrantInput) => ledgerApi.grantToTeam(classroomId, input),
    onSuccess: () => invalidateEconomy(qc, classroomId),
  })
}

export function useRedeem(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (input: RedeemInput) => ledgerApi.redeem(classroomId, input),
    onSuccess: () => {
      invalidateEconomy(qc, classroomId)
      qc.invalidateQueries({ queryKey: qk.benefits(classroomId) })
      toast.success("Benefit redeemed.")
    },
    onError: notifyError,
  })
}

export function useTopup(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ amount, notes }: { amount: number; notes?: string }) =>
      ledgerApi.topup(classroomId, amount, notes),
    onSuccess: () => {
      invalidateEconomy(qc, classroomId)
      toast.success("Vault topped up.")
    },
    onError: notifyError,
  })
}

export function useReverse(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: ({ id, reason }: { id: string; reason?: string }) =>
      ledgerApi.reverse(id, reason),
    onSuccess: () => {
      invalidateEconomy(qc, classroomId)
      toast.success("Movement reversed.")
    },
    onError: notifyError,
  })
}

export function useScanQR(classroomId: string) {
  return useMutation({
    mutationFn: (code: string) => qrApi.scan(classroomId, code),
  })
}

// ---------------------------------------------------------------------------
// Student self-service
// ---------------------------------------------------------------------------

export function useMyClassrooms() {
  return useQuery({ queryKey: qk.myClassrooms, queryFn: meApi.classrooms })
}

export function useMyEnrollment(
  classroomId: string | undefined,
  options?: Partial<UseQueryOptions<Awaited<ReturnType<typeof meApi.enrollment>>>>,
) {
  return useQuery({
    queryKey: qk.myEnrollment(classroomId!),
    queryFn: () => meApi.enrollment(classroomId!),
    enabled: Boolean(classroomId),
    ...options,
  })
}

export function useMyHistory(classroomId: string | undefined, filter: HistoryFilter = {}) {
  return useQuery({
    queryKey: [...qk.myHistory(classroomId!), filter],
    queryFn: () => meApi.transactions(classroomId!, filter),
    enabled: Boolean(classroomId),
  })
}

export function useMyMedals(classroomId: string | undefined) {
  return useQuery({
    queryKey: qk.myMedals(classroomId!),
    queryFn: () => meApi.medals(classroomId!),
    enabled: Boolean(classroomId),
  })
}

export function useMyBenefits(classroomId: string | undefined) {
  return useQuery({
    queryKey: qk.myBenefits(classroomId!),
    queryFn: () => meApi.benefits(classroomId!),
    enabled: Boolean(classroomId),
  })
}

/**
 * The student's rotating identity code (RN-13). It is only minted while the
 * dialog is open, is never cached, and is refetched on demand — a stale code in
 * a query cache would be a code that no longer works.
 */
export function useMyQRToken(classroomId: string, enabled: boolean) {
  return useQuery({
    queryKey: [...qk.myEnrollment(classroomId), "qr"],
    queryFn: () => meApi.issueQR(classroomId),
    enabled,
    gcTime: 0,
    staleTime: 0,
    refetchOnWindowFocus: false,
  })
}

export function useJoinClassroom() {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (inviteCode: string) => meApi.join(inviteCode),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.myClassrooms })
      toast.success("You joined the course.")
    },
    onError: notifyError,
  })
}

export function useClaimQR(classroomId: string) {
  const qc = useQueryClient()
  return useMutation({
    mutationFn: (code: string) => meApi.claimQR(classroomId, code),
    onSuccess: () => {
      qc.invalidateQueries({ queryKey: qk.myEnrollment(classroomId) })
      qc.invalidateQueries({ queryKey: qk.myHistory(classroomId) })
    },
    onError: notifyError,
  })
}
