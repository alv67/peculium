package service

import (
	"context"
	"fmt"
	"maps"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/rs/zerolog/log"

	"github.com/alv67/peculium/internal/model"
	"github.com/alv67/peculium/internal/repository"
	"github.com/alv67/peculium/internal/series"
)

// Restore modes accepted by POST /backup/restore. "add" is non-destructive:
// every portfolio is imported as a new one. "replace" deletes the user's own
// portfolios first (CASCADE removes their transactions) and then imports.
const (
	BackupModeAdd     = "add"
	BackupModeReplace = "replace"
)

// ExportUserBackup builds the per-user backup bundle: the portfolios the
// user owns (each as a standard portfolio-export document with its full
// transaction history), the referenced assets' metadata, their exposure
// (regions/sectors/countries rows with the provenance source) and manual
// price rows, the user's base currency and the supported-currency entries
// in use. Shared portfolios (portfolio_shares) are not duplicated: the
// bundle carries only what belongs to the account. FX history and health
// events are provider data and are excluded, and so are Yahoo prices, which
// the price worker refetches.
func (s *Service) ExportUserBackup(ctx context.Context, userID uuid.UUID) (*model.UserBackup, error) {
	return buildUserBackup(ctx, s.repos, userID)
}

func buildUserBackup(ctx context.Context, rx *repository.Repository, userID uuid.UUID) (*model.UserBackup, error) {
	user, err := rx.User.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	portfolios, err := rx.Portfolio.FindByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	owned := make([]*model.Portfolio, 0, len(portfolios))
	for _, p := range portfolios {
		if p.UserID == userID {
			owned = append(owned, p)
		}
	}
	// Creation order (oldest first) makes the bundle read like the history
	// of the account and keeps consecutive exports diff-stable.
	sort.SliceStable(owned, func(i, j int) bool {
		if !owned[i].CreatedAt.Equal(owned[j].CreatedAt) {
			return owned[i].CreatedAt.Before(owned[j].CreatedAt)
		}
		return owned[i].ID.String() < owned[j].ID.String()
	})

	ids := make([]uuid.UUID, 0, len(owned))
	for _, p := range owned {
		ids = append(ids, p.ID)
	}
	var txs []model.TransactionWithAsset
	if len(ids) > 0 {
		txs, err = rx.Transaction.FindByPortfoliosAsc(ctx, ids)
		if err != nil {
			return nil, err
		}
	}
	txsByPortfolio := map[string][]model.TransactionWithAsset{}
	assetIDs := make([]uuid.UUID, 0, len(txs))
	assetSeen := map[uuid.UUID]bool{}
	for _, tx := range txs {
		txsByPortfolio[tx.PortfolioID.String()] = append(txsByPortfolio[tx.PortfolioID.String()], tx)
		if !assetSeen[tx.AssetID] {
			assetSeen[tx.AssetID] = true
			assetIDs = append(assetIDs, tx.AssetID)
		}
	}
	assets, err := rx.Asset.FindByIDs(ctx, assetIDs)
	if err != nil {
		return nil, err
	}
	sort.Slice(assets, func(i, j int) bool { return assets[i].Ticker < assets[j].Ticker })
	assetByID := make(map[uuid.UUID]*model.Asset, len(assets))
	for _, a := range assets {
		assetByID[a.ID] = a
	}

	exportedAt := time.Now().UTC()
	doc := &model.UserBackup{
		Version:    model.UserBackupVersion,
		ExportedAt: exportedAt,
		User: model.BackupUser{
			Email:        user.Email,
			Name:         user.Name,
			BaseCurrency: user.BaseCurrency,
		},
		Currencies: []model.ExportCurrency{},
		Portfolios: make([]model.PortfolioExport, 0, len(owned)),
		Assets:     make([]model.BackupAsset, 0, len(assets)),
	}

	currencySet := map[string]bool{}
	if user.BaseCurrency != "" {
		currencySet[user.BaseCurrency] = true
	}
	for _, p := range owned {
		if p.Currency != "" {
			currencySet[p.Currency] = true
		}
	}
	for _, a := range assets {
		if a.Currency != "" {
			currencySet[a.Currency] = true
		}
	}
	codes := slices.Sorted(maps.Keys(currencySet))
	for _, code := range codes {
		entry := model.ExportCurrency{Code: code}
		if cur, err := rx.Currency.Get(ctx, code); err != nil {
			return nil, err
		} else if cur != nil {
			entry.Name, entry.Symbol = cur.Name, cur.Symbol
		}
		doc.Currencies = append(doc.Currencies, entry)
	}

	for _, p := range owned {
		pfTxs := txsByPortfolio[p.ID.String()]
		pfAssets := make([]model.ExportAsset, 0, len(pfTxs))
		pfSeen := map[uuid.UUID]bool{}
		for _, tx := range pfTxs {
			if pfSeen[tx.AssetID] {
				continue
			}
			pfSeen[tx.AssetID] = true
			if a := assetByID[tx.AssetID]; a != nil {
				pfAssets = append(pfAssets, toExportAsset(a))
			}
		}
		sort.Slice(pfAssets, func(i, j int) bool { return pfAssets[i].Ticker < pfAssets[j].Ticker })
		pfTxsExport := make([]model.ExportTransaction, 0, len(pfTxs))
		for _, tx := range pfTxs {
			if _, ok := assetByID[tx.AssetID]; !ok {
				continue
			}
			pfTxsExport = append(pfTxsExport, model.ExportTransaction{
				Date:        tx.Date,
				Type:        tx.Type,
				AssetTicker: tx.AssetTicker,
				Quantity:    tx.Quantity,
				Price:       tx.Price,
				Fees:        tx.Fees,
				Notes:       tx.Notes,
			})
		}
		doc.Portfolios = append(doc.Portfolios, model.PortfolioExport{
			Version:    model.UserBackupVersion,
			ExportedAt: exportedAt,
			Portfolio: model.ExportPortfolio{
				Name:        p.Name,
				Description: p.Description,
				Currency:    p.Currency,
			},
			Assets:       pfAssets,
			Transactions: pfTxsExport,
		})
	}

	if len(assets) == 0 {
		return doc, nil
	}
	regionRows, err := rx.Exposure.FindRegionsByAssets(ctx, assetIDs)
	if err != nil {
		return nil, err
	}
	sectorRows, err := rx.Exposure.FindSectorsByAssets(ctx, assetIDs)
	if err != nil {
		return nil, err
	}
	countryRows, err := rx.Exposure.FindCountriesByAssets(ctx, assetIDs)
	if err != nil {
		return nil, err
	}
	provenance, err := rx.Exposure.FindProvenanceByAssets(ctx, assetIDs)
	if err != nil {
		return nil, err
	}
	manualPrices, err := rx.Price.FindManualByAssets(ctx, assetIDs)
	if err != nil {
		return nil, err
	}
	pricesByAsset := map[uuid.UUID][]model.ExportPrice{}
	for _, p := range manualPrices {
		pricesByAsset[p.AssetID] = append(pricesByAsset[p.AssetID], model.ExportPrice{
			Date:   p.Date,
			Open:   p.Open,
			High:   p.High,
			Low:    p.Low,
			Close:  p.Close,
			Volume: p.Volume,
			Source: p.Source,
		})
	}

	for _, a := range assets {
		key := a.ID.String()
		ba := model.BackupAsset{ExportAsset: toExportAsset(a)}
		if ex := buildBackupExposure(a.ISIN, regionRows[key], sectorRows[key], countryRows[key], provenance[key]); ex != nil {
			ba.Exposure = ex
		}
		if prices := pricesByAsset[a.ID]; len(prices) > 0 {
			ba.ManualPrices = prices
		}
		doc.Assets = append(doc.Assets, ba)
	}
	return doc, nil
}

