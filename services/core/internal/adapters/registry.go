package adapters

import (
	"context"
	"database/sql"
	"fmt"
)

type Options struct {
	DB           *sql.DB
	WorkspaceDir string
}

type Registry struct {
	byID map[string]Adapter
}

func NewRegistry(opts Options) *Registry {
	r := &Registry{byID: map[string]Adapter{}}
	r.Register(NewOllama(opts.DB))
	r.Register(NewCodex(opts.WorkspaceDir))
	r.Register(NewClaudeCode(opts.WorkspaceDir))
	return r
}

func (r *Registry) Register(a Adapter) {
	r.byID[a.Info(context.Background()).ID] = a
}

func (r *Registry) List(ctx context.Context) []AdapterInfo {
	out := make([]AdapterInfo, 0, len(r.byID))
	for _, a := range r.byID {
		out = append(out, a.Info(ctx))
	}
	return out
}

func (r *Registry) Get(id string) (Adapter, error) {
	a, ok := r.byID[id]
	if !ok {
		return nil, fmt.Errorf("unknown adapter %q", id)
	}
	return a, nil
}
