<script lang="ts">
  import type { Snippet } from 'svelte'
  import { isAdmin } from '$lib/stores/auth.svelte'
  import { t } from '$lib/i18n/index.svelte'
  import EmptyState from '../ui/EmptyState.svelte'

  /**
   * Admin-area gate (issue #57 Phase A): renders the wrapped page content
   * only while the signed-in account carries an admin-equivalent role
   * (`owner`/`admin`). Anyone else landing on the route — e.g. by typing
   * the URL — gets a forbidden empty state instead; the nav entries are
   * hidden upstream (`SidebarNav`/`BottomNav`/`CommandPalette`). The backend
   * enforces the same rule again on every `/admin/*` call (403), so this is
   * presentation-only gating.
   */
  let { children }: { children: Snippet } = $props()
</script>

{#if isAdmin()}
  {@render children()}
{:else}
  <div class="mx-auto max-w-3xl p-4 lg:p-6">
    <EmptyState
      dashed
      title={t('admin.forbiddenTitle')}
      description={t('admin.forbiddenHint')}
    />
  </div>
{/if}
