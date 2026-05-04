package forgek

import (
	"fmt"

	"forge/projectforge/services/core/internal/forgek/contextcompiler"
	"forge/projectforge/services/core/internal/forgek/court"
	"forge/projectforge/services/core/internal/forgek/kv"
	"forge/projectforge/services/core/internal/forgek/lymphatic"
	"forge/projectforge/services/core/internal/forgek/palace"
	forgekRuntime "forge/projectforge/services/core/internal/forgek/runtime"
	"forge/projectforge/services/core/internal/forgek/semantic"
	"forge/projectforge/services/core/internal/forgek/snapshots"
)

type KernelOptions struct {
	IDs   IDProvider
	Clock Clock
}

type Kernel struct {
	objects      *ObjectRegistry
	syscalls     *SyscallRegistry
	capabilities *CapabilityManager
	journal      *Journal
	court        *court.Service
	palace       *palace.Service
	semantic     *semantic.SemanticAlgebraService
	snapshots    *snapshots.Service
	context      *contextcompiler.Service
	kv           *kv.Service
	runtime      *forgekRuntime.Service
	lymphatic    *lymphatic.Service
	ids          IDProvider
	clock        Clock
}

func NewKernel(options KernelOptions) *Kernel {
	ids := options.IDs
	if ids == nil {
		ids = NewSequenceIDProvider(nil)
	}
	clock := options.Clock
	if clock == nil {
		clock = realClock{}
	}

	kernel := &Kernel{
		objects:      NewObjectRegistry(),
		syscalls:     NewSyscallRegistry(),
		capabilities: NewCapabilityManager(),
		journal:      NewJournal(ids),
		court:        court.NewService(),
		palace:       palace.NewService(),
		semantic:     semantic.NewSemanticAlgebraService(),
		snapshots:    snapshots.NewService(),
		context:      contextcompiler.NewService(),
		kv:           kv.NewService(),
		runtime:      forgekRuntime.NewService(),
		lymphatic:    lymphatic.NewService(),
		ids:          ids,
		clock:        clock,
	}
	kernel.registerCoreSyscalls()
	return kernel
}

func (k *Kernel) Objects() *ObjectRegistry {
	return k.objects
}

func (k *Kernel) Syscalls() *SyscallRegistry {
	return k.syscalls
}

func (k *Kernel) Capabilities() *CapabilityManager {
	return k.capabilities
}

func (k *Kernel) Journal() *Journal {
	return k.journal
}

func (k *Kernel) Court() *court.Service {
	return k.court
}

func (k *Kernel) Palace() *palace.Service {
	return k.palace
}

func (k *Kernel) Semantic() *semantic.SemanticAlgebraService {
	return k.semantic
}

func (k *Kernel) Snapshots() *snapshots.Service {
	return k.snapshots
}

func (k *Kernel) ContextCompiler() *contextcompiler.Service {
	return k.context
}

func (k *Kernel) KV() *kv.Service {
	return k.kv
}

func (k *Kernel) Runtime() *forgekRuntime.Service {
	return k.runtime
}

func (k *Kernel) Lymphatic() *lymphatic.Service {
	return k.lymphatic
}

func (k *Kernel) DispatchSyscall(request SyscallRequest) SyscallResult {
	definition, ok := k.syscalls.Lookup(request.Name)
	if !ok {
		return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrUnknownSyscall}
	}
	if definition.ValidateInput != nil {
		if err := definition.ValidateInput(request); err != nil {
			return SyscallResult{Success: false, SyscallName: request.Name, Error: err}
		}
	}
	if definition.SideEffects {
		allowed, capabilityRefs := k.capabilities.CanCall(request.ActorID, request.WorkspaceID, request.Name, true, k.clock.Now())
		if !allowed {
			return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrCapabilityDenied}
		}
		beforeJournalCount := k.journal.len()
		result := definition.Handler(k, request, capabilityRefs)
		if result.Success && definition.JournalRequired && k.journal.len() <= beforeJournalCount {
			return SyscallResult{Success: false, SyscallName: request.Name, Error: ErrJournalRequired}
		}
		return result
	}
	return definition.Handler(k, request, nil)
}

func (k *Kernel) SubmitNeuralProposal(_ NeuralProposal) SyscallResult {
	return SyscallResult{Success: false, Error: ErrProposalOnly}
}

func (k *Kernel) registerCoreSyscalls() {
	mustRegister := func(definition SyscallDefinition) {
		if err := k.syscalls.Register(definition); err != nil {
			panic(fmt.Sprintf("register syscall %s: %v", definition.Name, err))
		}
	}

	mustRegister(SyscallDefinition{
		Name:            SyscallCaseOpen,
		Version:         "v1",
		AllowedLanes:    []string{"arterial"},
		Deterministic:   true,
		SideEffects:     true,
		JournalRequired: true,
		Replayable:      true,
		ValidateInput:   validateCaseOpen,
		Handler:         handleCaseOpen,
	})
	mustRegister(SyscallDefinition{
		Name:            SyscallCaseUpdate,
		Version:         "v1",
		AllowedLanes:    []string{"arterial"},
		Deterministic:   true,
		SideEffects:     true,
		JournalRequired: true,
		Replayable:      true,
		ValidateInput:   validateCaseID,
		Handler:         handleCaseUpdate,
	})
	mustRegister(SyscallDefinition{
		Name:            SyscallCaseClose,
		Version:         "v1",
		AllowedLanes:    []string{"arterial"},
		Deterministic:   true,
		SideEffects:     true,
		JournalRequired: true,
		Replayable:      true,
		ValidateInput:   validateCaseID,
		Handler:         handleCaseClose,
	})
	mustRegister(SyscallDefinition{
		Name:          SyscallObjectGet,
		Version:       "v1",
		AllowedLanes:  []string{"arterial"},
		Deterministic: true,
		Replayable:    true,
		Handler:       handleObjectGet,
	})
	mustRegister(SyscallDefinition{
		Name:          SyscallObjectList,
		Version:       "v1",
		AllowedLanes:  []string{"arterial"},
		Deterministic: true,
		Replayable:    true,
		Handler:       handleObjectList,
	})
	mustRegister(SyscallDefinition{
		Name:            SyscallCapabilityGrant,
		Version:         "v1",
		AllowedLanes:    []string{"arterial"},
		Deterministic:   true,
		SideEffects:     true,
		JournalRequired: true,
		Replayable:      true,
		Handler:         handleCapabilityGrant,
	})
	k.registerCourtSyscalls(mustRegister)
	k.registerPalaceSyscalls(mustRegister)
	k.registerSemanticSyscalls(mustRegister)
	k.registerSnapshotSyscalls(mustRegister)
	k.registerContextSyscalls(mustRegister)
	k.registerKVSyscalls(mustRegister)
	k.registerRuntimeSyscalls(mustRegister)
	k.registerLymphaticSyscalls(mustRegister)
}
