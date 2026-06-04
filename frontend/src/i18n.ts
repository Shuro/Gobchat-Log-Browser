import { createI18n } from 'vue-i18n'
import en from './locales/en.json'
import de from './locales/de.json'

export type AppLocale = 'en' | 'de'

function normalize(locale: string | undefined): AppLocale {
  return locale === 'de' ? 'de' : 'en'
}

export const i18n = createI18n({
  legacy: false,
  locale: 'en',
  fallbackLocale: 'en',
  messages: { en, de },
})

export function setLocale(locale: string | undefined): void {
  i18n.global.locale.value = normalize(locale)
}

// mergeBackend folds the backend's localized strings (errors, notifications)
// into the active locale so the frontend can render them too.
export function mergeBackend(locale: string | undefined, messages: Record<string, string>): void {
  i18n.global.mergeLocaleMessage(normalize(locale), messages)
}
