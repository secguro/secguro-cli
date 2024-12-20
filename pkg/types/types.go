package types

type GitInfo struct {
	CommitHash         string
	CommitDate         string
	AuthorName         string
	AuthorEmailAddress string
	CommitSummary      string
	File               string
	Line               int
}

type UnifiedFinding struct {
	Detector             string
	IdOnExternalPlatform *string
	Rule                 string
	File                 string
	LineStart            int
	LineEnd              int
	ColumnStart          int
	ColumnEnd            int
	Match                string
	Hint                 string
	Severity             string
	GitInfo              *GitInfo
}

type DetectorTermination struct {
	Detector   string
	Successful bool
}
