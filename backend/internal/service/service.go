package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"golang.org/x/crypto/bcrypt"

	"github.com/alv67/peculium/internal/auth"
	"github.com/alv67/peculium/internal/cache"
	"github.com/alv67/peculium/internal/geo"
	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/price"
	"github.com/alv67/peculium/internal/repository"
	"github.com/alv67/peculium/internal/series"
	"github.com/rs/zerolog/log"
	"github.com/shopspring/decimal"
)

var (
	ErrInvalidCredentials = errors.New("invalid email or password")
	ErrEmailExists        = errors.New("email already registered")
	ErrNotFound           = errors.New("not found")
	ErrForbidden          = errors.New("forbidden")
	ErrAssetInUse         = errors.New("asset is used in transactions")
	ErrAssetNotFound      = errors.New("asset not found")
	ErrInvalidWeights     = errors.New("weights must sum to 100")
	ErrWeakPassword       = errors.New("password must be at least 8 characters")
	ErrInvalidInput       = errors.New("invalid input")
	ErrCurrencyExists     = errors.New("currency already in whitelist")
	ErrCurrencyInUse      = errors.New("currency is used by assets or portfolios")
	ErrCurrencyProtected  = errors.New("currency cannot be removed")
	ErrCurrencyNotManaged = errors.New("currency conversion not available")
	ErrInvalidAssetClass  = errors.New("invalid asset class")
	ErrInvalidPriceSource = errors.New("invalid price source")
	ErrNotETF             = errors.New("asset is not an ETF")
	ErrAssetExists        = errors.New("asset with this ticker already exists")
)

// AssetExistsError reports a duplicate ticker during creation and carries
// the already-stored asset so callers can surface its id.
type AssetExistsError struct {
	Existing *model.Asset
}

func (e *AssetExistsError) Error() string {
	return fmt.Sprintf("%s: %s", ErrAssetExists, e.Existing.Ticker)
}
func (e *AssetExistsError) Unwrap() error { return ErrAssetExists }

// assetClasses is the allowed set for asset_class plus a helper to derive a
// default class from the asset type.
var assetClasses = map[string]bool{
	"equity": true, "bond": true, "commodity": true, "currency": true,
	"crypto": true, "real_estate": true, "mixed": true, "other": true,
}

// defaultAssetClassForType maps an asset type to its default investment class.
func defaultAssetClassForType(t model.AssetType) string {
	switch t {
	case model.AssetTypeStock:
		return "equity"
	case model.AssetTypeBond:
		return "bond"
	case model.AssetTypeCommodity:
		return "commodity"
	case model.AssetTypeCrypto:
		return "crypto"
	case model.AssetTypeCash:
		return "currency"
	default:
		return "other"
	}
}

// priceSources is the allowed set for price_source, which controls how an
// asset's price data is obtained (Yahoo fetcher or manual/none).
var priceSources = map[string]bool{
	"yahoo": true, "manual": true, "none": true,
}

// isYahooPriced reports whether an asset's market data is fetched from Yahoo
// Finance. An empty price_source is the legacy default (the column postdates
// the first assets) and means Yahoo.
func isYahooPriced(a *model.Asset) bool {
	return a.PriceSource == "" || a.PriceSource == "yahoo"
}

// filterYahooAssets keeps only the Yahoo-priced assets, the only ones the
// fetcher may ever be called for: manual/none assets have no Yahoo data, so
// every history/split backfill for them fails and floods Health with errors.
func filterYahooAssets(assets []*model.Asset) []*model.Asset {
	yahoo := make([]*model.Asset, 0, len(assets))
	for _, a := range assets {
		if isYahooPriced(a) {
			yahoo = append(yahoo, a)
		}
	}
	return yahoo
}

const (
	cacheTTLStats  = 5 * time.Minute
	cacheTTLPrices = time.Hour
)

// yahooFetcher is the subset of *price.YahooFetcher consumed by the service.
// The interface lets tests stub the Yahoo provider calls.
type yahooFetcher interface {
	FetchAssetProfile(ctx context.Context, ticker string) (sector, industry, country string, err error)
	FetchAssetProfileExtended(ctx context.Context, ticker string) (sector, industry, country string, sectorWeightings []model.ExposureRow, err error)
	FetchMeta(ctx context.Context, ticker string) (*price.AssetMeta, error)
	FetchFXRate(ctx context.Context, quote string) (decimal.Decimal, error)
	RefreshFX(ctx context.Context) ([]price.FetchIssue, error)
	RefreshStale(ctx context.Context, assets []*model.Asset) (price.RefreshReport, error)
	RefreshStaleForPortfolio(ctx context.Context, portfolioID uuid.UUID) (price.RefreshReport, error)
	EnsureHistory(ctx context.Context, assets []price.HistoryAsset) error
	EnsureSplits(ctx context.Context, assets []*model.Asset) error
}

type Service struct {
	repos            *repository.Repository
	jwtAuth          *auth.JWTAuth
	fetcher          yahooFetcher
	etfFetcher       price.ETFFetcher
	lookupCacheTTL   time.Duration
	exposureCacheTTL time.Duration
	cache            *cache.Cache
	seriesMaxPoints  int
	stalePriceDays   int
	Health           *HealthService
}

func New(repos *repository.Repository, jwtAuth *auth.JWTAuth, fetcher yahooFetcher, etfFetcher price.ETFFetcher, lookupCacheTTL time.Duration, exposureCacheTTL time.Duration, c *cache.Cache, seriesMaxPoints int, stalePriceDays int, health *HealthService) *Service {
	if seriesMaxPoints <= 0 {
		seriesMaxPoints = 500
	}
	if stalePriceDays <= 0 {
		stalePriceDays = 7
	}
	return &Service{repos: repos, jwtAuth: jwtAuth, fetcher: fetcher, etfFetcher: etfFetcher, lookupCacheTTL: lookupCacheTTL, exposureCacheTTL: exposureCacheTTL, cache: c, seriesMaxPoints: seriesMaxPoints, stalePriceDays: stalePriceDays, Health: health}
}

// cached implements the read-through cache pattern: it reads the current data
// revision, returns the payload from Redis on a hit and otherwise computes and
// stores it. When bump is set the revision is advanced after compute so writes
// performed during the computation invalidate previously cached entries.
func cached[T any](c *cache.Cache, ctx context.Context, kind string, id string, ttl time.Duration, bump bool, compute func() (T, error)) (T, error) {
	rev, err := c.Rev(ctx)
	if err != nil {
		log.Warn().Err(err).Msg("cache rev read failed")
	}
	key := fmt.Sprintf("pc:%s:%s:%d", kind, id, rev)
	var v T
	if hit, err := c.GetJSON(ctx, key, &v); err != nil {
		log.Warn().Err(err).Msg("cache read failed")
	} else if hit {
		return v, nil
	}
	result, err := compute()
	if err != nil {
		return result, err
	}
	if bump {
		rev, err = c.Bump(ctx)
		if err != nil {
			log.Warn().Err(err).Msg("cache rev bump failed")
		}
		key = fmt.Sprintf("pc:%s:%s:%d", kind, id, rev)
	}
	if err := c.SetJSON(ctx, key, result, ttl); err != nil {
		log.Warn().Err(err).Msg("cache write failed")
	}
	return result, nil
}

// bumpRev advances the global data revision after any write so cached reads
// keyed on the previous revision are invalidated.
func (s *Service) bumpRev(ctx context.Context) {
	if _, err := s.cache.Bump(ctx); err != nil {
		log.Warn().Err(err).Msg("cache rev bump failed")
	}
}

func (s *Service) Register(ctx context.Context, email, name, password string) (*model.User, error) {
	existing, _ := s.repos.User.FindByEmail(ctx, email)
	if existing != nil {
		return nil, ErrEmailExists
	}
	return s.repos.User.Create(ctx, email, name, password)
}

func (s *Service) Login(ctx context.Context, email, password string) (*model.User, string, string, error) {
	user, err := s.repos.User.FindByEmail(ctx, email)
	if err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(password)); err != nil {
		return nil, "", "", ErrInvalidCredentials
	}

	accessToken, err := s.jwtAuth.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return nil, "", "", err
	}

	refreshToken, err := s.jwtAuth.GenerateRefreshToken(user.ID)
	if err != nil {
		return nil, "", "", err
	}

	return user, accessToken, refreshToken, nil
}

func (s *Service) RefreshToken(ctx context.Context, tokenString string) (string, string, error) {
	claims, err := s.jwtAuth.ValidateToken(tokenString)
	if err != nil {
		return "", "", ErrInvalidCredentials
	}
	if claims.TokenType != "refresh" {
		return "", "", ErrInvalidCredentials
	}

	user, err := s.repos.User.FindByID(ctx, claims.UserID)
	if err != nil {
		return "", "", ErrNotFound
	}

	accessToken, err := s.jwtAuth.GenerateAccessToken(user.ID, user.Email, string(user.Role))
	if err != nil {
		return "", "", err
	}

	refreshToken, err := s.jwtAuth.GenerateRefreshToken(user.ID)
	if err != nil {
		return "", "", err
	}

	return accessToken, refreshToken, nil
}

func (s *Service) GetCurrentUser(ctx context.Context, claims *auth.Claims) (*model.User, error) {
	return s.repos.User.FindByID(ctx, claims.UserID)
}

// UpdateProfile patches the user's name and email, and optionally the base
// currency used by the dashboard aggregations. An empty baseCurrency keeps
// the stored value; a non-empty one must be an enabled whitelist currency.
func (s *Service) UpdateProfile(ctx context.Context, userID uuid.UUID, name, email, baseCurrency string) (*model.User, error) {
	if name == "" {
		return nil, ErrInvalidInput
	}
	if email == "" || !strings.Contains(email, "@") {
		return nil, ErrInvalidInput
	}

	code := strings.ToUpper(strings.TrimSpace(baseCurrency))
	if code != "" {
		enabled, err := s.repos.Currency.EnabledByCodes(ctx, []string{code})
		if err != nil {
			return nil, err
		}
		if len(enabled) == 0 {
			return nil, fmt.Errorf("%w: %s is not an enabled currency", ErrInvalidInput, code)
		}
	}

	user, err := s.repos.User.FindByID(ctx, userID)
	if err != nil {
		return nil, ErrNotFound
	}

	if email != user.Email {
		existing, _ := s.repos.User.FindByEmail(ctx, email)
		if existing != nil && existing.ID != userID {
			return nil, ErrEmailExists
		}
		user.Email = email
	}
	user.Name = name
	if code != "" {
		user.BaseCurrency = code
	}

	if err := s.repos.User.Update(ctx, user); err != nil {
		return nil, err
	}
	return s.repos.User.FindByID(ctx, userID)
}

func (s *Service) ChangePassword(ctx context.Context, userID uuid.UUID, currentPassword, newPassword string) error {
	if len(newPassword) < 8 {
		return ErrWeakPassword
	}

	user, err := s.repos.User.FindByID(ctx, userID)
	if err != nil {
		return ErrNotFound
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(currentPassword)); err != nil {
		return ErrInvalidCredentials
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(newPassword), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	return s.repos.User.UpdatePassword(ctx, userID, string(hash))
}

func (s *Service) CreateAsset(ctx context.Context, asset *model.Asset) (*model.Asset, error) {
	if asset.AssetClass == "" {
		asset.AssetClass = defaultAssetClassForType(asset.Type)
	}
	if asset.PriceSource == "" {
		asset.PriceSource = "yahoo"
	}
	if !priceSources[asset.PriceSource] {
		return nil, ErrInvalidPriceSource
	}
	if asset.Country != "" {
		asset.Country = geo.NormalizeCountry(asset.Country)
		if asset.Country == "" {
			return nil, fmt.Errorf("%w: invalid country", ErrInvalidInput)
		}
	}
	existing, err := s.repos.Asset.FindByTicker(ctx, asset.Ticker)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, &AssetExistsError{Existing: existing}
	}

	created, err := s.repos.Asset.Create(ctx, asset)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			existing, lookupErr := s.repos.Asset.FindByTicker(ctx, asset.Ticker)
			if lookupErr != nil {
				return nil, err
			}
			if existing != nil {
				return nil, &AssetExistsError{Existing: existing}
			}
		}
		return nil, err
	}
	s.syncAssetBackground(created.ID)
	return created, nil
}

func (s *Service) GetAsset(ctx context.Context, id uuid.UUID) (*model.Asset, error) {
	return s.repos.Asset.FindByID(ctx, id)
}

// AssetSplits returns the stock split events for a single asset, sorted by date
// ascending. Missing assets yield ErrAssetNotFound; assets without splits yield
// an empty (non-nil) slice.
func (s *Service) AssetSplits(ctx context.Context, id uuid.UUID) ([]model.SplitInfo, error) {
	_, err := s.repos.Asset.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}

	splitRows, err := s.repos.Split.FindByAssets(ctx, []uuid.UUID{id})
	if err != nil {
		return nil, err
	}

	splits := []model.SplitInfo{}
	for _, sp := range splitRows {
		splits = append(splits, model.SplitInfo{
			Date:  series.DayOf(sp.Date),
			Ratio: fmt.Sprintf("%s:%s", sp.Numerator.String(), sp.Denominator.String()),
		})
	}
	sort.Slice(splits, func(i, j int) bool { return splits[i].Date.Before(splits[j].Date) })
	return splits, nil
}

// UpdateAsset merges the editable asset fields from the patch into the stored
// asset and persists the result. Only fields explicitly present in the patch
// are applied; for the required fields an empty value keeps the current one,
// while the optional string fields can be cleared by sending an empty string.
func (s *Service) UpdateAsset(ctx context.Context, id uuid.UUID, patch *model.AssetPatch) (*model.Asset, error) {
	existing, err := s.repos.Asset.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}

	if patch.Ticker != nil && *patch.Ticker != "" {
		existing.Ticker = *patch.Ticker
	}
	if patch.ISIN != nil {
		existing.ISIN = *patch.ISIN
	}
	if patch.Name != nil && *patch.Name != "" {
		existing.Name = *patch.Name
	}
	if patch.Type != nil && *patch.Type != "" {
		existing.Type = *patch.Type
	}
	if patch.AssetClass != nil {
		if *patch.AssetClass != "" && !assetClasses[*patch.AssetClass] {
			return nil, ErrInvalidAssetClass
		}
		existing.AssetClass = *patch.AssetClass
	}
	if patch.PriceSource != nil {
		if *patch.PriceSource != "" && !priceSources[*patch.PriceSource] {
			return nil, ErrInvalidPriceSource
		}
		existing.PriceSource = *patch.PriceSource
	}
	if patch.Country != nil {
		existing.Country = geo.NormalizeCountry(*patch.Country)
		// Non-empty values must resolve to an ISO alpha-2 code; an explicit
		// empty string clears the field (ETF without a country).
		if *patch.Country != "" && existing.Country == "" {
			return nil, fmt.Errorf("%w: invalid country", ErrInvalidInput)
		}
	}
	if patch.Currency != nil && *patch.Currency != "" {
		existing.Currency = *patch.Currency
	}
	if patch.Exchange != nil {
		existing.Exchange = *patch.Exchange
	}
	if patch.Sector != nil {
		existing.Sector = *patch.Sector
	}
	if patch.Industry != nil {
		existing.Industry = *patch.Industry
	}

	updated, err := s.repos.Asset.Update(ctx, existing)
	if err != nil {
		return nil, err
	}
	s.bumpRev(ctx)
	return updated, nil
}

// GetAssetQuote returns the headline metrics for the asset detail page: latest
// close and the percentage changes vs. 1 day, 1 week, 1 month, 1 year and
// year-to-date reference closes. Changes are zero when the reference price is
// missing.
func (s *Service) GetAssetQuote(ctx context.Context, id uuid.UUID) (*model.AssetQuote, error) {
	asset, err := s.repos.Asset.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}

	quote := &model.AssetQuote{Currency: asset.Currency}
	latest, err := s.repos.Price.FindLatest(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return quote, nil
		}
		return nil, err
	}

	quote.HasData = true
	quote.LastClose = latest.Close
	quote.LastDate = latest.Date

	now := time.Now()
	refDates := []time.Time{
		series.DayOf(now.AddDate(0, 0, -1)),
		series.DayOf(now.AddDate(0, 0, -7)),
		series.DayOf(now.AddDate(0, -1, 0)),
		series.DayOf(now.AddDate(-1, 0, 0)),
		series.DayOf(time.Date(now.Year(), 1, 1, 0, 0, 0, 0, time.UTC)),
	}

	refCloses, err := s.repos.Price.ReferenceCloses(ctx, id, refDates)
	if err != nil {
		return nil, err
	}

	changes := []struct {
		ref time.Time
		pct *decimal.Decimal
	}{
		{refDates[0], &quote.Change1D},
		{refDates[1], &quote.Change1W},
		{refDates[2], &quote.Change1M},
		{refDates[3], &quote.Change1Y},
		{refDates[4], &quote.ChangeYTD},
	}
	for _, c := range changes {
		refClose, ok := refCloses[c.ref]
		if !ok || refClose.IsZero() {
			continue
		}
		*c.pct = quote.LastClose.Sub(refClose).Div(refClose).Mul(decimal.NewFromInt(100))
	}
	return quote, nil
}

