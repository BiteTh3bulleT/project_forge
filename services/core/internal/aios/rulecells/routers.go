package rulecells

import "context"

type Router struct {
	engine *Engine
	lane   Lane
}

func NewRouter(engine *Engine, lane Lane) Router {
	return Router{engine: engine, lane: lane}
}

func (r Router) Run(ctx context.Context, phase Phase, inputID, inputType string, facts map[string]any, opts RunOptions) (RunResult, error) {
	if r.engine == nil {
		return RunResult{}, nil
	}
	return r.engine.Run(ctx, RunInput{
		Lane:      r.lane,
		Phase:     phase,
		InputID:   inputID,
		InputType: inputType,
		Facts:     facts,
	}, opts)
}

func ArterialRouter(engine *Engine) Router  { return NewRouter(engine, LaneArterial) }
func LymphaticRouter(engine *Engine) Router { return NewRouter(engine, LaneLymphatic) }
func KernelRouter(engine *Engine) Router    { return NewRouter(engine, LaneKernel) }
func RuntimeRouter(engine *Engine) Router   { return NewRouter(engine, LaneRuntime) }
func OperatorRouter(engine *Engine) Router  { return NewRouter(engine, LaneOperator) }
func NeuralRouter(engine *Engine) Router    { return NewRouter(engine, LaneNeural) }
