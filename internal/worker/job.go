package worker

import "github.com/Satvik01000/GitHub-PR-Review-Bot/internal/webhook"

type Job struct {
	Event *webhook.PullRequestEvent
}
