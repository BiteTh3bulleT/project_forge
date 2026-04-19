import type { ButtonHTMLAttributes, ReactNode } from "react";

export function Panel(props: {
  title: string;
  subtitle?: string;
  children: ReactNode;
  className?: string;
  actions?: ReactNode;
}) {
  return (
    <section className={joinClassName("forge-panel overflow-hidden", props.className)}>
      <header className="forge-panel__head">
        <div className="min-w-0 flex-1">
          <h2 className="forge-panel__title">{props.title}</h2>
          {props.subtitle ? <p className="forge-panel__sub">{props.subtitle}</p> : null}
        </div>
        {props.actions ? <div className="forge-panel__actions">{props.actions}</div> : null}
      </header>
      <div className="forge-panel__body">{props.children}</div>
    </section>
  );
}

type ForgeButtonProps = {
  children: ReactNode;
} & ButtonHTMLAttributes<HTMLButtonElement>;

function joinClassName(base: string, className?: string) {
  return className ? `${base} ${className}` : base;
}

export function GhostButton(props: ForgeButtonProps) {
  const { children, className, type = "button", ...rest } = props;
  return (
    <button type={type} className={joinClassName("forge-btn forge-btn--ghost", className)} {...rest}>
      {children}
    </button>
  );
}

export function PrimaryButton(props: ForgeButtonProps) {
  const { children, className, type = "button", ...rest } = props;
  return (
    <button type={type} className={joinClassName("forge-btn forge-btn--primary", className)} {...rest}>
      {children}
    </button>
  );
}
