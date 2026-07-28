import type { Locale } from "./i18n";

// Relative-time formatter for task timestamps. Returns "just now",
// "3 mins ago", "1 day ago", etc. Plural / singular handled inline so
// we don't pull in a date library for one call site.
//
// All timestamps in the API are int64 unix-millis (see model.go) — the
// caller passes that directly; `now` defaults to Date.now() but is
// injectable for tests.

const SECOND = 1000;
const MINUTE = 60 * SECOND;
const HOUR = 60 * MINUTE;
const DAY = 24 * HOUR;
const WEEK = 7 * DAY;
const MONTH = 30 * DAY;
const YEAR = 365 * DAY;

type Unit = "min" | "hour" | "day" | "week" | "month" | "year";

function pickDuration(value: number, unit: Unit, locale: Locale): string {
  const v = Math.floor(value);
  if (locale === "zh-CN") {
    const units: Record<Unit, string> = {
      min: "分钟",
      hour: "小时",
      day: "天",
      week: "周",
      month: "个月",
      year: "年",
    };
    return `${v} ${units[unit]}`;
  }
  const plural = v === 1 ? unit : `${unit}s`;
  return `${v} ${plural}`;
}

export function formatElapsedTime(
  timestampMs: number,
  now: number = Date.now(),
  locale: Locale = "en-US"
): string {
  const diff = now - timestampMs;
  // Future timestamps shouldn't happen in normal flow, but if a client
  // clock is skewed forward we don't want to render a negative time.
  if (diff < MINUTE) return locale === "zh-CN" ? "刚刚" : "just now";
  if (diff < HOUR) return pickDuration(diff / MINUTE, "min", locale);
  if (diff < DAY) return pickDuration(diff / HOUR, "hour", locale);
  if (diff < WEEK) return pickDuration(diff / DAY, "day", locale);
  if (diff < MONTH) return pickDuration(diff / WEEK, "week", locale);
  if (diff < YEAR) return pickDuration(diff / MONTH, "month", locale);
  return pickDuration(diff / YEAR, "year", locale);
}

export function formatRelativeTime(
  timestampMs: number,
  now: number = Date.now(),
  locale: Locale = "en-US"
): string {
  const elapsed = formatElapsedTime(timestampMs, now, locale);
  if (locale === "zh-CN") return elapsed === "刚刚" ? elapsed : `${elapsed}前`;
  return elapsed === "just now" ? elapsed : `${elapsed} ago`;
}
