package reporting

import (
	"errors"
	"fmt"
	"net/http"

	resty "github.com/go-resty/resty/v2"
	"secguro.com/secguro/pkg/config"
	"secguro.com/secguro/pkg/types"
)

const endpointSaveScan = "saveScan"

func ReportScan(unifiedFindings []types.UnifiedFinding) error {
	fmt.Print("Sending scan report to server...")

	urlEndpointSaveScan := config.ServerUrl + "/" + endpointSaveScan

	scanPostReq := ScanPostReq{
		ProjectName: "example project",
		Revision:    "example revision",
		Findings:    unifiedFindings,
	}

	result := ConfirmationRes{} //nolint: exhaustruct
	client := resty.New()
	response, err := client.R().
		SetHeader("Content-Type", "application/json").
		SetBody(scanPostReq).
		SetResult(&result).
		Post(urlEndpointSaveScan)

	if err != nil {
		return err
	}

	if response.StatusCode() != http.StatusCreated {
		return errors.New("received bad status code")
	}

	if result.Status != "created" {
		return errors.New("received bad status response")
	}

	fmt.Println("done")

	return nil
}