// buildBackupExposure serializes the stored exposure of one asset in the
// same shape AssetExposure uses: the rows per dimension plus the persisted
// provenance source of each saved dimension in the *_source fields. The
// rows are exported as stored (already normalized by the save path), but a
// dimension is only carried when it has rows, so an empty stored dimension
// never shadows the data of another source. An asset without a single
// exposure row returns nil: nothing to bundle.
func buildBackupExposure(isin string, regions, sectors, countries []model.ExposureRow, provenance map[string]model.ExposureProvenance) *model.AssetExposure {
	if len(regions) == 0 && len(sectors) == 0 && len(countries) == 0 {
		return nil
	}
	ex := &model.AssetExposure{ISIN: isin}
	if len(countries) > 0 {
		ex.Countries = countries
		ex.CountriesSource = provenance[model.ExposureDimensionCountries].Source
	}
	if len(regions) > 0 {
		ex.Regions = regions
		ex.RegionsSource = provenance[model.ExposureDimensionRegions].Source
	}
	if len(sectors) > 0 {
		ex.Sectors = sectors
		ex.SectorsSource = provenance[model.ExposureDimensionSectors].Source
	}
	return ex
}

func toExportAsset(a *model.Asset) model.ExportAsset {
	return model.ExportAsset{
		Ticker:      a.Ticker,
		Name:        a.Name,
		Type:        a.Type,
		Currency:    a.Currency,
		ISIN:        a.ISIN,
		PriceSource: a.PriceSource,
		AssetClass:  a.AssetClass,
	}
}

