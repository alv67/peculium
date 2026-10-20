<script lang="ts">
  import { Check, Monitor, Moon, Sun } from 'lucide-svelte'
  import { afterNavigate } from '$app/navigation'
  import { t } from '#lib/i18n/index.svelte.js'
  import { resolved, setThemeMode, theme, type ThemeMode } from '#lib/stores/theme.svelte.js'
  import Button from '../ui/Button.svelte'

  /**
   * Three-state theme picker (EPIC D.3): Light / Dark / System, bound to the
   * theme store (`setThemeMode` persists + syncs `<html class="dark">`).
   * Labels are translated (EPIC K.1b: the shared `theme.*` dictionary group,
   * the same keys the Preferences page uses), so the popup and the trigger
   * aria-label follow the locale.
   *
   * The trigger carries the icon of the *chosen* mode (never the resolved one
   * alone: "follow the OS" must stay visible as an intent); the popup exposes
   * the selection via `aria-pressed` and a check icon. While in `system` mode
   * the trigger's `aria-label` also announces the currently resolved theme.
   *
   * Popup behaviour: closes on Escape / outside pointerdown / route change;
   * Escape returns focus to the trigger. Sits on z-20 (dropdown tier of the
   * z-index scale in ../ui/utils.ts).
   */
  type IconType = typeof Sun

  const modes: { value: ThemeMode; label: string; icon: IconType }[] = $derived([
    { value: 'light', label: t('theme.light'), icon: Sun },
    { value: 'dark', label: t('theme.dark'), icon: Moon },
    { value: 'system', label: t('theme.system'), icon: Monitor },
  ])

  let open = $state(false);
  let root = $state<HTMLDivElement | null>(null);
  const current = $derived(modes.find((m) => m.value === theme.mode) ?? modes[0]);

  // Mid-sentence labels are lower-cased inside the composed aria-labels
  // ("Theme: dark" / "Tema: scuro"); the dictionary keys guarantee the
  // `theme.<mode>` lookups exist for every mode and resolved theme.
  const triggerLabel = $derived(
    theme.mode === 'system'
      ? t('theme.ariaSystem', {
          theme: t('theme.system').toLowerCase(),
          resolved: t(`theme.${resolved()}`).toLowerCase(),
        })
      : t('theme.aria', { theme: t(`theme.${current.value}`).toLowerCase() }),
  )

  function toggle(): void {
    open = !open
  }

  function select(mode: ThemeMode): void {
    setThemeMode(mode)
    open = false
  }

  function handleWindowKeydown(event: KeyboardEvent): void {
    if (!open) return
    if (event.key === 'Escape') {
      open = false
      // The first button inside the root is the trigger (ui/Button renders a
      // real <button> and `bind:this` on it would yield the component, not
      // the element). Escape after close → focus back where it came from.
      root?.querySelector('button')?.focus()
    }
  }

  afterNavigate(({ shallow, type }) => {
    if (shallow && type === 'goto') return;

    return open = false;
  });
  $effect(() => {
    if (!open) return
    function handlePointerdown(event: PointerEvent): void {
      if (root && event.target instanceof Node && !root.contains(event.target)) {
        open = false
      }
    }
    window.addEventListener('pointerdown', handlePointerdown)
    return () => window.removeEventListener('pointerdown', handlePointerdown)
  })
</script>

<svelte:window onkeydown={handleWindowKeydown}></svelte:window>

<div class="relative" bind:this={root}>
  <Button
    variant="ghost"
    size="icon"
    aria-label={triggerLabel}
    aria-haspopup="true"
    aria-expanded={open}
    onclick={toggle}
  >
    {@const Icon = current.icon}
    <Icon class="h-5 w-5" />
  </Button>

  {#if open}
    <div
      role="group"
      aria-label={t('theme.group')}
      class="absolute right-0 top-full z-20 mt-2 w-44 rounded-card border border-border bg-surface p-1 shadow-raised"
    >
      <div class="flex flex-col gap-0.5">
        {#each modes as item (item.value)}
          {@const Icon = item.icon}
          <Button
            variant="ghost"
            class="w-full"
            aria-pressed={theme.mode === item.value}
            onclick={() => select(item.value)}
          >
            <span class="flex w-full items-center gap-2">
              <Icon class="h-4 w-4 shrink-0" />
              <span class="flex-1 text-left">{item.label}</span>
              {#if theme.mode === item.value}
                <Check class="h-4 w-4 shrink-0 text-accent-text" />
              {/if}
            </span>
          </Button>
        {/each}
      </div>
    </div>
  {/if}
</div>
