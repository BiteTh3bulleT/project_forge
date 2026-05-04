package semantic

import "sync"

type SemanticAlgebraService struct {
	mu         sync.RWMutex
	objects    map[string]SemanticObject
	operations map[string]SemanticOperation
	registry   *OperatorRegistry
}

func NewSemanticAlgebraService() *SemanticAlgebraService {
	return &SemanticAlgebraService{
		objects:    make(map[string]SemanticObject),
		operations: make(map[string]SemanticOperation),
		registry:   NewDefaultOperatorRegistry(),
	}
}

func (s *SemanticAlgebraService) Registry() *OperatorRegistry {
	return s.registry
}

func (s *SemanticAlgebraService) ApplyOperation(request OperationRequest) (SemanticOperation, SemanticTransformResult, error) {
	if request.OperationType == "" {
		return SemanticOperation{}, SemanticTransformResult{}, ErrInvalidOperation
	}
	ctx := OperatorContext{
		OperationID:   request.OperationID,
		ResultID:      request.ResultID,
		OperationType: request.OperationType,
		WorkspaceID:   request.WorkspaceID,
		CaseID:        request.CaseID,
		InputObjects:  cloneObjects(request.InputObjects),
		Parameters:    cloneMap(request.Parameters),
		CreatedBy:     request.CreatedBy,
		CreatedAt:     request.CreatedAt,
		NextObjectID:  request.NextObjectID,
	}
	result, err := s.registry.Dispatch(ctx)
	if err != nil {
		return SemanticOperation{}, SemanticTransformResult{}, err
	}
	operation, err := NewSemanticOperation(SemanticOperationInput{
		OperationID:      request.OperationID,
		OperationType:    request.OperationType,
		WorkspaceID:      request.WorkspaceID,
		CaseID:           request.CaseID,
		InputObjectRefs:  objectIDs(request.InputObjects),
		OutputObjectRefs: result.OutputRefs,
		OperatorVersion:  "v1",
		Parameters:       request.Parameters,
		ReasoningSummary: request.OperationType + " applied",
		ProvenanceRefs:   result.ProvenanceRefs,
		CreatedBy:        request.CreatedBy,
		CreatedAt:        request.CreatedAt,
	})
	if err != nil {
		return SemanticOperation{}, SemanticTransformResult{}, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	s.operations[operation.OperationID] = operation.Clone()
	for _, object := range result.OutputObjects {
		s.objects[object.SemanticObjectID] = object.Clone()
	}
	return operation.Clone(), result.Clone(), nil
}

func (s *SemanticAlgebraService) Merge(request OperationRequest) (SemanticTransformResult, error) {
	return s.applyOnly(withOperationType(request, OperationMerge))
}

func (s *SemanticAlgebraService) Diff(request OperationRequest) (SemanticTransformResult, error) {
	return s.applyOnly(withOperationType(request, OperationDiff))
}

func (s *SemanticAlgebraService) Intersect(request OperationRequest) (SemanticTransformResult, error) {
	return s.applyOnly(withOperationType(request, OperationIntersect))
}

func (s *SemanticAlgebraService) Contradict(request OperationRequest) (SemanticTransformResult, error) {
	return s.applyOnly(withOperationType(request, OperationContradict))
}

func (s *SemanticAlgebraService) Supersede(request OperationRequest) (SemanticTransformResult, error) {
	return s.applyOnly(withOperationType(request, OperationSupersede))
}

func (s *SemanticAlgebraService) Compress(request OperationRequest) (SemanticTransformResult, error) {
	return s.applyOnly(withOperationType(request, OperationCompress))
}

func (s *SemanticAlgebraService) Derive(request OperationRequest) (SemanticTransformResult, error) {
	return s.applyOnly(withOperationType(request, OperationDerive))
}

func (s *SemanticAlgebraService) Promote(request OperationRequest) (SemanticTransformResult, error) {
	return s.applyOnly(withOperationType(request, OperationPromote))
}

func (s *SemanticAlgebraService) Demote(request OperationRequest) (SemanticTransformResult, error) {
	return s.applyOnly(withOperationType(request, OperationDemote))
}

func (s *SemanticAlgebraService) Expire(request OperationRequest) (SemanticTransformResult, error) {
	return s.applyOnly(withOperationType(request, OperationExpire))
}

func (s *SemanticAlgebraService) applyOnly(request OperationRequest) (SemanticTransformResult, error) {
	_, result, err := s.ApplyOperation(request)
	return result, err
}

func (s *SemanticAlgebraService) StoreOperation(operation SemanticOperation) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.operations[operation.OperationID] = operation.Clone()
}

func (s *SemanticAlgebraService) StoreObject(object SemanticObject) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.objects[object.SemanticObjectID] = object.Clone()
}

func (s *SemanticAlgebraService) GetOperation(operationID string) (SemanticOperation, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	operation, ok := s.operations[operationID]
	if !ok {
		return SemanticOperation{}, false
	}
	return operation.Clone(), true
}

func (s *SemanticAlgebraService) ListOperations(workspaceID string) []SemanticOperation {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]SemanticOperation, 0)
	for _, operation := range s.operations {
		if operation.WorkspaceID == workspaceID {
			out = append(out, operation.Clone())
		}
	}
	return out
}

func (s *SemanticAlgebraService) GetObject(objectID string) (SemanticObject, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	object, ok := s.objects[objectID]
	if !ok {
		return SemanticObject{}, false
	}
	return object.Clone(), true
}

func withOperationType(request OperationRequest, operationType string) OperationRequest {
	request.OperationType = operationType
	return request
}

func objectIDs(objects []SemanticObject) []string {
	ids := make([]string, 0, len(objects))
	for _, object := range objects {
		ids = append(ids, object.SemanticObjectID)
	}
	return ids
}
