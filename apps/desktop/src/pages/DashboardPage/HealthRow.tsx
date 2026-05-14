export function HealthRow(props: {
  label: string;
  value: number;
  tone: string;
  total: number;
}) {
  const pct =
    props.total <= 0 ? 0 : Math.round((props.value / props.total) * 100);
  return (
    <div>
      <div className="mb-1 flex items-center justify-between gap-3 text-xs">
        <span className="text-forge-mist/70">{props.label}</span>
        <span className="font-semibold text-forge-ash">
          {props.value} ({pct}%)
        </span>
      </div>
      <div className="forge-ops-progress">
        <span style={{ width: `${Math.max(4, pct)}%` }} />
      </div>
    </div>
  );
}
