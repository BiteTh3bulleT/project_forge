import type { ReactNode } from "react";

type FoldSectionProps = {
  title: string;
  subtitle?: string;
  defaultOpen?: boolean;
  children: ReactNode;
};

export function FoldSection(props: FoldSectionProps) {
  return (
    <details className="forge-fold" open={props.defaultOpen}>
      <summary className="forge-fold__summary">
        <span className="forge-fold__title">{props.title}</span>
        {props.subtitle ? (
          <span className="forge-fold__subtitle">{props.subtitle}</span>
        ) : null}
      </summary>
      <div className="forge-fold__body">{props.children}</div>
    </details>
  );
}