// FetchAssetProfile resolves sector, industry and issuer domicile country from
// Yahoo and persists them on the asset. The Yahoo error is propagated to the
// caller when the profile cannot be fetched.
func (s *Service) FetchAssetProfile(ctx context.Context, id uuid.UUID) (*model.Asset, error) {
	asset, err := s.repos.Asset.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}

	sector, industry, country, err := s.fetcher.FetchAssetProfile(ctx, asset.Ticker)
	if err != nil {
		return nil, err
	}
	asset.Sector = sector
	asset.Industry = industry
	// Only stock assets expose a single issuer country; keep existing values
	// when Yahoo does not report one.
	if asset.Type == model.AssetTypeStock && country != "" {
		asset.Country = geo.NormalizeCountry(country)
	}

	updated, err := s.repos.Asset.Update(ctx, asset)
	if err != nil {
		return nil, err
	}
	s.bumpRev(ctx)
	return updated, nil
}

// GetAssetExposure returns the country, region and sector weight distribution
// of an asset, together with the persisted `provenance` of each dimension
// (source + last update, only for dimensions saved at least once). The output
// always contains every canonical country, region and GICS sector, in
// canonical order, with zero weight when not stored. When no weights are
// stored for a dimension, stocks fall back to a single 100% entry derived from
// the asset country and sector.
func (s *Service) GetAssetExposure(ctx context.Context, id uuid.UUID) (*model.AssetExposure, error) {
	asset, err := s.repos.Asset.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}

	regions, err := s.repos.Exposure.FindRegions(ctx, id)
	if err != nil {
		return nil, err
	}
	sectors, err := s.repos.Exposure.FindSectors(ctx, id)
	if err != nil {
		return nil, err
	}
	countries, err := s.repos.Exposure.FindCountries(ctx, id)
	if err != nil {
		return nil, err
	}
	ex := s.buildExposure(asset, regions, sectors, countries)
	provenance, err := s.repos.Exposure.FindProvenance(ctx, id)
	if err != nil {
		return nil, err
	}
	ex.Provenance = provenance
	return ex, nil
}

// SaveAssetExposure validates and persists the weight distribution for an
// asset. Each dimension is validated and saved independently: a dimension that
// is absent from the body (nil slice) is left untouched. Sectors must sum to
// ~100 (tolerance 0.5). Regions must not exceed 100 (same tolerance): the UI
// hides the "Other / Not Classified" row, so a sum below 100 is accepted and
// the residual is injected into that bucket before persisting, keeping the
// stored regions summing to 100 for portfolio geography aggregation. Countries
// must not exceed 100 (same tolerance) with no lower bound. Rows with an empty
// name or a non-positive weight are ignored. Country rows must come from the
// canonical geo.Countries list; non-canonical names are skipped. Saving
// countries does NOT touch the regions dimension: regions are recomputed from
// countries only through the explicit derive endpoint
// (POST /assets/{id}/exposure/derive) or updated explicitly via {regions};
// explicit regions (e.g. the official ones returned by Morningstar) are always
// saved as sent.
// Each dimension present in the body may carry its provenance source via
// countries_source / regions_source / sectors_source (e.g. "morningstar",
// "justetf"); when the source is absent or empty the dimension is recorded as
// "manual". Provenance is persisted only for the dimensions actually saved and
// is returned back in the response `provenance` map.
// Returns the complete output built from the stored state.
func (s *Service) SaveAssetExposure(ctx context.Context, id uuid.UUID, exposure *model.AssetExposure) (*model.AssetExposure, error) {
	asset, err := s.repos.Asset.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}

	err = s.repos.WithTx(ctx, func(rx *repository.Repository) error {
		return saveExposureDimensions(ctx, rx, id, exposure)
	})
	if err != nil {
		return nil, err
	}
	s.bumpRev(ctx)

	// Ricostruisci l'output completo dallo stato persistito (la dimensione non
	// toccata mantiene i valori memorizzati, non quelli del body).
	storedRegions, err := s.repos.Exposure.FindRegions(ctx, id)
	if err != nil {
		return nil, err
	}
	storedSectors, err := s.repos.Exposure.FindSectors(ctx, id)
	if err != nil {
		return nil, err
	}
	storedCountries, err := s.repos.Exposure.FindCountries(ctx, id)
	if err != nil {
		return nil, err
	}
	saved := s.buildExposure(asset, storedRegions, storedSectors, storedCountries)
	provenance, err := s.repos.Exposure.FindProvenance(ctx, id)
	if err != nil {
		return nil, err
	}
	saved.Provenance = provenance
	return saved, nil
}

// saveExposureDimensions validates and persists each dimension present in the
// payload through rx. Dimensions are independent: a nil slice means "leave the
// stored dimension untouched". Saving countries never rewrites the regions
// dimension — the country→region aggregation only runs through the explicit
// derive endpoint (DeriveRegions) or when the caller saves explicit regions.
// Each dimension actually written also gets its provenance recorded
// (countries_source/regions_source/sectors_source from the payload, defaulting
// to "manual"); dimensions left untouched keep their stored provenance.
func saveExposureDimensions(ctx context.Context, rx *repository.Repository, id uuid.UUID, exposure *model.AssetExposure) error {
	if exposure.Regions != nil {
		prepared, err := prepareRegions(exposure.Regions)
		if err != nil {
			return err
		}
		if err := rx.Exposure.ReplaceRegions(ctx, id, prepared); err != nil {
			return err
		}
		if err := rx.Exposure.SetProvenance(ctx, id, model.ExposureDimensionRegions, sourceOrDefault(exposure.RegionsSource)); err != nil {
			return err
		}
	}
	if exposure.Sectors != nil {
		sectors := normalizeExposureRows(exposure.Sectors)
		if err := validateExposureWeights(sectors, weightSumExact100); err != nil {
			return err
		}
		if err := rx.Exposure.ReplaceSectors(ctx, id, sectors); err != nil {
			return err
		}
		if err := rx.Exposure.SetProvenance(ctx, id, model.ExposureDimensionSectors, sourceOrDefault(exposure.SectorsSource)); err != nil {
			return err
		}
	}
	if exposure.Countries != nil {
		prepared, err := prepareCountries(exposure.Countries)
		if err != nil {
			return err
		}
		if err := rx.Exposure.ReplaceCountries(ctx, id, prepared); err != nil {
			return err
		}
		if err := rx.Exposure.SetProvenance(ctx, id, model.ExposureDimensionCountries, sourceOrDefault(exposure.CountriesSource)); err != nil {
			return err
		}
	}
	return nil
}

// sourceOrDefault normalizes an optional provenance source coming from the PUT
// payload: a missing or empty source means the edit was manual.
func sourceOrDefault(s string) string {
	if s == "" {
		return "manual"
	}
	return s
}

// DeriveRegions aggregates raw country rows into canonical macro-regions for an
// asset and returns the canonical region list in display order (zero weight for
// regions without exposure). Nothing is persisted: it is a preview of how the
// country dimension maps onto the Morningstar-aligned region taxonomy.
func (s *Service) DeriveRegions(ctx context.Context, id uuid.UUID, countries []model.ExposureRow) ([]model.ExposureRow, error) {
	if _, err := s.repos.Asset.FindByID(ctx, id); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}
	// An empty country list yields the canonical regions zero-filled; a
	// non-empty list is aggregated with the residual absorbed into
	// "Other / Not Classified".
	var derived []model.ExposureRow
	if len(countries) > 0 {
		derived = price.AggregateRegions(countries)
	}
	return canonicalExposureRows(geo.Regions, derived, false, ""), nil
}

// FetchAssetExposure previews the sector exposure of an asset from Yahoo: it
// fetches sector/industry + sector weights and returns the canonical exposure
// built from the provider sectors and the stored regions/countries. Nothing is
// persisted (neither the profile fields nor the sector weights): saving
// happens only through PUT /assets/{id}/exposure.
func (s *Service) FetchAssetExposure(ctx context.Context, id uuid.UUID) (*model.AssetExposure, error) {
	asset, err := s.repos.Asset.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}

	sector, _, _, weightings, err := s.fetcher.FetchAssetProfileExtended(ctx, asset.Ticker)
	if err != nil {
		return nil, err
	}

	sectors := make([]model.ExposureRow, 0, len(weightings))
	for _, w := range weightings {
		if w.Name == "" || !w.Weight.IsPositive() {
			continue
		}
		sectors = append(sectors, w)
	}
	// Single stocks have no topHoldings weights: fall back to the assetProfile
	// sector at 100%.
	if len(sectors) == 0 && sector != "" {
		sectors = []model.ExposureRow{{Name: geo.NormalizeSector(sector), Weight: decimal.NewFromInt(100)}}
	}

	regions, err := s.repos.Exposure.FindRegions(ctx, id)
	if err != nil {
		return nil, err
	}
	countries, err := s.repos.Exposure.FindCountries(ctx, id)
	if err != nil {
		return nil, err
	}

	return s.buildExposure(asset, regions, sectors, countries), nil
}

// FetchETFExposure previews the country and sector exposure of an ETF from the
// python-service: it keeps the raw countries (normalized to ISO codes), derives
// macro-regions and canonical GICS sectors and returns the canonical exposure.
// Nothing from the provider is persisted (saving happens through
// PUT /assets/{id}/exposure); the only write is persisting the ISIN when the
// asset had none and it gets auto-resolved from the ticker.
// The raw JustETF payload is cached in the lookup cache under
// "exposure:justetf:<ISIN>" with TTL exposureCacheTTL: a hit skips the
// provider call entirely, while refresh=true bypasses the read and rewrites the
// cache after a successful fetch. Empty results (no countries) are not cached.
func (s *Service) FetchETFExposure(ctx context.Context, id uuid.UUID, refresh bool) (*model.AssetExposure, error) {
	asset, err := s.repos.Asset.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}
	if asset.Type != model.AssetTypeETF {
		return nil, ErrNotETF
	}
	if strings.TrimSpace(asset.ISIN) == "" {
		isin, err := s.resolveETFISIN(ctx, asset)
		if err != nil {
			return nil, err
		}
		asset.ISIN = isin
		if _, err := s.repos.Asset.Update(ctx, asset); err != nil {
			return nil, err
		}
		s.bumpRev(ctx)
	}

	var raw *model.AssetExposure
	if !refresh {
		raw, _ = s.getCachedExposure(ctx, "justetf", asset.ISIN)
	}
	if raw == nil {
		raw, err = s.etfFetcher.FetchExposure(ctx, asset.ISIN)
		if err != nil {
			return nil, err
		}
		if len(raw.Countries) > 0 {
			s.setCachedExposure(ctx, "justetf", asset.ISIN, raw)
		}
	}

	countries := normalizeCountries(raw.Countries)
	regions := price.AggregateRegions(raw.Countries)
	sectors := price.AggregateSectors(raw.Sectors)
	return s.buildExposure(asset, regions, sectors, countries), nil
}

// resolveETFISIN looks up the ISIN of an ETF from its ticker through the
// python-service (which strips the exchange suffix before querying JustETF) and
// picks the best result via price.BestMatch.
func (s *Service) resolveETFISIN(ctx context.Context, asset *model.Asset) (string, error) {
	ticker := strings.TrimSpace(asset.Ticker)
	if ticker == "" {
		return "", fmt.Errorf("%w: asset has no ticker to resolve an ISIN", ErrInvalidInput)
	}
	results, err := s.etfFetcher.SearchTicker(ctx, ticker)
	if err != nil {
		return "", err
	}
	best, ok := price.BestMatch(results, ticker, asset.Name)
	if !ok {
		return "", fmt.Errorf("%w: no ETF found for ticker %s", ErrNotFound, ticker)
	}
	return best.ISIN, nil
}

// buildExposure assembles the complete exposure output for an asset: every
// canonical country, region and GICS sector in canonical order, overlaid with
// the stored weights and the stock defaults when nothing is stored.
func (s *Service) buildExposure(asset *model.Asset, regions, sectors, countries []model.ExposureRow) *model.AssetExposure {
	return &model.AssetExposure{
		ISIN:      asset.ISIN,
		Countries: canonicalExposureRows(geo.Countries, countries, false, ""),
		Regions:   canonicalExposureRows(geo.Regions, regions, asset.Type == model.AssetTypeStock, geo.RegionForCountry(asset.Country)),
		Sectors:   canonicalExposureRows(geo.GICSSectors, sectors, asset.Type == model.AssetTypeStock, geo.NormalizeSector(asset.Sector)),
	}
}

// canonicalExposureRows returns one dimension of the complete canonical
// exposure for an asset: every canonical name in order with its stored weight
// (zero when absent), or a single 100% default for stocks with no stored rows.
func canonicalExposureRows(names []string, stored []model.ExposureRow, isStock bool, defaultName string) []model.ExposureRow {
	weights := make(map[string]decimal.Decimal, len(stored))
	for _, r := range stored {
		weights[r.Name] = r.Weight
	}
	out := make([]model.ExposureRow, 0, len(names))
	for _, name := range names {
		out = append(out, model.ExposureRow{Name: name, Weight: weights[name]})
	}
	if isStock && len(stored) == 0 && defaultName != "" {
		for i := range out {
			if out[i].Name == defaultName {
				out[i].Weight = decimal.NewFromInt(100)
				break
			}
		}
	}
	return out
}

// normalizeExposureRows drops rows with an empty name or a non-positive weight.
func normalizeExposureRows(rows []model.ExposureRow) []model.ExposureRow {
	out := make([]model.ExposureRow, 0, len(rows))
	for _, row := range rows {
		if row.Name == "" || !row.Weight.IsPositive() {
			continue
		}
		out = append(out, row)
	}
	return out
}

// weightSumRule selects how validateExposureWeights checks a dimension's total
// against 100.
type weightSumRule int

const (
	// weightSumExact100 requires the weights to sum to 100 ± 0.5 (sectors).
	weightSumExact100 weightSumRule = iota
	// weightSumMax100 enforces only the upper bound: sums up to 100.5 are
	// accepted and sums below 100 are valid (regions and countries).
	weightSumMax100
)

// validateExposureWeights checks a dimension's weights sum against 100 following
// the given rule: exactly 100 ± 0.5 (weightSumExact100) or at most 100.5
// (weightSumMax100). Rows are expected to be already normalized.
func validateExposureWeights(rows []model.ExposureRow, rule weightSumRule) error {
	sum := exposureWeightsSum(rows)
	tolerance := decimal.NewFromFloat(0.5)
	delta := sum.Sub(decimal.NewFromInt(100))
	switch rule {
	case weightSumMax100:
		if delta.GreaterThan(tolerance) {
			return ErrInvalidWeights
		}
	default: // weightSumExact100
		if delta.Abs().GreaterThan(tolerance) {
			return ErrInvalidWeights
		}
	}
	return nil
}

// exposureWeightsSum totals the weights of a dimension.
func exposureWeightsSum(rows []model.ExposureRow) decimal.Decimal {
	sum := decimal.Zero
	for _, row := range rows {
		sum = sum.Add(row.Weight)
	}
	return sum
}

// prepareRegions normalizes and validates the region dimension, then enforces
// the stored invariant that regions sum to 100: the UI hides the
// "Other / Not Classified" row and may legitimately send weights summing below
// 100 (only sums above 100.5 are rejected), so when no explicit Other row is
// present and the residual exceeds the 0.5 rounding tolerance it is injected
// into that bucket before persisting, mirroring price.AggregateRegions. An
// explicit Other row is kept as the client sent it.
func prepareRegions(rows []model.ExposureRow) ([]model.ExposureRow, error) {
	prepared := normalizeExposureRows(rows)
	if err := validateExposureWeights(prepared, weightSumMax100); err != nil {
		return nil, err
	}
	for _, row := range prepared {
		if row.Name == geo.OtherRegion {
			return prepared, nil
		}
	}
	residual := decimal.NewFromInt(100).Sub(exposureWeightsSum(prepared))
	if residual.GreaterThan(decimal.NewFromFloat(0.5)) {
		prepared = append(prepared, model.ExposureRow{Name: geo.OtherRegion, Weight: residual})
	}
	return prepared, nil
}

