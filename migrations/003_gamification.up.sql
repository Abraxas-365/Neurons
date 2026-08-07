-- ============================================================================
-- Neurons Gamification Schema
--
-- Core invariant (RN-01): every neuron balance, transaction, medal and catalog
-- entry is scoped to a single classroom. Nothing crosses classroom boundaries.
-- ============================================================================

-- ============================================================================
-- CLASSROOMS (Salón) — owns a vault (bóveda) of neurons
-- ============================================================================

CREATE TABLE classrooms (
    id VARCHAR(255) PRIMARY KEY,
    tenant_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    section VARCHAR(100),
    term VARCHAR(100),
    description TEXT,
    icon VARCHAR(255),
    invite_code VARCHAR(32) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'DRAFT',
    -- Vault: NULL vault_balance means unlimited issuance (RN-05 / decision 15.1)
    unlimited_issuance BOOLEAN NOT NULL DEFAULT FALSE,
    vault_balance BIGINT NOT NULL DEFAULT 0,
    total_granted BIGINT NOT NULL DEFAULT 0,
    total_redeemed BIGINT NOT NULL DEFAULT 0,
    -- Join policy: AUTO enrolls immediately, APPROVAL requires teacher review
    join_policy VARCHAR(50) NOT NULL DEFAULT 'AUTO',
    -- Window (seconds) during which a teacher may void a transaction; 0 = always
    void_window_seconds INTEGER NOT NULL DEFAULT 0,
    -- Amount above which the UI must ask for reconfirmation (§11.9)
    reconfirm_threshold INTEGER NOT NULL DEFAULT 10,
    allow_free_redemption BOOLEAN NOT NULL DEFAULT TRUE,
    ranking_public BOOLEAN NOT NULL DEFAULT FALSE,
    starts_at TIMESTAMP,
    ends_at TIMESTAMP,
    closed_at TIMESTAMP,
    created_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_classrooms_tenant FOREIGN KEY (tenant_id) REFERENCES tenants(id) ON DELETE CASCADE,
    CONSTRAINT fk_classrooms_creator FOREIGN KEY (created_by) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT uq_classrooms_invite_code UNIQUE (invite_code),
    CONSTRAINT chk_classroom_status CHECK (status IN ('DRAFT', 'ACTIVE', 'CLOSED', 'ARCHIVED')),
    CONSTRAINT chk_classroom_join_policy CHECK (join_policy IN ('AUTO', 'APPROVAL')),
    -- RN-05: the vault may never go negative
    CONSTRAINT chk_classroom_vault_non_negative CHECK (vault_balance >= 0)
);

CREATE INDEX idx_classrooms_tenant_id ON classrooms(tenant_id);
CREATE INDEX idx_classrooms_status ON classrooms(status);
CREATE INDEX idx_classrooms_created_by ON classrooms(created_by);

CREATE TRIGGER update_classrooms_updated_at BEFORE UPDATE ON classrooms
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE classrooms IS 'A course/section. Independent gamification universe: its own vault, balances, catalogs and ledger.';
COMMENT ON COLUMN classrooms.vault_balance IS 'Neurons available to grant. Ignored when unlimited_issuance is true.';

-- ============================================================================
-- CLASSROOM TEACHERS — a classroom may have several teachers (decision 15.9)
-- ============================================================================

CREATE TABLE classroom_teachers (
    classroom_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    role VARCHAR(50) NOT NULL DEFAULT 'ASSISTANT',
    -- Optional allowance for assistants ("jefe de práctica"): NULL = no cap
    grant_allowance BIGINT,
    granted_from_allowance BIGINT NOT NULL DEFAULT 0,
    added_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    PRIMARY KEY (classroom_id, user_id),
    CONSTRAINT fk_classroom_teachers_classroom FOREIGN KEY (classroom_id) REFERENCES classrooms(id) ON DELETE CASCADE,
    CONSTRAINT fk_classroom_teachers_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT chk_classroom_teacher_role CHECK (role IN ('OWNER', 'ASSISTANT'))
);

