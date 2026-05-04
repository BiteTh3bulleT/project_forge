package forgek

import (
	"sync"
	"time"
)

type CapabilityManager struct {
	mu           sync.RWMutex
	capabilities map[string]Capability
}

func NewCapabilityManager() *CapabilityManager {
	return &CapabilityManager{capabilities: make(map[string]Capability)}
}

// Grant is the Phase 1 bootstrap path for tests and local examples. Governed
// capability mutation can move behind capability.grant in a later phase.
func (m *CapabilityManager) Grant(capability Capability) error {
	if capability.CapabilityID == "" || capability.SubjectID == "" {
		return ErrInvalidInput
	}
	m.mu.Lock()
	defer m.mu.Unlock()

	m.capabilities[capability.CapabilityID] = cloneCapability(capability)
	return nil
}

func (m *CapabilityManager) CanCall(actorID, workspaceID, syscallName string, mutating bool, now time.Time) (bool, []string) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var refs []string
	for _, capability := range m.capabilities {
		if capability.SubjectID != actorID {
			continue
		}
		if capability.Expiration != nil && !now.Before(*capability.Expiration) {
			continue
		}
		if !contains(capability.AllowedSyscalls, syscallName) {
			continue
		}
		if !contains(capability.WorkspaceScope, workspaceID) && !contains(capability.WorkspaceScope, "*") {
			continue
		}
		if mutating && capability.MutationScope != MutationScopeCanonical {
			continue
		}
		refs = append(refs, capability.CapabilityID)
	}
	return len(refs) > 0, refs
}

func cloneCapability(capability Capability) Capability {
	capability.AllowedSyscalls = append([]string(nil), capability.AllowedSyscalls...)
	capability.WorkspaceScope = append([]string(nil), capability.WorkspaceScope...)
	if capability.Expiration != nil {
		expiration := *capability.Expiration
		capability.Expiration = &expiration
	}
	return capability
}

func contains(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
