import React from "react";
import ReactDOM from "react-dom/client";
import { App } from "./modules/app/App";
import { ToastProvider } from "./shared/ui/ToastProvider";
import "./styles/index.css";

ReactDOM.createRoot(document.getElementById("root") as HTMLElement).render(
  <React.StrictMode>
    <ToastProvider>
      <App />
    </ToastProvider>
  </React.StrictMode>
);