// RestoreUserBackup rebuilds a user's data from a backup bundle, typically
// on a freshly initialized server where the account has just been recreated
// by hand. mode selects the strategy: "add" (the default) imports every
// portfolio as a new one without touching existing data, "replace" deletes
// the portfolios the user owns first — CASCADE removes their transactions —
// and then imports. Assets are global and never deleted: an existing ticker
// is reused as-is, a missing one is created from the bundle. The whole write
// runs in a single transaction, then the series of every created portfolio
// is recomputed. There is no server-side confirmation for "replace": the UI
// confirms client-side before calling.
func (s *Service) RestoreUserBackup(ctx context.Context, userID uuid.UUID, doc *model.UserBackup, mode string) (*model.BackupRestoreSummary, error) {
	mode, err := normalizeBackupMode(mode)
	if err != nil {
		return nil, err
	}
	if err := validateUserBackup(doc); err != nil {
		return nil, err
	}
	summary := &model.BackupRestoreSummary{Mode: mode}
	var created []uuid.UUID
	err = s.repos.WithTx(ctx, func(rx *repository.Repository) error {
		var err error
		created, err = applyUserBackup(ctx, rx, userID, doc, mode, summary)
		return err
	})
	if err != nil {
		return nil, err
	}
	for _, id := range created {
		if err := series.Recompute(ctx, s.repos, id); err != nil {
			log.Warn().Err(err).Str("portfolio_id", id.String()).Msg("series recompute failed")
		}
	}
	s.bumpRev(ctx)
	// Restored assets arrive without market data: queue the global asset sync
	// so the worker backfills their history and splits, exactly like an import.
	if _, err := s.Jobs.Enqueue(ctx, &model.Job{
		Type:        model.JobTypeAssetSync,
		TargetType:  model.JobTargetGlobal,
		RequestedBy: &userID,
	}); err != nil {
		log.Warn().Err(err).Msg("post-restore asset sync enqueue failed")
	}
	return summary, nil
}

func normalizeBackupMode(mode string) (string, error) {
	mode = strings.ToLower(strings.TrimSpace(mode))
	if mode == "" {
		return BackupModeAdd, nil
	}
	if mode != BackupModeAdd && mode != BackupModeReplace {
		return "", fmt.Errorf("%w: backup mode must be add or replace", ErrInvalidInput)
	}
	return mode, nil
}

// validateUserBackup checks the document version (rejecting a bundle newer
// than the importer understands with an explicit message, like the
// portfolio importer does) and the portfolio-level invariants that mirror
// ImportPortfolio's checks.
func validateUserBackup(doc *model.UserBackup) error {
	if doc == nil {
		return ErrInvalidInput
	}
	if doc.Version > model.UserBackupVersion {
		return fmt.Errorf("%w: unsupported backup version %d", ErrInvalidInput, doc.Version)
	}
	if doc.Version != model.UserBackupVersion {
		return ErrInvalidInput
	}
	for i := range doc.Portfolios {
		pf := &doc.Portfolios[i]
		if pf.Version > model.UserBackupVersion {
			return fmt.Errorf("%w: unsupported portfolio export version %d", ErrInvalidInput, pf.Version)
		}
		if pf.Version != model.UserBackupVersion || strings.TrimSpace(pf.Portfolio.Name) == "" {
			return ErrInvalidInput
		}
	}
	for i := range doc.Assets {
		if strings.TrimSpace(doc.Assets[i].Ticker) == "" {
			return fmt.Errorf("%w: bundle asset without ticker", ErrInvalidInput)
		}
	}
	return nil
}

