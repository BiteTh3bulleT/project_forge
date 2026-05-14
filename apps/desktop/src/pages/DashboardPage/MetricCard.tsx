import {
  metricAccentClass,
  statusDotClass,
  statusPillClass,
} from "./dashboardData";

export function MetricCard(props: {
  label: string;
  value: string;
  detail: string;
  tone: string;
  spark?: boolean;
  sparkBad?: boolean;
}) {
  return (
    <div className="forge-ops-card min-h-[8.25rem] p-4">
      <span
        className={[
          "absolute inset-x-0 top-0 h-0.5",
          metricAccentClass(props.tone),
        ].join(" ")}
      />
      <div className="flex items-start justify-between gap-3">
        <div className="min-w-0">
          <div className="forge-ops-label">{props.label}</div>
          <div className="mt-2 truncate text-3xl font-semibold tracking-normal text-forge-ash">
            {props.value}
          </div>
        </div>
        {props.spark || props.sparkBad ? (
          <div
            className={
              props.sparkBad
                ? "forge-ops-sparkline forge-ops-sparkline--bad"
                : "forge-ops-sparkline"
            }
          />
        ) : (
          <span className={statusPillClass(props.tone)}>{props.tone}</span>
        )}
      </div>
      <div className="mt-3 flex items-center gap-2 text-xs text-forge-mist/65">
        <span
          className={[
            "h-1.5 w-1.5 shrink-0 rounded-full",
            statusDotClass(props.tone),
          ].join(" ")}
        />
        <span className="min-w-0 truncate">{props.detail}</span>
      </div>
    </div>
  );
}
