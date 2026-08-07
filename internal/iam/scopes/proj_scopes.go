package scopes

// Domain-specific scopes — Neurons gamification.
// They are merged into the global scope registry at init time.

const (
	// Classroom management (teacher-side)
	ScopeClassroomsAll    = "classrooms:*"
	ScopeClassroomsRead   = "classrooms:read"
	ScopeClassroomsWrite  = "classrooms:write"
	ScopeClassroomsDelete = "classrooms:delete"

	// Roster management
	ScopeEnrollmentsAll   = "enrollments:*"
	ScopeEnrollmentsRead  = "enrollments:read"
	ScopeEnrollmentsWrite = "enrollments:write"

	// Teams
	ScopeTeamsAll   = "teams:*"
	ScopeTeamsRead  = "teams:read"
	ScopeTeamsWrite = "teams:write"

	// Catalogs: reasons, benefits, medals
	ScopeCatalogAll   = "catalog:*"
	ScopeCatalogRead  = "catalog:read"
	ScopeCatalogWrite = "catalog:write"

	// Neuron movements
	ScopeNeuronsAll = "neurons:*"
	// Grant neurons from the classroom vault to students
	ScopeNeuronsGrant = "neurons:grant"
	// Receive neurons back from students
	ScopeNeuronsRedeem = "neurons:redeem"
	// Void/reverse a transaction
	ScopeNeuronsVoid = "neurons:void"
	// Read the classroom ledger and balances
	ScopeNeuronsRead = "neurons:read"

	// Medal awarding
	ScopeMedalsAward = "medals:award"

	// Student-side self-service (own balance, own QR, own history)
	ScopeStudentSelf = "student:self"
)

// DomainScopeCategories organizes domain-specific scopes by category.
var DomainScopeCategories = map[string][]string{
	"Classrooms": {
		ScopeClassroomsAll,
		ScopeClassroomsRead,
		ScopeClassroomsWrite,
		ScopeClassroomsDelete,
	},
	"Enrollments": {
		ScopeEnrollmentsAll,
		ScopeEnrollmentsRead,
		ScopeEnrollmentsWrite,
	},
	"Teams": {
		ScopeTeamsAll,
		ScopeTeamsRead,
		ScopeTeamsWrite,
	},
	"Catalog": {
		ScopeCatalogAll,
		ScopeCatalogRead,
		ScopeCatalogWrite,
	},
	"Neurons": {
		ScopeNeuronsAll,
		ScopeNeuronsGrant,
		ScopeNeuronsRedeem,
		ScopeNeuronsVoid,
		ScopeNeuronsRead,
		ScopeMedalsAward,
	},
	"Student": {
		ScopeStudentSelf,
	},
}

// DomainScopeDescriptions provides descriptions for domain scopes.
var DomainScopeDescriptions = map[string]string{
	ScopeClassroomsAll:    "Full access to classrooms",
	ScopeClassroomsRead:   "View classrooms",
	ScopeClassroomsWrite:  "Create and edit classrooms",
	ScopeClassroomsDelete: "Delete classrooms",

	ScopeEnrollmentsAll:   "Full access to the student roster",
	ScopeEnrollmentsRead:  "View enrolled students and their balances",
	ScopeEnrollmentsWrite: "Enroll, approve and withdraw students",

	ScopeTeamsAll:   "Full access to teams",
	ScopeTeamsRead:  "View teams",
	ScopeTeamsWrite: "Create and edit teams",

	ScopeCatalogAll:   "Full access to reasons, benefits and medals",
	ScopeCatalogRead:  "View reasons, benefits and medals",
	ScopeCatalogWrite: "Create and edit reasons, benefits and medals",

	ScopeNeuronsAll:    "Full access to neuron operations",
	ScopeNeuronsGrant:  "Grant neurons to students and teams",
	ScopeNeuronsRedeem: "Receive neurons returned by students",
	ScopeNeuronsVoid:   "Void or reverse transactions",
	ScopeNeuronsRead:   "View the classroom ledger and balances",
	ScopeMedalsAward:   "Award medals to students and teams",

	ScopeStudentSelf: "Access your own balances, history, QR and redemptions",
}

// TeacherScopes is the scope set granted to a teacher account.
var TeacherScopes = []string{
	ScopeClassroomsAll,
	ScopeEnrollmentsAll,
	ScopeTeamsAll,
	ScopeCatalogAll,
	ScopeNeuronsAll,
	ScopeMedalsAward,
	ScopeStudentSelf,
}

// StudentScopes is the scope set granted to a student account. Students may
// only ever touch their own data (§10.4).
var StudentScopes = []string{
	ScopeStudentSelf,
}
