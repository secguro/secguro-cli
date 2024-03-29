package scan

import (
	"fmt"
	"os"
	"strings"

	ignore "github.com/sabhiram/go-gitignore"
	"secguro.com/secguro/pkg/dependencies"
	"secguro.com/secguro/pkg/dependencycheck"
	"secguro.com/secguro/pkg/functional"
	"secguro.com/secguro/pkg/gitleaks"
	"secguro.com/secguro/pkg/ignoring"
	"secguro.com/secguro/pkg/output"
	"secguro.com/secguro/pkg/semgrep"
	"secguro.com/secguro/pkg/types"
)

const maxFindingsIndicatingExitCode = 250

func CommandScan(gitMode bool, disabledDetectors []string,
	printAsJson bool, outputDestination string, tolerance int) error {
	unifiedFindingsNotIgnored, err := PerformScan(gitMode, disabledDetectors)
	if err != nil {
		return err
	}

	err = writeOutput(gitMode, printAsJson, outputDestination, unifiedFindingsNotIgnored)
	if err != nil {
		return err
	}

	exitWithAppropriateExitCode(len(unifiedFindingsNotIgnored), tolerance)

	return nil
}

func PerformScan(gitMode bool, disabledDetectors []string) ([]types.UnifiedFinding, error) {
	fmt.Print("Downloading and extracting dependencies...")
	err := dependencies.InstallDependencies(disabledDetectors)
	if err != nil {
		return nil, err
	}
	fmt.Println("done")

	fmt.Print("Scanning...")
	unifiedFindings, err := getUnifiedFindings(gitMode, disabledDetectors)
	if err != nil {
		return nil, err
	}
	fmt.Println("done")

	unifiedFindingsNotIgnored, err := getFindingsNotIgnored(unifiedFindings)
	if err != nil {
		return nil, err
	}

	return unifiedFindingsNotIgnored, nil
}

func exitWithAppropriateExitCode(numberOfFindingsNotIgnored int, tolerance int) {
	if numberOfFindingsNotIgnored <= tolerance {
		os.Exit(0)
	}

	if numberOfFindingsNotIgnored > maxFindingsIndicatingExitCode {
		os.Exit(maxFindingsIndicatingExitCode)
	}

	os.Exit(numberOfFindingsNotIgnored)
}

func getUnifiedFindings(gitMode bool, disabledDetectors []string) ([]types.UnifiedFinding, error) {
	unifiedFindings := make([]types.UnifiedFinding, 0)

	if !functional.ArrayIncludes(disabledDetectors, "gitleaks") {
		unifiedFindingsGitleaks, err := gitleaks.GetGitleaksFindingsAsUnified(gitMode)
		if err != nil {
			return unifiedFindings, err
		}
		unifiedFindings = append(unifiedFindings, unifiedFindingsGitleaks...)
	}

	if !functional.ArrayIncludes(disabledDetectors, "semgrep") {
		unifiedFindingsSemgrep, err := semgrep.GetSemgrepFindingsAsUnified(gitMode)
		if err != nil {
			return unifiedFindings, err
		}
		unifiedFindings = append(unifiedFindings, unifiedFindingsSemgrep...)
	}

	if !functional.ArrayIncludes(disabledDetectors, "dependencycheck") {
		unifiedFindingsDependencycheck, err := dependencycheck.GetDependencycheckFindingsAsUnified(gitMode)
		if err != nil {
			return unifiedFindings, err
		}
		unifiedFindings = append(unifiedFindings, unifiedFindingsDependencycheck...)
	}

	return unifiedFindings, nil
}

func getFindingsNotIgnored(unifiedFindings []types.UnifiedFinding) ([]types.UnifiedFinding, error) { //nolint: cyclop
	lineBasedIgnoreInstructions := ignoring.GetLineBasedIgnoreInstructions(unifiedFindings)
	fileBasedIgnoreInstructions, err := ignoring.GetFileBasedIgnoreInstructions()
	if err != nil {
		return make([]types.UnifiedFinding, 0), err
	}

	ignoreInstructions := []ignoring.IgnoreInstruction{
		// Ignore .secguroignore and .secguroignore-secrets in case
		// a detector finds something in there in the future (does
		// not currently appear to be the case).
		{
			FilePath:   "/" + ignoring.IgnoreFileName,
			LineNumber: -1,
			Rules:      make([]string, 0),
		},
		{
			FilePath:   "/" + ignoring.SecretsIgnoreFileName,
			LineNumber: -1,
			Rules:      make([]string, 0),
		},
	}
	ignoreInstructions = append(ignoreInstructions, lineBasedIgnoreInstructions...)
	ignoreInstructions = append(ignoreInstructions, fileBasedIgnoreInstructions...)

	ignoredSecrets, err := ignoring.GetIgnoredSecrets()
	if err != nil {
		return make([]types.UnifiedFinding, 0), err
	}

	unifiedFindingsNotIgnored := functional.Filter(unifiedFindings, func(unifiedFinding types.UnifiedFinding) bool {
		// Filter findings based on rules ignored for specific paths as well as on specific lines.
		for _, ii := range ignoreInstructions {
			gitIgnoreMatcher := ignore.CompileIgnoreLines(ii.FilePath)
			if gitIgnoreMatcher.MatchesPath(unifiedFinding.File) &&
				(ii.LineNumber == unifiedFinding.LineStart || ii.LineNumber == -1) &&
				(len(ii.Rules) == 0 || functional.ArrayIncludes(ii.Rules, unifiedFinding.Rule)) {
				return false
			}
		}

		// Filter findings based on ignored secrets
		for _, ignoredSecret := range ignoredSecrets {
			if !IsSecretDetectionRule(unifiedFinding.Rule) {
				continue
			}

			if strings.Contains(unifiedFinding.Match, ignoredSecret) {
				return false
			}
		}

		return true
	})

	return unifiedFindingsNotIgnored, nil
}

func writeOutput(gitMode bool, printAsJson bool,
	outputDestination string, unifiedFindingsNotIgnored []types.UnifiedFinding) error {
	var outputString string
	if printAsJson {
		var err error
		outputString, err = output.PrintJson(unifiedFindingsNotIgnored, gitMode)
		if err != nil {
			return err
		}
	} else {
		outputString = output.PrintText(unifiedFindingsNotIgnored, gitMode)
	}

	if outputDestination == "" {
		fmt.Println("Findings:")
		fmt.Println(outputString)
	} else {
		const filePermissions = 0644
		err := os.WriteFile(outputDestination, []byte(outputString), filePermissions)
		if err != nil {
			return err
		}

		fmt.Println("Output written to: " + outputDestination)
	}

	return nil
}
