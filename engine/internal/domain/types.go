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
	TaskID        string     `json:"taskId"`
	CurrentPhase  Phase      `json:"currentPhase"`
	Status        FlowStatus `json:"status"`
	StateVersion  int64      `json:"stateVersion"`
	Round         int        `json:"round"`
	BudgetUsedUSD float64   `json:"budgetUsedUsd"`
	BudgetCapUSD  float64   `json:"budgetCapUsd"`
	LastEventSeq  int64      `json:"lastEventSeq"`
	UpdatedAtUnix int64      `json:"updatedAtUnix"`
}

// TransitionTrigger initiates a phase transition.
type TransitionTrigger struct {
	Action  string `json:"action"`
	Actor   string `json:"actor"`
	Payload []byte `json:"payload,omitempty"`
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
	ID          int64  `json:"id"`
	TaskID      string `json:"taskId"`
	SeqNo       int64  `json:"seqNo"`
	Phase       Phase  `json:"phase"`
	EventType   string `json:"eventType"`
	PayloadJSON string `json:"payloadJson"`
	CreatedAt   int64  `json:"createdAt"`
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
	Correctness     int `json:"correctness"`
	Security        int `json:"security"`
	Maintainability int `json:"maintainability"`
	Cost            int `json:"cost"`
	DeliveryRisk    int `json:"deliveryRisk"`
}

// Issue represents a problem found during review.
type Issue struct {
	Severity    string `json:"severity"`
	Location    string `json:"location"`
	Description string `json:"description"`
	Suggestion  string `json:"suggestion"`
	Evidence    string `json:"evidence"`
}

// ScoreCard is a structured review output from a reviewer.
type ScoreCard struct {
	ReviewID     string   `json:"reviewId"`
	TaskID       string   `json:"taskId"`
	Reviewer     string   `json:"reviewer"`
	Scores       Scores   `json:"scores"`
	Issues       []Issue  `json:"issues"`
	Alternatives []string `json:"alternatives"`
	Verdict      string   `json:"verdict"`
	CreatedAt    int64    `json:"createdAt"`
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
	Type      string   `json:"type"`
	Provider  Provider `json:"provider"`
	SessionID string   `json:"sessionId"`
	Payload   []byte   `json:"payload"`
}

// CostDelta records a cost increment.
type CostDelta struct {
	InputTokens  int64    `json:"inputTokens"`
	OutputTokens int64    `json:"outputTokens"`
	AmountUSD    float64  `json:"amountUsd"`
	Provider     Provider `json:"provider"`
	Phase        Phase    `json:"phase"`
	CreatedAt    int64    `json:"createdAt"`
}

// WorkerRef tracks an active worker instance.
type WorkerRef struct {
	WorkerID       string      `json:"workerId"`
	TaskID         string      `json:"taskId"`
	Phase          Phase       `json:"phase"`
	Role           string      `json:"role"`
	State          WorkerState `json:"state"`
	FileOwnership  []string    `json:"fileOwnership"`
	SoftTimeoutSec int         `json:"softTimeoutSec"`
	HardTimeoutSec int         `json:"hardTimeoutSec"`
	LastHeartbeat  int64       `json:"lastHeartbeat"`
	CreatedAtUnix  int64       `json:"createdAtUnix"`
}

// CapabilitySheet defines allowed operations for a task.
type CapabilitySheet struct {
	TaskID          string
	AllowedPaths    []string
	AllowedCommands []string
	DeniedPatterns  []string
	CreatedAtUnix   int64
}

// CostAction is the decision from the cost governor.
type CostAction string

const (
	CostContinue CostAction = "continue"
	CostWarn     CostAction = "warn"
	CostHalt     CostAction = "halt"
)
