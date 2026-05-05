package consensus

import "time"

func NewEvidenceRef(ref EvidenceRef) (EvidenceRef, error) {
	ref.EvidenceID = trim(ref.EvidenceID)
	ref.EvidenceType = EvidenceType(trim(string(ref.EvidenceType)))
	ref.Tier = EvidenceTier(trim(string(ref.Tier)))
	ref.Source = trim(ref.Source)
	ref.Locator = trim(ref.Locator)
	ref.FreshnessScore = clamp01(ref.FreshnessScore)
	ref.ReliabilityScore = clamp01(ref.ReliabilityScore)
	ref.SourceHash = trim(ref.SourceHash)
	ref.Metadata = CloneMap(ref.Metadata)
	if ref.RetrievedAt.IsZero() {
		ref.RetrievedAt = time.Time{}
	}
	if err := ValidateEvidenceRef(ref); err != nil {
		return EvidenceRef{}, err
	}
	return ref, nil
}

func ValidateEvidenceRef(ref EvidenceRef) error {
	if ref.EvidenceID == "" || !ValidEvidenceType(ref.EvidenceType) || !ValidEvidenceTier(ref.Tier) ||
		ref.Source == "" || ref.Locator == "" || ref.FreshnessScore < 0 || ref.FreshnessScore > 1 ||
		ref.ReliabilityScore < 0 || ref.ReliabilityScore > 1 || containsSecretMetadata(ref.Metadata) {
		return ErrInvalidEvidenceRef
	}
	if ref.Tier == EvidenceTier3 && ref.EvidenceType != EvidenceModelInference {
		return ErrInvalidEvidenceRef
	}
	return nil
}

func EvidenceTierWeight(tier EvidenceTier) float64 {
	switch tier {
	case EvidenceTier1:
		return 1
	case EvidenceTier2:
		return 0.75
	case EvidenceTier3:
		return 0.30
	default:
		return 0
	}
}

func EvidenceQuality(ref EvidenceRef) float64 {
	return EvidenceTierWeight(ref.Tier) * ref.ReliabilityScore
}