CREATE INDEX idx_classroom_teachers_user_id ON classroom_teachers(user_id);

COMMENT ON COLUMN classroom_teachers.grant_allowance IS 'Cap on neurons this assistant may grant. NULL = unlimited within vault.';

-- ============================================================================
-- TEAMS (Equipos)
-- ============================================================================

CREATE TABLE teams (
    id VARCHAR(255) PRIMARY KEY,
    classroom_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    color VARCHAR(32),
    icon VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_teams_classroom FOREIGN KEY (classroom_id) REFERENCES classrooms(id) ON DELETE CASCADE,
    CONSTRAINT uq_teams_name_classroom UNIQUE (classroom_id, name),
    CONSTRAINT chk_team_status CHECK (status IN ('ACTIVE', 'INACTIVE'))
);

CREATE INDEX idx_teams_classroom_id ON teams(classroom_id);

CREATE TRIGGER update_teams_updated_at BEFORE UPDATE ON teams
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- ENROLLMENTS (Matrícula) — the student's vault for one classroom
-- ============================================================================

CREATE TABLE enrollments (
    id VARCHAR(255) PRIMARY KEY,
    classroom_id VARCHAR(255) NOT NULL,
    user_id VARCHAR(255) NOT NULL,
    -- RN-02/RN-04: authoritative per-classroom balance, server-side only
    balance BIGINT NOT NULL DEFAULT 0,
    total_received BIGINT NOT NULL DEFAULT 0,
    total_returned BIGINT NOT NULL DEFAULT 0,
    team_id VARCHAR(255),
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    student_code VARCHAR(100),
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    left_at TIMESTAMP,
    last_activity_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_enrollments_classroom FOREIGN KEY (classroom_id) REFERENCES classrooms(id) ON DELETE CASCADE,
    CONSTRAINT fk_enrollments_user FOREIGN KEY (user_id) REFERENCES users(id) ON DELETE CASCADE,
    CONSTRAINT fk_enrollments_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE SET NULL,
    -- RN-18: one enrollment per user per classroom
    CONSTRAINT uq_enrollments_user_classroom UNIQUE (classroom_id, user_id),
    CONSTRAINT chk_enrollment_status CHECK (status IN ('PENDING', 'ACTIVE', 'WITHDRAWN')),
    -- MVP criterion 15: balances can never go negative
    CONSTRAINT chk_enrollment_balance_non_negative CHECK (balance >= 0)
);

CREATE INDEX idx_enrollments_classroom_id ON enrollments(classroom_id);
CREATE INDEX idx_enrollments_user_id ON enrollments(user_id);
CREATE INDEX idx_enrollments_team_id ON enrollments(team_id);
CREATE INDEX idx_enrollments_status ON enrollments(status);
CREATE INDEX idx_enrollments_balance ON enrollments(classroom_id, balance DESC);

CREATE TRIGGER update_enrollments_updated_at BEFORE UPDATE ON enrollments
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON TABLE enrollments IS 'Student membership in one classroom, holding that classroom''s neuron balance. RN-01: balances never cross classrooms.';

-- ============================================================================
-- TEAM MEMBERSHIP HISTORY — RN-16 / decision 15.10 keep the trail
-- ============================================================================

CREATE TABLE team_memberships (
    id VARCHAR(255) PRIMARY KEY,
    team_id VARCHAR(255) NOT NULL,
    enrollment_id VARCHAR(255) NOT NULL,
    is_coordinator BOOLEAN NOT NULL DEFAULT FALSE,
    joined_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    left_at TIMESTAMP,

    CONSTRAINT fk_team_memberships_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    CONSTRAINT fk_team_memberships_enrollment FOREIGN KEY (enrollment_id) REFERENCES enrollments(id) ON DELETE CASCADE
);

