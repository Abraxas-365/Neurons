/**
 * Types mirrored from the Go domain structs. Field names here must match the
 * `json:"..."` tags in internal/gamification/**; a typo is a silent runtime bug,
 * so keep this file in lockstep with the backend.
 */

// ---------------------------------------------------------------------------
// Envelopes
// ---------------------------------------------------------------------------

/** kernel.Paginated[T] */
export interface Paginated<T> {
  items: T[]
  pagination: {
    page: number
    page_size: number
    total: number
    pages: number
  }
  empty: boolean
}

/** The shape produced by globalErrorHandler in cmd/server.go. */
export interface ApiErrorBody {
  error: string
  code: string
  type?: string
  status: number
  request_id?: string
  details?: Record<string, unknown>
}

// ---------------------------------------------------------------------------
// Enums
// ---------------------------------------------------------------------------

export type ClassroomStatus = "DRAFT" | "ACTIVE" | "CLOSED" | "ARCHIVED"
export type JoinPolicy = "AUTO" | "APPROVAL"
export type EnrollmentStatus = "PENDING" | "ACTIVE" | "WITHDRAWN"

export type TxType =
  | "GRANT"
  | "REDEMPTION"
  | "GRANT_REVERSAL"
  | "REDEMPTION_REVERSAL"
  | "VAULT_TOPUP"
  | "ADJUSTMENT"

export type TxChannel = "QR" | "MANUAL" | "BULK" | "SYSTEM"
export type TxStatus = "APPLIED" | "REVERSED"
export type BatchType = "TEAM_GRANT" | "MULTI_GRANT"

export type ReasonScope = "INDIVIDUAL" | "TEAM" | "BOTH"
export type MedalType = "INDIVIDUAL" | "TEAM"

// ---------------------------------------------------------------------------
// Auth
// ---------------------------------------------------------------------------

export interface TenantOption {
  tenant_id: string
  company_name: string
  user_status: string
  auth_methods: { otp: boolean; oauth: boolean; oauth_provider?: string }
}

export interface UserDetails {
  id: string
  tenant_id: string
  name: string
  email: string
  picture?: string
  is_active: boolean
  scopes: string[]
  oauth_provider: string
}

export interface TokenResponse {
  access_token: string
  refresh_token: string
  token_type: string
  expires_in: number
  user: UserDetails
  tenant: { id: string; company_name: string; [k: string]: unknown }
}

// ---------------------------------------------------------------------------
// Classroom
// ---------------------------------------------------------------------------

export interface Classroom {
  id: string
  name: string
  section: string | null
  term: string | null
  description: string | null
  icon: string | null
  invite_code: string
  status: ClassroomStatus
  unlimited_issuance: boolean
  vault_balance: number
  total_granted: number
  total_redeemed: number
  /** total_granted - total_redeemed: what students collectively hold. */
  distributed: number
  join_policy: JoinPolicy
  void_window_seconds: number
  reconfirm_threshold: number
  allow_free_redemption: boolean
  ranking_public: boolean
  starts_at: string | null
  ends_at: string | null
  closed_at: string | null
  created_at: string
}

export interface CreateClassroomInput {
  name: string
  section?: string | null
  term?: string | null
  description?: string | null
  icon?: string | null
  initial_neurons?: number
  unlimited_issuance?: boolean
  join_policy?: JoinPolicy
  status?: ClassroomStatus
  reconfirm_threshold?: number
  void_window_seconds?: number
  allow_free_redemption?: boolean
  ranking_public?: boolean
}

export type UpdateClassroomInput = Partial<CreateClassroomInput>

export interface ClassroomTeacher {
  user_id: string
  name: string
  email: string
  role: string
  added_at: string
}

// ---------------------------------------------------------------------------
// Enrollment
// ---------------------------------------------------------------------------

export interface RosterEntry {
  id: string
  user_id: string
  name: string
  email: string
  student_code: string | null
  team_id: string | null
  team_name: string | null
  balance: number
  total_received: number
  total_returned: number
  medal_count: number
  status: EnrollmentStatus
  last_activity_at: string | null
  joined_at: string
}

/** The student's own view of a membership (§10.4). */
export interface MyEnrollment {
  id: string
  classroom_id: string
  classroom_name: string
  section: string | null
  term: string | null
  icon: string | null
  balance: number
  total_received: number
  total_returned: number
  team_id: string | null
  team_name: string | null
  status: EnrollmentStatus
  /** False once the course is closed: the UI must stop offering actions (RN-17). */
  classroom_open: boolean
  joined_at: string
}

