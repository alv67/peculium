/**
 * Latest-release lookup, done once at build time by reading the repository's
 * release notes directly — NO network access.
 *
 * GitHub Releases are created from `docs/RELEASE-NOTES.*.md`, so the notes
 * file is the single source of truth: the build parses the first version
 * heading (`^## (v\S+) — (.+)$`) to get the tag and the release date. If the
 * file or the heading is missing the build still succeeds and callers get
 * `null`, rendering a plain link to the GitHub Releases page instead.
 */

import { existsSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

export const GITHUB_REPO = 'https://github.com/alv67/peculium';
export const GITHUB_RELEASES_URL = `${GITHUB_REPO}/releases`;

/** Notes file location relative to the repository root. */
const NOTES_REL = join('docs', 'RELEASE-NOTES.en.md');

/**
 * Locate `docs/RELEASE-NOTES.en.md` by walking up from this module's URL.
 * A fixed relative path (e.g. `../../../docs/...`) cannot work: at build time
 * Astro bundles this module into `website/dist/.prerender/chunks/…`, so the
 * module lives at a different depth than `website/src/lib`. Walking up to the
 * first directory that contains the notes file finds the repository root in
 * dev, build and CI alike — without assuming `process.cwd()`.
 */
function findReleaseNotes(): string | null {
  let dir = dirname(fileURLToPath(import.meta.url));
  for (;;) {
    const candidate = join(dir, NOTES_REL);
    if (existsSync(candidate)) return candidate;
    const parent = dirname(dir);
    if (parent === dir) return null; // reached the filesystem root
    dir = parent;
  }
}

export interface LatestRelease {
  tag: string;
  url: string;
  /** Release date exactly as written in the notes heading (e.g. `23 Sep 2026`). */
  date: string | null;
}

export function readLatestRelease(): LatestRelease | null {
  try {
    const notesPath = findReleaseNotes();
    if (!notesPath) return null;
    const notes = readFileSync(notesPath, 'utf8');
    const heading = /^## (v\S+) — (.+)$/m.exec(notes);
    if (!heading) return null;
    const [, tag, date] = heading;
    return {
      tag,
      url: `${GITHUB_RELEASES_URL}/tag/${tag}`,
      date: date.trim(),
    };
  } catch {
    // Missing/unreadable notes file (e.g. partial checkout): degrade gracefully.
    return null;
  }
}