CREATE INDEX idx_team_memberships_team_id ON team_memberships(team_id);
CREATE INDEX idx_team_memberships_enrollment_id ON team_memberships(enrollment_id);

-- A student may hold only one ACTIVE team membership per classroom (decision 15.10).
-- Enforced here across the whole set of open memberships for an enrollment.
CREATE UNIQUE INDEX uq_team_memberships_active_enrollment
    ON team_memberships(enrollment_id) WHERE left_at IS NULL;

-- ============================================================================
-- REASONS (Motivos de entrega)
-- ============================================================================

CREATE TABLE reasons (
    id VARCHAR(255) PRIMARY KEY,
    classroom_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(255),
    suggested_amount INTEGER,
    scope VARCHAR(50) NOT NULL DEFAULT 'BOTH',
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_reasons_classroom FOREIGN KEY (classroom_id) REFERENCES classrooms(id) ON DELETE CASCADE,
    CONSTRAINT chk_reason_scope CHECK (scope IN ('INDIVIDUAL', 'TEAM', 'BOTH')),
    CONSTRAINT chk_reason_suggested_amount CHECK (suggested_amount IS NULL OR suggested_amount > 0)
);

CREATE INDEX idx_reasons_classroom_id ON reasons(classroom_id);
CREATE INDEX idx_reasons_active ON reasons(classroom_id, is_active);

CREATE TRIGGER update_reasons_updated_at BEFORE UPDATE ON reasons
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- BENEFITS (Beneficios / conceptos de devolución)
-- ============================================================================

CREATE TABLE benefits (
    id VARCHAR(255) PRIMARY KEY,
    classroom_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    icon VARCHAR(255),
    -- NULL cost = free-amount benefit; the student chooses how much (HU-063)
    cost INTEGER,
    max_uses INTEGER,
    uses_count INTEGER NOT NULL DEFAULT 0,
    max_uses_per_student INTEGER,
    requires_approval BOOLEAN NOT NULL DEFAULT FALSE,
    scope VARCHAR(50) NOT NULL DEFAULT 'INDIVIDUAL',
    conditions TEXT,
    available_from TIMESTAMP,
    available_until TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_benefits_classroom FOREIGN KEY (classroom_id) REFERENCES classrooms(id) ON DELETE CASCADE,
    CONSTRAINT chk_benefit_scope CHECK (scope IN ('INDIVIDUAL', 'TEAM')),
    CONSTRAINT chk_benefit_cost CHECK (cost IS NULL OR cost > 0),
    CONSTRAINT chk_benefit_max_uses CHECK (max_uses IS NULL OR max_uses > 0)
);

CREATE INDEX idx_benefits_classroom_id ON benefits(classroom_id);
CREATE INDEX idx_benefits_active ON benefits(classroom_id, is_active);

CREATE TRIGGER update_benefits_updated_at BEFORE UPDATE ON benefits
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

COMMENT ON COLUMN benefits.cost IS 'Fixed neuron price. NULL means the student picks the amount (free redemption).';

-- ============================================================================
-- MEDALS (Medallas)
-- ============================================================================

CREATE TABLE medals (
    id VARCHAR(255) PRIMARY KEY,
    classroom_id VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    description TEXT,
    image_url TEXT,
    icon VARCHAR(255),
    category VARCHAR(100),
    type VARCHAR(50) NOT NULL DEFAULT 'INDIVIDUAL',
    condition TEXT,
    max_awards INTEGER,
    -- Decision 15.5: medals may be awarded repeatedly unless capped
    repeatable BOOLEAN NOT NULL DEFAULT TRUE,
    -- RN-14: whether a team medal also shows on each member's profile
    show_on_member_profile BOOLEAN NOT NULL DEFAULT TRUE,
    visible_to_students BOOLEAN NOT NULL DEFAULT TRUE,
    available_from TIMESTAMP,
    available_until TIMESTAMP,
    is_active BOOLEAN NOT NULL DEFAULT TRUE,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_medals_classroom FOREIGN KEY (classroom_id) REFERENCES classrooms(id) ON DELETE CASCADE,
    CONSTRAINT chk_medal_type CHECK (type IN ('INDIVIDUAL', 'TEAM'))
);