// prepareCountries normalizes the country dimension: only rows whose name
// resolves to a canonical country are kept (anything else is dropped rather
// than failing hard) and the kept weights must not exceed 100 (+ the 0.5
// rounding tolerance). There is no lower bound, so a partial country list is a
// valid save; the regions dimension is left untouched (countries only feed the
// region aggregation through the explicit derive endpoint).
func prepareCountries(rows []model.ExposureRow) ([]model.ExposureRow, error) {
	normalized := normalizeExposureRows(rows)
	kept := normalized[:0]
	for _, row := range normalized {
		code := geo.NormalizeCountry(row.Name)
		if !geo.IsValidCountry(code) || !isCanonicalCountry(code) {
			continue
		}
		row.Name = code
		kept = append(kept, row)
	}
	if err := validateExposureWeights(kept, weightSumMax100); err != nil {
		return nil, err
	}
	return kept, nil
}

// isCanonicalCountry reports whether code is one of the canonical ISO codes.
func isCanonicalCountry(code string) bool {
	for _, c := range geo.Countries {
		if c == code {
			return true
		}
	}
	return false
}

// normalizeCountries normalizes raw country rows to canonical ISO codes,
// dropping empty names, non-positive weights and names that resolve to no
// canonical country.
func normalizeCountries(rows []model.ExposureRow) []model.ExposureRow {
	out := make([]model.ExposureRow, 0, len(rows))
	for _, row := range rows {
		if row.Name == "" || !row.Weight.IsPositive() {
			continue
		}
		code := geo.NormalizeCountry(row.Name)
		if !isCanonicalCountry(code) {
			continue
		}
		out = append(out, model.ExposureRow{Name: code, Weight: row.Weight})
	}
	return out
}

// FetchMorningstarExposure previews the country, region and sector exposure of
// an ETF from the python-service using Morningstar as the data source. When
// Morningstar returns official region rows they are kept as-is (they are
// already canonical); otherwise the regions are derived from the country rows.
// Nothing from the provider is persisted (saving happens through
// PUT /assets/{id}/exposure); the only write is persisting the ISIN when the
// asset had none and it gets auto-resolved from the ticker.
// The raw Morningstar payload is cached in the lookup cache under
// "exposure:morningstar:<ISIN>" with TTL exposureCacheTTL (a separate entry
// from the JustETF one): a hit skips the provider call entirely, while
// refresh=true bypasses the read and rewrites the cache after a successful
// fetch. Empty results (no countries) are not cached.
func (s *Service) FetchMorningstarExposure(ctx context.Context, id uuid.UUID, refresh bool) (*model.AssetExposure, error) {
	asset, err := s.repos.Asset.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrAssetNotFound
		}
		return nil, err
	}
	if asset.Type != model.AssetTypeETF {
		return nil, ErrNotETF
	}
	if strings.TrimSpace(asset.ISIN) == "" {
		isin, err := s.resolveETFISIN(ctx, asset)
		if err != nil {
			return nil, err
		}
		asset.ISIN = isin
		if _, err := s.repos.Asset.Update(ctx, asset); err != nil {
			return nil, err
		}
		s.bumpRev(ctx)
	}

	var raw *model.AssetExposure
	if !refresh {
		raw, _ = s.getCachedExposure(ctx, "morningstar", asset.ISIN)
	}
	if raw == nil {
		raw, err = s.etfFetcher.FetchMorningstarExposure(ctx, asset.ISIN)
		if err != nil {
			return nil, err
		}
		if len(raw.Countries) > 0 {
			s.setCachedExposure(ctx, "morningstar", asset.ISIN, raw)
		}
	}

	mapped := mapMorningstarExposure(raw)
	return s.buildExposure(asset, mapped.Regions, mapped.Sectors, mapped.Countries), nil
}

// mapMorningstarExposure builds the exposure payload to persist from the raw
// Morningstar rows. Official region rows are used as-is when present; otherwise
// the regions are derived from the raw country rows via price.AggregateRegions.
// Countries are normalized to canonical ISO codes and sectors are aggregated to
// canonical GICS names, as for the other exposure sources.
func mapMorningstarExposure(raw *model.AssetExposure) *model.AssetExposure {
	mapped := &model.AssetExposure{}
	if countries := normalizeCountries(raw.Countries); len(countries) > 0 {
		mapped.Countries = countries
	}
	if len(raw.Regions) > 0 {
		mapped.Regions = canonicalExposureRows(geo.Regions, raw.Regions, false, "")
	} else if derived := price.AggregateRegions(raw.Countries); len(derived) > 0 {
		mapped.Regions = canonicalExposureRows(geo.Regions, derived, false, "")
	}
	if sectors := price.AggregateSectors(raw.Sectors); len(sectors) > 0 {
		mapped.Sectors = sectors
	}
	return mapped
}

func (s *Service) SearchAssets(ctx context.Context, query string) ([]*model.Asset, error) {
	return s.repos.Asset.Search(ctx, query)
}

func (s *Service) GetAssetMeta(ctx context.Context, ticker string) (*price.AssetMeta, error) {
	key := "meta:" + strings.ToUpper(strings.TrimSpace(ticker))

	if cached, err := s.repos.Lookup.Get(ctx, key); err == nil {
		var m price.AssetMeta
		if err := json.Unmarshal(cached, &m); err == nil {
			return &m, nil
		}
	}

	meta, err := s.fetcher.FetchMeta(ctx, ticker)
	if err != nil {
		return nil, err
	}

	if data, err := json.Marshal(meta); err == nil {
		if err := s.repos.Lookup.Set(ctx, key, data, s.lookupCacheTTL); err != nil {
			log.Warn().Err(err).Str("ticker", key).Msg("failed to cache asset meta")
		}
	}

	return meta, nil
}

// exposureCacheKey builds the lookup cache key of a raw provider exposure
// payload: "exposure:<source>:<ISIN>" (source is "justetf" or "morningstar").
func exposureCacheKey(source, isin string) string {
	return "exposure:" + source + ":" + strings.ToUpper(strings.TrimSpace(isin))
}

// getCachedExposure reads a previously cached provider exposure. Any failure
// (miss, cache error, malformed payload) degrades to a plain miss so a broken
// cache never blocks a fetch.
func (s *Service) getCachedExposure(ctx context.Context, source, isin string) (*model.AssetExposure, bool) {
	if s.repos == nil || s.repos.Lookup == nil {
		return nil, false
	}
	data, err := s.repos.Lookup.Get(ctx, exposureCacheKey(source, isin))
	if err != nil {
		return nil, false
	}
	var ex model.AssetExposure
	if err := json.Unmarshal(data, &ex); err != nil {
		return nil, false
	}
	return &ex, true
}

// setCachedExposure stores a raw provider exposure payload under
// exposureCacheKey with the configured TTL (PECULIUM_EXPOSURE_CACHE_TTL). Failures
// are logged as warnings and never fail the surrounding request.
func (s *Service) setCachedExposure(ctx context.Context, source, isin string, ex *model.AssetExposure) {
	if s.repos == nil || s.repos.Lookup == nil || ex == nil {
		return
	}
	key := exposureCacheKey(source, isin)
	data, err := json.Marshal(ex)
	if err != nil {
		log.Warn().Err(err).Str("key", key).Msg("failed to encode exposure for cache")
		return
	}
	if err := s.repos.Lookup.Set(ctx, key, data, s.exposureCacheTTL); err != nil {
		log.Warn().Err(err).Str("key", key).Msg("failed to cache provider exposure")
	}
}

func (s *Service) DeleteAsset(ctx context.Context, id uuid.UUID) error {
	used, err := s.repos.Transaction.CountByAsset(ctx, id)
	if err != nil {
		return err
	}
	if used > 0 {
		return ErrAssetInUse
	}
	return s.repos.Asset.Delete(ctx, id)
}

func (s *Service) LookupAsset(ctx context.Context, query string) ([]price.AssetLookup, error) {
	key := strings.ToLower(strings.TrimSpace(query))
	if key == "" {
		return []price.AssetLookup{}, nil
	}

	if cached, err := s.repos.Lookup.Get(ctx, key); err == nil {
		var results []price.AssetLookup
		if err := json.Unmarshal(cached, &results); err == nil {
			return results, nil
		}
	}

	results, err := price.LookupAsset(ctx, query)
	if err != nil {
		// price.LookupAsset never surfaces HTTP statuses as typed errors, so
		// remote search failures are always recorded with code "error".
		s.recordSearchHealth(ctx, "search "+query+": "+err.Error(), "failure", "error")
		return nil, err
	}
	s.recordSearchHealth(ctx, "search "+query+": "+strconv.Itoa(len(results))+" results", "success", "")

	if data, err := json.Marshal(results); err == nil {
		if err := s.repos.Lookup.Set(ctx, key, data, s.lookupCacheTTL); err != nil {
			log.Warn().Err(err).Str("query", key).Msg("failed to cache lookup results")
		}
	}

	return results, nil
}

// recordSearchHealth logs a lookup health event. Health tracking is optional
// (nil in tests) and must never fail the surrounding lookup.
func (s *Service) recordSearchHealth(ctx context.Context, message, status, code string) {
	if s.Health == nil {
		return
	}
	if err := s.Health.RecordEvent(ctx, &model.HealthEvent{
		ID:        uuid.New(),
		EventType: "search",
		Status:    status,
		Code:      code,
		Message:   message,
		CreatedAt: time.Now().UTC(),
	}); err != nil {
		log.Warn().Err(err).Msg("failed to record search health event")
	}
}

// RefreshPrices refreshes stale stored closes for the given portfolio (or all
// assets when portfolioID is nil) plus the USD->X FX rates, hitting Yahoo
// Finance only when needed.
func (s *Service) RefreshPrices(ctx context.Context, portfolioID *uuid.UUID) (price.RefreshReport, error) {
	report := price.RefreshReport{Refreshed: []string{}, Issues: []price.FetchIssue{}}
	var err error
	if portfolioID != nil {
		report, err = s.fetcher.RefreshStaleForPortfolio(ctx, *portfolioID)
	} else {
		var assets []*model.Asset
		assets, err = s.repos.Asset.ListYahoo(ctx)
		if err != nil {
			return report, err
		}
		report, err = s.fetcher.RefreshStale(ctx, assets)
	}
	if err != nil {
		return report, err
	}
	fxIssues, err := s.fetcher.RefreshFX(ctx)
	if err != nil {
		return report, err
	}
	report.Issues = append(report.Issues, fxIssues...)
	report.RateLimited = false
	for _, iss := range report.Issues {
		if iss.Code == "rate_limited" {
			report.RateLimited = true
			break
		}
	}
	if err := series.RecomputeAll(ctx, s.repos); err != nil {
		log.Warn().Err(err).Msg("series recompute all failed")
	}
	s.bumpRev(ctx)
	report.FinishedAt = time.Now().UTC()
	return report, nil
}

func (s *Service) ListAssets(ctx context.Context) ([]*model.Asset, error) {
	return s.repos.Asset.List(ctx)
}

// ListCurrencies returns the enabled whitelisted currencies, ordered.
func (s *Service) ListCurrencies(ctx context.Context) ([]model.Currency, error) {
	return s.repos.Currency.ListEnabled(ctx)
}

// AddCurrency validates that a USD->code conversion is available on Yahoo and
// adds the currency to the enabled whitelist.
func (s *Service) AddCurrency(ctx context.Context, code, name string) (*model.Currency, error) {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return nil, fmt.Errorf("%w: empty currency code", ErrInvalidInput)
	}
	if err := s.ValidateCurrency(ctx, code); err != nil {
		return nil, err
	}

	existing, err := s.repos.Currency.Get(ctx, code)
	if err != nil {
		return nil, err
	}
	if existing != nil {
		return nil, ErrCurrencyExists
	}

	if name == "" {
		name = defaultCurrencyName(code)
	}

	currencies, err := s.repos.Currency.ListAll(ctx)
	if err != nil {
		return nil, err
	}

	c := &model.Currency{
		Code:    code,
		Name:    name,
		Enabled: true,
		Sort:    nextSort(currencies),
	}
	if err := s.repos.Currency.Create(ctx, c); err != nil {
		return nil, err
	}
	return c, nil
}

// DeleteCurrency removes a currency from the whitelist. Removal is blocked for
// the default USD base and for any currency still referenced by assets or
// portfolios.
func (s *Service) DeleteCurrency(ctx context.Context, code string) error {
	code = strings.ToUpper(strings.TrimSpace(code))
	if code == "" {
		return fmt.Errorf("%w: empty currency code", ErrInvalidInput)
	}
	if code == "USD" {
		return ErrCurrencyProtected
	}

	existing, err := s.repos.Currency.Get(ctx, code)
	if err != nil {
		return err
	}
	if existing == nil {
		return ErrNotFound
	}

	inUse, err := s.repos.Currency.CountInUse(ctx, code)
	if err != nil {
		return err
	}
	if inUse > 0 {
		return ErrCurrencyInUse
	}
	return s.repos.Currency.Delete(ctx, code)
}

// ValidateCurrency ensures a USD->code FX conversion is available on Yahoo, so
// the currency can actually be managed.
func (s *Service) ValidateCurrency(ctx context.Context, code string) error {
	if _, err := s.fetcher.FetchFXRate(ctx, code); err != nil {
		return fmt.Errorf("%w: USD->%s conversion unavailable", ErrCurrencyNotManaged, code)
	}
	return nil
}

func defaultCurrencyName(code string) string {
	if code == "USD" {
		return "US Dollar"
	}
	if code == "EUR" {
		return "Euro"
	}
	return code
}

func nextSort(currencies []model.Currency) int {
	max := 0
	for _, c := range currencies {
		if c.Sort > max {
			max = c.Sort
		}
	}
	return max + 1
}

func (s *Service) CreatePortfolio(ctx context.Context, userID uuid.UUID, name, description, currency string) (*model.Portfolio, error) {
	p := &model.Portfolio{
		UserID:      userID,
		Name:        name,
		Description: description,
		Currency:    currency,
	}
	created, err := s.repos.Portfolio.Create(ctx, p)
	if err != nil {
		return nil, err
	}
	s.bumpRev(ctx)
	return created, nil
}

func (s *Service) ListPortfolios(ctx context.Context, userID uuid.UUID) ([]*model.Portfolio, error) {
	return s.repos.Portfolio.FindByUser(ctx, userID)
}

func (s *Service) GetPortfolio(ctx context.Context, id uuid.UUID, userID uuid.UUID) (*model.Portfolio, error) {
	p, err := s.repos.Portfolio.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	if !s.canAccessPortfolio(ctx, p, userID) {
		return nil, ErrForbidden
	}
	return p, nil
}

func (s *Service) UpdatePortfolio(ctx context.Context, p *model.Portfolio) error {
	if err := s.repos.Portfolio.Update(ctx, p); err != nil {
		return err
	}
	if err := series.Recompute(ctx, s.repos, p.ID); err != nil {
		log.Warn().Err(err).Str("portfolio_id", p.ID.String()).Msg("series recompute failed")
	}
	s.bumpRev(ctx)
	return nil
}

func (s *Service) DeletePortfolio(ctx context.Context, id uuid.UUID) error {
	if err := s.repos.Portfolio.Delete(ctx, id); err != nil {
		return err
	}
	s.bumpRev(ctx)
	return nil
}

func (s *Service) AddTransaction(ctx context.Context, tx *model.Transaction) (*model.Transaction, error) {
	tx, err := s.repos.Transaction.Create(ctx, tx)
	if err != nil {
		return nil, err
	}
	if err := series.Recompute(ctx, s.repos, tx.PortfolioID); err != nil {
		log.Warn().Err(err).Str("portfolio_id", tx.PortfolioID.String()).Msg("series recompute failed")
	}
	s.bumpRev(ctx)
	return tx, nil
}

// defaultTransactionLimit and maxTransactionLimit bound the page size of the
// paginated transactions endpoint: unset means the default, anything above
// the max is clamped down.
const (
	defaultTransactionLimit = 20
	maxTransactionLimit     = 100
)

