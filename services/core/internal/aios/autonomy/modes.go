package autonomy

import "forge/projectforge/services/core/internal/aios/domain"

func SupportsSelfCommit(mode domain.AutonomyMode) bool {
	switch mode {
	case domain.AutonomyModeMaintain, domain.AutonomyModeMission:
		return true
	default:
		return false
	}
}

func IsObservationOnly(mode domain.AutonomyMode) bool {
	switch mode {
	case domain.AutonomyModeOff, domain.AutonomyModeObserve:
		return true
	default:
		return false
	}
}
