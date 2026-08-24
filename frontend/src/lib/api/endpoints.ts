import { API, http } from "./client"
import type {
  AwardMedalInput,
  Batch,
  Benefit,
  Classroom,
  ClassroomStats,
  ClassroomTeacher,
  CreateBenefitInput,
  CreateClassroomInput,
  CreateMedalInput,
  CreateReasonInput,
  CreateTeamInput,
  GrantInput,
  GrantResult,
  HistoryFilter,
  IssueGrantQRInput,
  IssuedQR,
  JoinResult,
  Medal,
  MedalAward,
  MyEnrollment,
  Paginated,
  RandomizeResult,
  RandomizeTeamsInput,
  RankingEntry,
  Reason,
  ReasonUsage,
  RedeemInput,
  InviteResult,
  InviteStudentEntry,
  RosterEntry,
  RosterFilter,
  ScannedStudent,
  StudentBenefit,
  Team,
  TeamDetail,
  TeamGrantInput,
  TenantOption,
  TokenResponse,
  Transaction,
  UserDetails,
  UpdateBenefitInput,
  UpdateClassroomInput,
  UpdateMedalInput,
  UpdateReasonInput,
} from "./types"

// ---------------------------------------------------------------------------
// Auth (passwordless OTP — these live outside /api/v1)
// ---------------------------------------------------------------------------

/**
 * Passwordless OTP. These sit outside /api/v1 because they are what mints the
 * token everything else needs.
 */
const AUTH = "/auth/passwordless"

export const authApi = {
  /** Which organizations does this email belong to? */
  tenants: (email: string) =>
    http
      .post<{ email: string; tenants: TenantOption[]; count: number }>(`${AUTH}/tenants`, {
        email,
      })
      .then((r) => r.tenants),

  initiateLogin: (email: string, tenant_id: string) =>
    http.post<{ message: string; email: string; expires_in_seconds: number }>(
      `${AUTH}/login/initiate`,
      { email, tenant_id },
    ),

  verifyLogin: (email: string, code: string, tenant_id: string) =>
    http.post<TokenResponse>(`${AUTH}/login/verify`, { email, code, tenant_id }),

  resendOtp: (email: string, tenant_id: string, purpose: "signup" | "login" = "login") =>
    http.post<{ message: string }>(`${AUTH}/resend-otp`, {
      email,
      tenant_id,
      purpose,
    }),

  /** Trades the refresh token for a new access token (OAuth handler group). */
  refresh: (refresh_token: string) =>
    http.post<TokenResponse>("/auth/refresh", { refresh_token }),

  /** Kicks off Google OAuth: backend replies with the URL to redirect to. */
  googleLogin: () =>
    http
      .post<{ auth_url: string; state: string }>("/auth/login", { provider: "google" })
      .then((r) => r.auth_url),

  /** Resolves the current session from the access token (set after OAuth callback). */
  me: () =>
    http.get<{ user: UserDetails; tenant: { id: string; company_name: string } }>("/auth/me"),
}

// ---------------------------------------------------------------------------
// Classrooms
// ---------------------------------------------------------------------------

export const classroomApi = {
  /**
   * The API paginates classrooms, but a teacher owns a handful of courses and the
   * picker shows them all, so the envelope is unwrapped here rather than leaking
   * a page cursor into every screen that just wants the list.
   */
  list: () =>
    http.get<Paginated<Classroom>>(`${API}/classrooms`).then((r) => r.items),

  get: (id: string) => http.get<Classroom>(`${API}/classrooms/${id}`),

  /** Preview a course from its invite code before committing to join. */
  byInviteCode: (code: string) =>
    http.get<{ id: string; name: string; section: string | null; term: string | null }>(
      `${API}/classrooms/by-code/${code}`,
    ),

  create: (input: CreateClassroomInput) =>
    http.post<Classroom>(`${API}/classrooms`, input),

  update: (id: string, input: UpdateClassroomInput) =>
    http.put<Classroom>(`${API}/classrooms/${id}`, input),

  remove: (id: string) => http.delete<void>(`${API}/classrooms/${id}`),

  teachers: (id: string) =>
    http
      .get<{ items: ClassroomTeacher[] }>(`${API}/classrooms/${id}/teachers`)
      .then((r) => r.items),

  addTeacher: (id: string, email: string, role?: string) =>
    http.post<ClassroomTeacher>(`${API}/classrooms/${id}/teachers`, { email, role }),

  removeTeacher: (id: string, userId: string) =>
    http.delete<void>(`${API}/classrooms/${id}/teachers/${userId}`),
}

// ---------------------------------------------------------------------------
// Enrollments / roster
// ---------------------------------------------------------------------------

