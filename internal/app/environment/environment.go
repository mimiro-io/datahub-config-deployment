package environment

import (
	"github.com/mimiro-io/datahub-config-deployment/internal/utils"
	"github.com/pterm/pterm"
	"os"
	"path/filepath"
	"strings"
)

type Environment struct {
	MimServer               string
	Token                   string
	RootPath                string
	IgnorePath              []string
	EnvironmentFile         string
	DryRun                  bool
	CreateManifestIfMissing bool
	AbortOnMissingSecret    bool
	EnableJsonOut           bool
	EnableManifest          bool
	LogFormat               string
}

func (env *Environment) GetConfigFiles() ([]string, error) {
	pterm.Info.Printf("Reading files from %s\n", env.RootPath)
	var files []string
	err := filepath.Walk(env.RootPath, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() && contains(env.IgnorePath, path) {
			pterm.Info.Println("ignoring path ", path)
			return filepath.SkipDir
		}
		if info.IsDir() {
			return nil
		}
		//this check is too strict for transforms
		if filepath.Ext(path) != ".json" {
			return nil
		}
		files = append(files, path)
		return nil
	})
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (env *Environment) GetEnvironmentVariables() (map[string]interface{}, error) {
	pterm.Info.Printf("Reading Env file %s\n", env.EnvironmentFile)
	vars, err := utils.ReadJsonFile(env.EnvironmentFile)
	if err != nil {
		pterm.Error.Println(err)
		return nil, err
	}
	return vars, nil
}

func (env *Environment) GetConfigType(path string) string {

	relPath, _ := filepath.Rel(env.RootPath, path)
	typeDir := strings.Split(relPath, string(os.PathSeparator))
	switch typeDir[0] {
	case "jobs":
		return "job"
	case "contents":
		return "content"
	case "transforms":
		return "transform"
	case "dataset":
		return "dataset"
	default:
		return "unknown"

	}
}

func contains(elems []string, v string) bool {
	for _, s := range elems {
		if v == s {
			return true
		}
	}
	return false
}
