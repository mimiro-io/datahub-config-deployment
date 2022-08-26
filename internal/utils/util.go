package utils

import (
	"bufio"
	"encoding/json"
	"errors"
	"github.com/pterm/pterm"
	"io"
	"io/ioutil"
	"os"
	"strings"
)

type ErrorDetails struct {
	File    string
	Line    int
	Col     int
	Message string
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
