<script lang="ts">
  import { cx } from './utils'

  /**
   * Accessible on/off switch (issue #57 Phase A, admin server settings).
   * A `role="switch"` button: the boolean lives in the caller's state
   * (`checked`), and `onchange` receives the next value — same one-way data
   * flow as the other controls, so the caller can apply optimistic updates
   * and roll back on failure. `aria-label` names the setting (use `Field`
   * only for visually-labelled inputs; the switch is typically paired with
   * its own heading in the settings row).
   */
  let {
    checked,
    onchange,
    disabled = false,
    ariaLabel,
    class: className = '',
  }: {
    checked: boolean
    /** Called with the next value when the user toggles the switch. */
    onchange: (next: boolean) => void
    disabled?: boolean
    /** Accessible name of the setting being toggled. */
    ariaLabel: string
    class?: string
  } = $props()
</script>

<button
  type="button"
  role="switch"
  aria-checked={checked}
  aria-label={ariaLabel}
  {disabled}
  onclick={() => onchange(!checked)}
  class={cx(
    'focus-ring relative inline-flex h-6 w-11 shrink-0 items-center rounded-full border transition-colors',
    'disabled:cursor-not-allowed disabled:opacity-50',
    checked ? 'border-accent bg-accent' : 'border-input bg-muted',
    className,
  )}
>
  <span
    aria-hidden="true"
    class={cx(
      'pointer-events-none inline-block h-4 w-4 transform rounded-full bg-surface shadow transition-transform',
      checked ? 'translate-x-6' : 'translate-x-1',
    )}
  ></span>
</button>
