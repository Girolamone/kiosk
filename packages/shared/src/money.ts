/**
 * Formats a price for display.
 *
 * Prices travel as integer minor units and are only ever divided at the last
 * possible moment, here. Doing it earlier means carrying a float around, and
 * a float cannot represent 0.10 exactly.
 */
export function formatMoney(cents: number, currency: string, locale?: string): string {
  return new Intl.NumberFormat(locale, {
    style: 'currency',
    currency,
  }).format(cents / 100)
}