CREATE INDEX idx_medals_classroom_id ON medals(classroom_id);

CREATE TRIGGER update_medals_updated_at BEFORE UPDATE ON medals
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- MEDAL AWARDS
-- ============================================================================

CREATE TABLE medal_awards (
    id VARCHAR(255) PRIMARY KEY,
    medal_id VARCHAR(255) NOT NULL,
    classroom_id VARCHAR(255) NOT NULL,
    -- Exactly one of enrollment_id / team_id is set
    enrollment_id VARCHAR(255),
    team_id VARCHAR(255),
    awarded_by VARCHAR(255) NOT NULL,
    note TEXT,
    awarded_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    revoked_at TIMESTAMP,

    CONSTRAINT fk_medal_awards_medal FOREIGN KEY (medal_id) REFERENCES medals(id) ON DELETE CASCADE,
    CONSTRAINT fk_medal_awards_classroom FOREIGN KEY (classroom_id) REFERENCES classrooms(id) ON DELETE CASCADE,
    CONSTRAINT fk_medal_awards_enrollment FOREIGN KEY (enrollment_id) REFERENCES enrollments(id) ON DELETE CASCADE,
    CONSTRAINT fk_medal_awards_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE CASCADE,
    CONSTRAINT fk_medal_awards_awarder FOREIGN KEY (awarded_by) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT chk_medal_award_target CHECK (
        (enrollment_id IS NOT NULL AND team_id IS NULL) OR
        (enrollment_id IS NULL AND team_id IS NOT NULL)
    )
);

CREATE INDEX idx_medal_awards_medal_id ON medal_awards(medal_id);
CREATE INDEX idx_medal_awards_enrollment_id ON medal_awards(enrollment_id);
CREATE INDEX idx_medal_awards_team_id ON medal_awards(team_id);
CREATE INDEX idx_medal_awards_classroom_id ON medal_awards(classroom_id);

-- ============================================================================
-- BATCHES — group several individual transactions under one logical operation
-- (HU-033 team grant, HU-052 multi-student grant)
-- ============================================================================

CREATE TABLE transaction_batches (
    id VARCHAR(255) PRIMARY KEY,
    classroom_id VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    team_id VARCHAR(255),
    amount_per_student INTEGER NOT NULL,
    recipient_count INTEGER NOT NULL,
    total_amount BIGINT NOT NULL,
    reason_id VARCHAR(255),
    reason_text VARCHAR(500),
    performed_by VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_batches_classroom FOREIGN KEY (classroom_id) REFERENCES classrooms(id) ON DELETE CASCADE,
    CONSTRAINT fk_batches_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE SET NULL,
    CONSTRAINT fk_batches_reason FOREIGN KEY (reason_id) REFERENCES reasons(id) ON DELETE SET NULL,
    CONSTRAINT fk_batches_performer FOREIGN KEY (performed_by) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT chk_batch_type CHECK (type IN ('TEAM_GRANT', 'MULTI_GRANT'))
);

CREATE INDEX idx_batches_classroom_id ON transaction_batches(classroom_id);

-- ============================================================================
-- TRANSACTIONS (Bitácora) — append-only ledger, RN-09 / RN-15
-- ============================================================================

