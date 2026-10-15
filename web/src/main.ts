import { createElement as h } from "react";
import { createRoot } from "react-dom/client";
import { QueryClient, QueryClientProvider } from "@tanstack/react-query";
import { Tube } from "./Tube.generated";
import { Context } from "./context";
import { useTube } from "./useTube";
import "./style.css";
const client = new QueryClient({
  defaultOptions: {
    queries: { retry: false, staleTime: 2000, refetchOnWindowFocus: false },
  },
});
function App() {
  const ui = useTube();
  return h(
    Context.Provider,
    { value: ui },
    h(Tube, { data: { app: ui.app }, onAction: ui.action }),
  );
}
createRoot(document.getElementById("root")!).render(
  h(QueryClientProvider, { client }, h(App)),
);
