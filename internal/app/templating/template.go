package templating

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-config-deployment/internal/utils"
	"html/template"
	"path/filepath"
	"regexp"
	"strings"
)

type Templating struct {
}

func NewTemplating() *Templating {
	return &Templating{}
}

func (t *Templating) ReplaceTemplateVariables(name string, jsonBytes []byte, variables map[string]interface{}) ([]byte, error) {
	tmpl, err := template.New(name).Parse(string(jsonBytes))
	if err != nil {
		return nil, err
	}

	var out bytes.Buffer
	err = tmpl.Execute(&out, variables)
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

func (t *Templating) ReplaceVariables(jsonBytes []byte, variables map[string]interface{}) ([]byte, error) {
	rawJson := string(jsonBytes)
	for key, value := range variables {
		inLinePattern := "[^\"]{{ " + key + " }}|{{ " + key + " }}[^\"]"
		r, _ := regexp.Compile(inLinePattern)
		matches := r.FindAllString(rawJson, -1)
		for _, match := range matches {
			replacedMatch := strings.Replace(match, "{{ "+key+" }}", value.(string), -1)
			rawJson = strings.Replace(rawJson, match, replacedMatch, -1)
		}
		wrappedValue, err := t.wrapWithType(value)
		if err != nil {
			return nil, err
		}
		rawJson = strings.Replace(rawJson, "\"{{ "+key+" }}\"", wrappedValue, -1)
	}
	return []byte(rawJson), nil
}
func (t *Templating) wrapWithType(inputValue interface{}) (string, error) {
	// Used to determine type of interface data and
	// to wrap the value to be inserted into a raw json string
	switch inputValue.(type) {
	case int:
		return fmt.Sprint(inputValue.(int)), nil
	case float64:
		return fmt.Sprint(inputValue.(float64)), nil
	case string:
		return "\"" + inputValue.(string) + "\"", nil
	default:
		jsonValue, err := json.Marshal(inputValue)
		if err != nil {
			return "", err
		}
		return string(jsonValue), nil
	}
}

func (t *Templating) ReplaceVariableLogic(jsonBytes []byte, rootPath string) ([]byte, error) {
	stringifiedJson := string(jsonBytes)

	r := regexp.MustCompile(`"{%\s(.+)\s%}"`)
	results := r.FindAllStringSubmatch(stringifiedJson, -1)

	for _, result := range results {
		logicParts := strings.Split(result[1], " ")
		partLength := len([]rune(logicParts[1]))

		// Possibly add other logic operators in the future
		if logicParts[0] == "include" {
			includePath := ""
			forceList := false
			if strings.HasPrefix(logicParts[1], "list('") {
				forceList = true
				// strip 'list(' and closing ')'
				includePath = logicParts[1][6 : partLength-2]

			} else if strings.HasPrefix(logicParts[1], "'") {
				// strip ' on both sides
				includePath = logicParts[1][1 : partLength-1]
			} else {
				return jsonBytes, fmt.Errorf("unable to parse include expression")
			}
			// Get files
			files, err := filepath.Glob(filepath.Join(rootPath, includePath))
			if err != nil {
				return jsonBytes, fmt.Errorf("failed to get files from path")
			}
			var output []map[string]interface{}
			for _, jsonFile := range files {
				jsonContent, err := utils.ReadJsonFile(jsonFile)
				if err != nil {
					return jsonBytes, fmt.Errorf("failed to read json from '%s'", jsonFile)
				}
				output = append(output, jsonContent)
			}
			var jsonOutput []byte
			if !forceList && len(output) == 1 {
				jsonOutput, err = json.Marshal(output[0])
			} else {
				jsonOutput, err = json.Marshal(output)
			}
			stringifiedJson = strings.Replace(stringifiedJson, result[0], string(jsonOutput), -1)
		}
	}
	return []byte(stringifiedJson), nil

}
