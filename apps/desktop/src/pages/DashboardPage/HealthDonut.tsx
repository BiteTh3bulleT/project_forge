import { sumHealth } from "./dashboardData";

export function HealthDonut(props: {
  segments: Array<{ value: number; tone: string }>;
}) {
  const total = sumHealth(props.segments);
  const basis = Math.max(total, 1);
  const ok =
    ((props.segments.find((item) => item.tone === "ok")?.value ?? 0) / basis) *
    100;
  const warn =
    ((props.segments.find((item) => item.tone === "warn")?.value ?? 0) /
      basis) *
    100;
  const style = {
    background:
      total > 0
        ? `conic-gradient(rgb(var(--forge-mint-rgb)) 0 ${ok}%, rgb(var(--forge-amber-rgb)) ${ok}% ${ok + warn}%, rgb(var(--forge-danger-rgb)) ${ok + warn}% 100%)`
        : "conic-gradient(rgba(255,255,255,0.12) 0 100%)",
  };
  return (
    <div
      className="mx-auto grid h-32 w-32 place-items-center rounded-full"
      style={style}
    >
      <div className="grid h-20 w-20 place-items-center rounded-full bg-[#0b0e12] text-center">
        <div>
          <div className="text-xl font-semibold text-forge-ash">{total}</div>
          <div className="text-[10px] uppercase tracking-[0.12em] text-forge-mist/55">
            signals
          </div>
        </div>
      </div>
    </div>
  );
}
