<script lang="ts">
  import type { ComponentType, Snippet, SvelteComponentTyped } from 'svelte'
  import type { IconProps } from 'lucide-svelte'
  import { cx } from './utils'

  /**
   * "Nothing here yet" block (EPIC D.2). `dashed` renders the boxed variant
   * used inside empty grids/tables; otherwise it is a plain centered column.
   */
  let {
    title,
    description = undefined,
    icon = undefined,
    action = undefined,
    dashed = false,
    class: className = '',
  }: {
    title: string
    description?: string
    /** lucide-svelte component, rendered at a muted 32px. Typed as the
     * legacy class shape lucide v1 actually ships (same recipe as the
     * `typeof Icon` aliases in the layout components). */
    icon?: ComponentType<SvelteComponentTyped<IconProps>>
    /** Optional CTA snippet (usually a `Button`). */
    action?: Snippet
    dashed?: boolean
    class?: string
  } = $props()
</script>

<div
  class={cx(
    'flex flex-col items-center justify-center gap-1 p-10 text-center',
    dashed && 'rounded-card border border-dashed border-border',
    className,
  )}
>
  {#if icon}
    {@const Icon = icon}
    <div class={cx('mb-1 text-muted-foreground', dashed && 'mb-2')}>
      <Icon class="h-8 w-8" aria-hidden="true" />
    </div>
  {/if}
  <p class="text-sm font-medium text-foreground">{title}</p>
  {#if description}
    <p class="text-sm text-muted-foreground">{description}</p>
  {/if}
  {#if action}
    <div class="mt-3">{@render action()}</div>
  {/if}
</div>
