import { describe, expect, it } from "vitest";

import { sanitizeMessageContent } from "./messageContent";

describe("sanitizeMessageContent", () => {
  it("renders embedded structured model errors as readable text", () => {
    const content =
      'model backend unavailable: {"error":{"message":"500 Internal Server Error: backend unavailable","type":"api_error","param":null,"code":null}} (MODEL_CHAT_RETRY_EXHAUSTED)';

    expect(sanitizeMessageContent(content)).toBe(
      "model backend unavailable: 500 Internal Server Error: backend unavailable (MODEL_CHAT_RETRY_EXHAUSTED)",
    );
  });

  it("leaves non-error JSON-shaped text intact", () => {
    const content = 'The payload is {"ok":true}.';

    expect(sanitizeMessageContent(content)).toBe(content);
  });
});
