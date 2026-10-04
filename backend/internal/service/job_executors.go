package service

import (
	"context"
	"fmt"

	"github.com/alv67/peculium/internal/model"
	"github.com/google/uuid"
	"github.com/rs/zerolog/log"
)

// RegisterJobExecutors binds the real queue executors to a runner. Each one
// only drives the same Service methods the HTTP handlers used to call
// inline, so queue work never duplicates business logic; it adds progress
// reporting and the terminal status classification. Health events are tagged
// with the job id automatically: JobRunner.RunOnce executes on a context
// bound with WithJobID, and the runner itself emits the job_started plus one
// terminal job event around every run. Executors deliberately record no
// duplicate per-item events: per-bar history saves, split saves and price
// issues are already logged by the fetchers, so a second identical row per
// failure would only double the health volume. The one exception is
// meta_backfill, whose per-asset profile failures have no fetcher-side event
// and are recorded here (BackfillAssetMeta).
func (s *Service) RegisterJobExecutors(runner *JobRunner) {
	runner.Register(model.JobTypePriceRefresh, s.execPriceRefresh)
	runner.Register(model.JobTypeHistoryBackfill, s.execHistoryBackfill)
	runner.Register(model.JobTypeMetaBackfill, s.execMetaBackfill)
}

// reportProgress persists processed/total. A failed progress write is only
// logged: it must not mask the executor's actual outcome.
func (s *Service) reportProgress(ctx context.Context, job *model.Job, processed, total int) {
	if err := s.repos.Job.UpdateProgress(ctx, job.ID, processed, total); err != nil {
		log.Warn().Err(err).Str("job_id", job.ID.String()).Msg("job progress update failed")
	}
}

func jobAssetID(job *model.Job) (uuid.UUID, error) {
	if job.TargetID == nil {
		return uuid.Nil, fmt.Errorf("%w: job %s has no asset target", ErrInvalidInput, job.ID)
	}
	return *job.TargetID, nil
}

func (s *Service) execPriceRefresh(ctx context.Context, job *model.Job) (string, error) {
	var portfolioID *uuid.UUID
	if job.TargetType == model.JobTargetPortfolio {
		portfolioID = job.TargetID
	}
	report, err := s.RefreshPrices(ctx, portfolioID)
	if err != nil {
		return "", err
	}
	done := len(report.Refreshed) + len(report.Issues)
	s.reportProgress(ctx, job, done, done)
	if len(report.Issues) > 0 {
		// Some quotes failed upstream (per-asset issues are in the health
		// log); the refresh still completed for the rest.
		return model.JobStatusPartial, nil
	}
	return model.JobStatusDone, nil
}

func (s *Service) execHistoryBackfill(ctx context.Context, job *model.Job) (string, error) {
	assetID, err := jobAssetID(job)
	if err != nil {
		return "", err
	}
	// The job's two work items: the full price history and the split
	// metadata, since splits belong to the asset profile as much as quotes.
	s.reportProgress(ctx, job, 0, 2)
	if err := s.BackfillAssetHistory(ctx, assetID); err != nil {
		return "", err
	}
	s.reportProgress(ctx, job, 1, 2)

	splitsFailed := false
	if asset, err := s.repos.Asset.FindByID(ctx, assetID); err == nil && isYahooPriced(asset) {
		if err := s.fetcher.EnsureSplits(ctx, []*model.Asset{asset}); err != nil {
			log.Warn().Err(err).Str("job_id", job.ID.String()).Msg("splits ensure failed during history backfill")
			splitsFailed = true
		}
	}
	s.reportProgress(ctx, job, 2, 2)
	if splitsFailed {
		return model.JobStatusPartial, nil
	}
	return model.JobStatusDone, nil
}

func (s *Service) execMetaBackfill(ctx context.Context, job *model.Job) (string, error) {
	report, err := s.BackfillAssetMeta(ctx, func(processed, total int) {
		s.reportProgress(ctx, job, processed, total)
	})
	if err != nil {
		return "", err
	}
	if report.Failed > 0 {
		return model.JobStatusPartial, nil
	}
	return model.JobStatusDone, nil
}
