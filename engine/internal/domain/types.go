// Package domain defines the core types for the Three-Body Engine workflow.
package domain

// Phase represents workflow phases A through G.
type Phase string

const (
	PhaseA Phase = "A"
	PhaseB Phase = "B"
	PhaseC Phase = "C"
	PhaseD Phase = "D"
	PhaseE Phase = "E"
	PhaseF Phase = "F"
	PhaseG Phase = "G"
)

// FlowStatus represents the current status of a workflow.
type FlowStatus string

const (
	StatusRunning FlowStatus = "running"
	StatusBlocked FlowStatus = "blocked"
	StatusFailed  FlowStatus = "failed"
	StatusDone    FlowStatus = "completed"
)

// FlowState holds the current state of a workflow task.
type FlowState struct {
	TaskID       string
	CurrentPhase Phase
	Status       FlowStatus
	StateVersion int64
	Round        int
	BudgetUsedUSD float64
	BudgetCapUSD  float64
	LastEventSeq  int64
	UpdatedAtUnix int64
}

// TransitionTrigger initiates a phase transition.
type TransitionTrigger struct {
	Action  string
	Actor   string
	Payload []byte
}

// GateDecision is the result of evaluating phase exit conditions.
type GateDecision struct {
	Allow      bool
	Blockers   []string
	Retryable  bool
	NextPhase  Phase
	RequireOps []string
}

// WorkerState represents the lifecycle state of a worker.
type WorkerState string

const (
	WorkerCreated     WorkerState = "created"
	WorkerRunning     WorkerState = "running"
	WorkerSoftTimeout WorkerState = "soft_timeout"
	WorkerHardTimeout WorkerState = "hard_timeout"
	WorkerReplaced    WorkerState = "replaced"
	WorkerDone        WorkerState = "done"
)

// WorkerSpec defines parameters for spawning a worker.
type WorkerSpec struct {
	TaskID         string
	Phase          Phase
	Role           string
	FileOwnership  []string
	DigestPath     string
	SoftTimeoutSec int
	HardTimeoutSec int
}

// Intent represents a planned file operation by a worker.
type Intent struct {
	IntentID    string
	TaskID      string
	WorkerID    string
	TargetFile  string
	Operation   string
	Status      string
	PreHash     string
	PostHash    string
	PayloadHash string
	LeaseUntil  int64
}

// ArtifactRef points to a versioned artifact in the task directory.
type ArtifactRef struct {
	ID      string
	Type    string
	Path    string
	Version int
	Hash    string
}

// Deadline defines soft and hard time limits.
type Deadline struct {
	Soft string
	Hard string
}

// ContextDigest is the lightweight index sent to workers.
type ContextDigest struct {
	TaskID          string
	PhaseID         string
	Objective       string
	Constraints     []string
	FileOwnership   []string
	Deadline        Deadline
	ArtifactRefs    []ArtifactRef
	CodingStandards string
}

// CompactionSlots are the 9 semantic slots that must survive compaction.
type CompactionSlots struct {
	TaskSpec           string
	AcceptanceCriteria string
	CurrentPhase       string
	OpenRisks          []string
	ActiveConstraints  []string
	FileOwnership      []string
	ArtifactRefs       []ArtifactRef
	PendingIntents     []string
	NextPhaseReqs      []string
}

// WorkflowEvent represents an event in the workflow event log.
type WorkflowEvent struct {
	ID          int64
	TaskID      string
	SeqNo       int64
	Phase       Phase
	EventType   string
	PayloadJSON string
	CreatedAt   int64
}

// PhaseSnapshot captures the state at a phase boundary.
type PhaseSnapshot struct {
	ID           int64
	TaskID       string
	Phase        Phase
	Round        int
	SnapshotJSON string
	Checksum     string
	CreatedAt    int64
}

// AuditRecord logs security and compliance events.
type AuditRecord struct {
	ID           string
	TaskID       string
	Category     string
	Actor        string
	Action       string
	RequestJSON  string
	DecisionJSON string
	Severity     string
	CreatedAt    int64
}

// Scores holds the 5-dimension review scores (1-5 each).
type Scores struct {
	Correctness     int
	Security        int
	Maintainability int
	Cost            int
	DeliveryRisk    int
}

// Issue represents a problem found during review.
type Issue struct {
	Severity    string
	Location    string
	Description string
	Suggestion  string
	Evidence    string
}

// ScoreCard is a structured review output from a reviewer.
type ScoreCard struct {
	ReviewID     string
	Reviewer     string
	Scores       Scores
	Issues       []Issue
	Alternatives []string
	Verdict      string
}

// ConsensusResult is the aggregated review decision.
type ConsensusResult struct {
	WeightedScore float64
	Blocking      bool
	BlockReasons  []string
	FinalVerdict  string
}

// Provider identifies a code agent provider.
type Provider string

const (
	ProviderClaude Provider = "claude"
	ProviderCodex  Provider = "codex"
	ProviderGemini Provider = "gemini"
)

// SessionConfig configures a code agent session.
type SessionConfig struct {
	TaskID      string
	Role        string
	Workspace   string
	Env         map[string]string
	TimeoutSec  int
	ContextFile string
}

// NormalizedEvent is a provider-agnostic event from a code agent session.
type NormalizedEvent struct {
	Type      string
	Provider  Provider
	SessionID string
	Payload   []byte
}

// CostDelta records a cost increment.
type CostDelta struct {
	InputTokens  int64
	OutputTokens int64
	AmountUSD    float64
	Provider     Provider
	Phase        Phase
}

// CostAction is the decision from the cost governor.
type CostAction string

const (
	CostContinue CostAction = "continue"
	CostWarn     CostAction = "warn"
	CostHalt     CostAction = "halt"
)
