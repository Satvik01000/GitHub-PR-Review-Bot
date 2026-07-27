package worker

import (
	"context"
	"log/slog"
	"sync"
)

type Pool struct {
	jobQueue   chan Job
	maxWorkers int
	wg         sync.WaitGroup
}

func NewPool(maxWorkers, queueSize int) *Pool {
	return &Pool{
		jobQueue:   make(chan Job, queueSize),
		maxWorkers: maxWorkers,
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

	//TODO
	// Step 5 (GitHub API diff fetch) & Step 6 (Agentic Engine execution) will live here!
}
