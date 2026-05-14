export function SectionRows(props: { rows: Array<[string, string]> }) {
  return (
    <div className="grid gap-2 md:grid-cols-2">
      {props.rows.map(([label, value]) => (
        <div
          key={label}
          className="flex min-h-10 items-center justify-between gap-3 rounded-md border border-forge-platinum/10 bg-black/20 px-3 py-2 text-xs"
        >
          <span className="text-forge-mist/65">{label}</span>
          <span className="min-w-0 truncate text-right font-semibold text-forge-ash">
            {value}
          </span>
        </div>
      ))}
    </div>
  );
}
