package utils

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/pterm/pterm"
	"io"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type ErrorDetails struct {
	File    string
	Line    int
	Col     int
	Message string
}

type DatasetResponse struct {
	Items            int      `json:"items"`
	Name             string   `json:"name"`
	PublicNamespaces []string `json:"publicNamespaces"`
}

func HandleError(err error) {
	if err != nil {
		printer := pterm.PrefixPrinter{
			MessageStyle: &pterm.ThemeDefault.ErrorMessageStyle,
			Prefix: pterm.Prefix{
				Style: &pterm.ThemeDefault.ErrorPrefixStyle,
				Text:  " ERROR ",
			},
			ShowLineNumber: false,
		}

		printer.Println(err.Error())
		pterm.Println()
		os.Exit(1)
	}
}

func ReadStdIn() ([]byte, error) {
	fi, err := os.Stdin.Stat()
	if err != nil {
		return nil, err
	}
	if fi.Mode()&os.ModeNamedPipe == 0 {
		return nil, errors.New("no input on stdin pipe")
	} else {
		reader := bufio.NewReader(os.Stdin)
		var output []byte

		for {
			input, err := reader.ReadByte()

			if err != nil && err == io.EOF {
				break
			}
			output = append(output, input)
		}

		return output, nil
	}
}

func VerifyPath(path string) error {
	rawFile, err := os.Open(path)
	if err != nil {
		return err
	}
	defer rawFile.Close()
	return nil
}

func ReadFile(path string) ([]byte, error) {
	rawFile, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer rawFile.Close()

	fileBytes, _ := ioutil.ReadAll(rawFile)
	return fileBytes, nil
}

func ReadJsonFile(path string) (map[string]interface{}, error) {
	fileBytes, err := ReadFile(path)
	if err != nil {
		return nil, err
	}

	var jsonContent map[string]interface{}
	err = json.Unmarshal(fileBytes, &jsonContent)
	if err != nil {
		return nil, err
	}
	return jsonContent, nil
}

func ReadJson(rawJson []byte) (map[string]interface{}, error) {
	var jsonContent map[string]interface{}

	err := json.Unmarshal(rawJson, &jsonContent)
	if err != nil {
		return nil, err
	}
	return jsonContent, nil
}

func LogCommand(args []string, logFormat string, comment string) {
	cmd := strings.Join(args, " ")
	switch logFormat {
	case "github":
		if comment != "" {
			pterm.DefaultBasicText.Printf("::warning:: Executing %s (%s)\n", cmd, comment)
		} else {
			pterm.DefaultBasicText.Printf("::warning:: Executing %s\n", cmd)
		}
	default:
		if comment != "" {
			pterm.Info.Printf(" > Executing %s (%s)\n", cmd, comment)
		} else {
			pterm.Info.Printf(" > Executing %s\n", cmd)
		}

	}

}

func LogPlain(logString string, logFormat string) {
	switch logFormat {
	case "github":
		pterm.DefaultBasicText.Printf("::warning::%s\n", logString)
	default:
		pterm.DefaultParagraph.Println(logString)
	}
}

func LogError(error ErrorDetails, logFormat string) {
	switch logFormat {
	case "github":
		pterm.DefaultBasicText.Printf("::error file=%s,line=%d,col=%d::%s\n", error.File, error.Line, error.Col, error.Message)
	default:
		pterm.DefaultParagraph.Println(error.Message)
	}
}

func MimCommand(cmd []string, logFormat string, dryRun bool) ([]byte, error) {
	cmdExec := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(cmd, " ")))
	LogCommand(cmd, logFormat, "")
	if dryRun {
		return nil, nil
	}
	return cmdExec.CombinedOutput()
}

func MimDatasetStore(datasetName string, payload []byte, logFormat string, dryRun bool) error {
	tmpFilename := "tmp_entity"
	cmd := []string{"mim", "dataset", "store", datasetName, "-f", tmpFilename}
	err := ioutil.WriteFile(tmpFilename, payload, 0644)
	if err != nil {
		pterm.Error.Println("Failed to write entity to temp file")
		return err
	}
	_, err = MimCommand(cmd, logFormat, dryRun)
	err2 := os.Remove(tmpFilename)
	if err2 != nil {
		pterm.Error.Println("Failed to remove tmp file for core entity")
		return err2
	}

	return err
}

func MimDatasetDelete(datasetName string, logFormat string, dryRun bool) error {
	cmd := []string{"mim", "dataset", "delete", datasetName, "-C=false"}
	output, err := MimCommand(cmd, logFormat, dryRun)
	if err != nil {
		pterm.Error.Printf("Failed to delete dataset '%s':\n%s\n", datasetName, string(output))
		return err
	}
	return nil
}

func MimDatasetGet(datasetName string, logFormat string, dryRun bool) (DatasetResponse, error) {
	cmd := []string{"mim", "dataset", "get", datasetName, "--json"}
	output, err := MimCommand(cmd, logFormat, dryRun)
	var dataset DatasetResponse
	if err != nil {
		pterm.Error.Printf("Failed to get dataset '%s' from datahub: %s\n", datasetName, string(output))
		return dataset, err
	}
	json.Unmarshal(output, dataset)
	return dataset, nil
}

func MimDatasetCreate(datasetName string, publicNamespaces []string, logFormat string, dryRun bool) error {
	cmd := []string{"mim", "dataset", "create", datasetName}
	if len(publicNamespaces) > 0 {
		cmd = []string{"mim", "dataset", "create", datasetName, "--publicNamespaces", fmt.Sprintf("'%s'", strings.Join(publicNamespaces, "','"))}
	}
	output, err := MimCommand(cmd, logFormat, dryRun)
	if err != nil {
		pterm.Error.Println("Failed to create dataset in datahub: ", string(output))
		return err
	}
	return nil

}