CREATE TABLE transactions (
    id VARCHAR(255) PRIMARY KEY,
    -- Short human-readable code shown in the UI
    code VARCHAR(32) NOT NULL,
    classroom_id VARCHAR(255) NOT NULL,
    type VARCHAR(50) NOT NULL,
    enrollment_id VARCHAR(255),
    team_id VARCHAR(255),
    batch_id VARCHAR(255),
    -- Always positive; `type` determines the direction of the movement
    amount BIGINT NOT NULL,
    reason_id VARCHAR(255),
    reason_text VARCHAR(500),
    benefit_id VARCHAR(255),
    benefit_text VARCHAR(500),
    -- Balance snapshots for auditability (§4.6)
    student_balance_before BIGINT,
    student_balance_after BIGINT,
    vault_balance_before BIGINT,
    vault_balance_after BIGINT,
    channel VARCHAR(50) NOT NULL DEFAULT 'MANUAL',
    status VARCHAR(50) NOT NULL DEFAULT 'APPLIED',
    -- RN-15: reversals point back at the transaction they undo
    reverses_transaction_id VARCHAR(255),
    reversed_by_transaction_id VARCHAR(255),
    -- Idempotency guard against double submits / duplicate QR scans (§11.3)
    idempotency_key VARCHAR(255),
    performed_by VARCHAR(255) NOT NULL,
    device_info VARCHAR(500),
    notes TEXT,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_transactions_classroom FOREIGN KEY (classroom_id) REFERENCES classrooms(id) ON DELETE CASCADE,
    CONSTRAINT fk_transactions_enrollment FOREIGN KEY (enrollment_id) REFERENCES enrollments(id) ON DELETE RESTRICT,
    CONSTRAINT fk_transactions_team FOREIGN KEY (team_id) REFERENCES teams(id) ON DELETE SET NULL,
    CONSTRAINT fk_transactions_batch FOREIGN KEY (batch_id) REFERENCES transaction_batches(id) ON DELETE SET NULL,
    CONSTRAINT fk_transactions_reason FOREIGN KEY (reason_id) REFERENCES reasons(id) ON DELETE SET NULL,
    CONSTRAINT fk_transactions_benefit FOREIGN KEY (benefit_id) REFERENCES benefits(id) ON DELETE SET NULL,
    CONSTRAINT fk_transactions_reverses FOREIGN KEY (reverses_transaction_id) REFERENCES transactions(id) ON DELETE RESTRICT,
    CONSTRAINT fk_transactions_reversed_by FOREIGN KEY (reversed_by_transaction_id) REFERENCES transactions(id) ON DELETE RESTRICT,
    CONSTRAINT fk_transactions_performer FOREIGN KEY (performed_by) REFERENCES users(id) ON DELETE RESTRICT,
    CONSTRAINT uq_transactions_code UNIQUE (code),
    CONSTRAINT chk_transaction_type CHECK (type IN (
        'GRANT', 'REDEMPTION', 'GRANT_REVERSAL', 'REDEMPTION_REVERSAL', 'VAULT_TOPUP', 'ADJUSTMENT'
    )),
    CONSTRAINT chk_transaction_channel CHECK (channel IN ('QR', 'MANUAL', 'BULK', 'SYSTEM')),
    CONSTRAINT chk_transaction_status CHECK (status IN ('APPLIED', 'REVERSED')),
    CONSTRAINT chk_transaction_amount_positive CHECK (amount > 0)
);

CREATE INDEX idx_transactions_classroom_id ON transactions(classroom_id, created_at DESC);
CREATE INDEX idx_transactions_enrollment_id ON transactions(enrollment_id, created_at DESC);
CREATE INDEX idx_transactions_team_id ON transactions(team_id);
CREATE INDEX idx_transactions_batch_id ON transactions(batch_id);
CREATE INDEX idx_transactions_type ON transactions(classroom_id, type);
CREATE INDEX idx_transactions_reason_id ON transactions(reason_id);
CREATE INDEX idx_transactions_benefit_id ON transactions(benefit_id);
CREATE INDEX idx_transactions_created_at ON transactions(created_at DESC);

-- §11.3: the same idempotency key can never produce two movements in a classroom
CREATE UNIQUE INDEX uq_transactions_idempotency
    ON transactions(classroom_id, idempotency_key) WHERE idempotency_key IS NOT NULL;

