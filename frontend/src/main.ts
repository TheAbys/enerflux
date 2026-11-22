import { createApp } from "vue";
import App from "./App.vue";
import { i18n } from "./i18n";
import { router } from "./router";
import "./styles.css";

const appRoot = document.querySelector<HTMLElement>("#vue-app");

if (appRoot) {
  createApp(App).use(i18n).use(router).mount(appRoot);
}