// applyUserBackup writes the bundle back through rx and returns the ids of
// the portfolios it created so the caller can recompute their series. The
// steps are ordered so the global pieces (assets, currencies) exist before
// the data that references them: portfolio transactions, exposure rows,
// manual prices and the user's base currency.
func applyUserBackup(ctx context.Context, rx *repository.Repository, userID uuid.UUID, doc *model.UserBackup, mode string, summary *model.BackupRestoreSummary) ([]uuid.UUID, error) {
	if mode == BackupModeReplace {
		portfolios, err := rx.Portfolio.FindByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		for _, p := range portfolios {
			if p.UserID != userID {
				continue // portfolios shared from other users are not the caller's to delete
			}
			if err := rx.Portfolio.Delete(ctx, p.ID); err != nil {
				return nil, err
			}
		}
	}

	assetCache := map[string]*model.Asset{}
	bundleMeta := map[string]*model.ExportAsset{}
	for i := range doc.Assets {
		ticker := strings.TrimSpace(doc.Assets[i].Ticker)
		key := strings.ToLower(ticker)
		if _, ok := bundleMeta[key]; !ok {
			bundleMeta[key] = &doc.Assets[i].ExportAsset
		}
	}
	resolve := func(ticker string, meta map[string]*model.ExportAsset) (*model.Asset, error) {
		_, known := assetCache[ticker]
		a, created, err := importBackupAsset(ctx, rx, assetCache, meta, ticker)
		if err != nil {
			return nil, err
		}
		if created {
			summary.AssetsCreated++
		} else if !known {
			summary.AssetsReused++
		}
		return a, nil
	}

	// Bundle-level assets are resolved first: they are the shared identity
	// the transactions and the asset-level data key by ticker into.
	for i := range doc.Assets {
		if _, err := resolve(strings.TrimSpace(doc.Assets[i].Ticker), bundleMeta); err != nil {
			return nil, err
		}
	}

	for _, c := range doc.Currencies {
		code := strings.TrimSpace(c.Code)
		if code == "" {
			continue
		}
		existing, err := rx.Currency.Get(ctx, code)
		if err != nil {
			return nil, err
		}
		if existing != nil {
			continue // the whitelist is global: never modify an existing entry
		}
		name := c.Name
		if name == "" {
			name = code
		}
		if err := rx.Currency.Create(ctx, &model.Currency{Code: code, Name: name, Symbol: c.Symbol, Enabled: true}); err != nil {
			return nil, err
		}
	}

	createdIDs := make([]uuid.UUID, 0, len(doc.Portfolios))
	for i := range doc.Portfolios {
		pf := &doc.Portfolios[i]
		for j := range pf.Assets {
			key := strings.ToLower(strings.TrimSpace(pf.Assets[j].Ticker))
			if _, ok := bundleMeta[key]; !ok {
				bundleMeta[key] = &pf.Assets[j]
			}
		}
		p, err := rx.Portfolio.Create(ctx, &model.Portfolio{
			UserID:      userID,
			Name:        strings.TrimSpace(pf.Portfolio.Name),
			Description: pf.Portfolio.Description,
			Currency:    pf.Portfolio.Currency,
		})
		if err != nil {
			return nil, err
		}
		createdIDs = append(createdIDs, p.ID)
		summary.PortfoliosCreated++

		for _, et := range pf.Transactions {
			ticker := strings.TrimSpace(et.AssetTicker)
			if ticker == "" {
				return nil, fmt.Errorf("%w: transaction without asset_ticker", ErrInvalidInput)
			}
			if !model.ValidTransactionType(string(et.Type)) {
				return nil, fmt.Errorf("%w: invalid transaction type %q", ErrInvalidInput, et.Type)
			}
			a, err := resolve(ticker, bundleMeta)
			if err != nil {
				return nil, err
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
				return nil, err
			}
			summary.TransactionsCreated++
		}
	}

	for i := range doc.Assets {
		ba := &doc.Assets[i]
		a := assetCache[strings.TrimSpace(ba.Ticker)]
		if a == nil {
			return nil, fmt.Errorf("%w: unresolvable bundle asset %q", ErrInvalidInput, ba.Ticker)
		}
		if ex := ba.Exposure; ex != nil {
			if len(ex.Countries) > 0 {
				if err := rx.Exposure.ReplaceCountries(ctx, a.ID, ex.Countries); err != nil {
					return nil, err
				}
				if err := rx.Exposure.SetProvenance(ctx, a.ID, model.ExposureDimensionCountries, sourceOrDefault(ex.CountriesSource)); err != nil {
					return nil, err
				}
			}
			if len(ex.Regions) > 0 {
				if err := rx.Exposure.ReplaceRegions(ctx, a.ID, ex.Regions); err != nil {
					return nil, err
				}
				if err := rx.Exposure.SetProvenance(ctx, a.ID, model.ExposureDimensionRegions, sourceOrDefault(ex.RegionsSource)); err != nil {
					return nil, err
				}
			}
			if len(ex.Sectors) > 0 {
				if err := rx.Exposure.ReplaceSectors(ctx, a.ID, ex.Sectors); err != nil {
					return nil, err
				}
				if err := rx.Exposure.SetProvenance(ctx, a.ID, model.ExposureDimensionSectors, sourceOrDefault(ex.SectorsSource)); err != nil {
					return nil, err
				}
			}
		}
		for _, pr := range ba.ManualPrices {
			source := pr.Source
			if source == "" {
				source = "manual"
			}
			if _, err := rx.Price.Create(ctx, &model.Price{
				AssetID: a.ID,
				Date:    pr.Date,
				Open:    pr.Open,
				High:    pr.High,
				Low:     pr.Low,
				Close:   pr.Close,
				Volume:  pr.Volume,
				Source:  source,
			}); err != nil {
				return nil, err
			}
		}
	}

	if cur := strings.TrimSpace(doc.User.BaseCurrency); cur != "" {
		u, err := rx.User.FindByID(ctx, userID)
		if err != nil {
			return nil, err
		}
		if u.BaseCurrency != cur {
			u.BaseCurrency = cur
			if err := rx.User.Update(ctx, u); err != nil {
				return nil, err
			}
		}
	}
	return createdIDs, nil
}