// ListTransactionsPaged returns one page of a portfolio's transactions plus
// the total count of the filtered set, newest first. The filter is optional
// in every dimension (zero value = unfiltered); a type outside the allowed
// set is rejected with ErrInvalidInput even though the HTTP parser already
// gates it. Negative limit/offset are rejected the same way; limit is
// defaulted and clamped as per the constants above. Ownership is enforced
// like on the other portfolio-scoped reads: missing portfolios yield
// ErrNotFound, someone else's portfolio yields ErrForbidden.
func (s *Service) ListTransactionsPaged(ctx context.Context, portfolioID, userID uuid.UUID, limit, offset int, filter model.TransactionFilter) (*model.TransactionPage, error) {
	if limit < 0 || offset < 0 {
		return nil, ErrInvalidInput
	}
	if filter.Type != "" && !model.ValidTransactionType(filter.Type) {
		return nil, ErrInvalidInput
	}
	if limit == 0 {
		limit = defaultTransactionLimit
	}
	if limit > maxTransactionLimit {
		limit = maxTransactionLimit
	}

	p, err := s.repos.Portfolio.FindByID(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !s.canAccessPortfolio(ctx, p, userID) {
		return nil, ErrForbidden
	}

	txs, err := s.repos.Transaction.FindByPortfolioPage(ctx, portfolioID, filter, limit, offset)
	if err != nil {
		return nil, err
	}
	if txs == nil {
		txs = []model.TransactionWithAsset{}
	}
	total, err := s.repos.Transaction.CountByPortfolio(ctx, portfolioID, filter)
	if err != nil {
		return nil, err
	}
	return &model.TransactionPage{
		Transactions: txs,
		Total:        total,
		Limit:        limit,
		Offset:       offset,
	}, nil
}

// UpdateTransaction edits a transaction after verifying the caller owns the
// portfolio the transaction belongs to.
func (s *Service) UpdateTransaction(ctx context.Context, userID uuid.UUID, tx *model.Transaction) error {
	existing, err := s.repos.Transaction.FindByID(ctx, tx.ID)
	if err != nil {
		return ErrNotFound
	}
	p, err := s.repos.Portfolio.FindByID(ctx, existing.PortfolioID)
	if err != nil {
		return err
	}
	if !s.canAccessPortfolio(ctx, p, userID) {
		return ErrForbidden
	}
	tx.PortfolioID = existing.PortfolioID
	if err := s.repos.Transaction.Update(ctx, tx); err != nil {
		return err
	}
	if err := series.Recompute(ctx, s.repos, existing.PortfolioID); err != nil {
		log.Warn().Err(err).Str("portfolio_id", existing.PortfolioID.String()).Msg("series recompute failed")
	}
	s.bumpRev(ctx)
	return nil
}

// DeleteTransaction removes a transaction after verifying the caller owns the
// portfolio it belongs to.
func (s *Service) DeleteTransaction(ctx context.Context, userID uuid.UUID, id uuid.UUID) error {
	existing, err := s.repos.Transaction.FindByID(ctx, id)
	if err != nil {
		return ErrNotFound
	}
	p, err := s.repos.Portfolio.FindByID(ctx, existing.PortfolioID)
	if err != nil {
		return err
	}
	if !s.canAccessPortfolio(ctx, p, userID) {
		return ErrForbidden
	}
	if err := s.repos.Transaction.Delete(ctx, id); err != nil {
		return err
	}
	if err := series.Recompute(ctx, s.repos, existing.PortfolioID); err != nil {
		log.Warn().Err(err).Str("portfolio_id", existing.PortfolioID.String()).Msg("series recompute failed")
	}
	s.bumpRev(ctx)
	return nil
}

func (s *Service) GetPortfolioSummary(ctx context.Context, portfolioID uuid.UUID) (*model.PortfolioSummary, error) {
	return cached(s.cache, ctx, "summary", portfolioID.String(), cacheTTLStats, false, func() (*model.PortfolioSummary, error) {
		p, err := s.repos.Portfolio.FindByID(ctx, portfolioID)
		if err != nil {
			return nil, err
		}
		holdings, err := s.repos.Portfolio.HoldingsDetailed(ctx, []uuid.UUID{portfolioID})
		if err != nil {
			return nil, err
		}
		rates, err := series.LoadRates(ctx, s.repos, holdings, p.Currency)
		if err != nil {
			return nil, err
		}

		summary := &model.PortfolioSummary{
			PortfolioID:   portfolioID.String(),
			PortfolioName: p.Name,
			AssetCount:    len(holdings),
		}
		var totalValue, totalCost, totalRealized decimal.Decimal
		summary.Holdings = make([]model.AssetHolding, 0, len(holdings))
		staleThreshold := time.Duration(s.stalePriceDays) * 24 * time.Hour
		for _, h := range holdings {
			totalRealized = totalRealized.Add(h.Realized)
			if h.Country == "" {
				summary.MissingCountry++
			}
			if h.Sector == "" {
				summary.MissingSector++
			}
			stale := h.PriceFetchedAt == nil || time.Since(*h.PriceFetchedAt) > staleThreshold
			if stale {
				summary.StaleCount++
			}
			ah := model.AssetHolding{
				AssetID:     h.AssetID,
				Ticker:      h.Ticker,
				Name:        h.Name,
				Currency:    h.Currency,
				Qty:         h.Qty,
				LastClose:   h.LastClose,
				Cost:        h.Cost,
				CostCCY:     h.CostCCY,
				Realized:    h.Realized,
				RealizedCCY: h.RealizedCCY,
				Stale:       stale,
				Closed:      !h.Qty.IsPositive(),
			}
			if h.HasPrice && h.Qty.IsPositive() {
				value := h.Qty.Mul(h.LastClose)
				ah.Value = value
				if factor, ok := series.FxFactor(rates, h.Currency, p.Currency); ok {
					ah.ValuePF = value.Mul(factor)
					totalCost = totalCost.Add(h.Cost)
					totalValue = totalValue.Add(value.Mul(factor))
				} else {
					ah.FXMissing = true
					summary.FXMissingCount++
					summary.FXMissingValue = summary.FXMissingValue.Add(value)
				}
				ah.Unrealized = ah.ValuePF.Sub(h.Cost)
				if h.Cost.IsPositive() {
					ah.ROI = ah.Unrealized.Div(h.Cost).Mul(decimal.NewFromInt(100))
				}
			}
			summary.Holdings = append(summary.Holdings, ah)
			// The active/closed breakdown mirrors the per-portfolio rules of
			// GetDashboard, without any conversion to a base currency: cost,
			// dividends, closed cost and proceeds are already expressed in
			// the portfolio currency and only the market value needs the
			// asset->portfolio factor. Only open lots of priced assets feed
			// the active invested, so neither the AVCO rounding residue of a
			// closed position nor the uncomparable cost of an unpriced one
			// inflates it. Dividends follow the position: still-open ones
			// (even partially sold or unpriced) stay in the active group,
			// fully closed ones fold into the proceeds. An unconvertible
			// value is skipped here and already reported by the FX-missing
			// bookkeeping of the flat fields above.
			summary.Closed.Invested = summary.Closed.Invested.Add(h.ClosedCost)
			summary.Closed.Proceeds = summary.Closed.Proceeds.Add(h.Proceeds)
			if h.HasPrice && h.Qty.IsPositive() {
				summary.Active.Invested = summary.Active.Invested.Add(h.Cost)
				value := h.Qty.Mul(h.LastClose)
				if factor, ok := series.FxFactor(rates, h.Currency, p.Currency); ok {
					summary.Active.Value = summary.Active.Value.Add(value.Mul(factor))
				}
			}
			if h.Qty.IsPositive() {
				summary.Active.Dividends = summary.Active.Dividends.Add(h.Dividends)
			} else {
				summary.Closed.Proceeds = summary.Closed.Proceeds.Add(h.Dividends)
			}
		}
		summary.TotalCost = totalCost
		summary.TotalValue = totalValue
		summary.GainLoss = totalValue.Sub(totalCost)
		summary.RealizedGL = totalRealized
		summary.UnrealizedGL = summary.GainLoss
		if totalCost.IsPositive() {
			summary.GainLossPct = summary.GainLoss.Div(totalCost).Mul(decimal.NewFromInt(100))
		}
		finalizeBreakdowns(&summary.Active, &summary.Closed)
		return summary, nil
	})
}

func (s *Service) GetPortfolioAllocation(ctx context.Context, portfolioID uuid.UUID) ([]*model.AssetAllocation, error) {
	return cached(s.cache, ctx, "allocation", portfolioID.String(), cacheTTLStats, false, func() ([]*model.AssetAllocation, error) {
		p, err := s.repos.Portfolio.FindByID(ctx, portfolioID)
		if err != nil {
			return nil, err
		}
		holdings, err := s.repos.Portfolio.HoldingsDetailed(ctx, []uuid.UUID{portfolioID})
		if err != nil {
			return nil, err
		}
		rates, err := series.LoadRates(ctx, s.repos, holdings, p.Currency)
		if err != nil {
			return nil, err
		}

		var allocs []*model.AssetAllocation
		var total decimal.Decimal
		for _, h := range holdings {
			if !h.Qty.IsPositive() {
				continue
			}
			if !h.HasPrice {
				continue
			}
			value := h.Qty.Mul(h.LastClose)
			factor, ok := series.FxFactor(rates, h.Currency, p.Currency)
			if ok {
				value = value.Mul(factor)
				total = total.Add(value)
			}
			allocs = append(allocs, &model.AssetAllocation{
				AssetID:   h.AssetID,
				Ticker:    h.Ticker,
				Name:      h.Name,
				Value:     value,
				FXMissing: !ok,
			})
		}
		for _, a := range allocs {
			if total.IsPositive() && !a.FXMissing {
				a.AllocPct = a.Value.Div(total).Mul(decimal.NewFromInt(100))
			}
		}
		return allocs, nil
	})
}

func (s *Service) GetPortfolioClassAllocation(ctx context.Context, portfolioID uuid.UUID) (*model.PortfolioClassAllocation, error) {
	return cached(s.cache, ctx, "allocation-class", portfolioID.String(), cacheTTLStats, false, func() (*model.PortfolioClassAllocation, error) {
		p, err := s.repos.Portfolio.FindByID(ctx, portfolioID)
		if err != nil {
			return nil, err
		}
		holdings, err := s.repos.Portfolio.HoldingsDetailed(ctx, []uuid.UUID{portfolioID})
		if err != nil {
			return nil, err
		}
		rates, err := series.LoadRates(ctx, s.repos, holdings, p.Currency)
		if err != nil {
			return nil, err
		}

		byClass := map[string]decimal.Decimal{}
		var total decimal.Decimal
		for _, h := range holdings {
			if !h.Qty.IsPositive() || !h.HasPrice {
				continue
			}
			value := h.Qty.Mul(h.LastClose)
			if factor, ok := series.FxFactor(rates, h.Currency, p.Currency); ok {
				value = value.Mul(factor)
			} else {
				continue
			}
			if !value.IsPositive() {
				continue
			}
			class := h.AssetClass
			if class == "" {
				class = "other"
			}
			byClass[class] = byClass[class].Add(value)
			total = total.Add(value)
		}

		classes := make([]*model.ClassAllocation, 0, len(byClass))
		for class, value := range byClass {
			classes = append(classes, &model.ClassAllocation{
				Class: class,
				Value: value,
			})
		}
		sort.Slice(classes, func(i, j int) bool { return classes[i].Value.GreaterThan(classes[j].Value) })
		for _, c := range classes {
			if total.IsPositive() {
				c.Weight = c.Value.Div(total).Mul(decimal.NewFromInt(100))
			}
		}
		return &model.PortfolioClassAllocation{Currency: p.Currency, Classes: classes}, nil
	})
}

func (s *Service) GetPortfolioGeographyAllocation(ctx context.Context, portfolioID uuid.UUID) (*model.PortfolioGeographyAllocation, error) {
	return cached(s.cache, ctx, "allocation-geography", portfolioID.String(), cacheTTLStats, false, func() (*model.PortfolioGeographyAllocation, error) {
		p, err := s.repos.Portfolio.FindByID(ctx, portfolioID)
		if err != nil {
			return nil, err
		}
		holdings, err := s.repos.Portfolio.HoldingsDetailed(ctx, []uuid.UUID{portfolioID})
		if err != nil {
			return nil, err
		}
		rates, err := series.LoadRates(ctx, s.repos, holdings, p.Currency)
		if err != nil {
			return nil, err
		}
		exposures, err := s.repos.Exposure.FindRegionsByAssets(ctx, holdingAssetIDs(holdings))
		if err != nil {
			return nil, err
		}
		countryExposures, err := s.repos.Exposure.FindCountriesByAssets(ctx, holdingAssetIDs(holdings))
		if err != nil {
			return nil, err
		}
		buckets, total, cov := buildBuckets(holdings, rates, p.Currency, geo.Regions, exposures,
			func(h *model.Holding) string { return geo.RegionForCountry(h.Country) })
		regions := make([]*model.RegionAllocation, 0, len(geo.Regions)+1)
		for _, name := range geo.Regions {
			regions = append(regions, &model.RegionAllocation{Region: name, Value: buckets[name]})
		}
		if buckets["Other"].IsPositive() {
			regions = append(regions, &model.RegionAllocation{Region: "Other", Value: buckets["Other"]})
		}
		for _, e := range regions {
			if total.IsPositive() {
				e.Weight = e.Value.Div(total).Mul(decimal.NewFromInt(100))
			}
		}
		cBuckets, cTotal, _ := buildBuckets(holdings, rates, p.Currency, geo.Countries, countryExposures,
			func(h *model.Holding) string { return h.Country })
		countries := make([]*model.CountryAllocation, 0, len(cBuckets))
		for name, value := range cBuckets {
			if !value.IsPositive() {
				continue
			}
			countries = append(countries, &model.CountryAllocation{Country: name, Value: value})
		}
		sort.Slice(countries, func(i, j int) bool { return countries[i].Value.GreaterThan(countries[j].Value) })
		for _, c := range countries {
			if cTotal.IsPositive() {
				c.Weight = c.Value.Div(cTotal).Mul(decimal.NewFromInt(100))
			}
		}
		return &model.PortfolioGeographyAllocation{Currency: p.Currency, Regions: regions, Countries: countries, Covered: cov.covered, Excluded: cov.excluded}, nil
	})
}

func (s *Service) GetPortfolioSectorAllocation(ctx context.Context, portfolioID uuid.UUID) (*model.PortfolioSectorAllocation, error) {
	return cached(s.cache, ctx, "allocation-sector", portfolioID.String(), cacheTTLStats, false, func() (*model.PortfolioSectorAllocation, error) {
		p, err := s.repos.Portfolio.FindByID(ctx, portfolioID)
		if err != nil {
			return nil, err
		}
		holdings, err := s.repos.Portfolio.HoldingsDetailed(ctx, []uuid.UUID{portfolioID})
		if err != nil {
			return nil, err
		}
		rates, err := series.LoadRates(ctx, s.repos, holdings, p.Currency)
		if err != nil {
			return nil, err
		}
		exposures, err := s.repos.Exposure.FindSectorsByAssets(ctx, holdingAssetIDs(holdings))
		if err != nil {
			return nil, err
		}
		buckets, total, cov := buildBuckets(holdings, rates, p.Currency, geo.GICSSectors, exposures,
			func(h *model.Holding) string { return geo.NormalizeSector(h.Sector) })
		sectors := make([]*model.SectorAllocation, 0, len(geo.GICSSectors)+1)
		for _, name := range geo.GICSSectors {
			sectors = append(sectors, &model.SectorAllocation{Sector: name, Value: buckets[name]})
		}
		if buckets["Other"].IsPositive() {
			sectors = append(sectors, &model.SectorAllocation{Sector: "Other", Value: buckets["Other"]})
		}
		for _, e := range sectors {
			if total.IsPositive() {
				e.Weight = e.Value.Div(total).Mul(decimal.NewFromInt(100))
			}
		}
		return &model.PortfolioSectorAllocation{Currency: p.Currency, Sectors: sectors, Covered: cov.covered, Excluded: cov.excluded}, nil
	})
}

// holdingAssetIDs returns the parsed asset ids of the holdings, skipping any
// that fail to parse.
func holdingAssetIDs(holdings []*model.Holding) []uuid.UUID {
	ids := make([]uuid.UUID, 0, len(holdings))
	for _, h := range holdings {
		if id, err := uuid.Parse(h.AssetID); err == nil {
			ids = append(ids, id)
		}
	}
	return ids
}

// exposureEligible reports whether a holding participates in the equity
// universe used by the geography and sector allocations. STRICT policy:
// stocks always, ETFs and mutual funds only when their asset class is
// equity or real estate; everything else (bonds, crypto, commodities,
// currencies, unclassified funds) is excluded by nature.
func exposureEligible(h *model.Holding) bool {
	switch h.Type {
	case model.AssetTypeStock:
		return true
	case model.AssetTypeETF, model.AssetTypeMutualFund:
		return h.AssetClass == "equity" || h.AssetClass == "real_estate"
	default:
		return false
	}
}

// exposureCoverage tracks the value of the eligible (equity) universe and of
// the excluded (non-equity) holdings for the geography/sector allocations.
type exposureCoverage struct {
	covered  decimal.Decimal // value of the eligible (equity) holdings
	excluded decimal.Decimal // value of the excluded (non-equity) holdings
}

// buildBuckets aggregates the weighted value of the holdings across one
// canonical exposure dimension. Every canonical name in order is keyed in the
// returned map (zero when nothing maps), and any value that could not be
// assigned lands in "Other". Only holdings eligible for the equity universe
// participate; their value is returned as coverage metadata.
func buildBuckets(holdings []*model.Holding, rates map[string]decimal.Decimal, currency string, names []string, exposures map[string][]model.ExposureRow, defaultName func(*model.Holding) string) (map[string]decimal.Decimal, decimal.Decimal, exposureCoverage) {
	byBucket := make(map[string]decimal.Decimal, len(names)+1)
	for _, name := range names {
		byBucket[name] = decimal.Zero
	}
	var total decimal.Decimal
	var cov exposureCoverage
	for _, h := range holdings {
		if !h.Qty.IsPositive() || !h.HasPrice {
			continue
		}
		value := h.Qty.Mul(h.LastClose)
		factor, ok := series.FxFactor(rates, h.Currency, currency)
		if !ok {
			continue
		}
		value = value.Mul(factor)
		if !value.IsPositive() {
			continue
		}
		if !exposureEligible(h) {
			cov.excluded = cov.excluded.Add(value)
			continue
		}
		cov.covered = cov.covered.Add(value)
		total = total.Add(value)

		isStock := h.Type == model.AssetTypeStock
		rows := canonicalExposureRows(names, exposures[h.AssetID], isStock, defaultName(h))
		var sum decimal.Decimal
		for _, r := range rows {
			sum = sum.Add(r.Weight)
		}
		if !sum.IsPositive() {
			byBucket["Other"] = byBucket["Other"].Add(value)
			continue
		}
		for _, r := range rows {
			byBucket[r.Name] = byBucket[r.Name].Add(value.Mul(r.Weight).Div(decimal.NewFromInt(100)))
		}
	}
	return byBucket, total, cov
}

// allocationDrillDim describes one drillable allocation dimension: the
// canonical bucket names, the per-asset stock default and the stored-rows
// loader, mirroring the dimension wiring of the geography/sector
// aggregations. A nil loader marks the class dimension, which has no
// exposure rows: the asset's whole value sits in its own class.
type allocationDrillDim struct {
	names       []string
	defaultName func(*model.Holding) string
	rows        func(ctx context.Context, repos *repository.Repository, holdings []*model.Holding) (map[string][]model.ExposureRow, error)
}

var allocationDrillDims = map[string]allocationDrillDim{
	"class": {},
	"region": {
		names:       geo.Regions,
		defaultName: func(h *model.Holding) string { return geo.RegionForCountry(h.Country) },
		rows: func(ctx context.Context, repos *repository.Repository, holdings []*model.Holding) (map[string][]model.ExposureRow, error) {
			return repos.Exposure.FindRegionsByAssets(ctx, holdingAssetIDs(holdings))
		},
	},
	"country": {
		names:       geo.Countries,
		defaultName: func(h *model.Holding) string { return h.Country },
		rows: func(ctx context.Context, repos *repository.Repository, holdings []*model.Holding) (map[string][]model.ExposureRow, error) {
			return repos.Exposure.FindCountriesByAssets(ctx, holdingAssetIDs(holdings))
		},
	},
	"sector": {
		names:       geo.GICSSectors,
		defaultName: func(h *model.Holding) string { return geo.NormalizeSector(h.Sector) },
		rows: func(ctx context.Context, repos *repository.Repository, holdings []*model.Holding) (map[string][]model.ExposureRow, error) {
			return repos.Exposure.FindSectorsByAssets(ctx, holdingAssetIDs(holdings))
		},
	},
}

// drillBucketWeight returns the holding's weight (in percentage points) in
// one allocation bucket with the same semantics as buildBuckets: only
// equity-universe eligible holdings participate on the exposure dimensions
// (the class dimension has no eligibility filter), stocks without stored
// rows default to 100% of their canonical default bucket, and when the
// canonical rows carry no weight at all the whole value falls into "Other".
// fullValue mirrors the other buildBuckets branch: the buckets that receive
// the asset's whole value (the class itself and the "Other" fallback) are
// fed with the raw value, without the value*weight/100 rounding.
func drillBucketWeight(h *model.Holding, d allocationDrillDim, key string, stored []model.ExposureRow) (weight decimal.Decimal, fullValue bool) {
	if d.rows == nil {
		class := h.AssetClass
		if class == "" {
			class = "other"
		}
		if class != key {
			return decimal.Zero, false
		}
		return decimal.NewFromInt(100), true
	}
	if !exposureEligible(h) {
		return decimal.Zero, false
	}
	rows := canonicalExposureRows(d.names, stored, h.Type == model.AssetTypeStock, d.defaultName(h))
	var sum decimal.Decimal
	for _, r := range rows {
		sum = sum.Add(r.Weight)
	}
	if !sum.IsPositive() {
		if key == "Other" {
			return decimal.NewFromInt(100), true
		}
		return decimal.Zero, false
	}
	for _, r := range rows {
		if r.Name == key {
			return r.Weight, false
		}
	}
	return decimal.Zero, false
}

// buildAllocationDrill values the holdings in the reference currency with
// the same rules the allocation aggregations use (priced, positive quantity,
// available FX factor, positive value) and keeps, per asset, the ones
// contributing to the bucket. The same asset held in several portfolios
// merges into a single entry whose value is the sum; its weight is the
// asset's exposure weight, identical across portfolios. The bucket total is
// the sum of the contributions, so it matches the corresponding bucket of
// the aggregation output.
func (s *Service) buildAllocationDrill(ctx context.Context, d allocationDrillDim, dim, key, currency string, holdings []*model.Holding) (*model.AllocationDrill, error) {
	rates, err := series.LoadRates(ctx, s.repos, holdings, currency)
	if err != nil {
		return nil, err
	}
	var exposures map[string][]model.ExposureRow
	if d.rows != nil {
		if exposures, err = d.rows(ctx, s.repos, holdings); err != nil {
			return nil, err
		}
	}

	type drillAgg struct {
		holding      *model.Holding
		value        decimal.Decimal
		weight       decimal.Decimal
		contribution decimal.Decimal
	}
	byAsset := map[string]*drillAgg{}
	for _, h := range holdings {
		if !h.Qty.IsPositive() || !h.HasPrice {
			continue
		}
		value := h.Qty.Mul(h.LastClose)
		factor, ok := series.FxFactor(rates, h.Currency, currency)
		if !ok {
			continue
		}
		value = value.Mul(factor)
		if !value.IsPositive() {
			continue
		}
		weight, fullValue := drillBucketWeight(h, d, key, exposures[h.AssetID])
		if !weight.IsPositive() {
			continue
		}
		contribution := value
		if !fullValue {
			contribution = value.Mul(weight).Div(decimal.NewFromInt(100))
		}
		agg, ok := byAsset[h.AssetID]
		if !ok {
			agg = &drillAgg{holding: h, weight: weight}
			byAsset[h.AssetID] = agg
		}
		agg.value = agg.value.Add(value)
		agg.contribution = agg.contribution.Add(contribution)
	}

	assets := make([]*model.AllocationDrillAsset, 0, len(byAsset))
	var total decimal.Decimal
	for assetID, agg := range byAsset {
		if !agg.contribution.IsPositive() {
			continue
		}
		total = total.Add(agg.contribution)
		assets = append(assets, &model.AllocationDrillAsset{
			AssetID:      assetID,
			Ticker:       agg.holding.Ticker,
			Name:         agg.holding.Name,
			Value:        agg.value,
			Weight:       agg.weight,
			Contribution: agg.contribution,
		})
	}
	sort.Slice(assets, func(i, j int) bool {
		if !assets[i].Contribution.Equal(assets[j].Contribution) {
			return assets[i].Contribution.GreaterThan(assets[j].Contribution)
		}
		return assets[i].AssetID < assets[j].AssetID
	})
	return &model.AllocationDrill{Currency: currency, Dim: dim, Key: key, Total: total, Assets: assets}, nil
}

// GetPortfolioAllocationDrill returns the per-asset contributions behind one
// bucket of the portfolio's allocation (dim: class, country, region or
// sector; key: the bucket name, e.g. an "Other" fallback bucket too) valued
// in the portfolio currency. A missing portfolio yields ErrNotFound;
// ownership is enforced by the caller like on the other portfolio-scoped
// allocation reads.
func (s *Service) GetPortfolioAllocationDrill(ctx context.Context, portfolioID uuid.UUID, dim, key string) (*model.AllocationDrill, error) {
	d, ok := allocationDrillDims[dim]
	if !ok || key == "" {
		return nil, ErrInvalidInput
	}
	return cached(s.cache, ctx, "allocation-drill", portfolioID.String()+":"+dim+":"+key, cacheTTLStats, false, func() (*model.AllocationDrill, error) {
		p, err := s.repos.Portfolio.FindByID(ctx, portfolioID)
		if err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return nil, ErrNotFound
			}
			return nil, err
		}
		holdings, err := s.repos.Portfolio.HoldingsDetailed(ctx, []uuid.UUID{portfolioID})
		if err != nil {
			return nil, err
		}
		return s.buildAllocationDrill(ctx, d, dim, key, p.Currency, holdings)
	})
}

