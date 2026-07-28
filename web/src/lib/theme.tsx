import {
  createContext,
  useContext,
  useEffect,
  useMemo,
  useState,
  type ReactNode,
} from "react";

export type Theme = "dracula" | "paper";

export const THEME_STORAGE_KEY = "runengram.theme";

export function resolveTheme(value: string | null): Theme {
  return value === "paper" ? "paper" : "dracula";
}

function storedTheme(): Theme {
  if (typeof window === "undefined") return "dracula";
  const storage =
    typeof window.localStorage?.getItem === "function" ? window.localStorage : null;
  return resolveTheme(storage?.getItem(THEME_STORAGE_KEY) ?? null);
}

type ThemeValue = {
  theme: Theme;
  setTheme: (theme: Theme) => void;
  toggleTheme: () => void;
};

const ThemeContext = createContext<ThemeValue>({
  theme: "dracula",
  setTheme: () => undefined,
  toggleTheme: () => undefined,
});

export function ThemeProvider({ children }: { children: ReactNode }) {
  const [theme, setThemeState] = useState<Theme>(storedTheme);

  const setTheme = (next: Theme) => {
    if (typeof window.localStorage?.setItem === "function") {
      window.localStorage.setItem(THEME_STORAGE_KEY, next);
    }
    setThemeState(next);
  };

  useEffect(() => {
    document.documentElement.dataset.theme = theme;
    document.documentElement.style.colorScheme = theme === "dracula" ? "dark" : "light";
  }, [theme]);

  const value = useMemo<ThemeValue>(
    () => ({
      theme,
      setTheme,
      toggleTheme: () => setTheme(theme === "dracula" ? "paper" : "dracula"),
    }),
    [theme]
  );

  return <ThemeContext.Provider value={value}>{children}</ThemeContext.Provider>;
}

export function useTheme(): ThemeValue {
  return useContext(ThemeContext);
}
