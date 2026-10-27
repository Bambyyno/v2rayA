import { createApp } from "vue";
import { createPinia } from "pinia";
import AppShell from "@/AppShell.vue";
import { forgetLoopbackBackend } from "@/api/client";
import i18n from "@/plugins/i18n";
import "@/plugins/dayjs";
import { vuetify } from "@/theme";

function initializeBackendAddress(): void {
  // A saved http://localhost:2017 points the panel at the viewer's own
  // machine. The page's host is the service that served it.
  forgetLoopbackBackend(window.location.href);
  // Reverse-proxy deployments serve the GUI and API under the same path prefix.
  const address = localStorage.getItem("backendAddress");
  const prefix = window.location.pathname.match(
    /^(.*)\/(?:login|setting|log|server|rule|running)?\/?$/,
  )?.[1];
  if (
    prefix &&
    prefix !== "/" &&
    (!address || (address.startsWith("/") && !address.startsWith("//")))
  ) {
    localStorage.setItem("backendAddress", prefix);
  } else if (address === null) {
    localStorage.setItem("backendAddress", "");
  }
}

export function buildApp() {
  initializeBackendAddress();
  return createApp(AppShell).use(createPinia()).use(vuetify).use(i18n);
}
