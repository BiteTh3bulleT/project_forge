import type { ReactNode } from "react";

type ToastTone = "info" | "success" | "warning" | "danger";

const toneClass: Record<ToastTone, string> = {
  danger: "border-forge-ember/30 bg-forge-ember/10 text-forge-ash",
  info: "border-forge-electric/25 bg-forge-electric/10 text-forge-ash",
  success: "border-emerald-300/30 bg-emerald-300/10 text-forge-ash",
  warning: "border-amber-300/30 bg-amber-300/10 text-forge-ash",
};

export function Toast(props: {
  children: ReactNode;
  tone?: ToastTone;
  className?: string;
}) {
  const tone = props.tone ?? "info";
  const liveRole = tone === "danger" ? "alert" : "status";
  return (
    <div
      className={[
        "rounded-md border p-3 text-sm",
        toneClass[tone],
        props.className,
      ]
        .filter(Boolean)
        .join(" ")}
      role={liveRole}
      aria-live={tone === "danger" ? "assertive" : "polite"}
    >
      {props.children}
    </div>
  );
}
