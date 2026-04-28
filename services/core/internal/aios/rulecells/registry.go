package rulecells

type Registry struct {
	packs []RulePack
}

func NewRegistry(packs ...RulePack) (*Registry, error) {
	r := &Registry{}
	for _, pack := range packs {
		if err := r.RegisterPack(pack); err != nil {
			return nil, err
		}
	}
	return r, nil
}

func (r *Registry) RegisterPack(pack RulePack) error {
	if err := validatePacks([]RulePack{pack}); err != nil {
		return err
	}
	r.packs = append(r.packs, pack)
	return nil
}

func (r *Registry) Packs() []RulePack {
	if r == nil {
		return nil
	}
	return append([]RulePack(nil), r.packs...)
}
