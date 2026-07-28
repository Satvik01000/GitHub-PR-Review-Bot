package domain

type PullRequestEvent struct {
	Action       string       `json:"action"`
	Number       int          `json:"number"`
	PullRequest  PullRequest  `json:"pull_request"`
	Repository   Repository   `json:"repository"`
	Installation Installation `json:"installation"`
}

type Installation struct {
	ID int64 `json:"id"`
}

type PullRequest struct {
	Title string `json:"title"`
	Body  string `json:"body"`
	Head  Commit `json:"head"`
	Base  Commit `json:"base"`
}

type Commit struct {
	SHA string `json:"sha"`
}

type Repository struct {
	Name  string `json:"name"`
	Owner Owner  `json:"owner"`
}

type Owner struct {
	Login string `json:"login"`
}
