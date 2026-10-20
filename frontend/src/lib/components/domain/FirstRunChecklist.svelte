<script lang="ts">
  import { resolve } from '$app/paths'
  import { ArrowRight, Check } from 'lucide-svelte'
  import { t } from '#lib/i18n/index.svelte.js'
  import Card from '../ui/Card.svelte'
  import { cx } from '../ui/utils'

  /**
   * First-run checklist (EPIC K.3a, decision D8): a single ordered-list card
   * replacing the plain EmptyState on a fresh vault — create a portfolio →
   * add an asset → record a transaction, each step deep-linking to the page
   * where the action happens (transactions are recorded inside a portfolio,
   * so step 3 also points at /portfolios; the FAB action sheet already covers
   * the ≤2-tap path on phones).
   *
   * Progress is derived exclusively from what the dashboard payload already
   * knows (no extra calls). The page hides the whole card as soon as
   * portfolios exist, so at render time the portfolio step is normally the
   * current one and the others pending — the derivations stay data-driven
   * anyway so the component never lies when the payload is fresher than the
   * moment the branch was taken.
   */
  let { portfolioCount = 0, assetCount = 0, transactionSeen = false

  /** True when any portfolio already shows buy/sell activity. */
   }: { 
    portfolioCount?: number;
    assetCount?: number;
    transactionSeen?: boolean
  } = $props()

  const steps = $derived([
    {
      key: 'portfolio',
      title: t('checklist.stepPortfolio'),
      hint: t('checklist.stepPortfolioHint'),
      href: resolve('portfolios'),
      done: portfolioCount > 0
    },
    {
      key: 'asset',
      title: t('checklist.stepAsset'),
      hint: t('checklist.stepAssetHint'),
      href: resolve('assets'),
      done: assetCount > 0
    },
    {
      key: 'transaction',
      title: t('checklist.stepTransaction'),
      hint: t('checklist.stepTransactionHint'),
      href: resolve('portfolios'),
      done: transactionSeen
    }
  ]);
  const currentIndex = $derived(steps.findIndex((s) => !s.done));
</script>

<Card class="p-6">
  <h2 class="font-semibold">{t('checklist.title')}</h2>
  <p class="mt-1 text-sm text-muted-foreground">{t('checklist.intro')}</p>
  <ol class="mt-4 space-y-3">
    {#each steps as step, i (step.key)}
      {@const current = !step.done && i === currentIndex}
      <li class="flex items-center gap-3">
        {#if step.done}
          <span
            class="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-positive/10 text-positive"
            aria-hidden="true"
          >
            <Check class="h-4 w-4" />
          </span>
        {:else}
          <span
            class={cx(
              'flex h-7 w-7 shrink-0 items-center justify-center rounded-full text-sm font-semibold tabular-nums',
              current ? 'bg-accent text-accent-foreground' : 'border border-border text-muted-foreground',
            )}
            aria-hidden="true"
          >
            {i + 1}
          </span>
        {/if}
        <div class="min-w-0 flex-1">
          <!-- `resolve()`-built href lives in the derived steps array, which
               the navigation rule cannot trace (see Button.svelte's note). -->
          <!-- eslint-disable svelte/no-navigation-without-resolve -->
          <a
            href={step.href}
            class={cx(
              'focus-ring rounded text-sm font-medium hover:underline',
              step.done || current ? 'text-accent-text' : 'text-muted-foreground',
            )}
          >
            {step.title}
          </a>
          <!-- eslint-enable svelte/no-navigation-without-resolve -->
          <p class="text-xs text-muted-foreground">{step.hint}</p>
        </div>
        <!-- Step state carried in text, not only by icon/color (a11y). -->
        <span class="sr-only">
          {step.done ? t('checklist.done') : current ? t('checklist.current') : t('checklist.pending')}
        </span>
        {#if current}
          <ArrowRight class="h-4 w-4 shrink-0 text-accent-text" aria-hidden="true" />
        {/if}
      </li>
    {/each}
  </ol>
</Card>
