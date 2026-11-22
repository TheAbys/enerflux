import { createRouter, createWebHistory, type RouteRecordRaw } from "vue-router";

type Domain = "water" | "electricity";

export interface PageMeta {
  titleKey: string;
  labelKey: string;
  rootKey: string;
  statusKey: string;
  rootTo?: string;
  dot: boolean;
  updated: boolean;
}

declare module "vue-router" {
  interface RouteMeta extends PageMeta {}
}

const routes: RouteRecordRaw[] = [
  {
    path: "/",
    component: () => import("./pages/OverviewPage.vue"),
    meta: { titleKey: "navigation.overview", labelKey: "navigation.overview", rootKey: "app.workspace", statusKey: "app.resourceSummary", dot: true, updated: true } satisfies PageMeta,
  },
  {
    path: "/water",
    component: () => import("./pages/ResourcePage.vue"),
    props: { domain: "water" satisfies Domain },
    meta: { titleKey: "navigation.water", labelKey: "navigation.water", rootKey: "navigation.overview", rootTo: "/", statusKey: "app.meterReadings", dot: false, updated: true } satisfies PageMeta,
  },
  {
    path: "/electricity",
    component: () => import("./pages/ResourcePage.vue"),
    props: { domain: "electricity" satisfies Domain },
    meta: { titleKey: "navigation.electricity", labelKey: "navigation.electricity", rootKey: "navigation.overview", rootTo: "/", statusKey: "app.powerMeasurements", dot: false, updated: true } satisfies PageMeta,
  },
  {
    path: "/heating",
    component: () => import("./pages/HeatingPage.vue"),
    meta: { titleKey: "heating.title", labelKey: "heating.breadcrumb", rootKey: "navigation.overview", rootTo: "/", statusKey: "app.stockHistory", dot: true, updated: true } satisfies PageMeta,
  },
  {
    path: "/settings",
    component: () => import("./pages/SettingsPage.vue"),
    meta: { titleKey: "navigation.configuration", labelKey: "navigation.configuration", rootKey: "app.workspace", statusKey: "app.postgresStorage", dot: true, updated: false } satisfies PageMeta,
  },
  { path: "/pellets", redirect: "/heating" },
];

export const router = createRouter({
  history: createWebHistory(),
  routes,
  scrollBehavior: () => ({ top: 0 }),
});
