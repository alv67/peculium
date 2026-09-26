<script lang="ts">
  import type { HTMLAttributes } from 'svelte/elements'
  import { t } from '$lib/i18n/index.svelte'
  import { cx } from './utils'

  /**
   * Inline loading indicator. Announced via `role="status"` so screen readers
   * pick it up when it appears (e.g. inside a loading `Button`).
   */
  let {
    size = 'md',
    class: className = '',
    ...rest
  }: { size?: 'sm' | 'md' | 'lg' } & HTMLAttributes<HTMLSpanElement> = $props()

  const sizes = {
    sm: 'h-3.5 w-3.5',
    md: 'h-5 w-5',
    lg: 'h-6 w-6',
  } as const
</script>

<span
  role="status"
  aria-label={t('a11y.loading')}
  class={cx('inline-flex shrink-0 animate-spin text-current', sizes[size], className)}
  {...rest}
>
  <svg viewBox="0 0 24 24" fill="none" class="h-full w-full" aria-hidden="true">
    <circle cx="12" cy="12" r="10" stroke="currentColor" stroke-opacity="0.25" stroke-width="4" />
    <path
      d="M12 2a10 10 0 0 1 10 10"
      stroke="currentColor"
      stroke-width="4"
      stroke-linecap="round"
    />
  </svg>
</span>
