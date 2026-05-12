export function hostLabelForMonitorOrdinal(ordinal: number) {
  return ordinal === 0 ? "main" : `forge-monitor-${ordinal + 1}`;
}
