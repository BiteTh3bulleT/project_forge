export function isOptionalEndpointMissing(error: unknown): boolean {
  const message =
    error instanceof Error
      ? error.message.toLowerCase()
      : String(error).toLowerCase();
  return message.includes("404") || message.includes("not found");
}