export interface JoinResult {
  enrollment_id: string
  classroom_id: string
  status: EnrollmentStatus
  balance: number
}

/** One row of a bulk invite (HU-020 / HU-021). */
export interface InviteStudentEntry {
  email: string
  name?: string
  student_code?: string | null
}

/** Per-row outcome so partial failures can be shown before a retry. */
export interface InviteResult {
  email: string
  status: "ENROLLED" | "ALREADY_ENROLLED" | "INVITED" | "ERROR"
  detail?: string
}

export interface RosterFilter {
  search?: string
  status?: EnrollmentStatus
  team_id?: string
  sort_by?: string
  sort_dir?: "asc" | "desc"
  page?: number
  page_size?: number
}

// ---------------------------------------------------------------------------
// Team
// ---------------------------------------------------------------------------

export type TeamStatus = "ACTIVE" | "ARCHIVED"

/** team.TeamSummary — what list endpoints return. */
export interface Team {
  id: string
  name: string
  description: string | null
  color: string | null
  icon: string | null
  status: TeamStatus
  member_count: number
  created_at: string
}

export interface TeamMember {
  enrollment_id: string
  user_id: string
  name: string
  email: string
  balance: number
  is_coordinator: boolean
  joined_at: string
}

export interface TeamDetail extends Team {
  members: TeamMember[]
}

export interface CreateTeamInput {
  name: string
  description?: string | null
  color?: string | null
  icon?: string | null
  enrollment_ids?: string[]
}

export interface RandomizeTeamsInput {
  team_count?: number | null
  team_size?: number | null
  name_prefix?: string
  keep_together?: string[][]
  keep_apart?: string[][]
  /** True returns a proposal without persisting it, so the teacher can reroll. */
  preview?: boolean
}

export interface RandomizeResult {
  teams: { id?: string; name: string; members: TeamMember[] }[]
  persisted: boolean
  unplaced: TeamMember[]
}

// ---------------------------------------------------------------------------
// Catalog: reasons, benefits, medals
// ---------------------------------------------------------------------------

