export function arrayOrEmpty<T>(value: readonly T[] | null | undefined): T[];
export function arrayOrEmpty<T = unknown>(value: unknown): T[];
export function arrayOrEmpty<T>(value: unknown): T[] {
  return Array.isArray(value) ? (value as T[]) : [];
}
