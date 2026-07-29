package worker

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/ai"
	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/github"
)

type Pool struct {
	jobQueue     chan Job
	maxWorkers   int
	wg           sync.WaitGroup
	githubClient *github.Client
	aiReviewer   ai.Reviewer
}

func NewPool(maxWorkers, queueSize int, githubClient *github.Client, aiReviewer ai.Reviewer) *Pool {
	return &Pool{
		jobQueue:     make(chan Job, queueSize),
		maxWorkers:   maxWorkers,
		githubClient: githubClient,
		aiReviewer:   aiReviewer,
	}
}

func (p *Pool) Start(ctx context.Context) {
	for i := 1; i <= p.maxWorkers; i++ {
		p.wg.Add(1)
		go p.worker(ctx, i)
	}
	slog.Info("Worker pool started", "max_workers", p.maxWorkers, "queue_capacity", cap(p.jobQueue))
}

func (p *Pool) worker(ctx context.Context, workerId int) {
	defer p.wg.Done()

	slog.Debug("Worker started", "worker_id", workerId)

	for {
		select {
		case <-ctx.Done():
			slog.Debug("Worker received cancellation context", "worker_id", workerId)
			return
		case job, ok := <-p.jobQueue:
			if !ok {
				slog.Debug("Job queue closed, worker exiting", "worker_id", workerId)
				return
			}
			p.processJob(workerId, job)
		}
	}
}

func (p *Pool) Enqueue(job Job) bool {
	select {
	case p.jobQueue <- job:
		slog.Info("Job enqueued successfully", "pr_number", job.Event.Number)
		return true
	default:
		slog.Error("Worker queue full! Dropping job", "pr_number", job.Event.Number)
		return false
	}
}

func (p *Pool) Stop() {
	slog.Info("Stopping worker pool, draining remaining jobs...")
	close(p.jobQueue)
	p.wg.Wait()
	slog.Info("Worker pool stopped cleanly")
}

func (p *Pool) processJob(workerId int, job Job) {
	slog.Info("Worker picked up job",
		"worker_id", workerId,
		"pr_number", job.Event.Number,
		"action", job.Event.Action,
		"repo", job.Event.Repository.Name,
		"owner", job.Event.Repository.Owner.Login,
	)

	ctx := context.Background()

	// 1. Get Installation Token
	token, err := p.githubClient.GetInstallationToken(ctx, job.Event.Installation.ID)
	if err != nil {
		slog.Error("Failed to fetch installation token", "error", err, "pr_number", job.Event.Number)
		return
	}

	// 2. Handle Reopened Action
	if job.Event.Action == "reopened" {
		_ = p.githubClient.PostPRReview(ctx, token,
			job.Event.Repository.Owner.Login,
			job.Event.Repository.Name,
			job.Event.Number,
			"Check the previous Review.",
		)
		return
	}

	// 3. Immediately trigger "In Progress" spinner UI on PR page
	checkRunID, err := p.githubClient.CreateCheckRun(ctx, token,
		job.Event.Repository.Owner.Login,
		job.Event.Repository.Name,
		job.Event.PullRequest.Head.SHA,
	)
	if err != nil {
		slog.Warn("Failed to create check run UI status", "error", err)
	}

	// Clean up check status on completion (always pass "success" so merge is never blocked)
	defer func() {
		if checkRunID != 0 {
			_ = p.githubClient.CompleteCheckRun(ctx, token,
				job.Event.Repository.Owner.Login,
				job.Event.Repository.Name,
				checkRunID,
				"success",
			)
		}
	}()

	// 4. Fetch diff
	diff, err := p.githubClient.GetPullRequestDiff(ctx,
		token,
		job.Event.Repository.Owner.Login,
		job.Event.Repository.Name,
		job.Event.Number,
	)
	if err != nil {
		slog.Error("Failed to fetch PR diff", "error", err, "pr_number", job.Event.Number)
		return
	}

	// 5. Generate AI Review
	review, err := p.aiReviewer.GenerateReview(
		ctx,
		job.Event.PullRequest.Title,
		job.Event.PullRequest.Body,
		diff,
	)
	if err != nil {
		slog.Error("Failed to generate AI review", "error", err, "pr_number", job.Event.Number)
		return
	}

	// 6. Post PR Review
	err = p.githubClient.PostPRReview(
		ctx,
		token,
		job.Event.Repository.Owner.Login,
		job.Event.Repository.Name,
		job.Event.Number,
		review,
	)
	if err != nil {
		slog.Error("Failed to post PR review comment", "error", err, "pr_number", job.Event.Number)
		return
	}

	slog.Info("Successfully posted AI review to PR!", "pr_number", job.Event.Number)
}
