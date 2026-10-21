<script lang="ts">
  /**
   * Small pill that tells the user where the weights shown in a box come from
   * and — when the dimension has actually been persisted — when it was last
   * updated (e.g. "da Morningstar (2026-09-05)"). Unknown or null sources
   * render nothing (e.g. a dimension never persisted and never prefilled).
   * A null/empty `updatedAt` means the shown source is not persisted yet
   * (unsaved prefill or fresh manual edit): the badge shows the label only,
   * and the date appears after the next successful save.
   *
   * Built on the design-system `Badge` (EPIC D.2): the outline variant gives
   * the pill border, while `bg-surface` + a muted text tone keep the original
   * subtle look; the provenance identity dots stay on the chart tokens.
   */
  import Badge from '#lib/components/ui/Badge.svelte'
  import { t, type MessageKey } from '#lib/i18n/index.svelte.js'

  let {
    source = null as string | null,
    updatedAt = null as string | null,
  }: { source?: string | null; updatedAt?: string | null } = $props()

  interface Provenance {
    /** `provenance.*Label` key: the pill text. */
    labelKey: MessageKey
    /** Swatch background class: the chart tokens double as the provenance
     *  identity colors so the dot keeps working in both themes. */
    dot: string
    /** `provenance.*Desc` key: long explanation exposed via title/aria. */
    descKey: MessageKey
  }

  // Keys (not strings) so both halves re-render when the locale flips
  // (EPIC K bug-fix: the badge is user-visible copy).
  const PROVENANCE: Record<string, Provenance> = {
    manual: {
      labelKey: 'provenance.manualLabel',
      dot: 'bg-chart-muted',
      descKey: 'provenance.manualDesc',
    },
    justetf: {
      labelKey: 'provenance.justetfLabel',
      dot: 'bg-chart-11',
      descKey: 'provenance.justetfDesc',
    },
    morningstar: {
      labelKey: 'provenance.morningstarLabel',
      dot: 'bg-chart-3',
      descKey: 'provenance.morningstarDesc',
    },
    'morningstar-regions': {
      labelKey: 'provenance.morningstarRegionsLabel',
      dot: 'bg-chart-3',
      descKey: 'provenance.morningstarRegionsDesc',
    },
    yahoo: {
      labelKey: 'provenance.yahooLabel',
      dot: 'bg-chart-12',
      descKey: 'provenance.yahooDesc',
    },
    derived: {
      labelKey: 'provenance.derivedLabel',
      dot: 'bg-chart-5',
      descKey: 'provenance.derivedDesc',
    },
    'derived-etf': {
      labelKey: 'provenance.derivedEtfLabel',
      dot: 'bg-chart-5',
      descKey: 'provenance.derivedEtfDesc',
    },
    issuer_country: {
      labelKey: 'provenance.issuerCountryLabel',
      dot: 'bg-chart-5',
      descKey: 'provenance.issuerCountryDesc',
    },
  }

  const info = $derived(source ? PROVENANCE[source] ?? null : null)

  /** YYYY-MM-DD part of the last persisted update; '' when not persisted yet. */
  const date = $derived(updatedAt ? updatedAt.slice(0, 10) : '')
  const label = $derived(info ? (date ? `${t(info.labelKey)} (${date})` : t(info.labelKey)) : '')
  const hint = $derived(
    info
      ? date
        ? `${t(info.descKey)} — ${t('provenance.updatedTo', { date })}`
        : t(info.descKey)
      : '',
  )
</script>

{#if info}
  <Badge
    variant="outline"
    class="bg-surface px-2 whitespace-nowrap text-muted-foreground"
    title={hint}
    aria-label={`${label}: ${hint}`}
  >
    <span class="h-1.5 w-1.5 shrink-0 rounded-full {info.dot}"></span>
    {label}
  </Badge>
{/if}
