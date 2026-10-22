package model

import (
	"time"

	"github.com/shopspring/decimal"
)

type PortfolioSummary struct {
	PortfolioID   string `json:"portfolio_id"`
	PortfolioName string `json:"portfolio_name"`
	// Active/Closed are the same roll-ups the dashboard exposes per portfolio
	// (see PortfolioPerformanceSummary), in the portfolio currency.
	Active         ActiveBreakdown `json:"active"`
	Closed         ClosedBreakdown `json:"closed"`
	TotalValue     decimal.Decimal `json:"total_value"`
	TotalCost      decimal.Decimal `json:"total_cost"`
	GainLoss       decimal.Decimal `json:"gain_loss"`
	GainLossPct    decimal.Decimal `json:"gain_loss_pct"`
	RealizedGL     decimal.Decimal `json:"realized_gl"`
	UnrealizedGL   decimal.Decimal `json:"unrealized_gl"`
	AssetCount     int             `json:"asset_count"`
	FXMissingCount int             `json:"fx_missing_count"`
	FXMissingValue decimal.Decimal `json:"fx_missing_value"`
	MissingCountry int             `json:"missing_country"`
	MissingSector  int             `json:"missing_sector"`
	StaleCount     int             `json:"stale_count"`
	Holdings       []AssetHolding  `json:"holdings"`
}

type AssetAllocation struct {
	AssetID   string          `json:"asset_id"`
	Ticker    string          `json:"ticker"`
	Name      string          `json:"name"`
	Value     decimal.Decimal `json:"value"`
	AllocPct  decimal.Decimal `json:"alloc_pct"`
	FXMissing bool            `json:"fx_missing"`
}

// ClassAllocation is one investment-class bucket within a portfolio's class
// allocation. Weight is in percentage (sum = 100).
type ClassAllocation struct {
	Class  string          `json:"class"`
	Value  decimal.Decimal `json:"value"`
	Weight decimal.Decimal `json:"weight"`
}

// PortfolioClassAllocation groups the class allocation of a portfolio in its
// reference currency.
type PortfolioClassAllocation struct {
	Currency string             `json:"currency"`
	Classes  []*ClassAllocation `json:"classes"`
}

// RegionAllocation is one macro-region bucket within a portfolio's geographic
// allocation. Weight is in percentage (sum over all buckets = 100).
type RegionAllocation struct {
	Region string          `json:"region"`
	Value  decimal.Decimal `json:"value"`
	Weight decimal.Decimal `json:"weight"`
}

// PortfolioGeographyAllocation groups the geographic allocation of a portfolio
// in its reference currency. Countries carries the equity-only per-country
// exposure with the same semantics as the dashboard's countries: only the
// non-zero buckets, sorted by value descending, weights summing to 100.
type PortfolioGeographyAllocation struct {
	Currency  string               `json:"currency"`
	Regions   []*RegionAllocation  `json:"regions"`
	Countries []*CountryAllocation `json:"countries"`
	Covered   decimal.Decimal      `json:"covered_value"`
	Excluded  decimal.Decimal      `json:"excluded_value"`
}

// CountryAllocation is one country bucket within the vault's country
// exposure. Weight is in percentage (sum over the non-zero buckets = 100).
type CountryAllocation struct {
	Country string          `json:"country"` // ISO alpha-2 code
	Value   decimal.Decimal `json:"value"`
	Weight  decimal.Decimal `json:"weight"`
}

// SectorAllocation is one GICS sector bucket within a portfolio's sector
// allocation. Weight is in percentage (sum over all buckets = 100).
type SectorAllocation struct {
	Sector string          `json:"sector"`
	Value  decimal.Decimal `json:"value"`
	Weight decimal.Decimal `json:"weight"`
}

// PortfolioSectorAllocation groups the sector allocation of a portfolio in its
// reference currency.
type PortfolioSectorAllocation struct {
	Currency string              `json:"currency"`
	Sectors  []*SectorAllocation `json:"sectors"`
	Covered  decimal.Decimal     `json:"covered_value"`
	Excluded decimal.Decimal     `json:"excluded_value"`
}

type PortfolioPerformance struct {
	Date  time.Time       `json:"date"`
	Value decimal.Decimal `json:"value"`
}

