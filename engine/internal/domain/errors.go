package domain

import "fmt"

// EngineError is the unified error type for the engine.
// Each error has a numeric code and human-readable message.
type EngineError struct {
	Code    int
	Message string
}

// Error implements the error interface.
func (e *EngineError) Error() string {
	return fmt.Sprintf("engine error %d: %s", e.Code, e.Message)
}

// NewEngineError creates a new EngineError.
func NewEngineError(code int, msg string) *EngineError {
	return &EngineError{Code: code, Message: msg}
}

// WrapEngineError creates an EngineError that includes a cause.
func WrapEngineError(code int, msg string, cause error) *EngineError {
	return &EngineError{Code: code, Message: fmt.Sprintf("%s: %v", msg, cause)}
}

// ---- Engine / FSM / Gate errors (-32010 to -32039) ----

var (
	ErrInvalidTransition = &EngineError{Code: -32010, Message: "invalid phase transition"}
	ErrPhaseGateFailed   = &EngineError{Code: -32011, Message: "phase gate evaluation failed"}
	ErrFlowNotFound      = &EngineError{Code: -32012, Message: "workflow not found"}
	ErrFlowAlreadyDone   = &EngineError{Code: -32013, Message: "workflow already completed"}
	ErrFlowBlocked       = &EngineError{Code: -32014, Message: "workflow is blocked"}
	ErrOptimisticLock    = &EngineError{Code: -32015, Message: "optimistic lock conflict: state was modified concurrently"}
	ErrInvalidPhase      = &EngineError{Code: -32016, Message: "invalid phase value"}
	ErrGateNotRegistered = &EngineError{Code: -32017, Message: "no gate registered for phase"}
	ErrFSMNotStarted     = &EngineError{Code: -32018, Message: "workflow has not been started"}
	ErrDuplicateTask     = &EngineError{Code: -32019, Message: "task already exists"}
)

// ---- Worker / Supervisor / Intent errors (-32040 to -32069) ----

var (
	ErrWorkerNotFound     = &EngineError{Code: -32040, Message: "worker not found"}
	ErrWorkerTimeout      = &EngineError{Code: -32041, Message: "worker exceeded timeout"}
	ErrIntentConflict     = &EngineError{Code: -32042, Message: "intent conflicts with existing intent"}
	ErrIntentNotFound     = &EngineError{Code: -32043, Message: "intent not found"}
	ErrWorkerReplaced     = &EngineError{Code: -32044, Message: "worker was replaced"}
	ErrLeaseExpired       = &EngineError{Code: -32045, Message: "intent lease has expired"}
	ErrFileOwnership      = &EngineError{Code: -32046, Message: "file ownership violation"}
	ErrWorkerLimitReached  = &EngineError{Code: -32047, Message: "maximum concurrent workers reached"}
	ErrIntentHashMismatch  = &EngineError{Code: -32048, Message: "intent pre-hash does not match current file"}
	ErrCompactionInvalid   = &EngineError{Code: -32049, Message: "compaction slots validation failed"}
	ErrWorkerAlreadyDone   = &EngineError{Code: -32050, Message: "worker is already in terminal state"}
)

// ---- MCP / Bridge errors (-32070 to -32099) ----

var (
	ErrMCPConnectionFailed = &EngineError{Code: -32070, Message: "MCP connection failed"}
	ErrMCPTimeout          = &EngineError{Code: -32071, Message: "MCP request timed out"}
	ErrMCPInvalidResponse  = &EngineError{Code: -32072, Message: "MCP returned invalid response"}
	ErrBridgeNotReady      = &EngineError{Code: -32073, Message: "bridge is not ready"}
	ErrSessionNotFound     = &EngineError{Code: -32074, Message: "code agent session not found"}
	ErrProviderUnavailable = &EngineError{Code: -32075, Message: "code agent provider unavailable"}
)

// ---- Guard / Permission errors (-32100 to -32129) ----

var (
	ErrPermissionDenied   = &EngineError{Code: -32100, Message: "permission denied"}
	ErrBudgetExceeded     = &EngineError{Code: -32101, Message: "budget limit exceeded"}
	ErrBudgetWarning      = &EngineError{Code: -32102, Message: "budget warning threshold reached"}
	ErrRateLimitExceeded  = &EngineError{Code: -32103, Message: "rate limit exceeded"}
	ErrForbiddenOperation = &EngineError{Code: -32104, Message: "operation is forbidden in current context"}
	ErrMaxRoundsExceeded  = &EngineError{Code: -32105, Message: "maximum review rounds exceeded"}
)

// ---- Review / Consensus errors (-32160 to -32189) ----

var (
	ErrScoreCardInvalid = &EngineError{Code: -32160, Message: "score card validation failed"}
	ErrConsensusNoCards = &EngineError{Code: -32161, Message: "consensus requires at least one score card"}
)

// ---- Store / Recovery / Config errors (-32130 to -32159) ----

var (
	ErrStoreInit       = &EngineError{Code: -32130, Message: "failed to initialize store"}
	ErrStoreQuery      = &EngineError{Code: -32131, Message: "store query failed"}
	ErrStoreWrite      = &EngineError{Code: -32132, Message: "store write failed"}
	ErrSchemaMigration = &EngineError{Code: -32133, Message: "schema migration failed"}
	ErrSnapshotCorrupt = &EngineError{Code: -32134, Message: "snapshot checksum mismatch"}
	ErrRecoveryFailed  = &EngineError{Code: -32135, Message: "recovery from snapshot failed"}
	ErrConfigInvalid   = &EngineError{Code: -32136, Message: "invalid configuration"}
	ErrDuplicateEvent  = &EngineError{Code: -32137, Message: "duplicate event sequence number"}
)
