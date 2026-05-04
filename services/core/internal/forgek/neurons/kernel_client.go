package neurons

import "forge/projectforge/services/core/internal/forgek"

type KernelDispatcher interface {
	DispatchSyscall(request forgek.SyscallRequest) forgek.SyscallResult
}

type KernelSyscallClient struct {
	kernel  KernelDispatcher
	actorID string
}

func NewKernelSyscallClient(kernel KernelDispatcher, actorID string) KernelSyscallClient {
	return KernelSyscallClient{kernel: kernel, actorID: actorID}
}

func (c KernelSyscallClient) Submit(request SyscallRequest) forgek.SyscallResult {
	return c.kernel.DispatchSyscall(forgek.SyscallRequest{
		Name:        request.SyscallName,
		ActorID:     c.actorID,
		WorkspaceID: request.WorkspaceID,
		CaseID:      request.CaseID,
		Input:       cloneMap(request.Payload),
	})
}