export const enrollmentApi = {
  roster: (classroomId: string, filter: RosterFilter = {}) =>
    http.get<Paginated<RosterEntry>>(
      `${API}/classrooms/${classroomId}/students`,
      filter as Record<string, unknown>,
    ),

  /**
   * The API enrolls each row independently and reports a per-row outcome, so a
   * bad email in a pasted list never rejects the whole batch (HU-021).
   */
  invite: (classroomId: string, students: InviteStudentEntry[]) =>
    http
      .post<{ results: InviteResult[] }>(`${API}/classrooms/${classroomId}/students`, {
        students,
      })
      .then((r) => r.results),

  update: (enrollmentId: string, input: { student_code?: string | null }) =>
    http.put<RosterEntry>(`${API}/enrollments/${enrollmentId}`, input),

  approve: (enrollmentId: string) =>
    http.post<RosterEntry>(`${API}/enrollments/${enrollmentId}/approve`),

  withdraw: (enrollmentId: string) =>
    http.post<RosterEntry>(`${API}/enrollments/${enrollmentId}/withdraw`),

  setTeam: (enrollmentId: string, teamId: string | null) =>
    http.put<RosterEntry>(`${API}/enrollments/${enrollmentId}/team`, {
      team_id: teamId,
    }),
}

/** Student self-service. Every route here is scoped to the caller (§10.4). */
export const meApi = {
  join: (inviteCode: string) =>
    http.post<JoinResult>(`${API}/me/classrooms/join`, {
      invite_code: inviteCode,
    }),

  classrooms: () =>
    http.get<{ items: MyEnrollment[] }>(`${API}/me/classrooms`).then((r) => r.items),

  enrollment: (classroomId: string) =>
    http.get<MyEnrollment>(`${API}/me/classrooms/${classroomId}/enrollment`),

  transactions: (classroomId: string, filter: HistoryFilter = {}) =>
    http.get<Paginated<Transaction>>(
      `${API}/me/classrooms/${classroomId}/transactions`,
      filter as Record<string, unknown>,
    ),

  medals: (classroomId: string) =>
    http
      .get<{ items: MedalAward[] }>(`${API}/me/classrooms/${classroomId}/medals`)
      .then((r) => r.items),

  benefits: (classroomId: string) =>
    http
      .get<{ items: StudentBenefit[] }>(`${API}/me/classrooms/${classroomId}/benefits`)
      .then((r) => r.items),

  team: (teamId: string) => http.get<TeamDetail>(`${API}/me/teams/${teamId}`),

  /** Mint a short-lived code for the teacher to scan (RN-13). */
  issueQR: (classroomId: string) =>
    http.post<IssuedQR>(`${API}/me/classrooms/${classroomId}/qr`),

  /** Claim neurons from a code the teacher projected on screen. */
  claimQR: (classroomId: string, code: string) =>
    http.post<Transaction>(`${API}/me/classrooms/${classroomId}/qr/claim`, { code }),
}

// ---------------------------------------------------------------------------
// Teams
// ---------------------------------------------------------------------------

export const teamApi = {
  list: (classroomId: string) =>
    http
      .get<{ items: Team[] }>(`${API}/classrooms/${classroomId}/teams`)
      .then((r) => r.items),

  create: (classroomId: string, input: CreateTeamInput) =>
    http.post<Team>(`${API}/classrooms/${classroomId}/teams`, input),

  /** HU-034: auto-build balanced teams, optionally as a discardable preview. */
  randomize: (classroomId: string, input: RandomizeTeamsInput) =>
    http.post<RandomizeResult>(
      `${API}/classrooms/${classroomId}/teams/randomize`,
      input,
    ),

  detail: (teamId: string) => http.get<TeamDetail>(`${API}/teams/${teamId}`),

  update: (teamId: string, input: Partial<CreateTeamInput>) =>
    http.put<Team>(`${API}/teams/${teamId}`, input),

  remove: (teamId: string) => http.delete<void>(`${API}/teams/${teamId}`),

  setMembers: (teamId: string, enrollmentIds: string[]) =>
    http.put<TeamDetail>(`${API}/teams/${teamId}/members`, {
      enrollment_ids: enrollmentIds,
    }),

  setCoordinator: (teamId: string, enrollmentId: string | null) =>
    http.put<TeamDetail>(`${API}/teams/${teamId}/coordinator`, {
      enrollment_id: enrollmentId,
    }),
}

// ---------------------------------------------------------------------------
// Catalog
// ---------------------------------------------------------------------------

export const reasonApi = {
  list: (classroomId: string, activeOnly = false) =>
    http
      .get<{ items: Reason[] }>(`${API}/classrooms/${classroomId}/reasons`, {
        active_only: activeOnly,
      })
      .then((r) => r.items),

  create: (classroomId: string, input: CreateReasonInput) =>
    http.post<Reason>(`${API}/classrooms/${classroomId}/reasons`, input),

  update: (id: string, input: UpdateReasonInput) =>
    http.put<Reason>(`${API}/reasons/${id}`, input),

  remove: (id: string) => http.delete<void>(`${API}/reasons/${id}`),
}

