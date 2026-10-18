<script lang="ts">
  import { onMount } from 'svelte'
  import { goto } from '$app/navigation'
  import { resolve } from '$app/paths'
  import Spinner from '$lib/components/ui/Spinner.svelte'

  /**
   * `/admin` index (issue #57 Phase A): the management area opens directly on
   * the user list — the two admin pages are also linked from the sidebar and
   * the command palette, so no intermediate landing is needed. `replaceState`
   * keeps `/admin` out of the history; a non-admin reaching this URL is
   * redirected as well and meets the `AdminGate` forbidden state on the
   * Users page. The spinner only flashes for the moment of the redirect.
   */
  onMount(() => {
    void goto(resolve('/admin/users'), { replaceState: true, noScroll: true })
  })
</script>

<div class="flex justify-center py-12">
  <Spinner size="lg" class="text-accent-text" />
</div>
