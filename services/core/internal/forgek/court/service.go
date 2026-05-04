package court

import "sync"

type Service struct {
	mu             sync.RWMutex
	exhibits       map[string]Exhibit
	rulings        map[string]Ruling
	contradictions map[string]Contradiction
	supersessions  map[string]Supersession
}

func NewService() *Service {
	return &Service{
		exhibits:       make(map[string]Exhibit),
		rulings:        make(map[string]Ruling),
		contradictions: make(map[string]Contradiction),
		supersessions:  make(map[string]Supersession),
	}
}

func (s *Service) SubmitExhibit(exhibit Exhibit) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exhibits[exhibit.ExhibitID] = exhibit.Clone()
}

func (s *Service) GetExhibit(exhibitID string) (Exhibit, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	exhibit, ok := s.exhibits[exhibitID]
	if !ok {
		return Exhibit{}, false
	}
	return exhibit.Clone(), true
}

func (s *Service) UpdateExhibit(exhibit Exhibit) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.exhibits[exhibit.ExhibitID] = exhibit.Clone()
}

func (s *Service) CreateRuling(ruling Ruling) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rulings[ruling.RulingID] = ruling.Clone()
}

func (s *Service) RegisterContradiction(contradiction Contradiction) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.contradictions[contradiction.ContradictionID] = contradiction.Clone()
}

func (s *Service) RegisterSupersession(supersession Supersession) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.supersessions[supersession.SupersessionID] = supersession
}

func (s *Service) ListCaseExhibits(caseID string) []Exhibit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Exhibit, 0)
	for _, exhibit := range s.exhibits {
		if exhibit.CaseID == caseID {
			out = append(out, exhibit.Clone())
		}
	}
	return out
}

func (s *Service) ListAdmittedExhibits(caseID string) []Exhibit {
	return s.listExhibitsByStatus(caseID, StatusAdmitted)
}

func (s *Service) ListRejectedExhibits(caseID string) []Exhibit {
	return s.listExhibitsByStatus(caseID, StatusRejected)
}

func (s *Service) listExhibitsByStatus(caseID, status string) []Exhibit {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Exhibit, 0)
	for _, exhibit := range s.exhibits {
		if exhibit.CaseID == caseID && exhibit.AdmissibilityStatus == status {
			out = append(out, exhibit.Clone())
		}
	}
	return out
}

func (s *Service) ListCaseRulings(caseID string) []Ruling {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Ruling, 0)
	for _, ruling := range s.rulings {
		if ruling.CaseID == caseID {
			out = append(out, ruling.Clone())
		}
	}
	return out
}

func (s *Service) ListCaseContradictions(caseID string) []Contradiction {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Contradiction, 0)
	for _, contradiction := range s.contradictions {
		if contradiction.CaseID == caseID {
			out = append(out, contradiction.Clone())
		}
	}
	return out
}
