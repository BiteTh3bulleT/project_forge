import { Component, type ErrorInfo, type ReactNode } from "react";

type Props = { children: ReactNode; resetKey: string };

type State = { message: string | null };

/**
 * Catches render errors so a single bad page does not leave a blank shell.
 * `resetKey` should change on navigation so a recovered route can render again.
 */
export class ForgeErrorBoundary extends Component<Props, State> {
  state: State = { message: null };

  static getDerivedStateFromError(err: unknown): State {
    return { message: err instanceof Error ? err.message : String(err) };
  }

  componentDidCatch(err: unknown, info: ErrorInfo) {
    console.error("[FORGE] view render error", err, info.componentStack);
  }

  componentDidUpdate(prevProps: Props) {
    if (prevProps.resetKey !== this.props.resetKey) {
      this.setState({ message: null });
    }
  }

  render() {
    if (this.state.message) {
      return (
        <div className="forge-panel forge-status-glow border-forge-ember/30 bg-forge-iron/90">
          <header className="forge-panel__head">
            <div>
              <h2 className="forge-panel__title text-forge-emberSoft">View error</h2>
              <p className="forge-panel__sub">Rendering stopped for this route. Check the console for the stack trace.</p>
            </div>
          </header>
          <div className="forge-panel__body">
            <pre className="max-h-[min(50vh,28rem)] overflow-auto whitespace-pre-wrap rounded-md border border-white/10 bg-black/40 p-3 font-mono text-[11px] leading-relaxed text-forge-mist">
              {this.state.message}
            </pre>
          </div>
        </div>
      );
    }
    return this.props.children;
  }
}
