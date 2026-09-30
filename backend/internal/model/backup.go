package model

import (
	"time"

	"github.com/shopspring/decimal"
)

// UserBackupVersion is the format version written by the per-user backup
// exporter. The document is additive like the portfolio export format:
// optional fields may be absent on import (tolerated with sensible
// defaults), and a document whose version is newer than what the importer
// understands is rejected explicitly.
const UserBackupVersion = 1

// UserBackup is the per-user JSON bundle: everything needed to rebuild a
// user's data on a freshly initialized server after the account has been
// recreated manually. Portfolios are carried as standard portfolio-export
// documents (each entry is independently importable by the portfolio import
// endpoint); the asset-level data they share lives once per ticker at the
// top level, because assets are global. FX history and health events are
// provider data and are deliberately excluded, and so are Yahoo prices,
// which the price worker refetches.
type UserBackup struct {
	Version    int               `json:"version"`
	ExportedAt time.Time         `json:"exported_at"`
	User       BackupUser        `json:"user"`
	Currencies []ExportCurrency  `json:"currencies,omitempty"`
	Portfolios []PortfolioExport `json:"portfolios"`
	Assets     []BackupAsset     `json:"assets"`
}

// BackupUser identifies the account the bundle came from (email and name
// are informational) and carries the only user setting restore applies: the
// base currency.
type BackupUser struct {
	Email        string `json:"email,omitempty"`
	Name         string `json:"name,omitempty"`
	BaseCurrency string `json:"base_currency,omitempty"`
}

// ExportCurrency is one supported-currency whitelist entry in use: the
// user's base currency plus every portfolio and asset currency. Restore
// adds missing codes and never modifies existing entries, since the
// whitelist is global.
type ExportCurrency struct {
	Code   string `json:"code"`
	Name   string `json:"name,omitempty"`
	Symbol string `json:"symbol,omitempty"`
}

// BackupAsset is the shared, asset-level part of the bundle: the identity
// metadata exactly as the portfolio export writes it, plus the exposure
// (countries/regions/sectors rows with their provenance source) and the
// manual price rows, which are user input no provider can refetch.
type BackupAsset struct {
	ExportAsset
	Exposure     *AssetExposure `json:"exposure,omitempty"`
	ManualPrices []ExportPrice  `json:"manual_prices,omitempty"`
}

// ExportPrice is one manual price bar keyed by (asset ticker, date). Source
// is the stored price source ("manual" for every exported row); it is
// optional on import and falls back to "manual".
type ExportPrice struct {
	Date   time.Time       `json:"date"`
	Open   decimal.Decimal `json:"open"`
	High   decimal.Decimal `json:"high"`
	Low    decimal.Decimal `json:"low"`
	Close  decimal.Decimal `json:"close"`
	Volume int64           `json:"volume,omitempty"`
	Source string          `json:"source,omitempty"`
}

// BackupRestoreSummary is the response of a user backup restore: the mode
// actually applied and what it produced.
type BackupRestoreSummary struct {
	Mode                string `json:"mode"`
	PortfoliosCreated   int    `json:"portfolios_created"`
	TransactionsCreated int    `json:"transactions_created"`
	AssetsCreated       int    `json:"assets_created"`
	AssetsReused        int    `json:"assets_reused"`
}
