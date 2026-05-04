package forgek

import "sync"

type ObjectRegistry struct {
	mu      sync.RWMutex
	objects map[string]KernelObject
	cases   map[string]CasePacket
}

func NewObjectRegistry() *ObjectRegistry {
	return &ObjectRegistry{
		objects: make(map[string]KernelObject),
		cases:   make(map[string]CasePacket),
	}
}

func (r *ObjectRegistry) GetObject(objectID string) (KernelObject, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	obj, ok := r.objects[objectID]
	if !ok {
		return KernelObject{}, false
	}
	return cloneKernelObject(obj), true
}

func (r *ObjectRegistry) ListObjects() []KernelObject {
	r.mu.RLock()
	defer r.mu.RUnlock()

	out := make([]KernelObject, 0, len(r.objects))
	for _, obj := range r.objects {
		out = append(out, cloneKernelObject(obj))
	}
	return out
}

func (r *ObjectRegistry) getCase(caseID string) (CasePacket, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	cp, ok := r.cases[caseID]
	if !ok {
		return CasePacket{}, false
	}
	return cloneCasePacket(cp), true
}

func (r *ObjectRegistry) putCase(obj KernelObject, cp CasePacket) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.objects[obj.ObjectID] = cloneKernelObject(obj)
	r.cases[cp.CaseID] = cloneCasePacket(cp)
}

func (r *ObjectRegistry) putObject(obj KernelObject) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.objects[obj.ObjectID] = cloneKernelObject(obj)
}

func cloneKernelObject(obj KernelObject) KernelObject {
	obj.State = cloneMap(obj.State)
	obj.SourceRefs = append([]string(nil), obj.SourceRefs...)
	obj.CapabilityScope = append([]string(nil), obj.CapabilityScope...)
	obj.JournalRefs = append([]string(nil), obj.JournalRefs...)
	return obj
}

func cloneCasePacket(cp CasePacket) CasePacket {
	cp.ObjectRefs = append([]string(nil), cp.ObjectRefs...)
	cp.JournalRefs = append([]string(nil), cp.JournalRefs...)
	cp.SubmittedExhibitRefs = append([]string(nil), cp.SubmittedExhibitRefs...)
	cp.AdmittedExhibitRefs = append([]string(nil), cp.AdmittedExhibitRefs...)
	cp.RejectedExhibitRefs = append([]string(nil), cp.RejectedExhibitRefs...)
	cp.RulingRefs = append([]string(nil), cp.RulingRefs...)
	cp.ContradictionRefs = append([]string(nil), cp.ContradictionRefs...)
	cp.SupersessionRefs = append([]string(nil), cp.SupersessionRefs...)
	cp.PalaceRouteRefs = append([]string(nil), cp.PalaceRouteRefs...)
	cp.CandidateObjectRefs = append([]string(nil), cp.CandidateObjectRefs...)
	if cp.ClosedAt != nil {
		closedAt := *cp.ClosedAt
		cp.ClosedAt = &closedAt
	}
	return cp
}

func cloneMap(in map[string]any) map[string]any {
	if in == nil {
		return nil
	}
	out := make(map[string]any, len(in))
	for key, value := range in {
		switch typed := value.(type) {
		case []string:
			out[key] = append([]string(nil), typed...)
		case map[string]int:
			stats := make(map[string]int, len(typed))
			for statKey, statValue := range typed {
				stats[statKey] = statValue
			}
			out[key] = stats
		default:
			out[key] = value
		}
	}
	return out
}
