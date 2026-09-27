/**
 * Site chrome strings (nav, footer, hero) for the two locales.
 * Page body copy lives inline in the single-locale page files under src/pages.
 */

export type Locale = 'en' | 'it';

export const LOCALES: Locale[] = ['en', 'it'];

/** Nav keys in display order → locale-root-relative paths. */
export const NAV = {
  home: '/',
  install: '/install',
  releases: '/releases',
  features: '/features',
} as const;

export type NavKey = keyof typeof NAV;

interface Strings {
  siteName: string;
  navLabel: Record<NavKey, string>;
  switchLangLabel: string;
  githubLabel: string;
  hero: {
    tagline: string;
    lede: string;
    ctaInstall: string;
    ctaRelease: string;
    ghRepo: string;
    releaseNote: string;
  };
  releasesError: { text: string; linkText: string };
  cardsHeading: string;
  features: { title: string; body: string }[];
  teasers: {
    installTitle: string;
    installBody: string;
    installCta: string;
    releasesTitle: string;
    releasesBody: string;
    releasesCta: string;
  };
  footer: {
    blurb: string;
    project: string;
    sourceCode: string;
    /** Author-credit label; the name itself is a literal in the layout. */
    author: string;
    copyright: string;
  };
  comingSoon: string;
  screenshotSoon: string;
}

export const UI: Record<Locale, Strings> = {
  en: {
    siteName: 'Peculium',
    navLabel: {
      home: 'Home',
      install: 'Install',
      releases: 'Releases',
      features: 'Features',
    },
    switchLangLabel: 'Italiano',
    githubLabel: 'GitHub repository',
    hero: {
      tagline: 'Peculium — Your wealth, self-hosted.',
      lede:
        'A self-hosted, multi-user personal finance and investment suite for your homelab. ' +
        'Track portfolios, transactions and live market data — your data stays on your server.',
      ctaInstall: 'Install Peculium',
      ctaRelease: 'Latest release',
      ghRepo: 'View on GitHub',
      releaseNote: 'v{version} on GitHub',
    },
    releasesError: {
      text: 'Release information is unavailable right now.',
      linkText: 'See all releases on GitHub',
    },
    cardsHeading: 'Everything you need to run a serious portfolio',
    features: [
      {
        title: 'Self-hosted',
        body:
          'The full stack runs on your machine with a single docker compose up. No third-party service ever sees your positions.',
      },
      {
        title: 'Multi-user',
        body:
          'Real accounts and per-user workspaces — share the instance with your family, keep your portfolios separate.',
      },
      {
        title: 'Investment tracking',
        body:
          'Buy, sell and dividend transactions across multiple portfolios, with ticker and asset metadata kept for you.',
      },
      {
        title: 'Live market data',
        body:
          'Quotes pulled from Yahoo Finance, price-history backfill and asset lookup by ticker, with editable exchange and ISIN metadata.',
      },
      {
        title: 'Performance & ROI',
        body:
          'Total value, gain/loss, per-asset ROI and historical performance charts, right on the dashboard.',
      },
      {
        title: 'Allocation analysis',
        body:
          'Aggregation by asset class, geography and sector with donut charts, including ETF exposure imported per holding.',
      },
    ],
    teasers: {
      installTitle: 'Running in minutes',
      installBody:
        'Peculium ships ready-to-run container images — pull them and bring the whole stack up with one command.',
      installCta: 'Read the install guide',
      releasesTitle: 'Stay current',
      releasesBody:
        'Every published version comes with end-user release notes, mirrored in English and Italian.',
      releasesCta: 'Browse releases',
    },
    footer: {
      blurb: 'A self-hosted personal finance & investment suite for your homelab.',
      project: 'Project',
      sourceCode: 'Source code',
      author: 'Created by',
      copyright: '© {year} Peculium contributors · MIT License',
    },
    comingSoon: 'This page is being written — full guide coming soon.',
    screenshotSoon: 'Screenshot coming soon',
  },
  it: {
    siteName: 'Peculium',
    navLabel: {
      home: 'Home',
      install: 'Installazione',
      releases: 'Release',
      features: 'Funzionalità',
    },
    switchLangLabel: 'English',
    githubLabel: 'Repository GitHub',
    hero: {
      // Localized rendering of the EN brand tagline
      // (`Peculium — Your wealth, self-hosted.`).
      tagline: 'Peculium — Il tuo patrimonio, self-hosted.',
      lede:
        'Una suite self-hosted e multi-utente per la finanza personale e gli investimenti, pensata per il tuo homelab. ' +
        'Monitora portfolio, transazioni e dati di mercato in tempo reale: i tuoi dati restano sul tuo server.',
      ctaInstall: 'Installa Peculium',
      ctaRelease: 'Ultima release',
      ghRepo: 'Vedi su GitHub',
      releaseNote: 'v{version} su GitHub',
    },
    releasesError: {
      text: 'In questo momento le informazioni sulle release non sono disponibili.',
      linkText: 'Vedi tutte le release su GitHub',
    },
    cardsHeading: 'Tutto ciò che serve per gestire un portfolio serio',
    features: [
      {
        title: 'Self-hosted',
        body:
          "L'intera applicazione gira sulla tua macchina con un singolo docker compose up. Nessun servizio di terze parti vede mai le tue posizioni.",
      },
      {
        title: 'Multi-utente',
        body:
          'Account reali e spazi di lavoro per utente — condividi l’istanza con la tua famiglia, mantenendo i portfolio separati.',
      },
      {
        title: 'Tracciamento degli investimenti',
        body:
          'Transazioni di acquisto, vendita e dividendi su più portfolio, con ticker e metadati degli asset gestiti per te.',
      },
      {
        title: 'Dati di mercato live',
        body:
          'Prezzi da Yahoo Finance, backfill dello storico e ricerca asset per ticker, con metadati di exchange e ISIN modificabili.',
      },
      {
        title: 'Performance e ROI',
        body:
          "Valore totale, utile/perdita, ROI per asset e storico delle performance, direttamente in dashboard.",
      },
      {
        title: 'Analisi di allocazione',
        body:
          'Aggregazione per classe di asset, area geografica e settore con grafici a ciambella, esposizione ETF inclusa.',
      },
    ],
    teasers: {
      installTitle: 'In esecuzione in pochi minuti',
      installBody:
        'Peculium pubblica immagini container pronte all’uso: scaricali e avvia tutta la stack con un solo comando.',
      installCta: "Leggi la guida all'installazione",
      releasesTitle: 'Sempre aggiornato',
      releasesBody:
        'Ogni versione pubblicata arriva con note di release per l’utente finale, in inglese e in italiano.',
      releasesCta: 'Vedi le release',
    },
    footer: {
      blurb: 'Una suite self-hosted per la finanza personale e gli investimenti del tuo homelab.',
      project: 'Progetto',
      sourceCode: 'Codice sorgente',
      author: 'Creato da',
      copyright: '© {year} Contributori di Peculium · Licenza MIT',
    },
    comingSoon: 'Questa pagina è in lavorazione — guida completa in arrivo.',
    screenshotSoon: 'Screenshot in arrivo',
  },
};