// importBackupAsset resolves one asset during an import: a stored asset with
// the same ticker is reused as-is (assets are global and shared, so the
// import never overwrites their identity metadata), a missing one is created
// from the exported metadata. The cache ensures each ticker is resolved at
// most once per import; importNew reports whether the asset was created here.
//
// The defaults below keep documents exported by older app versions
// importable: those files predate fields like price_source/asset_class and
// may omit any optional value, so every missing piece is filled with a
// constraint-satisfying default instead of failing the insert. An unknown or
// absent price_source falls back to "yahoo" rather than erroring, the same
// default Service.CreateAsset applies.
func importBackupAsset(ctx context.Context, rx *repository.Repository, cache map[string]*model.Asset, metaByTicker map[string]*model.ExportAsset, ticker string) (*model.Asset, bool, error) {
	if a, ok := cache[ticker]; ok {
		return a, false, nil
	}
	a, err := rx.Asset.FindByTicker(ctx, ticker)
	if err != nil {
		return nil, false, err
	}
	created := false
	if a == nil {
		a = &model.Asset{Ticker: ticker, Name: ticker, Type: model.AssetTypeStock, Currency: "USD", PriceSource: "yahoo"}
		if ea := metaByTicker[strings.ToLower(ticker)]; ea != nil {
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
		}
		if a.AssetClass == "" {
			a.AssetClass = defaultAssetClassForType(a.Type)
		}
		a, err = rx.Asset.Create(ctx, a)
		if err != nil {
			return nil, false, err
		}
		created = true
	}
	cache[ticker] = a
	return a, created, nil
}
