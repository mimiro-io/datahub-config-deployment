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

func NewMim(env *environment.Environment) *MimConfig {
	return &MimConfig{Env: env}
}

func (m *MimConfig) MimCommand(cmd []string) ([]byte, error) {
	cmdExec := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(cmd, " ")))

	if m.Env.DryRun {
		return nil, nil
	}

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
	output, err := m.MimCommand(cmd)
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
	output, err := m.MimCommand(cmd)
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
	json.Unmarshal(output, dataset)
	return dataset, nil
}

func (m *MimConfig) MimDatasetCreate(datasetName string, publicNamespaces []string) error {
	cmd := []string{"mim", "dataset", "create", datasetName}
	if len(publicNamespaces) > 0 {
		cmd = []string{"mim", "dataset", "create", datasetName, "--publicNamespaces", fmt.Sprintf("'%s'", strings.Join(publicNamespaces, "','"))}
	}
	m.CmdOutputs = append(m.CmdOutputs, strings.Join(cmd, " "))
	utils.LogCommand(cmd, m.Env.LogFormat, "")
	output, err := m.MimCommand(cmd)
	if err != nil {
		pterm.Error.Println("Failed to create dataset in datahub: ", string(output))
		return err
	}
	return nil

}
