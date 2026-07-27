import { useCallback, useEffect, useState } from 'react'

export type Theme = 'light' | 'dark'

const STORAGE_KEY = 'kiosk.theme'

/** Matches the duration in index.css. */
const FADE_MS = 280

function systemPreference(): Theme {
  return window.matchMedia('(prefers-color-scheme: dark)').matches ? 'dark' : 'light'
}

function stored(): Theme | null {
  try {
    const value = localStorage.getItem(STORAGE_KEY)
    return value === 'light' || value === 'dark' ? value : null
  } catch {
    return null
  }
}

/**
 * Light or dark, remembered.
 *
 * The starting point is whatever the operating system already says, because
 * somebody who has set their machine to dark has asked once already and
 * should not have to ask again here. An explicit choice overrides it and is
 * remembered.
 */
export function useTheme() {
  const [theme, setTheme] = useState<Theme>(() => stored() ?? systemPreference())

  useEffect(() => {
    document.documentElement.dataset.theme = theme
    try {
      localStorage.setItem(STORAGE_KEY, theme)
    } catch {
      /* private browsing; the choice just will not survive a reload */
    }
  }, [theme])

  // Follow the system while the visitor has not expressed a preference of
  // their own, so switching the OS to dark at sunset switches this too.
  useEffect(() => {
    if (stored()) return
    const media = window.matchMedia('(prefers-color-scheme: dark)')
    const onChange = (event: MediaQueryListEvent) => setTheme(event.matches ? 'dark' : 'light')
    media.addEventListener('change', onChange)
    return () => media.removeEventListener('change', onChange)
  }, [])

  const toggle = useCallback(() => {
    // The fade is switched on for the length of the change and then off
    // again, so it never applies to anything else that repaints.
    const root = document.documentElement
    root.setAttribute('data-theme-transition', '')
    window.setTimeout(() => root.removeAttribute('data-theme-transition'), FADE_MS + 40)

    setTheme((current) => (current === 'dark' ? 'light' : 'dark'))
  }, [])

  return { theme, toggle }
}
