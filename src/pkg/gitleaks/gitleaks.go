package gitleaks

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"

	"secguro.com/secguro/pkg/config"
	"secguro.com/secguro/pkg/dependencies"
	"secguro.com/secguro/pkg/functional"
	"secguro.com/secguro/pkg/git"
	"secguro.com/secguro/pkg/types"
)

type GitleaksFinding struct {
	RuleID      string
	File        string
	StartLine   int
	EndLine     int
	StartColumn int
	EndColumn   int
	Match       string
	Commit      string
	Date        string
	Author      string
	Email       string
	Message     string
}

func convertGitleaksFindingToUnifiedFinding(gitMode bool,
	gitleaksFinding GitleaksFinding) (types.UnifiedFinding, error) {
	gitInfo, err := git.GetGitInfo(gitMode, gitleaksFinding.Commit,
		gitleaksFinding.File, gitleaksFinding.StartLine, false)
	if err != nil {
		return types.UnifiedFinding{}, err
	}

	currentLocationGitInfo, err := git.GetGitInfo(gitMode, gitleaksFinding.Commit,
		gitleaksFinding.File, gitleaksFinding.StartLine, true)
	if err != nil {
		return types.UnifiedFinding{}, err
	}

	unifiedFinding := types.UnifiedFinding{
		Detector:    "gitleaks",
		Rule:        gitleaksFinding.RuleID,
		File:        gitleaksFinding.File,
		LineStart:   gitleaksFinding.StartLine,
		LineEnd:     gitleaksFinding.EndLine,
		ColumnStart: gitleaksFinding.StartColumn,
		ColumnEnd:   gitleaksFinding.EndColumn,
		Match:       gitleaksFinding.Match,
		Hint:        "",
		GitInfo:     gitInfo,
	}

	if currentLocationGitInfo != nil {
		latestCommitHash, err := git.GetLatestCommitHash()
		if err != nil {
			return types.UnifiedFinding{}, err
		}

		if currentLocationGitInfo.CommitHash == latestCommitHash {
			unifiedFinding.File = currentLocationGitInfo.File
			unifiedFinding.LineStart = currentLocationGitInfo.Line
			unifiedFinding.LineEnd =
				currentLocationGitInfo.Line + gitleaksFinding.EndLine - gitleaksFinding.StartLine
		} else {
			unifiedFinding.File = ""
			unifiedFinding.LineStart = -1
			unifiedFinding.LineEnd = -1
		}
	}

	return unifiedFinding, nil
}

func getGitleaksOutputJson(gitMode bool) ([]byte, error) {
	gitleaksOutputJsonPath := dependencies.DependenciesDir + "/gitleaksOutput.json"

	cmd := (func() *exec.Cmd {
		if gitMode {
			// secguro-ignore-next-line
			return exec.Command(dependencies.DependenciesDir+"/gitleaks/gitleaks",
				"detect", "--report-format", "json", "--report-path", gitleaksOutputJsonPath)
		} else {
			// secguro-ignore-next-line
			return exec.Command(dependencies.DependenciesDir+"/gitleaks/gitleaks",
				"detect", "--no-git", "--report-format", "json", "--report-path", gitleaksOutputJsonPath)
		}
	})()
	cmd.Dir = config.DirectoryToScan
	// Ignore error because this is expected to deliver an exit code not equal to 0 and write to stderr.
	out, _ := cmd.Output()
	if out == nil {
		return nil, errors.New("did not receive output from gitleaks")
	}

	gitleaksOutputJson, err := os.ReadFile(gitleaksOutputJsonPath)

	return gitleaksOutputJson, err
}

func GetGitleaksFindingsAsUnified(gitMode bool) ([]types.UnifiedFinding, error) {
	gitleaksOutputJson, err := getGitleaksOutputJson(gitMode)
	if err != nil {
		return nil, err
	}

	var gitleaksFindings []GitleaksFinding
	err = json.Unmarshal(gitleaksOutputJson, &gitleaksFindings)
	if err != nil {
		return nil, err
	}

	unifiedFindings, err := functional.MapWithError(gitleaksFindings,
		func(gitleaksFinding GitleaksFinding) (types.UnifiedFinding, error) {
			return convertGitleaksFindingToUnifiedFinding(gitMode, gitleaksFinding)
		})
	if err != nil {
		return make([]types.UnifiedFinding, 0), err
	}

	return unifiedFindings, nil
}