export interface Reason {
  id: string
  classroom_id: string
  name: string
  description: string | null
  icon: string | null
  /** Prefills the grant amount; the teacher can still override it. */
  suggested_amount: number | null
  scope: ReasonScope
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateReasonInput {
  name: string
  description?: string | null
  icon?: string | null
  suggested_amount?: number | null
  scope: ReasonScope
}

export type UpdateReasonInput = Partial<CreateReasonInput> & { is_active?: boolean }

export interface Benefit {
  id: string
  classroom_id: string
  name: string
  description: string | null
  icon: string | null
  /** null means the student chooses the amount (HU-063). */
  cost: number | null
  max_uses: number | null
  uses_count: number
  max_uses_per_student: number | null
  requires_approval: boolean
  scope: ReasonScope
  conditions: string | null
  available_from: string | null
  available_until: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

/** benefit.StudentView — the catalog as a student sees it (HU-061). */
export interface StudentBenefit {
  id: string
  name: string
  description: string | null
  icon: string | null
  cost: number | null
  requires_approval: boolean
  conditions: string | null
  remaining_uses: number | null
  available: boolean
  /** False when the student cannot currently afford it (RN-04). */
  affordable: boolean
  available_until: string | null
}

export interface CreateBenefitInput {
  name: string
  description?: string | null
  icon?: string | null
  cost?: number | null
  max_uses?: number | null
  max_uses_per_student?: number | null
  requires_approval?: boolean
  scope?: ReasonScope
  conditions?: string | null
  available_from?: string | null
  available_until?: string | null
}

export type UpdateBenefitInput = Partial<CreateBenefitInput> & { is_active?: boolean }

export interface Medal {
  id: string
  classroom_id: string
  name: string
  description: string | null
  image_url: string | null
  icon: string | null
  category: string | null
  type: MedalType
  condition: string | null
  max_awards: number | null
  /** Decision 15.5: may the same student earn this more than once? */
  repeatable: boolean
  /** RN-14: team medals also surface on each member's profile. */
  show_on_member_profile: boolean
  visible_to_students: boolean
  available_from: string | null
  available_until: string | null
  is_active: boolean
  created_at: string
  updated_at: string
}

export interface CreateMedalInput {
  name: string
  description?: string | null
  image_url?: string | null
  icon?: string | null
  category?: string | null
  type: MedalType
  condition?: string | null
  max_awards?: number | null
  repeatable?: boolean
  show_on_member_profile?: boolean
  visible_to_students?: boolean
  available_from?: string | null
  available_until?: string | null
}

export type UpdateMedalInput = Partial<CreateMedalInput> & { is_active?: boolean }

export interface AwardMedalInput {
  enrollment_ids?: string[]
  team_id?: string | null
  note?: string | null
}

export interface MedalAward {
  id: string
  medal_id: string
  classroom_id: string
  enrollment_id: string | null
  team_id: string | null
  awarded_by: string
  note: string | null
  awarded_at: string
  revoked_at: string | null
  medal_name?: string
  medal_icon?: string | null
  medal_image_url?: string | null
  student_name?: string | null
  team_name?: string | null
}

// ---------------------------------------------------------------------------
// Ledger
// ---------------------------------------------------------------------------

export interface Transaction {
  id: string
  code: string
  classroom_id: string
  type: TxType
  enrollment_id: string | null
  team_id: string | null
  batch_id: string | null
  amount: number
  reason_id: string | null
  reason_text: string | null
  benefit_id: string | null
  benefit_text: string | null
  student_balance_before: number | null
  student_balance_after: number | null
  vault_balance_before: number | null
  vault_balance_after: number | null
  channel: TxChannel
  status: TxStatus
  reverses_transaction_id: string | null
  reversed_by_transaction_id: string | null
  performed_by: string
  notes: string | null
  created_at: string
  student_name?: string
  team_name?: string
  performer_name?: string
}

export interface GrantResult {
  batch_id?: string
  transactions: Transaction[]
  recipients: number
  amount_each: number
  total_amount: number
  vault_balance: number
}

export interface GrantInput {
  enrollment_ids: string[]
  amount: number
  reason_id?: string | null
  reason_text?: string | null
  notes?: string | null
  channel?: TxChannel
  idempotency_key?: string | null
  /** Required when the amount exceeds the classroom's reconfirm threshold. */
  confirmed?: boolean
}

export interface TeamGrantInput {
  team_id: string
  amount: number
  reason_id?: string | null
  reason_text?: string | null
  notes?: string | null
  idempotency_key?: string | null
  confirmed?: boolean
}

export interface RedeemInput {
  enrollment_id: string
  benefit_id?: string | null
  benefit_text?: string | null
  /** Only for free-amount benefits, where cost is null. */
  amount?: number | null
  notes?: string | null
  channel?: TxChannel
  idempotency_key?: string | null
}

export interface ClassroomStats {
  vault_balance: number
  unlimited_issuance: boolean
  total_granted: number
  total_redeemed: number
  in_circulation: number
  active_students: number
  transactions: number
}

export interface RankingEntry {
  position: number
  enrollment_id: string
  student_name: string
  total_received: number
  medal_count: number
}

export interface ReasonUsage {
  reason_id: string | null
  reason_name: string
  uses: number
  total_amount: number
}

export interface HistoryFilter {
  enrollment_id?: string
  team_id?: string
  type?: TxType
  reason_id?: string
  benefit_id?: string
  from?: string
  to?: string
  page?: number
  page_size?: number
}

export interface Batch {
  id: string
  classroom_id: string
  type: BatchType
  team_id: string | null
  amount_per_student: number
  recipient_count: number
  total_amount: number
  reason_id: string | null
  reason_text: string | null
  performed_by: string
  created_at: string
}

// ---------------------------------------------------------------------------
// QR
// ---------------------------------------------------------------------------

/** qr.Issued — what a QR display screen needs: the code plus its countdown. */
export interface IssuedQR {
  code: string
  kind: "STUDENT" | "GRANT"
  expires_at: string
  ttl_seconds: number
}

export interface IssueGrantQRInput {
  amount: number
  reason_id?: string | null
  reason_text?: string | null
  ttl_seconds?: number
  /** 0 = unlimited for the life of the code. */
  max_claims?: number
}

/** What the teacher's scanner gets back after resolving a student token. */
export interface ScannedStudent {
  enrollment_id: string
  classroom_id: string
  student_name: string
  student_email: string
  balance: number
  team_id: string | null
  team_name: string | null
  /** Use as the idempotency_key of the grant that follows (§11.3). */
  grant_key: string
}
