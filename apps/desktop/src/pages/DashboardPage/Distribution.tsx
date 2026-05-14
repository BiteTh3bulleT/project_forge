export function Distribution(props: {
  title: string;
  counts: Record<string, number>;
}) {
  const entries = Object.entries(props.counts).sort((a, b) => b[1] - a[1]);
  const total = entries.reduce((sum, item) => sum + item[1], 0);
  return (
    <div>
      <div className="forge-ops-label">{props.title}</div>
      <div className="mt-3 space-y-3">
        {entries.length === 0 ? (
          <div className="text-xs text-forge-mist/65">No data.</div>
        ) : null}
        {entries.slice(0, 6).map(([label, value]) => {
          const pct = total <= 0 ? 0 : Math.round((value / total) * 100);
          return (
            <div key={label}>
              <div className="mb-1 flex justify-between text-xs">
                <span className="text-forge-mist/70">{label}</span>
                <span className="font-semibold text-forge-ash">{value}</span>
              </div>
              <div className="forge-ops-progress">
                <span style={{ width: `${Math.max(4, pct)}%` }} />
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );
}
