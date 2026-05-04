package forgek

import "sync"

type SyscallHandler func(*Kernel, SyscallRequest, []string) SyscallResult

type SyscallDefinition struct {
	Name                string
	Version             string
	RequiredPermissions []string
	AllowedLanes        []string
	Deterministic       bool
	SideEffects         bool
	JournalRequired     bool
	Replayable          bool
	ValidateInput       func(SyscallRequest) error
	Handler             SyscallHandler
}

type SyscallRegistry struct {
	mu      sync.RWMutex
	entries map[string]SyscallDefinition
}

func NewSyscallRegistry() *SyscallRegistry {
	return &SyscallRegistry{entries: make(map[string]SyscallDefinition)}
}

func (r *SyscallRegistry) Register(definition SyscallDefinition) error {
	if definition.Name == "" || definition.Handler == nil {
		return ErrInvalidInput
	}
	r.mu.Lock()
	defer r.mu.Unlock()

	r.entries[definition.Name] = definition
	return nil
}

func (r *SyscallRegistry) Lookup(name string) (SyscallDefinition, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	definition, ok := r.entries[name]
	return definition, ok
}