type AssetROI struct {
	AssetID       string          `json:"asset_id"`
	Ticker        string          `json:"ticker"`
	Name          string          `json:"name"`
	ROI           decimal.Decimal `json:"roi"`
	TotalInvested decimal.Decimal `json:"total_invested"`
	CurrentValue  decimal.Decimal `json:"current_value"`
	Realized      decimal.Decimal `json:"realized"`
	FXMissing     bool            `json:"fx_missing"`
}

// Holding is one position of an asset within a portfolio.
type Holding struct {
	PortfolioID string          `json:"-"`
	AssetID     string          `json:"asset_id"`
	Ticker      string          `json:"ticker"`
	Name        string          `json:"name"`
	Currency    string          `json:"currency"`
	Qty         decimal.Decimal `json:"qty"`
	Cost        decimal.Decimal `json:"cost"`     // cost basis in portfolio currency
	CostCCY     decimal.Decimal `json:"cost_ccy"` // cost basis in asset currency
	Realized    decimal.Decimal `json:"realized"` // realized P&L in portfolio currency
	RealizedCCY decimal.Decimal `json:"realized_ccy"`

	ClosedCost    decimal.Decimal `json:"closed_cost"`     // AVCO cost of sold lots, portfolio currency
	ClosedCostCCY decimal.Decimal `json:"closed_cost_ccy"` // AVCO cost of sold lots, asset currency
	Proceeds      decimal.Decimal `json:"proceeds"`        // net sale proceeds, portfolio currency
	ProceedsCCY   decimal.Decimal `json:"proceeds_ccy"`    // net sale proceeds, asset currency
	Dividends     decimal.Decimal `json:"dividends"`       // dividends, portfolio currency
	DividendsCCY  decimal.Decimal `json:"dividends_ccy"`   // dividends, asset currency

	AvgCost        decimal.Decimal `json:"avg_cost"`
	LastClose      decimal.Decimal `json:"last_close"` // latest close in asset currency
	HasPrice       bool            `json:"has_price"`
	Country        string          `json:"country"`
	IssuerCountry  string          `json:"-"`
	Sector         string          `json:"sector,omitempty"`
	AssetClass     string          `json:"asset_class,omitempty"`
	Type           AssetType       `json:"-"`
	QuoteDivisor   decimal.Decimal `json:"-"` // the asset's, divides qty×price into cash
	PriceFetchedAt *time.Time      `json:"price_fetched_at,omitempty"`
}

// Value is the holding's market value in asset currency: quantity × last
// close divided by the asset's quote divisor (a bond quoted per 100). It is
// the canonical money figure every allocation/summary/ROI path aggregates;
// an unset or 1 divisor keeps it exactly quantity × last close.
func (h *Holding) Value() decimal.Decimal {
	return TradeAmount(h.Qty, h.LastClose, h.QuoteDivisor)
}

type AssetHolding struct {
	AssetID     string          `json:"asset_id"`
	Ticker      string          `json:"ticker"`
	Name        string          `json:"name"`
	Currency    string          `json:"currency"`
	Qty         decimal.Decimal `json:"qty"`
	LastClose   decimal.Decimal `json:"last_close"` // latest close in asset currency
	Cost        decimal.Decimal `json:"cost"`
	CostCCY     decimal.Decimal `json:"cost_ccy"`
	Value       decimal.Decimal `json:"value"`
	ValuePF     decimal.Decimal `json:"value_pf"`
	Realized    decimal.Decimal `json:"realized"`
	RealizedCCY decimal.Decimal `json:"realized_ccy"`
	Unrealized  decimal.Decimal `json:"unrealized"`
	ROI         decimal.Decimal `json:"roi"`
	FXMissing   bool            `json:"fx_missing"`
	Stale       bool            `json:"stale"`
	Closed      bool            `json:"closed"`
}

type CurrencyPerformance struct {
	Currency    string          `json:"currency"`
	Invested    decimal.Decimal `json:"invested"`
	Value       decimal.Decimal `json:"value"`
	GainLoss    decimal.Decimal `json:"gain_loss"`
	GainLossPct decimal.Decimal `json:"gain_loss_pct"`
	Realized    decimal.Decimal `json:"realized"`
}

