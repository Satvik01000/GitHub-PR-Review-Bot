package worker

import (
	"context"
	"log/slog"
	"sync"

	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/github"
)

type Pool struct {
	jobQueue     chan Job
	maxWorkers   int
	wg           sync.WaitGroup
	githubClient *github.Client
}

func NewPool(maxWorkers, queueSize int, githubClient *github.Client) *Pool {
	return &Pool{
		jobQueue:     make(chan Job, queueSize),
		maxWorkers:   maxWorkers,
		githubClient: githubClient,
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
		"repo", job.Event.Repository.Name,
		"owner", job.Event.Repository.Owner.Login,
	)

	ctx := context.Background()

	token, err := p.githubClient.GetInstallationToken(ctx, job.Event.Installation.ID)
	if err != nil {
		slog.Error("Failed to fetch installation token", "error", err, "pr_number", job.Event.Number)
		return
	}

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

	slog.Info("Successfully retrieved PR diff",
		"pr_number", job.Event.Number,
		"diff_size_bytes", len(diff),
	)
	//TODO
	// Step 6 (Agentic Engine execution) will live here!
}