export const benefitApi = {
  list: (classroomId: string, activeOnly = false) =>
    http
      .get<{ items: Benefit[] }>(`${API}/classrooms/${classroomId}/benefits`, {
        active_only: activeOnly,
      })
      .then((r) => r.items),

  create: (classroomId: string, input: CreateBenefitInput) =>
    http.post<Benefit>(`${API}/classrooms/${classroomId}/benefits`, input),

  update: (id: string, input: UpdateBenefitInput) =>
    http.put<Benefit>(`${API}/benefits/${id}`, input),

  remove: (id: string) => http.delete<void>(`${API}/benefits/${id}`),
}

export const medalApi = {
  list: (classroomId: string, activeOnly = false) =>
    http
      .get<{ items: Medal[] }>(`${API}/classrooms/${classroomId}/medals`, {
        active_only: activeOnly,
      })
      .then((r) => r.items),

  create: (classroomId: string, input: CreateMedalInput) =>
    http.post<Medal>(`${API}/classrooms/${classroomId}/medals`, input),

  update: (id: string, input: UpdateMedalInput) =>
    http.put<Medal>(`${API}/medals/${id}`, input),

  remove: (id: string) => http.delete<void>(`${API}/medals/${id}`),

  awards: (classroomId: string) =>
    http
      .get<{ items: MedalAward[] }>(`${API}/classrooms/${classroomId}/medals/awards`)
      .then((r) => r.items),

  award: (medalId: string, input: AwardMedalInput) =>
    http.post<MedalAward[]>(`${API}/medals/${medalId}/awards`, input),

  revoke: (awardId: string) =>
    http.delete<void>(`${API}/medal-awards/${awardId}`),
}

// ---------------------------------------------------------------------------
// Ledger
// ---------------------------------------------------------------------------

export const ledgerApi = {
  /** Flows A and D: hand neurons to one or many students. */
  grant: (classroomId: string, input: GrantInput) =>
    http.post<GrantResult>(`${API}/classrooms/${classroomId}/grants`, input),

  /** Flow B: every member receives the full amount, not a split (RN-07). */
  grantToTeam: (classroomId: string, input: TeamGrantInput) =>
    http.post<GrantResult>(`${API}/classrooms/${classroomId}/team-grants`, input),

  /** Flow C: neurons go back to the vault in exchange for a benefit. */
  redeem: (classroomId: string, input: RedeemInput) =>
    http.post<Transaction>(`${API}/classrooms/${classroomId}/redemptions`, input),

  topup: (classroomId: string, amount: number, notes?: string) =>
    http.post<Transaction>(`${API}/classrooms/${classroomId}/vault/topup`, {
      amount,
      notes,
    }),

  history: (classroomId: string, filter: HistoryFilter = {}) =>
    http.get<Paginated<Transaction>>(
      `${API}/classrooms/${classroomId}/transactions`,
      filter as Record<string, unknown>,
    ),

  stats: (classroomId: string) =>
    http.get<ClassroomStats>(`${API}/classrooms/${classroomId}/stats`),

  reasonUsage: (classroomId: string) =>
    http
      .get<{ items: ReasonUsage[] }>(`${API}/classrooms/${classroomId}/reports/reasons`)
      .then((r) => r.items),

  ranking: (classroomId: string, limit = 20) =>
    http
      .get<{ items: RankingEntry[] }>(`${API}/classrooms/${classroomId}/ranking`, {
        limit,
      })
      .then((r) => r.items),

  studentHistory: (enrollmentId: string, filter: HistoryFilter = {}) =>
    http.get<Paginated<Transaction>>(
      `${API}/enrollments/${enrollmentId}/transactions`,
      filter as Record<string, unknown>,
    ),

  get: (id: string) => http.get<Transaction>(`${API}/transactions/${id}`),

  /** RN-15: never deletes, always writes a compensating entry. */
  reverse: (id: string, reason?: string) =>
    http.post<Transaction>(`${API}/transactions/${id}/reversal`, { reason }),

  batch: (batchId: string) => http.get<Batch>(`${API}/batches/${batchId}`),
}

// ---------------------------------------------------------------------------
// QR (teacher side)
// ---------------------------------------------------------------------------

export const qrApi = {
  /**
   * Step 2 of flow A: resolve a scanned student code to a person. This does not
   * move neurons — the returned `grant_key` must be passed as the grant's
   * idempotency_key so a double tap on confirm pays only once (§11.3).
   */
  scan: (classroomId: string, code: string) =>
    http.post<ScannedStudent>(`${API}/classrooms/${classroomId}/qr/scan`, { code }),

  /** HU-050: project a code that many students can claim at once. */
  issueGrant: (classroomId: string, input: IssueGrantQRInput) =>
    http.post<IssuedQR>(`${API}/classrooms/${classroomId}/qr/grants`, input),
}