// GetDashboardAllocationDrill returns the per-asset contributions behind one
// bucket of the vault-wide allocation in the user's base currency (default
// EUR when unset), aggregating the holdings of all the user's portfolios by
// asset with the same semantics as GetDashboardAllocation.
func (s *Service) GetDashboardAllocationDrill(ctx context.Context, userID uuid.UUID, dim, key string) (*model.AllocationDrill, error) {
	d, ok := allocationDrillDims[dim]
	if !ok || key == "" {
		return nil, ErrInvalidInput
	}
	return cached(s.cache, ctx, "allocation-drill", userID.String()+":"+dim+":"+key, cacheTTLStats, false, func() (*model.AllocationDrill, error) {
		user, err := s.repos.User.FindByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		currency := user.BaseCurrency
		if currency == "" {
			currency = "EUR"
		}
		portfolios, err := s.repos.Portfolio.FindByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		ids := make([]uuid.UUID, 0, len(portfolios))
		for _, p := range portfolios {
			ids = append(ids, p.ID)
		}
		holdings, err := s.repos.Portfolio.HoldingsDetailed(ctx, ids)
		if err != nil {
			return nil, err
		}
		return s.buildAllocationDrill(ctx, d, dim, key, currency, holdings)
	})
}

func (s *Service) GetPortfolioPerformance(ctx context.Context, portfolioID uuid.UUID) ([]*model.PortfolioPerformance, error) {
	return cached(s.cache, ctx, "performance", portfolioID.String(), cacheTTLStats, false, func() ([]*model.PortfolioPerformance, error) {
		p, err := s.repos.Portfolio.FindByID(ctx, portfolioID)
		if err != nil {
			return nil, err
		}
		agg, err := s.repos.Series.FindPortfolioAgg(ctx, p.ID)
		if err != nil {
			return nil, err
		}
		perf := make([]*model.PortfolioPerformance, 0, len(agg))
		for _, pt := range agg {
			perf = append(perf, &model.PortfolioPerformance{Date: pt.Date, Value: pt.MarketValue})
		}
		if len(perf) > s.seriesMaxPoints {
			vals := make([]model.PortfolioPerformance, 0, len(perf))
			for _, p := range perf {
				vals = append(vals, *p)
			}
			vals = series.PortfolioPerformance(vals, s.seriesMaxPoints)
			perf = make([]*model.PortfolioPerformance, 0, len(vals))
			for i := range vals {
				perf = append(perf, &vals[i])
			}
		}
		return perf, nil
	})
}

func (s *Service) GetPortfolioROI(ctx context.Context, portfolioID uuid.UUID) ([]*model.AssetROI, error) {
	return cached(s.cache, ctx, "roi", portfolioID.String(), cacheTTLStats, false, func() ([]*model.AssetROI, error) {
		p, err := s.repos.Portfolio.FindByID(ctx, portfolioID)
		if err != nil {
			return nil, err
		}
		holdings, err := s.repos.Portfolio.HoldingsDetailed(ctx, []uuid.UUID{portfolioID})
		if err != nil {
			return nil, err
		}
		rates, err := series.LoadRates(ctx, s.repos, holdings, p.Currency)
		if err != nil {
			return nil, err
		}

		var rois []*model.AssetROI
		for _, h := range holdings {
			roi := &model.AssetROI{
				AssetID:       h.AssetID,
				Ticker:        h.Ticker,
				Name:          h.Name,
				TotalInvested: h.Cost,
				Realized:      h.Realized,
			}
			if h.HasPrice {
				value := h.Qty.Mul(h.LastClose)
				if factor, ok := series.FxFactor(rates, h.Currency, p.Currency); ok {
					roi.CurrentValue = value.Mul(factor)
				} else {
					roi.FXMissing = true
				}
			}
			if roi.TotalInvested.IsPositive() {
				roi.ROI = roi.CurrentValue.Sub(roi.TotalInvested).Div(roi.TotalInvested).Mul(decimal.NewFromInt(100))
			}
			rois = append(rois, roi)
		}
		return rois, nil
	})
}

// GetPortfolioHistory returns the running AVCO cost basis, market value and
// realized P&L per date for a portfolio and for each asset in it.
func (s *Service) GetPortfolioHistory(ctx context.Context, portfolioID uuid.UUID) (*model.PortfolioPositionHistory, error) {
	return cached(s.cache, ctx, "history", portfolioID.String(), cacheTTLStats, true, func() (*model.PortfolioPositionHistory, error) {
		p, err := s.repos.Portfolio.FindByID(ctx, portfolioID)
		if err != nil {
			return nil, err
		}
		txs, err := s.repos.Transaction.FindByPortfoliosAsc(ctx, []uuid.UUID{portfolioID})
		if err != nil {
			return nil, err
		}

		txByAsset := map[uuid.UUID][]model.TransactionWithAsset{}
		for _, tx := range txs {
			txByAsset[tx.AssetID] = append(txByAsset[tx.AssetID], tx)
		}
		// Only Yahoo-priced assets may be sent to the fetcher: manual/none
		// assets would fail the history/split backfill and flood the Health
		// page with errors. The rest of the computation still covers every
		// asset from its stored data.
		txAssetIDs := make([]uuid.UUID, 0, len(txByAsset))
		for aid := range txByAsset {
			txAssetIDs = append(txAssetIDs, aid)
		}
		txAssets, err := s.repos.Asset.FindByIDs(ctx, txAssetIDs)
		if err != nil {
			return nil, err
		}
		yahooByID := make(map[uuid.UUID]*model.Asset, len(txAssets))
		for _, a := range filterYahooAssets(txAssets) {
			yahooByID[a.ID] = a
		}
		historyAssets := make([]price.HistoryAsset, 0, len(yahooByID))
		assetPtrs := make([]*model.Asset, 0, len(yahooByID))
		for aid, assetTxs := range txByAsset {
			a, ok := yahooByID[aid]
			if !ok {
				continue
			}
			historyAssets = append(historyAssets, price.HistoryAsset{
				ID:     aid,
				Ticker: a.Ticker,
				From:   series.DayOf(assetTxs[0].Date),
			})
			assetPtrs = append(assetPtrs, a)
		}
		if err := s.fetcher.EnsureHistory(ctx, historyAssets); err != nil {
			log.Warn().Err(err).Msg("history ensure failed")
		}
		if err := s.fetcher.EnsureSplits(ctx, assetPtrs); err != nil {
			log.Warn().Err(err).Msg("splits ensure failed")
		}
		if err := series.Recompute(ctx, s.repos, portfolioID); err != nil {
			log.Warn().Err(err).Str("portfolio_id", portfolioID.String()).Msg("series recompute failed")
		}

		agg, err := s.repos.Series.FindPortfolioAgg(ctx, portfolioID)
		if err != nil {
			return nil, err
		}
		assets, err := s.repos.Series.FindPortfolio(ctx, portfolioID)
		if err != nil {
			return nil, err
		}
		agg = series.PositionPoints(agg, s.seriesMaxPoints)
		for i := range assets {
			assets[i].Series = series.PositionPoints(assets[i].Series, s.seriesMaxPoints)
		}
		firstTxByAsset := map[uuid.UUID]time.Time{}
		for aid, assetTxs := range txByAsset {
			firstTxByAsset[aid] = series.DayOf(assetTxs[0].Date)
		}
		assetIDs := make([]uuid.UUID, 0, len(assets))
		for _, a := range assets {
			if id, err := uuid.Parse(a.AssetID); err == nil {
				assetIDs = append(assetIDs, id)
			}
		}
		splitRows, err := s.repos.Split.FindByAssets(ctx, assetIDs)
		if err != nil {
			return nil, err
		}
		splitSeen := map[time.Time]bool{}
		aggSplits := []model.SplitInfo{}
		for _, sp := range splitRows {
			d := series.DayOf(sp.Date)
			if firstTx, ok := firstTxByAsset[sp.AssetID]; ok && !d.After(firstTx) {
				continue
			}
			if !splitSeen[d] {
				splitSeen[d] = true
				aggSplits = append(aggSplits, model.SplitInfo{
					Date:  d,
					Ratio: fmt.Sprintf("%s:%s", sp.Numerator.String(), sp.Denominator.String()),
				})
			}
		}
		sort.Slice(aggSplits, func(i, j int) bool { return aggSplits[i].Date.Before(aggSplits[j].Date) })
		return &model.PortfolioPositionHistory{
			PortfolioID:   portfolioID.String(),
			PortfolioName: p.Name,
			Currency:      p.Currency,
			Series:        agg,
			Assets:        assets,
			Splits:        aggSplits,
		}, nil
	})
}

