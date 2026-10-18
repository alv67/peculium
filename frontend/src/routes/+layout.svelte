<script lang="ts">
  import '../app.css'
  // Self-hosted UI fonts (EPIC K.1a / D5, no CDN): Inter for the interface,
  // JetBrains Mono for tickers/ISINs. Stacks in tailwind.config.js.
  import '@fontsource/inter/400.css'
  import '@fontsource/inter/500.css'
  import '@fontsource/inter/600.css'
  import '@fontsource/inter/700.css'
  import '@fontsource/jetbrains-mono/400.css'
  import '@fontsource/jetbrains-mono/500.css'
  import { onMount } from 'svelte'
  import { page } from '$app/state'
  import { resolve } from '$app/paths'
  import { goto } from '$app/navigation'
  import { auth, initAuth } from '$lib/stores/auth.svelte'
  import '$lib/stores/theme.svelte' // side effect: theme listeners + <html class="dark"> sync
  import '$lib/i18n/index.svelte' // side effect: locale listeners + <html lang> sync (K.1b)
  import { assetApi, jobsApi } from '$lib/services/api'
  import { markDataSynced } from '$lib/stores/priceRefresh.svelte'
  import AppShell from '$lib/components/layout/AppShell.svelte'
  import Spinner from '$lib/components/ui/Spinner.svelte'
  import Toaster from '$lib/components/Toaster.svelte'

  let { children } = $props()

  let synced = $state(false)

  onMount(() => {
    initAuth()
  })

  // Queue the global asset sync (splits + history for every Yahoo asset) and,
  // when the worker finishes, invalidate the frontend cache and bump the
  // refresh revision so price-derived pages refetch on their own.
  async function syncAssets(): Promise<void> {
    try {
      const enqueued = await assetApi.sync()
      const job = await jobsApi.wait(enqueued.job_id)
      if (job.status === 'done' || job.status === 'partial') markDataSynced()
    } catch {
      // keep going; individual pages backfill what they need
    }
  }

  $effect(() => {
    if (auth.user && !synced) {
      synced = true
      void syncAssets()
    }
  })

  $effect(() => {
    if (auth.isLoading) return
    if (!auth.user && page.url.pathname !== '/login') {
      goto(resolve('/login'), { replaceState: true })
    } else if (auth.user && page.url.pathname === '/login') {
      goto(resolve('/'), { replaceState: true })
    }
  })
</script>

{#if auth.isLoading}
  <div class="grid h-dvh place-items-center bg-background">
    <Spinner size="lg" class="text-accent-text" />
  </div>
{:else if auth.user}
  <AppShell>{@render children()}</AppShell>
{:else}
  {@render children()}
{/if}

<Toaster />
