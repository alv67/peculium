<script lang="ts">
  import { formatCurrency, formatSignedPercent } from '#lib/format.js'
  import { pnlColorClass } from '#lib/ui-colors.js'
  import { cx } from './utils'

  /**
   * Canonical gain/loss renderer (EPIC K.1c, decision D6): the single place
   * where P/L is formatted — never colour alone. Every non-zero value carries
   * an explicit sign, a ▲▼ glyph and the semantic `positive`/`negative` tone
   * from `ui-colors.ts`; zero/empty stay neutral `muted-foreground`, never
   * red (K.1c rule). A visually-hidden "positive"/"negative" word gives the
   * same information to screen readers, whose reading of bare "+"/"−"/▲▼
   * varies between products.
   *
   * `kind='currency'` formats via `formatCurrency` (unsigned) with the sign
   * prefixed manually; `kind='percent'` delegates to `formatSignedPercent`,
   * which already carries the sign — inputs are percentage numbers, not
   * fractions (9.1 → "+9.10%"). Null/blank/unparseable values render an em
   * dash. Digits use `tabular-nums` per the design rules.
   *
   * The English sr-only words follow the pre-i18n primitives (Spinner's
   * "Loading", Modal's "Close dialog") and move through the D1 dictionary at
   * adoption time; the future CVD palette toggle (D6, K.5) is expected to
   * flip colors inside this component only.
   */
  let {
    value,
    kind = 'currency',
    currency = 'USD',
    showArrow = true,
    size = 'base',
    class: className = '',
  }: {
    value: number | string | null | undefined
    kind?: 'currency' | 'percent'
    /** ISO code, only used by `kind='currency'`. */
    currency?: string
    /** ▲/▼ glyph next to the signed number (hidden for zero/empty). */
    showArrow?: boolean
    size?: 'sm' | 'base' | 'lg'
    class?: string
  } = $props()

  const sizes = {
    sm: 'text-xs',
    base: 'text-sm',
    lg: 'text-lg',
  } as const

  /** One pass: normalise → sign → text → color (keeps TS narrowing honest). */
  const pnl = $derived.by(() => {
    const n = value === null || value === undefined || value === '' ? null : Number(value)
    if (n === null || !Number.isFinite(n)) {
      return { sign: 'empty', text: '—', color: 'text-muted-foreground' }
    }
    const sign = n > 0 ? 'positive' : n < 0 ? 'negative' : 'zero'
    const text =
      kind === 'percent'
        ? formatSignedPercent(n)
        : `${n > 0 ? '+' : n < 0 ? '-' : ''}${formatCurrency(Math.abs(n), currency)}`
    return { sign, text, color: pnlColorClass(n, 'text-muted-foreground') }
  })
</script>

<span
  class={cx('inline-flex items-center gap-1 tabular-nums', sizes[size], pnl.color, className)}
>
  {#if showArrow && (pnl.sign === 'positive' || pnl.sign === 'negative')}
    <span aria-hidden="true" class="text-[0.8em] leading-none">
      {pnl.sign === 'positive' ? '▲' : '▼'}
    </span>
  {/if}
  {#if pnl.sign === 'positive' || pnl.sign === 'negative'}
    <span class="sr-only">{pnl.sign}</span>
  {/if}
  {pnl.text}
</span>
