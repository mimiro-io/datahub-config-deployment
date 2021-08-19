package app

import (
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"github.com/mimiro-io/datahub-config-deployment/internal/app/environment"
	"github.com/mimiro-io/datahub-config-deployment/internal/utils"
	"github.com/pterm/pterm"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
)

type ManifestConfig struct {
	Env *environment.Environment
}

type Manifest struct {
	Id         string            `json:"id"`
	Manifest   map[string]config `json:"manifest"`
	Operations []operation       `json:"operations"`
}

type config struct {
	Id              string                 `json:"id"`
	Path            string                 `json:"path"`
	Digest          string                 `json:"digest"`
	Type            string                 `json:"type"`
	JsonContent     map[string]interface{} `json:"jsonContent"`
	TransformDigest string                 `json:"transformDigest"`
}

type operation struct {
	Config         config `json:"config"`
	ConfigPath     string `json:"configPath"`
	Action         string `json:"action"`
	HasTransform   bool   `json:"hasTransform"`
	RequireDataset bool   `json:"requireDataset"`
}

func NewManifest(env *environment.Environment) *ManifestConfig {
	return &ManifestConfig{Env: env}
}

func hasTransform(JsonContent map[string]interface{}) bool {
	value, exist := JsonContent["transform"]
	return exist && value != nil
}

func requireDataset(JsonContent map[string]interface{}) bool {
	_, value := JsonContent["requireDataset"]
	return value
}

func determineSinkDataset(jsonContent map[string]interface{}) string {
	return jsonContent["sink"].(map[string]interface{})["Name"].(string)
}

func getTransformDigest(path string) (string, error) {
	fileBytes, err := utils.ReadFile(path)
	if err != nil {
		return "", err
	}
	hasher := md5.New()
	hasher.Write(fileBytes)
	hash := hex.EncodeToString(hasher.Sum(nil))
	return hash, nil
}

func createDigest(jsonContent map[string]interface{}) (string, error) {
	// create md5 hash
	b, err := json.Marshal(jsonContent)
	if err != nil {
		return "", err
	}
	hasher := md5.New()
	hasher.Write(b)
	hash := hex.EncodeToString(hasher.Sum(nil))
	return hash, err
}

func (m *ManifestConfig) getManifestFromDatahub() (*Manifest, error) {
	args := []string{
		"mim", "content", "show", "DatahubConfigManifest", "--json",
	}

	utils.LogCommand(args, "default")

	cmdMim := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(args, " ")))
	output, err := cmdMim.CombinedOutput()
	if err != nil {
		pterm.Error.Println("Command to get manifest from datahub failed with error: ", string(output), err)
		return nil, err
	}
	manifest := &Manifest{}
	err = json.Unmarshal(output, manifest)
	if err != nil {
		fmt.Println("Failed to unmarshal manifest: ", err)
		return nil, err
	}
	return manifest, err
}

func (m *ManifestConfig) writeManifestToDatahub(input string) error {

	// Create temp file with string payload
	tmpFileName := "tmp.json"
	d1 := []byte(input)
	err := ioutil.WriteFile(tmpFileName, d1, 0644)
	if err != nil {
		return err
	}

	args := []string{
		"mim", "content", "add", "--file=tmp.json",
	}

	cmdMim := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(args, " ")))
	if err := cmdMim.Run(); err != nil {
		pterm.Warning.Println("Failed to write manifest to datahub: ", err)
		return err
	}

	// Remove temp file
	err = os.Remove(tmpFileName)
	if err != nil {
		return err
	}

	return nil
}

func diffManifest(previousManifest *Manifest, currentManifest Manifest) []operation {
	var operations []operation

	for key, config := range currentManifest.Manifest {
		var action string
		previous, exist := previousManifest.Manifest[key]
		hasTransform := hasTransform(config.JsonContent)
		requireDataset := requireDataset(config.JsonContent)
		if hasTransform {
			if config.TransformDigest != previous.TransformDigest {
				action = "update"
			}
		}
		if !exist {
			action = "add"
		} else if previous.Digest != config.Digest {
			action = "update"

		}
		if action == "add" || action == "update" {
			op := operation{
				Config:         config,
				ConfigPath:     config.Path,
				Action:         action,
				HasTransform:   hasTransform,
				RequireDataset: requireDataset,
			}
			operations = append(operations, op)
		}
	}

	for key, config := range previousManifest.Manifest {
		_, exist := currentManifest.Manifest[key]
		if !exist {
			op := operation{
				Config:     config,
				ConfigPath: key,
				Action:     "delete",
			}
			operations = append(operations, op)
		}
	}
	return operations
}