// ExportPortfolio builds the JSON document for a portfolio: portfolio
// metadata, the assets referenced by its transactions and the full transaction
// history. Assets are referenced by ticker so the file is human-readable and
// editable. Historical prices and split events are asset-level market data and
// are not exported (they are re-synced from the data provider).
func (s *Service) ExportPortfolio(ctx context.Context, portfolioID uuid.UUID, userID uuid.UUID) (*model.PortfolioExport, error) {
	p, err := s.repos.Portfolio.FindByID(ctx, portfolioID)
	if err != nil {
		return nil, err
	}
	if !s.canAccessPortfolio(ctx, p, userID) {
		return nil, ErrForbidden
	}
	txs, err := s.repos.Transaction.FindByPortfoliosAsc(ctx, []uuid.UUID{portfolioID})
	if err != nil {
		return nil, err
	}

	assetIDs := make([]uuid.UUID, 0, len(txs))
	seen := map[uuid.UUID]bool{}
	for _, tx := range txs {
		if !seen[tx.AssetID] {
			seen[tx.AssetID] = true
			assetIDs = append(assetIDs, tx.AssetID)
		}
	}
	assets, err := s.repos.Asset.FindByIDs(ctx, assetIDs)
	if err != nil {
		return nil, err
	}
	assetByID := map[uuid.UUID]*model.Asset{}
	for _, a := range assets {
		assetByID[a.ID] = a
	}

	doc := &model.PortfolioExport{
		Version:    1,
		ExportedAt: time.Now().UTC(),
		Portfolio: model.ExportPortfolio{
			Name:        p.Name,
			Description: p.Description,
			Currency:    p.Currency,
		},
		Assets:       make([]model.ExportAsset, 0, len(assets)),
		Transactions: make([]model.ExportTransaction, 0, len(txs)),
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Ticker < assets[j].Ticker })
	for _, a := range assets {
		doc.Assets = append(doc.Assets, model.ExportAsset{
			Ticker:      a.Ticker,
			Name:        a.Name,
			Type:        a.Type,
			Currency:    a.Currency,
			ISIN:        a.ISIN,
			PriceSource: a.PriceSource,
			AssetClass:  a.AssetClass,
		})
	}
	for _, tx := range txs {
		a := assetByID[tx.AssetID]
		if a == nil {
			continue
		}
		doc.Transactions = append(doc.Transactions, model.ExportTransaction{
			Date:        tx.Date,
			Type:        tx.Type,
			AssetTicker: a.Ticker,
			Quantity:    tx.Quantity,
			Price:       tx.Price,
			Fees:        tx.Fees,
			Notes:       tx.Notes,
		})
	}
	return doc, nil
}

// ImportPortfolio restores a portfolio from an exported document. Assets are
// matched by ticker and created if missing. In "new" mode a fresh portfolio is
// created with the given name (falling back to the document's). In "overwrite"
// mode the target portfolio is deleted and recreated from the document. Both
// paths run atomically.
func (s *Service) ImportPortfolio(ctx context.Context, userID uuid.UUID, doc *model.PortfolioExport, mode, name string, targetID *uuid.UUID) (*model.Portfolio, error) {
	if doc == nil {
		return nil, ErrInvalidInput
	}
	if doc.Version > 1 {
		return nil, fmt.Errorf("%w: unsupported export version %d", ErrInvalidInput, doc.Version)
	}
	if doc.Version != 1 {
		return nil, ErrInvalidInput
	}
	if strings.TrimSpace(doc.Portfolio.Name) == "" {
		return nil, ErrInvalidInput
	}
	if mode == "" {
		mode = "new"
	}
	if mode != "new" && mode != "overwrite" {
		return nil, ErrInvalidInput
	}
	if mode == "overwrite" && targetID == nil {
		return nil, ErrInvalidInput
	}

	var created *model.Portfolio
	err := s.repos.WithTx(ctx, func(rx *repository.Repository) error {
		if mode == "overwrite" {
			target, err := rx.Portfolio.FindByID(ctx, *targetID)
			if err != nil {
				return err
			}
			if !s.canAccessPortfolio(ctx, target, userID) {
				return ErrForbidden
			}
			if err := rx.Portfolio.Delete(ctx, *targetID); err != nil {
				return err
			}
		}

		assetByTicker := map[string]*model.Asset{}
		// The defaults below keep documents exported by older app versions
		// importable: those files predate fields like price_source/asset_class
		// and may omit any optional value, so every missing piece is filled
		// with a constraint-satisfying default instead of failing the insert.
		// An unknown or absent price_source falls back to "yahoo" rather than
		// erroring, the same default Service.CreateAsset applies.
		createAsset := func(ticker string) (*model.Asset, error) {
			if a, ok := assetByTicker[ticker]; ok {
				return a, nil
			}
			a, err := rx.Asset.FindByTicker(ctx, ticker)
			if err != nil {
				return nil, err
			}
			if a == nil {
				a = &model.Asset{Ticker: ticker, Name: ticker, Type: model.AssetTypeStock, Currency: "USD", PriceSource: "yahoo"}
				for i := range doc.Assets {
					ea := doc.Assets[i]
					if !strings.EqualFold(ea.Ticker, ticker) {
						continue
					}
					if ea.Name != "" {
						a.Name = ea.Name
					}
					a.ISIN = ea.ISIN
					if ea.Type != "" {
						a.Type = ea.Type
					}
					if ea.Currency != "" {
						a.Currency = ea.Currency
					}
					if ea.AssetClass != "" {
						a.AssetClass = ea.AssetClass
					}
					if priceSources[ea.PriceSource] {
						a.PriceSource = ea.PriceSource
					}
					break
				}
				if a.AssetClass == "" {
					a.AssetClass = defaultAssetClassForType(a.Type)
				}
				a, err = rx.Asset.Create(ctx, a)
				if err != nil {
					return nil, err
				}
			}
			assetByTicker[ticker] = a
			return a, nil
		}
		for _, ea := range doc.Assets {
			if _, err := createAsset(strings.TrimSpace(ea.Ticker)); err != nil {
				return err
			}
		}

		pname := strings.TrimSpace(name)
		if pname == "" {
			pname = doc.Portfolio.Name
		}
		p, err := rx.Portfolio.Create(ctx, &model.Portfolio{
			UserID:      userID,
			Name:        pname,
			Description: doc.Portfolio.Description,
			Currency:    doc.Portfolio.Currency,
		})
		if err != nil {
			return err
		}
		created = p

		for _, et := range doc.Transactions {
			if strings.TrimSpace(et.AssetTicker) == "" {
				return fmt.Errorf("%w: transaction without asset_ticker", ErrInvalidInput)
			}
			if !model.ValidTransactionType(string(et.Type)) {
				return fmt.Errorf("%w: invalid transaction type %q", ErrInvalidInput, et.Type)
			}
			a, err := createAsset(strings.TrimSpace(et.AssetTicker))
			if err != nil {
				return err
			}
			if _, err := rx.Transaction.Create(ctx, &model.Transaction{
				PortfolioID: p.ID,
				AssetID:     a.ID,
				Type:        et.Type,
				Quantity:    et.Quantity,
				Price:       et.Price,
				Fees:        et.Fees,
				Date:        et.Date,
				Notes:       et.Notes,
			}); err != nil {
				return err
			}
		}
		return nil
	})
	if err != nil {
		return nil, err
	}
	if err := series.Recompute(ctx, s.repos, created.ID); err != nil {
		log.Warn().Err(err).Str("portfolio_id", created.ID.String()).Msg("series recompute failed")
	}
	s.bumpRev(ctx)
	return created, nil
}

// SyncAssetData refreshes asset-level market data for every Yahoo-priced
// asset independently of any portfolio: split events are re-checked and the
// price history is brought up to date. It is meant to run once per app load.
func (s *Service) SyncAssetData(ctx context.Context) error {
	assets, err := s.repos.Asset.ListYahoo(ctx)
	if err != nil {
		return err
	}
	ids := make([]uuid.UUID, 0, len(assets))
	for _, a := range assets {
		ids = append(ids, a.ID)
	}
	if err := s.syncAssets(ctx, ids); err != nil {
		return err
	}
	// Newly fetched prices change every series; recompute so charts and the
	// dashboard reflect the fresh data immediately instead of waiting for the
	// next worker tick.
	if err := series.RecomputeAll(ctx, s.repos); err != nil {
		return err
	}
	// Invalidate cached reads so the fresh prices/series are visible right away.
	s.bumpRev(ctx)
	return nil
}

func (s *Service) syncAssets(ctx context.Context, ids []uuid.UUID) error {
	if len(ids) == 0 {
		return nil
	}
	assets, err := s.repos.Asset.FindByIDs(ctx, ids)
	if err != nil {
		return err
	}
	// Defensive filter: only Yahoo-priced assets may reach the fetcher, so
	// every caller (not just SyncAssetData) is safe from Health-flooding
	// failures on manual/none assets.
	assets = filterYahooAssets(assets)
	if len(assets) == 0 {
		return nil
	}
	firstDates, err := s.repos.Transaction.MinDateByAsset(ctx, ids)
	if err != nil {
		return err
	}
	historyAssets := make([]price.HistoryAsset, 0, len(assets))
	for _, a := range assets {
		from := series.DayOf(time.Now())
		if fd, ok := firstDates[a.ID]; ok {
			from = series.DayOf(fd)
		}
		historyAssets = append(historyAssets, price.HistoryAsset{ID: a.ID, Ticker: a.Ticker, From: from, Full: !a.HistoryBackfilled})
	}
	if err := s.fetcher.EnsureSplits(ctx, assets); err != nil {
		return err
	}
	return s.fetcher.EnsureHistory(ctx, historyAssets)
}

// BackfillAssetHistory forces a full price-history backfill for a single
// asset, regardless of stored data. Used by the asset detail page. Assets
// priced outside Yahoo (manual/none) are a silent no-op: they have no Yahoo
// data, so a backfill would only produce a Health error.
func (s *Service) BackfillAssetHistory(ctx context.Context, id uuid.UUID) error {
	asset, err := s.repos.Asset.FindByID(ctx, id)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrAssetNotFound
		}
		return err
	}
	if !isYahooPriced(asset) {
		return nil
	}
	if err := s.fetcher.EnsureHistory(ctx, []price.HistoryAsset{{ID: asset.ID, Ticker: asset.Ticker, Full: true}}); err != nil {
		return err
	}
	s.bumpRev(ctx)
	return nil
}

// BackfillAssetMeta reconciles issuer country and GICS sector for every stock
// asset. For stocks the Yahoo assetProfile.country is the source of truth for
// the exposure country (the issuer's domicile), so it is applied whenever Yahoo
// reports a recognizable value — filling empty fields and correcting legacy
// values derived from the old exchange-based logic. A manual override stays
// possible via PATCH, but a subsequent backfill restores the issuer domicile.
// Sector is normalized to its canonical GICS form, so legacy non-canonical
// values are corrected too. Failures are reported per asset without aborting.
func (s *Service) BackfillAssetMeta(ctx context.Context) (*model.MetaBackfillReport, error) {
	assets, err := s.repos.Asset.AllStocks(ctx)
	if err != nil {
		return nil, err
	}

	report := &model.MetaBackfillReport{
		Processed: len(assets),
		Errors:    []string{},
	}
	for _, a := range assets {
		sector, industry, country, err := s.fetcher.FetchAssetProfile(ctx, a.Ticker)
		if err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", a.Ticker, err))
			continue
		}

		changed := false

		// Country: Yahoo is the source of truth when it recognizes the issuer
		// domicile. Only when Yahoo reports no country do we fall back to the
		// exchange-derived value for stocks that have none stored.
		if c := geo.NormalizeCountry(country); c != "" {
			if a.Country != c {
				a.Country = c
				report.UpdatedCountry++
				changed = true
			}
		} else if a.Country == "" {
			if c := price.ExchangeCountry(a.Exchange); c != "" {
				a.Country = c
				report.UpdatedCountry++
				changed = true
			}
		}

		if sec := geo.NormalizeSector(sector); sec != "" && a.Sector != sec {
			a.Sector = sec
			report.UpdatedSector++
			changed = true
		}
		if industry != "" && a.Industry != industry {
			a.Industry = industry
			changed = true
		}

		if !changed {
			continue
		}
		if _, err := s.repos.Asset.Update(ctx, a); err != nil {
			report.Failed++
			report.Errors = append(report.Errors, fmt.Sprintf("%s: %v", a.Ticker, err))
		}
	}
	s.bumpRev(ctx)
	return report, nil
}

func (s *Service) syncAssetBackground(assetID uuid.UUID) {
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
		defer cancel()
		if err := s.syncAssets(ctx, []uuid.UUID{assetID}); err != nil {
			log.Warn().Err(err).Str("asset_id", assetID.String()).Msg("asset background sync failed")
			return
		}
		s.bumpRev(ctx)
	}()
}

// roundAmount strips the rounding residues accumulated by AVCO division
// (shopspring/decimal works at 16 decimal places) from the dashboard amounts.
func roundAmount(d decimal.Decimal) decimal.Decimal {
	return d.Round(8)
}

// finalizeBreakdowns rounds the aggregated active/closed amounts and
// recomputes the derived fields from the rounded inputs, so residues never
// surface in the dashboard figures or in their percentages.
func finalizeBreakdowns(active *model.ActiveBreakdown, closed *model.ClosedBreakdown) {
	active.Invested = roundAmount(active.Invested)
	active.Value = roundAmount(active.Value)
	active.Dividends = roundAmount(active.Dividends)
	active.GainLoss = roundAmount(active.Value.Sub(active.Invested))
	if active.Invested.IsPositive() {
		active.GainLossPct = roundAmount(active.GainLoss.Div(active.Invested).Mul(decimal.NewFromInt(100)))
	}
	closed.Invested = roundAmount(closed.Invested)
	closed.Proceeds = roundAmount(closed.Proceeds)
	closed.Realized = roundAmount(closed.Proceeds.Sub(closed.Invested))
	if closed.Invested.IsPositive() {
		closed.RealizedPct = roundAmount(closed.Realized.Div(closed.Invested).Mul(decimal.NewFromInt(100)))
	}
}

