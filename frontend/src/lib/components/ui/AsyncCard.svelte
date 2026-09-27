<script lang="ts">
  import type { Snippet } from 'svelte'
  import { AlertTriangle } from 'lucide-svelte'
  import { t } from '$lib/i18n/index.svelte'
  import Button from './Button.svelte'
  import EmptyState from './EmptyState.svelte'
  import Skeleton from './Skeleton.svelte'
  import { cx } from './utils'

  /**
   * Per-card async state shell (EPIC K.1c): the four states of one content
   * block — loading (shape-matched `Skeleton`s, redesign spec §8.4), error
   * (one-line cause + Retry), empty (`EmptyState`) and data — without ever
   * blanking the page: each card on a dashboard fetches independently and
   * owns its own failure surface.
   *
   * State precedence: `loading` → `error` → `empty` → content snippet, so a
   * retry (caller re-sets `loading`) hides the stale error. The default
   * loading/empty visuals are generic card shapes/text; `skeleton` and
   * `emptyContent` snippets override them once a phase needs final-geometry
   * placeholders (zero-CLS rule).
   *
    * The built-in retry and empty-state strings resolve through the i18n
    * dictionary; `retryLabel` and `emptyContent` override them per call site
    * without replacing the default visuals.
    */
  let {
    loading = false,
    error = null,
    empty = false,
    onRetry = undefined,
    retryLabel = t('common.retry'),
    class: className = '',
    children = undefined,
    emptyContent = undefined,
    skeleton = undefined,
  }: {
    loading: boolean
    /** One-line cause; a failing fetch should pass `e.message` or similar. */
    error?: string | null
    empty?: boolean
    /** When provided, the error state offers this Retry action. */
    onRetry?: () => void
    retryLabel?: string
    class?: string
    /** Data state. */
    children?: Snippet
    /** Overrides the default `EmptyState`. */
    emptyContent?: Snippet
    /** Overrides the default skeleton shapes. */
    skeleton?: Snippet
  } = $props()
</script>

<div class={cx(className)} aria-busy={loading || undefined}>
  {#if loading}
    {#if skeleton}
      {@render skeleton()}
    {:else}
      {@render defaultSkeleton()}
    {/if}
  {:else if error}
    {@render errorState()}
  {:else if empty}
    {#if emptyContent}
      {@render emptyContent()}
    {:else}
      {@render defaultEmpty()}
    {/if}
  {:else}
    {@render children?.()}
  {/if}
</div>

{#snippet defaultSkeleton()}
  <div class="space-y-3 p-1" aria-hidden="true">
    <Skeleton class="h-5 w-1/3" />
    <Skeleton class="h-4 w-2/3" />
    <Skeleton class="h-4 w-1/2" />
    <Skeleton class="h-4 w-3/5" />
  </div>
{/snippet}

{#snippet errorState()}
  <div
    class="flex min-h-[6rem] flex-col items-center justify-center gap-2 p-4 text-center"
    role="alert"
  >
    <div class="flex items-center gap-2 text-negative">
      <AlertTriangle class="h-4 w-4 shrink-0" aria-hidden="true" />
      <p class="text-sm text-foreground">{error}</p>
    </div>
    {#if onRetry}
      <Button variant="secondary" size="sm" onclick={onRetry}>{retryLabel}</Button>
    {/if}
  </div>
{/snippet}

{#snippet defaultEmpty()}
  <EmptyState title={t('common.nothingHere')} />
{/snippet}
