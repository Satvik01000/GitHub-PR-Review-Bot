package worker

import (
	"github.com/Satvik01000/GitHub-PR-Review-Bot/internal/domain"
)

type Job struct {
	Event *domain.PullRequestEvent
}
