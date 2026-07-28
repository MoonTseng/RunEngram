import { describe, expect, it } from "vitest";
import { formatElapsedTime, formatRelativeTime } from "./time";

describe("localized task time", () => {
  const now = Date.parse("2026-07-24T12:00:00.000Z");

  it("formats elapsed time in zh-CN", () => {
    const timestamp = Date.parse("2026-07-24T11:15:00.000Z");

    expect(formatElapsedTime(timestamp, now, "zh-CN")).toBe("45 分钟");
  });

  it("formats relative time in zh-CN", () => {
    const timestamp = Date.parse("2026-07-24T11:57:00.000Z");

    expect(formatRelativeTime(timestamp, now, "zh-CN")).toBe("3 分钟前");
  });
});
