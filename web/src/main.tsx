import React from "react";
import ReactDOM from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { NuqsAdapter } from "nuqs/adapters/react";
import App from "./App";
import "./index.css";
import { ThemeProvider } from "./lib/theme";

const queryClient = new QueryClient({
  defaultOptions: {
    queries: {
      // Tasks change often — keep a tight refetch cadence so the kanban
      // reflects mutations made via the CLI in another shell quickly.
      staleTime: 5_000,
      refetchInterval: 10_000,
    },
  },
});

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <NuqsAdapter>
      <QueryClientProvider client={queryClient}>
        <ThemeProvider>
          <App />
        </ThemeProvider>
      </QueryClientProvider>
    </NuqsAdapter>
  </React.StrictMode>
);
