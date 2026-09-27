/**
 * Release-notes loading, done at build time by reading the repository's
 * release notes directly — NO network access.
 *
 * GitHub Releases are created from `docs/RELEASE-NOTES.en.md` /
 * `docs/RELEASE-NOTES.it.md`, so the notes files are the single source of
 * truth: the website parses them per locale at build time and renders them
 * as-is. If a notes file is missing or has no version heading, the build
 * still succeeds — `readReleaseNotes` returns `[]` and `readLatestRelease`
 * returns `null`, and callers degrade to a plain link to GitHub Releases.
 *
 * The parser understands the notes format defined by the project's release
 * process — nothing more:
 *   - `## vX.Y.Z — <date>` starts a release entry (`<date>` is kept verbatim,
 *     including any trailing note such as "(first official release)");
 *   - `### <heading>` opens a section inside the entry (e.g. `Features`,
 *     `Fixes`, `Nuove funzionalità`, `Correzioni`);
 *   - `- <text>` lines are section bullets, rendered as PLAIN TEXT — Astro
 *     HTML-escapes interpolated strings, and no inline markdown is parsed;
 *   - anything before the first version heading and any `## ` section that is
 *     not a version (notably `## Unreleased`) is SKIPPED: the website shows
 *     released versions only, and an unreleased section would announce
 *     features that do not exist yet. It becomes visible when the release
 *     process renames it to `## vX.Y.Z — <date>`.
 */

import { existsSync, readFileSync } from 'node:fs';
import { dirname, join } from 'node:path';
import { fileURLToPath } from 'node:url';

export const GITHUB_REPO = 'https://github.com/alv67/peculium';
export const GITHUB_RELEASES_URL = `${GITHUB_REPO}/releases`;

const VERSION_HEADING = /^## (v\S+) — (.+)$/;
const SECTION_HEADING = /^### (.+)$/;
const BULLET = /^- (.+)$/;

/** Locate a repository-root-relative file by walking up from this module. */
function findRepoFile(relPath: string): string | null {
  // A fixed relative path (e.g. `../../../docs/...`) cannot work: at build
  // time Astro bundles this module into `website/dist/.prerender/chunks/…`,
  // so the module lives at a different depth than `website/src/lib`. Walking
  // up to the first directory that contains the file finds the repository
  // root in dev, build and CI alike — without assuming `process.cwd()`.
  let dir = dirname(fileURLToPath(import.meta.url));
  for (;;) {
    const candidate = join(dir, relPath);
    if (existsSync(candidate)) return candidate;
    const parent = dirname(dir);
    if (parent === dir) return null; // reached the filesystem root
    dir = parent;
  }
}

export function releaseTagUrl(tag: string): string {
  return `${GITHUB_RELEASES_URL}/tag/${tag}`;
}

export interface ReleaseSection {
  /** Section heading exactly as written in the notes (already localized). */
  heading: string;
  /** Bullet items as plain text. */
  items: string[];
}

export interface ReleaseEntry {
  tag: string;
  /** Text after the em dash in the version heading, verbatim. */
  date: string;
  url: string;
  sections: ReleaseSection[];
}

/** Parse `docs/RELEASE-NOTES.<locale>.md` into released entries (newest first). */
export function readReleaseNotes(locale: 'en' | 'it'): ReleaseEntry[] {
  try {
    const notesPath = findRepoFile(join('docs', `RELEASE-NOTES.${locale}.md`));
    if (!notesPath) return [];
    const entries: ReleaseEntry[] = [];
    let current: ReleaseEntry | null = null;
    let section: ReleaseSection | null = null;
    for (const line of readFileSync(notesPath, 'utf8').split(/\r?\n/)) {
      const version = VERSION_HEADING.exec(line);
      if (version) {
        current = {
          tag: version[1],
          date: version[2].trim(),
          url: releaseTagUrl(version[1]),
          sections: [],
        };
        section = null;
        entries.push(current);
        continue;
      }
      if (/^## /.test(line)) {
        // Any other h2 (`## Unreleased`, `## Something`): close the current
        // entry so its bullets can't leak into the next one.
        current = null;
        section = null;
        continue;
      }
      if (!current) continue;
      const sub = SECTION_HEADING.exec(line);
      if (sub) {
        section = { heading: sub[1].trim(), items: [] };
        current.sections.push(section);
        continue;
      }
      const item = BULLET.exec(line);
      if (item && section) section.items.push(item[1].trim());
    }
    return entries;
  } catch {
    // Missing/unreadable notes file (e.g. partial checkout): degrade gracefully.
    return [];
  }
}

export interface LatestRelease {
  tag: string;
  url: string;
  /** Release date exactly as written in the notes heading (e.g. `23 Sep 2026`). */
  date: string | null;
}

/** Newest released version, from the same per-locale notes file. */
export function readLatestRelease(locale: 'en' | 'it'): LatestRelease | null {
  const [latest] = readReleaseNotes(locale);
  if (!latest) return null;
  return { tag: latest.tag, url: latest.url, date: latest.date };
}
