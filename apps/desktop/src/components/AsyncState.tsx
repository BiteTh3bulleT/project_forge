import type { ReactNode } from "react";

export function AsyncState(props: {
  error?: string | null;
  loading?: boolean;
  loadingText?: string;
  empty?: boolean;
  emptyTitle?: string;
  emptyDetail?: string;
  children?: ReactNode;
}) {
  if (props.error) {
    return (
      <div
        className="rounded border border-forge-ember/30 bg-forge-ember/10 p-3 text-sm text-forge-ash"
        role="alert"
      >
        {props.error}
      </div>
    );
  }
  if (props.loading) {
    return (
      <div className="text-sm text-forge-mist" role="status">
        {props.loadingText ?? "Loading"}
      </div>
    );
  }
  if (props.empty) {
    return (
      <div className="forge-ops-card border-dashed p-4 text-sm">
        {props.emptyTitle ? (
          <div className="font-semibold text-forge-ash">{props.emptyTitle}</div>
        ) : null}
        {props.emptyDetail ? (
          <div className="mt-1 text-xs leading-5 text-forge-mist/70">
            {props.emptyDetail}
          </div>
        ) : null}
      </div>
    );
  }
  return <>{props.children}</>;
}
