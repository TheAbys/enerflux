import { createI18n } from "vue-i18n";
import { watch } from "vue";
import de from "./messages/de";
import en, { type LocaleMessages } from "./messages/en";

export type Locale = "de" | "en";

const storageKey = "enerflux.locale";

function initialLocale(): Locale {
  try {
    const saved = localStorage.getItem(storageKey);
    if (saved === "en" || saved === "de") return saved;
  } catch {
    // Keep German as the default when browser storage is unavailable.
  }
  return "de";
}

export const i18n = createI18n<LocaleMessages, Locale, false>({
  legacy: false,
  globalInjection: true,
  locale: initialLocale(),
  fallbackLocale: "de",
  messages: { de, en },
});

declare module "vue-i18n" {
  export interface DefineLocaleMessage extends LocaleMessages {}
}

watch(i18n.global.locale, (locale) => {
  document.documentElement.lang = locale;
  try {
    localStorage.setItem(storageKey, locale);
  } catch {
    // The language remains active for this session if browser storage is unavailable.
  }
}, { immediate: true });

export function intlLocale(): string {
  return i18n.global.locale.value === "de" ? "de-DE" : "en-GB";
}
