package semantic

import (
	"sync"
	"time"
)

type OperatorHandler func(OperatorContext) (SemanticTransformResult, error)

type OperatorDefinition struct {
	OperationType         string
	Version               string
	Deterministic         bool
	InputRequirements     []string
	OutputType            string
	MutatesCanonicalState bool
	RequiresSyscall       bool
	Description           string
	Handler               OperatorHandler
}

type OperatorContext struct {
	OperationID   string
	ResultID      string
	OperationType string
	WorkspaceID   string
	CaseID        string
	InputObjects  []SemanticObject
	Parameters    map[string]any
	CreatedBy     string
	CreatedAt     time.Time
	NextObjectID  func() string
}

type OperatorRegistry struct {
	mu        sync.RWMutex
	operators map[string]OperatorDefinition
}

func NewOperatorRegistry() *OperatorRegistry {
	return &OperatorRegistry{operators: make(map[string]OperatorDefinition)}
}

func (r *OperatorRegistry) Register(definition OperatorDefinition) error {
	if definition.OperationType == "" || definition.Handler == nil {
		return ErrInvalidOperation
	}
	r.mu.Lock()
	defer r.mu.Unlock()
	r.operators[definition.OperationType] = definition
	return nil
}

func (r *OperatorRegistry) Get(operationType string) (OperatorDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	definition, ok := r.operators[operationType]
	return definition, ok
}

func (r *OperatorRegistry) List() []OperatorDefinition {
	r.mu.RLock()
	defer r.mu.RUnlock()
	out := make([]OperatorDefinition, 0, len(r.operators))
	for _, definition := range r.operators {
		out = append(out, definition)
	}
	return out
}

func (r *OperatorRegistry) Dispatch(ctx OperatorContext) (SemanticTransformResult, error) {
	definition, ok := r.Get(ctx.OperationType)
	if !ok {
		return SemanticTransformResult{}, ErrUnknownOperator
	}
	if definition.Deterministic != true {
		return SemanticTransformResult{}, ErrInvalidOperation
	}
	return definition.Handler(ctx)
}