// ActiveBreakdown is the roll-up of the open (still held) portions of the
// positions: the cost still carried by lots not sold, the market value of the
// remaining quantity and the dividends of the positions that are still open
// (even if partially sold).
type ActiveBreakdown struct {
	Invested    decimal.Decimal `json:"invested"`
	Value       decimal.Decimal `json:"value"`
	GainLoss    decimal.Decimal `json:"gain_loss"`
	GainLossPct decimal.Decimal `json:"gain_loss_pct"`
	Dividends   decimal.Decimal `json:"dividends"`
}

// ClosedBreakdown is the roll-up of the closed (already sold) lot portions.
// Invested is the AVCO cost of the sold lots, Proceeds the net sale proceeds
// plus the dividends of the fully closed positions (a position with no
// remaining quantity folds its distributions here), Realized the difference
// (proceeds - invested) and RealizedPct the realized return in percentage.
type ClosedBreakdown struct {
	Invested    decimal.Decimal `json:"invested"`     // AVCO cost of closed lots
	Proceeds    decimal.Decimal `json:"proceeds"`     // net sale proceeds + dividends of fully closed positions
	Realized    decimal.Decimal `json:"realized"`     // proceeds - invested
	RealizedPct decimal.Decimal `json:"realized_pct"` // realized / invested * 100
}

type PortfolioPerformanceSummary struct {
	PortfolioID   string          `json:"portfolio_id"`
	PortfolioName string          `json:"portfolio_name"`
	Currency      string          `json:"currency"`
	Active        ActiveBreakdown `json:"active"`
	Closed        ClosedBreakdown `json:"closed"`
	AssetCount    int             `json:"asset_count"`
	FXMissing     int             `json:"fx_missing"`
}

type AssetPerformance struct {
	AssetID    string          `json:"asset_id"`
	Ticker     string          `json:"ticker"`
	Name       string          `json:"name"`
	Currency   string          `json:"currency"`
	Qty        decimal.Decimal `json:"qty"`
	Invested   decimal.Decimal `json:"invested"` // in asset currency
	Value      decimal.Decimal `json:"value"`    // in asset currency
	GainLoss   decimal.Decimal `json:"gain_loss"`
	ROI        decimal.Decimal `json:"roi"`
	FXMissing  bool            `json:"fx_missing"`
	ValuePF    decimal.Decimal `json:"value_pf"` // consolidated in portfolio currency
	Realized   decimal.Decimal `json:"realized"` // in asset currency
	RealizedPF decimal.Decimal `json:"realized_pf"`
}

type PortfolioAssets struct {
	PortfolioID   string             `json:"portfolio_id"`
	PortfolioName string             `json:"portfolio_name"`
	Currency      string             `json:"currency"`
	Assets        []AssetPerformance `json:"assets"`
}

type PositionPoint struct {
	Date        time.Time       `json:"date"`
	Qty         decimal.Decimal `json:"qty"`
	CostBasis   decimal.Decimal `json:"cost_basis"`
	MarketValue decimal.Decimal `json:"market_value"`
	Realized    decimal.Decimal `json:"realized"`
}

type SplitInfo struct {
	Date  time.Time `json:"date"`
	Ratio string    `json:"ratio"`
}

type AssetPositionSeries struct {
	AssetID  string          `json:"asset_id"`
	Ticker   string          `json:"ticker"`
	Name     string          `json:"name"`
	Currency string          `json:"currency"`
	Series   []PositionPoint `json:"series"`
	Splits   []SplitInfo     `json:"splits"`
}

type PortfolioPositionHistory struct {
	PortfolioID   string                `json:"portfolio_id"`
	PortfolioName string                `json:"portfolio_name"`
	Currency      string                `json:"currency"`
	Series        []PositionPoint       `json:"series"`
	Assets        []AssetPositionSeries `json:"assets"`
	Splits        []SplitInfo           `json:"splits"`
}

// DashboardSummary is the vault-wide roll-up of every portfolio converted to
// the user's base currency, split into the active (open lots, plus the
// dividends of positions still held) and closed (sold lots, whose proceeds
// fold in the dividends of the fully closed positions) breakdowns. Amounts
// whose FX rate is missing are excluded from the totals and reported through
// FXMissingCount/FXMissingValue.
type DashboardSummary struct {
	Currency       string          `json:"currency"`
	Active         ActiveBreakdown `json:"active"`
	Closed         ClosedBreakdown `json:"closed"`
	FXMissingCount int             `json:"fx_missing_count"`
	FXMissingValue decimal.Decimal `json:"fx_missing_value"`
}

