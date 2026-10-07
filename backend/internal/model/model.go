package model

import (
	"time"

	"github.com/google/uuid"
	"github.com/shopspring/decimal"
)

type Role string

const (
	RoleOwner  Role = "owner"
	RoleAdmin  Role = "admin"
	RoleEditor Role = "editor"
	RoleViewer Role = "viewer"
)

// IsAdmin reports whether a role grants administrator privileges. The legacy
// owner role of pre-admin installs is admin-equivalent.
func (r Role) IsAdmin() bool { return r == RoleOwner || r == RoleAdmin }

type Status string

const (
	StatusActive   Status = "active"
	StatusPending  Status = "pending"
	StatusDisabled Status = "disabled"
)

type User struct {
	ID           uuid.UUID `json:"id"`
	Email        string    `json:"email"`
	Name         string    `json:"name"`
	PasswordHash string    `json:"-"`
	Role         Role      `json:"role"`
	Status       Status    `json:"status"`
	BaseCurrency string    `json:"base_currency"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

// ServerSettings is the singleton row (id = 1) of server-wide options edited
// from the admin area.
type ServerSettings struct {
	AutoApproveRegistrations bool      `json:"auto_approve_registrations"`
	UpdatedAt                time.Time `json:"updated_at"`
}

type Portfolio struct {
	ID          uuid.UUID `json:"id"`
	UserID      uuid.UUID `json:"user_id"`
	Name        string    `json:"name"`
	Description string    `json:"description,omitempty"`
	Currency    string    `json:"currency"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type AssetType string

const (
	AssetTypeStock      AssetType = "stock"
	AssetTypeETF        AssetType = "etf"
	AssetTypeBond       AssetType = "bond"
	AssetTypeMutualFund AssetType = "mutual_fund"
	AssetTypeCrypto     AssetType = "crypto"
	AssetTypeCommodity  AssetType = "commodity"
	AssetTypeCash       AssetType = "cash"
)

type Asset struct {
	ID                uuid.UUID        `json:"id"`
	Ticker            string           `json:"ticker"`
	ISIN              string           `json:"isin,omitempty"`
	Name              string           `json:"name"`
	Type              AssetType        `json:"type"`
	AssetClass        string           `json:"asset_class"`
	PriceSource       string           `json:"price_source"`
	Country           string           `json:"country,omitempty"`
	Currency          string           `json:"currency"`
	Exchange          string           `json:"exchange,omitempty"`
	Sector            string           `json:"sector,omitempty"`
	Industry          string           `json:"industry,omitempty"`
	CreatedAt         time.Time        `json:"created_at"`
	PriceFetchedAt    *time.Time       `json:"price_fetched_at,omitempty"`
	HistoryBackfilled bool             `json:"history_backfilled"`
	DataStatus        *AssetDataStatus `json:"data_status,omitempty"`
}

// AssetDataStatus is the derived completeness of an asset's market data
// (never stored): whether its history is fully downloaded, still syncing,
// partial or missing. Set only on the single-asset GET.
type AssetDataStatus struct {
	State     string     `json:"state"`
	JobID     *uuid.UUID `json:"job_id,omitempty"`
	JobStatus string     `json:"job_status,omitempty"`
	Processed int        `json:"processed,omitempty"`
	Total     int        `json:"total,omitempty"`
}

const (
	AssetDataComplete = "complete" // history_backfilled is true
	AssetDataSyncing  = "syncing"  // an open job covers this asset
	AssetDataPartial  = "partial"  // not backfilled but some prices stored
	AssetDataMissing  = "missing"  // not backfilled, no prices
)

// AssetQuote holds the headline metrics shown on the asset detail page. It is
// asset-scoped and independent of any portfolio.
type AssetQuote struct {
	Currency  string          `json:"currency"`
	LastClose decimal.Decimal `json:"last_close"`
	LastDate  time.Time       `json:"last_date"`
	Change1D  decimal.Decimal `json:"change_1d"`
	Change1W  decimal.Decimal `json:"change_1w"`
	Change1M  decimal.Decimal `json:"change_1m"`
	Change1Y  decimal.Decimal `json:"change_1y"`
	ChangeYTD decimal.Decimal `json:"change_ytd"`
	HasData   bool            `json:"has_data"`
}

// AssetPatch is the body accepted by PATCH /assets/{id}. Pointer fields
// distinguish "not provided" (nil) from an explicit value, so string fields can
// be cleared by sending an empty string.
type AssetPatch struct {
	Ticker      *string    `json:"ticker"`
	ISIN        *string    `json:"isin"`
	Name        *string    `json:"name"`
	Type        *AssetType `json:"type"`
	AssetClass  *string    `json:"asset_class"`
	PriceSource *string    `json:"price_source"`
	Country     *string    `json:"country"`
	Currency    *string    `json:"currency"`
	Exchange    *string    `json:"exchange"`
	Sector      *string    `json:"sector"`
	Industry    *string    `json:"industry"`
}

// ExposureRow è una singola voce di peso percentuale per una dimensione
// (regione o settore). Weight è in percentuale (somma = 100).
type ExposureRow struct {
	Name   string          `json:"name"`
	Weight decimal.Decimal `json:"weight"`
}

// Dimensioni di esposizione con provenienza persistita.
const (
	ExposureDimensionCountries = "countries"
	ExposureDimensionRegions   = "regions"
	ExposureDimensionSectors   = "sectors"
)

// ExposureProvenance è la sorgente e la data dell'ultimo aggiornamento di una
// dimensione di esposizione (es. "morningstar", "manual", "justetf").
type ExposureProvenance struct {
	Source    string    `json:"source"`
	UpdatedAt time.Time `json:"updated_at"`
}

// AssetExposure contiene la distribuzione per paese, macro-regione e settore di
// un asset. I campi *_source sono input del PUT: la sorgente da associare a
// ciascuna dimensione salvata (vuota ⇒ "manual"). Provenance è output: la
// provenienza persistita per dimensione, presente solo per le dimensioni
// salvate almeno una volta.
type AssetExposure struct {
	ISIN      string        `json:"isin,omitempty"`
	Countries []ExposureRow `json:"countries"`
	Regions   []ExposureRow `json:"regions"`
	Sectors   []ExposureRow `json:"sectors"`

	CountriesSource string `json:"countries_source,omitempty"`
	RegionsSource   string `json:"regions_source,omitempty"`
	SectorsSource   string `json:"sectors_source,omitempty"`

	Provenance map[string]ExposureProvenance `json:"provenance,omitempty"`
}

type Currency struct {
	Code      string    `json:"code"`
	Name      string    `json:"name"`
	Symbol    string    `json:"symbol"`
	Enabled   bool      `json:"enabled"`
	Sort      int       `json:"sort"`
	CreatedAt time.Time `json:"created_at"`
}

type TransactionType string

const (
	TxBuy      TransactionType = "buy"
	TxSell     TransactionType = "sell"
	TxDividend TransactionType = "dividend"
	TxSplit    TransactionType = "split"
	TxFee      TransactionType = "fee"
)

type Transaction struct {
	ID          uuid.UUID       `json:"id"`
	PortfolioID uuid.UUID       `json:"portfolio_id"`
	AssetID     uuid.UUID       `json:"asset_id"`
	Type        TransactionType `json:"type"`
	Quantity    decimal.Decimal `json:"quantity"`
	Price       decimal.Decimal `json:"price"`
	Fees        decimal.Decimal `json:"fees,omitempty"`
	Date        time.Time       `json:"date"`
	Notes       string          `json:"notes,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type TransactionWithAsset struct {
	ID          uuid.UUID       `json:"id"`
	PortfolioID uuid.UUID       `json:"portfolio_id"`
	AssetID     uuid.UUID       `json:"asset_id"`
	AssetTicker string          `json:"asset_ticker"`
	AssetName   string          `json:"asset_name"`
	Type        TransactionType `json:"type"`
	Quantity    decimal.Decimal `json:"quantity"`
	Price       decimal.Decimal `json:"price"`
	Fees        decimal.Decimal `json:"fees,omitempty"`
	Date        time.Time       `json:"date"`
	Notes       string          `json:"notes,omitempty"`
	CreatedAt   time.Time       `json:"created_at"`
}

type TransactionPage struct {
	Transactions []TransactionWithAsset `json:"transactions"`
	Total        int64                  `json:"total"`
	Limit        int                    `json:"limit"`
	Offset       int                    `json:"offset"`
}

type Price struct {
	ID        uuid.UUID       `json:"id"`
	AssetID   uuid.UUID       `json:"asset_id"`
	Date      time.Time       `json:"date"`
	Open      decimal.Decimal `json:"open"`
	High      decimal.Decimal `json:"high"`
	Low       decimal.Decimal `json:"low"`
	Close     decimal.Decimal `json:"close"`
	Volume    int64           `json:"volume"`
	Source    string          `json:"source"`
	CreatedAt time.Time       `json:"created_at"`
}

// FXRatePoint is one daily base->quote rate in the stored FX history series.
type FXRatePoint struct {
	Date time.Time       `json:"date"`
	Rate decimal.Decimal `json:"rate"`
}

type PortfolioShare struct {
	PortfolioID uuid.UUID `json:"portfolio_id"`
	UserID      uuid.UUID `json:"user_id"`
	Role        Role      `json:"role"`
	CreatedAt   time.Time `json:"created_at"`
}

type Split struct {
	AssetID     uuid.UUID       `json:"asset_id"`
	Date        time.Time       `json:"date"`
	Numerator   decimal.Decimal `json:"numerator"`
	Denominator decimal.Decimal `json:"denominator"`
}

// PortfolioExport is the JSON document produced by portfolio export and
// consumed by portfolio import. It is designed to be human-readable and
// editable: assets and transactions are referenced by ticker.
//
// Version is the document format version (export currently writes 1). Fields
// added later on are optional and additive (omitempty): importers must
// tolerate their absence and apply sensible defaults, so documents written by
// older app versions keep importing. A document whose Version is newer than
// what the importer understands is rejected explicitly.
type PortfolioExport struct {
	Version      int                 `json:"version"`
	ExportedAt   time.Time           `json:"exported_at"`
	Portfolio    ExportPortfolio     `json:"portfolio"`
	Assets       []ExportAsset       `json:"assets"`
	Transactions []ExportTransaction `json:"transactions"`
}

type ExportPortfolio struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Currency    string `json:"currency"`
}

type ExportAsset struct {
	Ticker   string    `json:"ticker"`
	Name     string    `json:"name"`
	Type     AssetType `json:"type"`
	Currency string    `json:"currency"`
	ISIN     string    `json:"isin,omitempty"`
	// PriceSource and AssetClass are additive (version-1-compatible) fields:
	// they are omitted when empty and fall back to importer defaults when
	// missing from an older document.
	PriceSource string `json:"price_source,omitempty"`
	AssetClass  string `json:"asset_class,omitempty"`
}

type ExportTransaction struct {
	Date        time.Time       `json:"date"`
	Type        TransactionType `json:"type"`
	AssetTicker string          `json:"asset_ticker"`
	Quantity    decimal.Decimal `json:"quantity"`
	Price       decimal.Decimal `json:"price"`
	Fees        decimal.Decimal `json:"fees,omitempty"`
	Notes       string          `json:"notes,omitempty"`
}
