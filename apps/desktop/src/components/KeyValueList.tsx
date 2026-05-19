import type { ReactNode } from "react";

type KeyValueRow = {
  label: ReactNode;
  value: ReactNode;
};

export function KeyValueList(props: {
  rows: KeyValueRow[];
  empty?: ReactNode;
  className?: string;
}) {
  if (props.rows.length === 0) {
    return props.empty ? (
      <div className="rounded border border-dashed border-forge-platinum/10 bg-black/15 p-3 text-sm text-forge-mist/65">
        {props.empty}
      </div>
    ) : null;
  }

  return (
    <dl className={["grid gap-2 md:grid-cols-2", props.className].filter(Boolean).join(" ")}>
      {props.rows.map((row, index) => (
        <div
          key={index}
          className="flex min-h-10 items-center justify-between gap-3 rounded-md border border-forge-platinum/10 bg-black/20 px-3 py-2 text-xs"
        >
          <dt className="text-forge-mist/65">{row.label}</dt>
          <dd className="min-w-0 truncate text-right font-semibold text-forge-ash">
            {row.value}
          </dd>
        </div>
      ))}
    </dl>
  );
}
