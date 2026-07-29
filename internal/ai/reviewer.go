package ai

import "context"

type Reviewer interface {
	GenerateReview(ctx context.Context, prTitle, prDescription, diff string) (string, error)
}
