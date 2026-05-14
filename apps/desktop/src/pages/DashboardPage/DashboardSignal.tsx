export function DashboardSignal(props: {
  label: string;
  value: string;
  alignRight?: boolean;
}) {
  return (
    <div
      className={[
        "flex min-w-0 flex-wrap items-center justify-between gap-2 rounded border border-forge-platinum/10 bg-black/20 px-3 py-2",
        props.alignRight ? "sm:text-right" : "",
      ].join(" ")}
    >
      <span className="forge-ops-label shrink-0">{props.label}</span>
      <span className="w-full min-w-0 break-all font-mono text-forge-ash sm:w-auto sm:text-right">
        {props.value}
      </span>
    </div>
  );
}