// GetDashboard returns the consolidated dashboard for a user: performance
// grouped by currency, per-portfolio summaries and assets grouped per
// portfolio.
func (s *Service) GetDashboard(ctx context.Context, userID uuid.UUID) (*model.Dashboard, error) {
	return cached(s.cache, ctx, "dash", userID.String(), cacheTTLStats, false, func() (*model.Dashboard, error) {
		user, err := s.repos.User.FindByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		baseCurrency := user.BaseCurrency
		if baseCurrency == "" {
			baseCurrency = "EUR"
		}
		portfolios, err := s.repos.Portfolio.FindByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		dash := &model.Dashboard{
			BaseCurrency:   baseCurrency,
			ByCurrency:     []model.CurrencyPerformance{},
			Portfolios:     []model.PortfolioPerformanceSummary{},
			Assets:         []model.PortfolioAssets{},
			InvestedAssets: []model.InvestedAsset{},
		}
		if len(portfolios) == 0 {
			return dash, nil
		}

		ids := make([]uuid.UUID, 0, len(portfolios))
		for _, p := range portfolios {
			ids = append(ids, p.ID)
		}
		holdings, err := s.repos.Portfolio.HoldingsDetailed(ctx, ids)
		if err != nil {
			return nil, err
		}
		pfCurrencies := make([]string, 0, len(portfolios))
		for _, p := range portfolios {
			if p.Currency != "" {
				pfCurrencies = append(pfCurrencies, p.Currency)
			}
		}
		rates, err := series.LoadRates(ctx, s.repos, holdings, baseCurrency, pfCurrencies...)
		if err != nil {
			return nil, err
		}

		byID := make(map[uuid.UUID]*model.Portfolio, len(portfolios))
		for _, p := range portfolios {
			byID[p.ID] = p
		}

		byCurrency := map[string]*model.CurrencyPerformance{}
		investedByAsset := map[string]*model.InvestedAsset{}
		summary := &model.DashboardSummary{Currency: baseCurrency}
		for _, h := range holdings {
			p := byID[mustUUID(h.PortfolioID)]
			if p == nil {
				continue
			}
			cp := byCurrency[h.Currency]
			if cp == nil {
				cp = &model.CurrencyPerformance{Currency: h.Currency}
				byCurrency[h.Currency] = cp
			}
			cp.Invested = cp.Invested.Add(h.CostCCY)
			cp.Realized = cp.Realized.Add(h.RealizedCCY)
			if h.HasPrice {
				cp.Value = cp.Value.Add(h.Qty.Mul(h.LastClose))
			}

			// Summary totals are converted per amount, mirroring the
			// FX-missing semantics of GetPortfolioSummary: an amount whose
			// rate is missing is skipped from the totals, counted in
			// FXMissingCount and added raw to FXMissingValue (currencies of
			// skipped amounts may differ, so the flag count is what matters;
			// zero amounts are not counted).
			pfFactor, pfOK := series.FxFactor(rates, p.Currency, baseCurrency)
			addPF := func(dst *decimal.Decimal, amt decimal.Decimal) {
				if pfOK {
					*dst = dst.Add(amt.Mul(pfFactor))
				} else if !amt.IsZero() {
					summary.FXMissingCount++
					summary.FXMissingValue = summary.FXMissingValue.Add(amt)
				}
			}
			addPF(&summary.Closed.Invested, h.ClosedCost)
			addPF(&summary.Closed.Proceeds, h.Proceeds)
			// Only open lots of priced assets feed the active invested: a fully
			// closed position can still carry a negligible cost basis left by
			// AVCO division rounding, and an unpriced asset has no market
			// value to compare its cost with (counting it would fake a -100%
			// loss). Dividends follow the position: still-open ones (even
			// partially sold or unpriced) stay in the active group, fully
			// closed ones fold into the proceeds.
			if h.HasPrice && h.Qty.IsPositive() {
				addPF(&summary.Active.Invested, h.Cost)
			}
			if h.Qty.IsPositive() {
				addPF(&summary.Active.Dividends, h.Dividends)
			} else {
				addPF(&summary.Closed.Proceeds, h.Dividends)
			}
			if h.HasPrice && h.Qty.IsPositive() {
				value := h.Qty.Mul(h.LastClose)
				if factor, ok := series.FxFactor(rates, h.Currency, baseCurrency); ok {
					summary.Active.Value = summary.Active.Value.Add(value.Mul(factor))
				} else {
					summary.FXMissingCount++
					summary.FXMissingValue = summary.FXMissingValue.Add(value)
				}
			}
			// The consolidated invested-assets list aggregates the open
			// positions by asset across the portfolios, in the base
			// currency: a holding whose portfolio factor is missing is
			// skipped like the unconvertible summary amounts, a priced
			// asset whose own factor is missing is carried at cost (no fake
			// loss) and so is an unpriced one, flagged by has_price.
			if h.Qty.IsPositive() && pfOK {
				ia := investedByAsset[h.AssetID]
				if ia == nil {
					ia = &model.InvestedAsset{
						AssetID:  h.AssetID,
						Ticker:   h.Ticker,
						Name:     h.Name,
						Currency: h.Currency,
					}
					investedByAsset[h.AssetID] = ia
				}
				invested := h.Cost.Mul(pfFactor)
				value := invested
				if h.HasPrice {
					if factor, ok := series.FxFactor(rates, h.Currency, baseCurrency); ok {
						value = h.Qty.Mul(h.LastClose).Mul(factor)
					}
					ia.HasPrice = true
				}
				ia.Invested = ia.Invested.Add(invested)
				ia.Value = ia.Value.Add(value)
			}
		}
		finalizeBreakdowns(&summary.Active, &summary.Closed)
		dash.Summary = summary
		for _, cp := range byCurrency {
			cp.GainLoss = cp.Value.Sub(cp.Invested)
			if cp.Invested.IsPositive() {
				cp.GainLossPct = cp.GainLoss.Div(cp.Invested).Mul(decimal.NewFromInt(100))
			}
			dash.ByCurrency = append(dash.ByCurrency, *cp)
		}

		investedAssets := make([]*model.InvestedAsset, 0, len(investedByAsset))
		for _, ia := range investedByAsset {
			investedAssets = append(investedAssets, ia)
		}
		sort.Slice(investedAssets, func(i, j int) bool {
			if investedAssets[i].Value.Equal(investedAssets[j].Value) {
				return investedAssets[i].Ticker < investedAssets[j].Ticker
			}
			return investedAssets[i].Value.GreaterThan(investedAssets[j].Value)
		})
		dash.InvestedAssets = make([]model.InvestedAsset, 0, len(investedAssets))
		for _, ia := range investedAssets {
			ia.Invested = roundAmount(ia.Invested)
			ia.Value = roundAmount(ia.Value)
			ia.GainLoss = roundAmount(ia.Value.Sub(ia.Invested))
			if ia.Invested.IsPositive() {
				ia.GainLossPct = roundAmount(ia.GainLoss.Div(ia.Invested).Mul(decimal.NewFromInt(100)))
			}
			dash.InvestedAssets = append(dash.InvestedAssets, *ia)
		}

		assetsByPF := map[uuid.UUID][]model.AssetPerformance{}
		holdingsByPF := map[uuid.UUID][]*model.Holding{}
		for _, h := range holdings {
			pfID := mustUUID(h.PortfolioID)
			p := byID[pfID]
			holdingsByPF[pfID] = append(holdingsByPF[pfID], h)
			ap := model.AssetPerformance{
				AssetID:    h.AssetID,
				Ticker:     h.Ticker,
				Name:       h.Name,
				Currency:   h.Currency,
				Qty:        h.Qty,
				Invested:   h.CostCCY,
				Realized:   h.RealizedCCY,
				RealizedPF: h.Realized,
			}
			if h.HasPrice && h.Qty.IsPositive() {
				value := h.Qty.Mul(h.LastClose)
				ap.Value = value
				ap.GainLoss = value.Sub(h.CostCCY)
				if h.CostCCY.IsPositive() {
					ap.ROI = ap.GainLoss.Div(h.CostCCY).Mul(decimal.NewFromInt(100))
				}
				if p != nil {
					if factor, ok := series.FxFactor(rates, h.Currency, p.Currency); ok {
						ap.ValuePF = value.Mul(factor)
					} else {
						ap.FXMissing = true
					}
				}
			}
			assetsByPF[pfID] = append(assetsByPF[pfID], ap)
		}

		for _, p := range portfolios {
			ps := &model.PortfolioPerformanceSummary{
				PortfolioID:   p.ID.String(),
				PortfolioName: p.Name,
				Currency:      p.Currency,
				AssetCount:    len(assetsByPF[p.ID]),
			}
			for _, h := range holdingsByPF[p.ID] {
				ps.Closed.Invested = ps.Closed.Invested.Add(h.ClosedCost)
				ps.Closed.Proceeds = ps.Closed.Proceeds.Add(h.Proceeds)
				// Same guards as the vault summary: only open lots of priced
				// assets feed the active invested, so neither the AVCO
				// rounding residue of a closed position nor the uncomparable
				// cost of an unpriced one inflates it. Dividends follow the
				// position: still-open ones stay in the active group, fully
				// closed ones fold into the proceeds.
				if h.HasPrice && h.Qty.IsPositive() {
					ps.Active.Invested = ps.Active.Invested.Add(h.Cost)
				}
				if h.Qty.IsPositive() {
					ps.Active.Dividends = ps.Active.Dividends.Add(h.Dividends)
				} else {
					ps.Closed.Proceeds = ps.Closed.Proceeds.Add(h.Dividends)
				}
				if h.HasPrice && h.Qty.IsPositive() {
					value := h.Qty.Mul(h.LastClose)
					if factor, ok := series.FxFactor(rates, h.Currency, p.Currency); ok {
						ps.Active.Value = ps.Active.Value.Add(value.Mul(factor))
					} else {
						ps.FXMissing++
					}
				}
			}
			finalizeBreakdowns(&ps.Active, &ps.Closed)
			dash.Portfolios = append(dash.Portfolios, *ps)
			dash.Assets = append(dash.Assets, model.PortfolioAssets{
				PortfolioID:   p.ID.String(),
				PortfolioName: p.Name,
				Currency:      p.Currency,
				Assets:        assetsByPF[p.ID],
			})
		}

		return dash, nil
	})
}

// GetDashboardPerformance returns the vault-wide true time-weighted return
// (TWR) chart of the user in their base currency, bucketed by month or year.
// Returns are measured daily and linked geometrically: each day carries
// r(d) = (V(d) - V(d-1) - flow(d)) / V(d-1) when V(d-1) is positive and is
// skipped otherwise (the first day and the gaps of a fully liquidated vault
// measure nothing), where V(d) is the market value only: mv_priced(d), the
// sum of the priced assets' market values, plus bond_at_cost(d), the cost
// basis of the unpriced assets still held (already zero once fully sold),
// all converted to the base currency. flow(d) is the day's external cash
// flows at end of day: buy +(qty*price + fees), sell -(qty*price - fees),
// standalone fee +feeAmount (qty*price, or the price alone when no quantity
// is set), dividend -divAmount (income taken out) and split 0; a flow whose
// FX rate is missing on its transaction date is skipped. Realized P&L never
// enters V: it is already captured by the sell flow. The bucket return is
// the geometric linking of its days' factors, Π(1 + r(d)) - 1, in percentage,
// and twr the cumulative product Π(1 + r) - 1 across all buckets so far.
// invested is the net capital: the cumulative flow excluding dividends (the
// income part) up to the bucket's last date, and value V there. The daily
// points come from the materialized per-asset series, converted per date
// through USD-pivoted FX rates: while an asset's factor is missing its
// converted value is forward-filled (zero before the first successful
// conversion), and every asset keeps carrying its last converted value on
// the dates where it has no new point. Dates with no points produce no
// bucket.
func (s *Service) GetDashboardPerformance(ctx context.Context, userID uuid.UUID, granularity string) (*model.DashboardPerformance, error) {
	if granularity != "month" && granularity != "year" {
		return nil, fmt.Errorf("%w: granularity must be month or year", ErrInvalidInput)
	}
	return cached(s.cache, ctx, "dash-perf", userID.String()+":"+granularity, cacheTTLStats, false, func() (*model.DashboardPerformance, error) {
		user, err := s.repos.User.FindByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		baseCurrency := user.BaseCurrency
		if baseCurrency == "" {
			baseCurrency = "EUR"
		}
		perf := &model.DashboardPerformance{
			Currency:    baseCurrency,
			Granularity: granularity,
			Buckets:     []model.PerformanceBucket{},
		}
		portfolios, err := s.repos.Portfolio.FindByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if len(portfolios) == 0 {
			return perf, nil
		}

		ids := make([]uuid.UUID, 0, len(portfolios))
		for _, p := range portfolios {
			ids = append(ids, p.ID)
		}
		holdings, err := s.repos.Portfolio.HoldingsDetailed(ctx, ids)
		if err != nil {
			return nil, err
		}
		// HasPrice is asset-level: it tells whether the asset has any price
		// row. Priced assets contribute their market value to V, unpriced
		// ones their cost basis (a bond carried at cost, never a fake total
		// loss). The holding currency is the asset currency the cash flows
		// are converted from.
		pricedAsset := map[uuid.UUID]bool{}
		assetCurrency := map[uuid.UUID]string{}
		for _, h := range holdings {
			assetID := mustUUID(h.AssetID)
			pricedAsset[assetID] = h.HasPrice
			if h.Currency != "" {
				assetCurrency[assetID] = h.Currency
			}
		}

		quoteSet := map[string]bool{baseCurrency: true}
		for _, p := range portfolios {
			if p.Currency != "" {
				quoteSet[p.Currency] = true
			}
		}
		for _, cur := range assetCurrency {
			quoteSet[cur] = true
		}
		quotes := make([]string, 0, len(quoteSet))
		for c := range quoteSet {
			quotes = append(quotes, c)
		}
		dr, err := series.LoadDateRates(ctx, s.repos, "USD", quotes)
		if err != nil {
			return nil, err
		}

		// Daily market value in base currency: each asset walks its series as
		// a forward-fill stream, contributing its last converted point — the
		// market value when priced, the cost basis when unpriced — to every
		// date after its first point. A missing conversion keeps the last
		// converted value (zero before the first successful one).
		type assetWalk struct {
			pts    []model.PositionPoint
			pos    int
			priced bool
			from   string
			last   decimal.Decimal
		}
		var walks []*assetWalk
		dateSet := map[time.Time]bool{}
		for _, p := range portfolios {
			assetSeries, err := s.repos.Series.FindPortfolio(ctx, p.ID)
			if err != nil {
				return nil, err
			}
			for _, a := range assetSeries {
				walks = append(walks, &assetWalk{
					pts:    a.Series,
					priced: pricedAsset[mustUUID(a.AssetID)],
					from:   p.Currency,
				})
				for _, pt := range a.Series {
					dateSet[series.DayOf(pt.Date)] = true
				}
			}
		}

		// External cash flows per day in base currency. A transaction whose
		// asset currency or FX rate is unknown cannot be converted and is
		// skipped entirely. capital is the same flow without the dividend
		// part: distributions are income, not deployed capital, so they move
		// the return but never the invested.
		type dayCash struct{ flow, capital decimal.Decimal }
		flows := map[time.Time]*dayCash{}
		txs, err := s.repos.Transaction.FindByPortfoliosAsc(ctx, ids)
		if err != nil {
			return nil, err
		}
		for _, tx := range txs {
			var flow decimal.Decimal
			switch tx.Type {
			case model.TxBuy:
				flow = tx.Quantity.Mul(tx.Price).Add(tx.Fees)
			case model.TxSell:
				flow = tx.Quantity.Mul(tx.Price).Sub(tx.Fees).Neg()
			case model.TxFee:
				flow = tx.Price
				if tx.Quantity.IsPositive() {
					flow = tx.Quantity.Mul(tx.Price)
				}
			case model.TxDividend:
				flow = tx.Price
				if tx.Quantity.IsPositive() {
					flow = tx.Quantity.Mul(tx.Price)
				}
				flow = flow.Neg()
			default:
				continue
			}
			factor, ok := dr.Factor(assetCurrency[tx.AssetID], baseCurrency, tx.Date)
			if !ok {
				continue
			}
			flow = flow.Mul(factor)
			capital := flow
			if tx.Type == model.TxDividend {
				capital = decimal.Zero
			}
			d := series.DayOf(tx.Date)
			dateSet[d] = true
			day, seen := flows[d]
			if !seen {
				day = &dayCash{}
				flows[d] = day
			}
			day.flow = day.flow.Add(flow)
			day.capital = day.capital.Add(capital)
		}

		// The walk runs over the union of the market-observed dates and the
		// flow dates. Dates are ascending and bucket periods are monotonic
		// over them, so every time the period changes the previous bucket is
		// sealed on its last date.
		dates := make([]time.Time, 0, len(dateSet))
		for d := range dateSet {
			dates = append(dates, d)
		}
		sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

		one := decimal.NewFromInt(1)
		hundred := decimal.NewFromInt(100)
		buckets := make([]model.PerformanceBucket, 0, len(dates))
		bucketFactor, twrFactor := one, one
		var period string
		var invested, endValue, prevV decimal.Decimal
		flush := func() {
			if period == "" {
				return
			}
			buckets = append(buckets, model.PerformanceBucket{
				Period:   period,
				Return:   bucketFactor.Sub(one).Mul(hundred).Round(4),
				TWR:      twrFactor.Sub(one).Mul(hundred).Round(4),
				Invested: roundAmount(invested),
				Value:    roundAmount(endValue),
			})
		}
		for _, d := range dates {
			v := decimal.Zero
			for _, w := range walks {
				for w.pos < len(w.pts) && !series.DayOf(w.pts[w.pos].Date).After(d) {
					pt := w.pts[w.pos]
					if factor, ok := dr.Factor(w.from, baseCurrency, pt.Date); ok {
						if w.priced {
							w.last = pt.MarketValue.Mul(factor)
						} else {
							w.last = pt.CostBasis.Mul(factor)
						}
					}
					w.pos++
				}
				v = v.Add(w.last)
			}
			var flow, capital decimal.Decimal
			if c, ok := flows[d]; ok {
				flow, capital = c.flow, c.capital
			}
			if p := performancePeriod(d, granularity); p != period {
				flush()
				period = p
				bucketFactor = one
			}
			invested = invested.Add(capital)
			// A day whose previous value is not positive (the vault's first
			// days and the gaps of a fully liquidated one) measures no
			// return: it is skipped, not turned into a fake ±100%.
			if prevV.IsPositive() {
				factor := v.Sub(prevV).Sub(flow).Div(prevV).Add(one)
				bucketFactor = bucketFactor.Mul(factor)
				twrFactor = twrFactor.Mul(factor)
			}
			endValue = v
			prevV = v
		}
		flush()
		perf.Buckets = buckets
		return perf, nil
	})
}