COMMENT ON TABLE transactions IS 'Append-only ledger. Rows are never updated except to link a reversal (RN-15).';
COMMENT ON COLUMN transactions.amount IS 'Always positive. The type field carries the direction.';

-- ============================================================================
-- BENEFIT REQUESTS (HU-064) — approval workflow, separate from the neuron move
-- ============================================================================

CREATE TABLE benefit_requests (
    id VARCHAR(255) PRIMARY KEY,
    classroom_id VARCHAR(255) NOT NULL,
    enrollment_id VARCHAR(255) NOT NULL,
    benefit_id VARCHAR(255),
    benefit_text VARCHAR(500),
    amount BIGINT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    -- The redemption transaction, created when neurons actually move
    transaction_id VARCHAR(255),
    -- Whether neurons were already debited (held) while awaiting approval
    neurons_held BOOLEAN NOT NULL DEFAULT FALSE,
    student_note TEXT,
    resolution_note TEXT,
    resolved_by VARCHAR(255),
    resolved_at TIMESTAMP,
    expires_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_benefit_requests_classroom FOREIGN KEY (classroom_id) REFERENCES classrooms(id) ON DELETE CASCADE,
    CONSTRAINT fk_benefit_requests_enrollment FOREIGN KEY (enrollment_id) REFERENCES enrollments(id) ON DELETE CASCADE,
    CONSTRAINT fk_benefit_requests_benefit FOREIGN KEY (benefit_id) REFERENCES benefits(id) ON DELETE SET NULL,
    CONSTRAINT fk_benefit_requests_transaction FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE SET NULL,
    CONSTRAINT fk_benefit_requests_resolver FOREIGN KEY (resolved_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_benefit_request_status CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED', 'EXPIRED', 'CANCELED')),
    CONSTRAINT chk_benefit_request_amount CHECK (amount > 0)
);

CREATE INDEX idx_benefit_requests_classroom_id ON benefit_requests(classroom_id, status);
CREATE INDEX idx_benefit_requests_enrollment_id ON benefit_requests(enrollment_id);

CREATE TRIGGER update_benefit_requests_updated_at BEFORE UPDATE ON benefit_requests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();

-- ============================================================================
-- CORRECTION REQUESTS (HU-091) — student disputes a movement
-- ============================================================================

CREATE TABLE correction_requests (
    id VARCHAR(255) PRIMARY KEY,
    classroom_id VARCHAR(255) NOT NULL,
    enrollment_id VARCHAR(255) NOT NULL,
    transaction_id VARCHAR(255) NOT NULL,
    student_note TEXT NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'PENDING',
    resolution_note TEXT,
    resolved_by VARCHAR(255),
    resolved_at TIMESTAMP,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,

    CONSTRAINT fk_correction_requests_classroom FOREIGN KEY (classroom_id) REFERENCES classrooms(id) ON DELETE CASCADE,
    CONSTRAINT fk_correction_requests_enrollment FOREIGN KEY (enrollment_id) REFERENCES enrollments(id) ON DELETE CASCADE,
    CONSTRAINT fk_correction_requests_transaction FOREIGN KEY (transaction_id) REFERENCES transactions(id) ON DELETE CASCADE,
    CONSTRAINT fk_correction_requests_resolver FOREIGN KEY (resolved_by) REFERENCES users(id) ON DELETE SET NULL,
    CONSTRAINT chk_correction_request_status CHECK (status IN ('PENDING', 'APPROVED', 'REJECTED'))
);

CREATE INDEX idx_correction_requests_classroom_id ON correction_requests(classroom_id, status);
CREATE INDEX idx_correction_requests_enrollment_id ON correction_requests(enrollment_id);

CREATE TRIGGER update_correction_requests_updated_at BEFORE UPDATE ON correction_requests
    FOR EACH ROW EXECUTE FUNCTION update_updated_at_column();
