import { defineCollection, z } from 'astro:content';
import { glob } from 'astro/loaders';

/**
 * User-manual chapters, one markdown file per chapter per locale, under
 * `src/content/docs/{en,it}/manual/`. Entry ids look like `en/manual/quickstart`;
 * the locale is the first path segment and the chapter slug the last, so the
 * `{chapter}` route param matches across locales and the language switcher
 * links the same chapter in the other language.
 *
 * `order` drives the chapter list in the landing page and the sidebar TOC;
 * chapters are added one at a time as the manual grows.
 */
const manual = defineCollection({
  loader: glob({ pattern: '**/*.md', base: './src/content/docs' }),
  schema: z.object({
    title: z.string(),
    order: z.number(),
    description: z.string(),
  }),
});

export const collections = { manual };
