import { authApi, type User } from '$lib/services/api'

export const auth = $state({
  user: null as User | null,
  isLoading: true,
})

/** Admin gate for the shell chrome (issue #57 Phase A): `owner` and `admin`
 * are admin-equivalent server-side, and this mirrors that rule so the
 * sidebar/bottom-nav/palette can hide the admin entries from everyone else
 * (module-level `$derived` is not allowed in `.svelte.ts`, but calling this
 * during any render tracks `auth.user`, so the UI updates live when the
 * role changes). The `/admin/*` pages re-check it client-side and the
 * backend enforces it again with a 403. */
export function isAdmin(): boolean {
  return auth.user !== null && (auth.user.role === 'owner' || auth.user.role === 'admin')
}

export async function initAuth(): Promise<void> {
  const token = localStorage.getItem('access_token')
  if (!token) {
    auth.isLoading = false
    return
  }
  try {
    const res = await authApi.me()
    auth.user = res
  } catch {
    localStorage.removeItem('access_token')
    localStorage.removeItem('refresh_token')
  } finally {
    auth.isLoading = false
  }
}

export async function login(email: string, password: string): Promise<void> {
  const res = await authApi.login(email, password)
  localStorage.setItem('access_token', res.access_token)
  localStorage.setItem('refresh_token', res.refresh_token)
  auth.user = res.user
  window.location.replace('/')
}

/** Creates the account and returns it: the caller inspects `status` to tell
 * the user whether approval is pending (auto-approve off, #57 Phase A). */
export async function register(email: string, name: string, password: string): Promise<User> {
  return authApi.register(email, name, password)
}

export async function updateProfile(
  name: string,
  email: string,
  baseCurrency?: string,
): Promise<void> {
  // `baseCurrency` is optional: when omitted the PATCH body simply lacks
  // `base_currency` and the backend keeps the stored value (EPIC I.1).
  const user = await authApi.updateProfile({ name, email, base_currency: baseCurrency })
  auth.user = user
}

export function logout(): void {
  localStorage.removeItem('access_token')
  localStorage.removeItem('refresh_token')
  auth.user = null
  window.location.replace('/login')
}
