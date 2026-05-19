import type { ReactNode } from "react";

function cx(parts: Array<string | false | null | undefined>) {
  return parts.filter(Boolean).join(" ");
}

export function OpsPanel(props: {
  title: string;
  subtitle?: string;
  actions?: ReactNode;
  children: ReactNode;
  className?: string;
  bodyClassName?: string;
}) {
  return (
    <section className={cx(["forge-ops-panel", props.className])}>
      <div className="forge-ops-panel__head">
        <div>
          <div className="forge-ops-title">{props.title}</div>
          {props.subtitle ? (
            <div className="mt-1 text-xs text-forge-mist/65">
              {props.subtitle}
            </div>
          ) : null}
        </div>
        {props.actions ? (
          <div className="flex flex-wrap items-center gap-2">
            {props.actions}
          </div>
        ) : null}
      </div>
      <div className={cx(["forge-ops-panel__body", props.bodyClassName])}>
        {props.children}
      </div>
    </section>
  );
}
