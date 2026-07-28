import { cleanup, render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import {
  THEME_STORAGE_KEY,
  ThemeProvider,
  resolveTheme,
  useTheme,
} from "./theme";

function Probe() {
  const { theme, toggleTheme } = useTheme();
  return (
    <button type="button" onClick={toggleTheme}>
      {theme}
    </button>
  );
}

describe("theme", () => {
  const values = new Map<string, string>();

  beforeEach(() => {
    values.clear();
    vi.stubGlobal("localStorage", {
      getItem: (key: string) => values.get(key) ?? null,
      setItem: (key: string, value: string) => values.set(key, value),
      removeItem: (key: string) => values.delete(key),
      clear: () => values.clear(),
    });
  });

  afterEach(() => {
    cleanup();
    vi.unstubAllGlobals();
    delete document.documentElement.dataset.theme;
  });

  it("defaults to Dracula and only accepts paper as the light preference", () => {
    expect(resolveTheme(null)).toBe("dracula");
    expect(resolveTheme("unknown")).toBe("dracula");
    expect(resolveTheme("paper")).toBe("paper");
  });

  it("persists an explicit theme choice and updates the document", async () => {
    const user = userEvent.setup();
    render(
      <ThemeProvider>
        <Probe />
      </ThemeProvider>
    );

    expect(screen.getByRole("button", { name: "dracula" })).toBeTruthy();
    expect(document.documentElement.dataset.theme).toBe("dracula");

    await user.click(screen.getByRole("button", { name: "dracula" }));

    expect(screen.getByRole("button", { name: "paper" })).toBeTruthy();
    expect(window.localStorage.getItem(THEME_STORAGE_KEY)).toBe("paper");
    expect(document.documentElement.dataset.theme).toBe("paper");
  });
});