func performancePeriod(d time.Time, granularity string) string {
	if granularity == "year" {
		return d.Format("2006")
	}
	return d.Format("2006-01")
}

// GetPortfolioPerformanceBuckets returns the true time-weighted return (TWR)
// chart of a single portfolio in the portfolio's own currency, bucketed by
// month or year, with the same daily model as GetDashboardPerformance:
// r(d) = (V(d) - V(d-1) - flow(d)) / V(d-1) on the market value V (priced
// assets at market value, unpriced ones carried at cost), external flows at
// the transaction-date FX and geometric linking into buckets. It is simpler
// than the vault-wide chart because nothing is converted to a base currency:
// the materialized per-asset series is already denominated in the portfolio
// currency, so only the cash flows — recorded in the asset currency — are
// converted to it, and a transaction whose FX rate or asset currency is
// unknown is skipped. Ownership is enforced before the cache lookup so a
// cached bucket set is never served to a caller who cannot access the
// portfolio.
func (s *Service) GetPortfolioPerformanceBuckets(ctx context.Context, portfolioID, userID uuid.UUID, granularity string) (*model.DashboardPerformance, error) {
	if granularity != "month" && granularity != "year" {
		return nil, fmt.Errorf("%w: granularity must be month or year", ErrInvalidInput)
	}
	p, err := s.repos.Portfolio.FindByID(ctx, portfolioID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	if !s.canAccessPortfolio(ctx, p, userID) {
		return nil, ErrForbidden
	}
	currency := p.Currency
	return cached(s.cache, ctx, "pf-perf", portfolioID.String()+":"+granularity, cacheTTLStats, false, func() (*model.DashboardPerformance, error) {
		perf := &model.DashboardPerformance{
			Currency:    currency,
			Granularity: granularity,
			Buckets:     []model.PerformanceBucket{},
		}
		holdings, err := s.repos.Portfolio.HoldingsDetailed(ctx, []uuid.UUID{portfolioID})
		if err != nil {
			return nil, err
		}
		// HasPrice is asset-level: priced assets contribute their market
		// value to V, unpriced ones their cost basis. The holding currency
		// is the asset currency the cash flows are converted from.
		pricedAsset := map[uuid.UUID]bool{}
		assetCurrency := map[uuid.UUID]string{}
		for _, h := range holdings {
			assetID := mustUUID(h.AssetID)
			pricedAsset[assetID] = h.HasPrice
			if h.Currency != "" {
				assetCurrency[assetID] = h.Currency
			}
		}

		quoteSet := map[string]bool{}
		if currency != "" {
			quoteSet[currency] = true
		}
		for _, cur := range assetCurrency {
			quoteSet[cur] = true
		}
		quotes := make([]string, 0, len(quoteSet))
		for c := range quoteSet {
			quotes = append(quotes, c)
		}
		dr, err := series.LoadDateRates(ctx, s.repos, "USD", quotes)
		if err != nil {
			return nil, err
		}

		// Daily market value in the portfolio currency: each asset walks its
		// series as a forward-fill stream, contributing its last point — the
		// market value when priced, the cost basis when unpriced — to every
		// date after its first one. The series is already in the portfolio
		// currency, so no per-date conversion is needed.
		type assetWalk struct {
			pts    []model.PositionPoint
			pos    int
			priced bool
			last   decimal.Decimal
		}
		assetSeries, err := s.repos.Series.FindPortfolio(ctx, portfolioID)
		if err != nil {
			return nil, err
		}
		var walks []*assetWalk
		dateSet := map[time.Time]bool{}
		for _, a := range assetSeries {
			walks = append(walks, &assetWalk{
				pts:    a.Series,
				priced: pricedAsset[mustUUID(a.AssetID)],
			})
			for _, pt := range a.Series {
				dateSet[series.DayOf(pt.Date)] = true
			}
		}

		// External cash flows per day in the portfolio currency. capital is
		// the same flow without the dividend part: distributions are income,
		// not deployed capital, so they move the return but never the
		// invested.
		type dayCash struct{ flow, capital decimal.Decimal }
		flows := map[time.Time]*dayCash{}
		txs, err := s.repos.Transaction.FindByPortfoliosAsc(ctx, []uuid.UUID{portfolioID})
		if err != nil {
			return nil, err
		}
		for _, tx := range txs {
			var flow decimal.Decimal
			switch tx.Type {
			case model.TxBuy:
				flow = tx.Quantity.Mul(tx.Price).Add(tx.Fees)
			case model.TxSell:
				flow = tx.Quantity.Mul(tx.Price).Sub(tx.Fees).Neg()
			case model.TxFee:
				flow = tx.Price
				if tx.Quantity.IsPositive() {
					flow = tx.Quantity.Mul(tx.Price)
				}
			case model.TxDividend:
				flow = tx.Price
				if tx.Quantity.IsPositive() {
					flow = tx.Quantity.Mul(tx.Price)
				}
				flow = flow.Neg()
			default:
				continue
			}
			factor, ok := dr.Factor(assetCurrency[tx.AssetID], currency, tx.Date)
			if !ok {
				continue
			}
			flow = flow.Mul(factor)
			capital := flow
			if tx.Type == model.TxDividend {
				capital = decimal.Zero
			}
			d := series.DayOf(tx.Date)
			dateSet[d] = true
			day, seen := flows[d]
			if !seen {
				day = &dayCash{}
				flows[d] = day
			}
			day.flow = day.flow.Add(flow)
			day.capital = day.capital.Add(capital)
		}

		// The walk runs over the union of the market-observed dates and the
		// flow dates. Dates are ascending and bucket periods are monotonic
		// over them, so every time the period changes the previous bucket is
		// sealed on its last date.
		dates := make([]time.Time, 0, len(dateSet))
		for d := range dateSet {
			dates = append(dates, d)
		}
		sort.Slice(dates, func(i, j int) bool { return dates[i].Before(dates[j]) })

		one := decimal.NewFromInt(1)
		hundred := decimal.NewFromInt(100)
		buckets := make([]model.PerformanceBucket, 0, len(dates))
		bucketFactor, twrFactor := one, one
		var period string
		var invested, endValue, prevV decimal.Decimal
		flush := func() {
			if period == "" {
				return
			}
			buckets = append(buckets, model.PerformanceBucket{
				Period:   period,
				Return:   bucketFactor.Sub(one).Mul(hundred).Round(4),
				TWR:      twrFactor.Sub(one).Mul(hundred).Round(4),
				Invested: roundAmount(invested),
				Value:    roundAmount(endValue),
			})
		}
		for _, d := range dates {
			v := decimal.Zero
			for _, w := range walks {
				for w.pos < len(w.pts) && !series.DayOf(w.pts[w.pos].Date).After(d) {
					pt := w.pts[w.pos]
					if w.priced {
						w.last = pt.MarketValue
					} else {
						w.last = pt.CostBasis
					}
					w.pos++
				}
				v = v.Add(w.last)
			}
			var flow, capital decimal.Decimal
			if c, ok := flows[d]; ok {
				flow, capital = c.flow, c.capital
			}
			if p := performancePeriod(d, granularity); p != period {
				flush()
				period = p
				bucketFactor = one
			}
			invested = invested.Add(capital)
			// A day whose previous value is not positive (the portfolio's
			// first days and the gaps of a fully liquidated one) measures no
			// return: it is skipped, not turned into a fake ±100%.
			if prevV.IsPositive() {
				factor := v.Sub(prevV).Sub(flow).Div(prevV).Add(one)
				bucketFactor = bucketFactor.Mul(factor)
				twrFactor = twrFactor.Mul(factor)
			}
			endValue = v
			prevV = v
		}
		flush()
		perf.Buckets = buckets
		return perf, nil
	})
}

// GetDashboardAllocation returns the user's whole-vault class, geographic,
// country and sector allocation in their base currency, aggregating holdings
// across all portfolios.
func (s *Service) GetDashboardAllocation(ctx context.Context, userID uuid.UUID) (*model.DashboardAllocation, error) {
	return cached(s.cache, ctx, "dash-allocation", userID.String(), cacheTTLStats, false, func() (*model.DashboardAllocation, error) {
		user, err := s.repos.User.FindByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		baseCurrency := user.BaseCurrency
		if baseCurrency == "" {
			baseCurrency = "EUR"
		}
		portfolios, err := s.repos.Portfolio.FindByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if len(portfolios) == 0 {
			return &model.DashboardAllocation{
				Currency:  baseCurrency,
				Classes:   []*model.ClassAllocation{},
				Regions:   []*model.RegionAllocation{},
				Countries: []*model.CountryAllocation{},
				Sectors:   []*model.SectorAllocation{},
				Covered:   decimal.Zero,
				Excluded:  decimal.Zero,
			}, nil
		}

		ids := make([]uuid.UUID, 0, len(portfolios))
		for _, p := range portfolios {
			ids = append(ids, p.ID)
		}
		holdings, err := s.repos.Portfolio.HoldingsDetailed(ctx, ids)
		if err != nil {
			return nil, err
		}
		rates, err := series.LoadRates(ctx, s.repos, holdings, baseCurrency)
		if err != nil {
			return nil, err
		}

		byClass := map[string]decimal.Decimal{}
		var classTotal decimal.Decimal
		for _, h := range holdings {
			if !h.Qty.IsPositive() || !h.HasPrice {
				continue
			}
			value := h.Qty.Mul(h.LastClose)
			factor, ok := series.FxFactor(rates, h.Currency, baseCurrency)
			if !ok {
				continue
			}
			value = value.Mul(factor)
			if !value.IsPositive() {
				continue
			}
			class := h.AssetClass
			if class == "" {
				class = "other"
			}
			byClass[class] = byClass[class].Add(value)
			classTotal = classTotal.Add(value)
		}
		classes := make([]*model.ClassAllocation, 0, len(byClass))
		for class, value := range byClass {
			classes = append(classes, &model.ClassAllocation{
				Class: class,
				Value: value,
			})
		}
		sort.Slice(classes, func(i, j int) bool { return classes[i].Value.GreaterThan(classes[j].Value) })
		for _, c := range classes {
			if classTotal.IsPositive() {
				c.Weight = c.Value.Div(classTotal).Mul(decimal.NewFromInt(100))
			}
		}

		geoExposures, err := s.repos.Exposure.FindRegionsByAssets(ctx, holdingAssetIDs(holdings))
		if err != nil {
			return nil, err
		}
		secExposures, err := s.repos.Exposure.FindSectorsByAssets(ctx, holdingAssetIDs(holdings))
		if err != nil {
			return nil, err
		}
		countryExposures, err := s.repos.Exposure.FindCountriesByAssets(ctx, holdingAssetIDs(holdings))
		if err != nil {
			return nil, err
		}

		gBuckets, gTotal, gCov := buildBuckets(holdings, rates, baseCurrency, geo.Regions, geoExposures,
			func(h *model.Holding) string { return geo.RegionForCountry(h.Country) })
		regions := make([]*model.RegionAllocation, 0, len(geo.Regions)+1)
		for _, name := range geo.Regions {
			regions = append(regions, &model.RegionAllocation{Region: name, Value: gBuckets[name]})
		}
		if gBuckets["Other"].IsPositive() {
			regions = append(regions, &model.RegionAllocation{Region: "Other", Value: gBuckets["Other"]})
		}
		for _, e := range regions {
			if gTotal.IsPositive() {
				e.Weight = e.Value.Div(gTotal).Mul(decimal.NewFromInt(100))
			}
		}

		sBuckets, sTotal, _ := buildBuckets(holdings, rates, baseCurrency, geo.GICSSectors, secExposures,
			func(h *model.Holding) string { return geo.NormalizeSector(h.Sector) })
		sectors := make([]*model.SectorAllocation, 0, len(geo.GICSSectors)+1)
		for _, name := range geo.GICSSectors {
			sectors = append(sectors, &model.SectorAllocation{Sector: name, Value: sBuckets[name]})
		}
		if sBuckets["Other"].IsPositive() {
			sectors = append(sectors, &model.SectorAllocation{Sector: "Other", Value: sBuckets["Other"]})
		}
		for _, e := range sectors {
			if sTotal.IsPositive() {
				e.Weight = e.Value.Div(sTotal).Mul(decimal.NewFromInt(100))
			}
		}

		cBuckets, cTotal, _ := buildBuckets(holdings, rates, baseCurrency, geo.Countries, countryExposures,
			func(h *model.Holding) string { return h.Country })
		countries := make([]*model.CountryAllocation, 0, len(cBuckets))
		for name, value := range cBuckets {
			if !value.IsPositive() {
				continue
			}
			countries = append(countries, &model.CountryAllocation{Country: name, Value: value})
		}
		sort.Slice(countries, func(i, j int) bool { return countries[i].Value.GreaterThan(countries[j].Value) })
		for _, c := range countries {
			if cTotal.IsPositive() {
				c.Weight = c.Value.Div(cTotal).Mul(decimal.NewFromInt(100))
			}
		}

		return &model.DashboardAllocation{
			Currency:  baseCurrency,
			Classes:   classes,
			Regions:   regions,
			Countries: countries,
			Sectors:   sectors,
			Covered:   gCov.covered,
			Excluded:  gCov.excluded,
		}, nil
	})
}

func mustUUID(s string) uuid.UUID {
	id, _ := uuid.Parse(s)
	return id
}

func (s *Service) GetPrices(ctx context.Context, assetID uuid.UUID, full bool) ([]*model.Price, error) {
	kind := "prices"
	if full {
		kind = "prices_full"
	}
	return cached(s.cache, ctx, kind, assetID.String(), cacheTTLPrices, false, func() ([]*model.Price, error) {
		prices, err := s.repos.Price.FindByAsset(ctx, assetID)
		if err != nil {
			return nil, err
		}
		vals := make([]model.Price, 0, len(prices))
		for i := len(prices) - 1; i >= 0; i-- {
			vals = append(vals, *prices[i])
		}
		if !full {
			vals = series.Prices(vals, s.seriesMaxPoints)
		}
		out := make([]*model.Price, 0, len(vals))
		for i := len(vals) - 1; i >= 0; i-- {
			out = append(out, &vals[i])
		}
		return out, nil
	})
}

func (s *Service) canAccessPortfolio(ctx context.Context, portfolio *model.Portfolio, userID uuid.UUID) bool {
	if portfolio.UserID == userID {
		return true
	}
	return false // TODO: check portfolio_shares
}
