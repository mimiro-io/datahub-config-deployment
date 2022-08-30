package app

import (
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

type MimConfig struct {
	Env        *environment.Environment
	CmdOutputs []string
}

type Entity struct {
	Id         string                 `json:"id"`
	Recorded   int64                  `json:"recorded,omitempty"`
	Deleted    bool                   `json:"deleted,omitempty"`
	Refs       map[string]interface{} `json:"refs,omitempty"`
	Props      map[string]interface{} `json:"props,omitempty"`
	Namespaces map[string]interface{} `json:"namespaces,omitempty"`
}

type DatasetResponse struct {
	Items            int      `json:"items"`
	Name             string   `json:"name"`
	PublicNamespaces []string `json:"publicNamespaces"`
}

func NewMim(env *environment.Environment) *MimConfig {
	return &MimConfig{Env: env}
}

func (m *MimConfig) MimCommand(cmd []string) ([]byte, error) {
	cmdExec := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(cmd, " ")))
	return cmdExec.CombinedOutput()
}

func (m *MimConfig) MimDatasetStore(datasetName string, payload []byte) ([]byte, error) {
	tmpFilename := "tmp_entity"
	cmd := []string{"mim", "dataset", "store", datasetName, "-f", tmpFilename}
	err := ioutil.WriteFile(tmpFilename, payload, 0644)
	if err != nil {
		pterm.Error.Println("Failed to write entity to temp file")
		return nil, err
	}
	m.CmdOutputs = append(m.CmdOutputs, strings.Join(cmd, " "))
	utils.LogCommand(cmd, m.Env.LogFormat, "")
	var output []byte
	if !m.Env.DryRun {
		output, err = m.MimCommand(cmd)
	}

	err2 := os.Remove(tmpFilename)
	if err2 != nil {
		pterm.Error.Println("Failed to remove tmp file for core entity")
		return nil, err2
	}

	return output, err
}

func (m *MimConfig) MimDatasetDelete(datasetName string) error {
	cmd := []string{"mim", "dataset", "delete", datasetName, "-C=false"}
	m.CmdOutputs = append(m.CmdOutputs, strings.Join(cmd, " "))
	utils.LogCommand(cmd, m.Env.LogFormat, "")
	var output []byte
	var err error
	if !m.Env.DryRun {
		output, err = m.MimCommand(cmd)
	}
	if err != nil {
		pterm.Error.Printf("Failed to delete dataset '%s':\n%s\n", datasetName, string(output))
		return err
	}
	return nil
}

func (m *MimConfig) MimDatasetGet(datasetName string) (DatasetResponse, error) {
	cmd := []string{"mim", "dataset", "get", datasetName, "--json"}
	output, err := m.MimCommand(cmd)
	var dataset DatasetResponse
	if err != nil {
		pterm.Error.Printf("Failed to get dataset '%s' from datahub: %s\n", datasetName, string(output))
		return dataset, err
	}
	err = json.Unmarshal(output, &dataset)
	if err != nil {
		pterm.Error.Println("Failed to unmarshal dataset response: ", string(output))
		return dataset, err
	}
	return dataset, nil
}

func (m *MimConfig) MimDatasetCreate(datasetName string, publicNamespaces []string) ([]byte, error) {
	cmd := []string{"mim", "dataset", "create", datasetName}
	if len(publicNamespaces) > 0 {
		cmd = []string{"mim", "dataset", "create", datasetName, "--publicNamespaces", fmt.Sprintf("'%s'", strings.Join(publicNamespaces, "','"))}
	}
	m.CmdOutputs = append(m.CmdOutputs, strings.Join(cmd, " "))
	utils.LogCommand(cmd, m.Env.LogFormat, "")
	var output []byte
	var err error
	if !m.Env.DryRun {
		output, err = m.MimCommand(cmd)
	}
	if err != nil {
		pterm.Error.Println("Failed to create dataset in datahub: ", string(output))
		return output, err
	}
	return output, nil

}

func (m *MimConfig) MimDatasetEntities(datasetName string) ([]Entity, error) {
	cmd := []string{"mim", "dataset", "entities", datasetName, "--json", "--limit=40000"}
	output, err := m.MimCommand(cmd)
	if err != nil {
		pterm.Error.Println("Failed to get dataset entities from datahub: ", string(output))
		return nil, err
	}
	var entities []Entity
	err = json.Unmarshal(output, &entities)
	if err != nil {
		pterm.Error.Println("Failed to unmarshal dataset entities response: ", string(output))
	}
	return entities, err

}

func (m *MimConfig) MimJobAdd(fileName string, transform string) ([]byte, error) {
	cmd := []string{"mim", "job", "add", "-f", fileName}
	if transform != "" {
		cmd = []string{"mim", "job", "add", "-f", fileName, "-t", transform}
	}
	var output []byte
	var err error
	m.CmdOutputs = append(m.CmdOutputs, strings.Join(cmd, " "))
	utils.LogCommand(cmd, m.Env.LogFormat, "")
	if !m.Env.DryRun {
		output, err = m.MimCommand(cmd)
	}
	if err != nil {
		pterm.Error.Println("Failed to write job to datahub: ", string(output))
	}
	return output, err
}

func (m *MimConfig) MimJobDelete(jobId string) ([]byte, error) {
	cmd := []string{"mim", "job", "delete", jobId, "-C=false"}
	m.CmdOutputs = append(m.CmdOutputs, strings.Join(cmd, " "))
	utils.LogCommand(cmd, m.Env.LogFormat, "")
	var output []byte
	var err error
	if !m.Env.DryRun {
		output, err = m.MimCommand(cmd)
	}
	if err != nil {
		pterm.Error.Printf("Failed to delete job '%s':\n%s\n", jobId, string(output))
	}
	return output, err
}

func (m *MimConfig) MimContentAdd(fileName string) ([]byte, error) {
	cmd := []string{"mim", "content", "add", "-f", fileName}
	var output []byte
	var err error
	m.CmdOutputs = append(m.CmdOutputs, strings.Join(cmd, " "))
	utils.LogCommand(cmd, m.Env.LogFormat, "")
	if !m.Env.DryRun {
		output, err = m.MimCommand(cmd)
	}
	if err != nil {
		pterm.Error.Println("Failed to write content to datahub: ", string(output))
	}
	return output, err
}

func (m *MimConfig) MimContentDelete(contentId string) ([]byte, error) {
	cmd := []string{"mim", "content", "delete", contentId, "-C=false"}
	m.CmdOutputs = append(m.CmdOutputs, strings.Join(cmd, " "))
	utils.LogCommand(cmd, m.Env.LogFormat, "")
	var output []byte
	var err error
	if !m.Env.DryRun {
		output, err = m.MimCommand(cmd)
	}
	if err != nil {
		pterm.Error.Printf("Failed to delete content '%s':\n%s\n", contentId, string(output))
	}
	return output, err
}
