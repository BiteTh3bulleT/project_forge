package storagebackend

type BackendCapabilities struct {
	DurableRelational      bool
	Transactional          bool
	AdvisoryLocksAvailable bool
	VectorIndex            bool
	EphemeralCache         bool
	CanonicalTruthAllowed  bool
}

type InfrastructureKind string

const (
	InfrastructureRedis  InfrastructureKind = "redis"
	InfrastructureQdrant InfrastructureKind = "qdrant"
)

func CapabilitiesFor(kind BackendKind) BackendCapabilities {
	switch kind {
	case BackendPostgres:
		return BackendCapabilities{
			DurableRelational:      true,
			Transactional:          true,
			AdvisoryLocksAvailable: true,
			CanonicalTruthAllowed:  true,
		}
	case BackendSQLite:
		fallthrough
	default:
		return BackendCapabilities{
			DurableRelational:     true,
			Transactional:         true,
			CanonicalTruthAllowed: true,
		}
	}
}

func InfrastructureCapabilities(kind InfrastructureKind) BackendCapabilities {
	switch kind {
	case InfrastructureRedis:
		return BackendCapabilities{EphemeralCache: true}
	case InfrastructureQdrant:
		return BackendCapabilities{VectorIndex: true}
	default:
		return BackendCapabilities{}
	}
}

func CanBeCanonicalTruth(caps BackendCapabilities) bool {
	return caps.CanonicalTruthAllowed && caps.DurableRelational && caps.Transactional
}