// InvestedAsset is one open asset aggregated across all the user's portfolios,
// expressed in the base currency.
type InvestedAsset struct {
	AssetID     string          `json:"asset_id"`
	Ticker      string          `json:"ticker"`
	Name        string          `json:"name"`
	Currency    string          `json:"currency"`
	Invested    decimal.Decimal `json:"invested"`  // cost of the open quantity
	Value       decimal.Decimal `json:"value"`     // market value (cost when there is no price)
	GainLoss    decimal.Decimal `json:"gain_loss"` // value - invested
	GainLossPct decimal.Decimal `json:"gain_loss_pct"`
	HasPrice    bool            `json:"has_price"`
}

type Dashboard struct {
	BaseCurrency   string                        `json:"base_currency"`
	Summary        *DashboardSummary             `json:"summary,omitempty"`
	ByCurrency     []CurrencyPerformance         `json:"by_currency"`
	Portfolios     []PortfolioPerformanceSummary `json:"portfolios"`
	Assets         []PortfolioAssets             `json:"assets"`
	InvestedAssets []InvestedAsset               `json:"invested_assets"`
}

// PerformanceBucket is one month or year of the dashboard performance chart:
// Period is "YYYY-MM" for monthly and "YYYY" for yearly granularity, Return
// is the bucket's true time-weighted return in percentage, the geometric
// linking of its daily TWR returns, TWR the cumulative time-weighted return
// compounded up to this bucket, Invested the net capital deployed in the
// vault (cumulative external flows excluding dividends) at the bucket's last
// date and Value the market value there (priced assets at market value,
// unpriced ones at cost). Amounts are in the user's base currency.
type PerformanceBucket struct {
	Period   string          `json:"period"`
	Return   decimal.Decimal `json:"return"`
	TWR      decimal.Decimal `json:"twr"`
	Invested decimal.Decimal `json:"invested"`
	Value    decimal.Decimal `json:"value"`
}

// DashboardPerformance is the vault-wide time-weighted return chart across
// all the user's portfolios, converted to their base currency and bucketed by
// month or year.
type DashboardPerformance struct {
	Currency    string              `json:"currency"`
	Granularity string              `json:"granularity"`
	Buckets     []PerformanceBucket `json:"buckets"`
}

// DashboardAllocation groups the class, geographic, country and sector
// allocation of a user's whole vault, aggregated across all portfolios and
// converted to the user's base currency.
type DashboardAllocation struct {
	Currency  string               `json:"currency"`
	Classes   []*ClassAllocation   `json:"classes"`
	Regions   []*RegionAllocation  `json:"regions"`
	Countries []*CountryAllocation `json:"countries"`
	Sectors   []*SectorAllocation  `json:"sectors"`
	Covered   decimal.Decimal      `json:"covered_value"`
	Excluded  decimal.Decimal      `json:"excluded_value"`
}

// AllocationDrillAsset is one asset contributing to an allocation bucket:
// its market value in the drill currency, its exposure weight for the bucket
// (percentage points) and the value it places in the bucket
// (value * weight / 100).
type AllocationDrillAsset struct {
	AssetID      string          `json:"asset_id"`
	Ticker       string          `json:"ticker"`
	Name         string          `json:"name"`
	Value        decimal.Decimal `json:"value"`
	Weight       decimal.Decimal `json:"weight"`
	Contribution decimal.Decimal `json:"contribution"`
}

// AllocationDrill is the drill-down of one allocation bucket: dim is class,
// country, region or sector and key the bucket name; Assets carries the
// contributing assets sorted by contribution descending (only positive
// ones) and Total is their sum, i.e. the bucket value of the corresponding
// aggregation.
type AllocationDrill struct {
	Currency string                  `json:"currency"`
	Dim      string                  `json:"dim"`
	Key      string                  `json:"key"`
	Total    decimal.Decimal         `json:"total"`
	Assets   []*AllocationDrillAsset `json:"assets"`
}
