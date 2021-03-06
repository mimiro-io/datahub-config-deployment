package app

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/mimiro-io/datahub-config-deployment/internal/app/environment"
	"github.com/mimiro-io/datahub-config-deployment/internal/app/templating"
	"github.com/mimiro-io/datahub-config-deployment/internal/utils"
	"github.com/pterm/pterm"
	"github.com/spf13/cobra"
	"github.com/tidwall/pretty"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

type App struct {
	Env *environment.Environment
	T   *templating.Templating
	M   *ManifestConfig
}

func NewApp(cmd *cobra.Command, args []string) (*App, error) {
	// lets validate and set up our environment
	enableJsonOut, _ := cmd.Flags().GetBool("json")
	if enableJsonOut {
		pterm.DisableOutput()
	}

	datahub, _ := cmd.Flags().GetString("datahub")
	if datahub == "" && len(args) > 0 {
		datahub = args[0]
	}
	if datahub == "" {
		return nil, errors.New("URL for DataHub is missing")
	}

	token, _ := cmd.Flags().GetString("token")
	stdIn, _ := cmd.Flags().GetBool("token-stdin")
	if token == "" && stdIn { // token is missing, and should be expected from stdin
		themBytes, err := utils.ReadStdIn()
		if err != nil {
			return nil, err
		}
		if themBytes != nil {
			token = string(themBytes)
		}
	}
	if token == "" {
		pterm.Warning.Println("No token provided in param or StdIn, assuming no token is needed")
	}

	path, _ := cmd.Flags().GetString("path")
	ignorePath, _ := cmd.Flags().GetStringArray("ignorePath")
	env, _ := cmd.Flags().GetString("env")
	err := verifyEnv(path, env)
	if err != nil {
		return nil, err
	}

	dryRun, _ := cmd.Flags().GetBool("dry-run")
	manifest, _ := cmd.Flags().GetBool("create-manifest")
	abort, _ := cmd.Flags().GetBool("abort-missing-secret")
	enableManifest, _ := cmd.Flags().GetBool("display-manifest")
	logFormat, _ := cmd.Flags().GetString("log-format")

	e := &environment.Environment{
		MimServer:               datahub,
		Token:                   token,
		RootPath:                path,
		IgnorePath:              ignorePath,
		EnvironmentFile:         env,
		DryRun:                  dryRun,
		CreateManifestIfMissing: manifest,
		AbortOnMissingSecret:    abort,
		EnableManifest:          enableManifest,
		EnableJsonOut:           enableJsonOut,
		LogFormat:               logFormat,
	}

	return &App{
		Env: e,
		T:   templating.NewTemplating(),
		M:   NewManifest(e),
	}, nil
}

// verifyEnv makes sure the config path and the env path is correct
func verifyEnv(path string, env string) error {
	if path == "" {
		return errors.New("path is missing")
	}
	if env == "" {
		return errors.New("path to env variables is missing")
	}

	err := utils.VerifyPath(path)
	if err != nil {
		return err
	}
	err = utils.VerifyPath(env)
	if err != nil {
		return err
	}
	return nil
}

func (app *App) Run() error {
	files, err := app.Env.GetConfigFiles()
	if err != nil {
		return err
	}

	variables, err := app.Env.GetEnvironmentVariables()
	if err != nil {
		return err
	}

	err = app.loginMimCli()
	if err != nil {
		return err
	}
	return app.doStuff(files, variables)
}

func (app *App) loginMimCli() error {
	args := []string{
		"mim", "login", "add", "--alias=deploy", fmt.Sprintf("--server=%s", app.Env.MimServer),
	}
	utils.LogCommand(args, "default", "")
	if app.Env.Token != "" {
		args = append(args, fmt.Sprintf("--type token --token=%s", app.Env.Token))
	}

	cmdMim1 := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(args, " ")))
	output, err := cmdMim1.CombinedOutput()
	if err != nil {
		pterm.Error.Println("Failed to add login alias: ", string(output), err.Error())
		return err
	}

	cmdMim2 := exec.Command("/bin/bash", "-c", "mim login deploy")
	output, err = cmdMim2.CombinedOutput()
	if err != nil {
		pterm.Error.Println("Failed to login mim: ", string(output), err.Error())
		return err
	}
	return nil
}

func (app *App) doStuff(files []string, variables map[string]interface{}) error {
	var fileConfigs map[string]config
	fileConfigs = make(map[string]config)

	for i := 0; i < len(files); i++ {
		pterm.Info.Printf(" > Processing %s\n", files[i])
		rawJson, _ := utils.ReadFile(files[i])

		updatedJson, err := app.T.ReplaceVariableLogic(rawJson, app.Env.RootPath)
		if err != nil {
			return err
		}
		updatedJson, err = app.T.ReplaceVariables(updatedJson, variables)
		if err != nil {
			return err
		}
		jsonContent, err := utils.ReadJson(updatedJson)
		if err != nil {
			return err
		}
		fileType, exist := jsonContent["type"].(string)
		if !exist {
			continue
		}
		if fileType == "job" || fileType == "content" {

			var transformDigest string
			if hasTransform(jsonContent) {
				transformPath := jsonContent["transform"].(map[string]interface{})["Path"].(string)
				transformFullPath := filepath.Join(app.Env.RootPath, "transforms", transformPath)
				transformDigest, err = getTransformDigest(transformFullPath)
				if err != nil {
					return err
				}
			}

			// Get relative path
			relPath, err := filepath.Rel(app.Env.RootPath, files[i])
			if err != nil {
				pterm.Error.Println("Failed to determine relative path for ", files[i])
				return err
			}

			// create md5 digest for each file
			digest, err := createDigest(jsonContent)
			if err != nil {
				return err
			}
			configType := app.Env.GetConfigType(files[i])
			jsonId, exist := jsonContent["id"].(string)
			if !exist {
				jsonId = ""
			}
			jsonTitle, exist := jsonContent["title"].(string)
			if !exist {
				jsonTitle = ""
			}
			contentInstance := config{
				Path:            relPath,
				JsonContent:     jsonContent,
				Digest:          digest,
				Type:            configType,
				Id:              jsonId,
				Title:           jsonTitle,
				TransformDigest: transformDigest,
			}
			fileConfigs[jsonId] = contentInstance
			//break
		}
	}

	currentManifest := Manifest{
		Id:       "DatahubConfigManifest",
		Manifest: fileConfigs,
	}

	previousManifest, err := app.M.getManifestFromDatahub()
	if err != nil {
		if app.Env.CreateManifestIfMissing {
			pterm.Warning.Println("Unable to read manifest from datahub. Assuming first run.")
			previousManifest = new(Manifest) // To avoid empty pointer in diff
		} else {
			return nil
		}
	}

	operations := diffManifest(previousManifest, currentManifest)
	currentManifest.Operations = operations
	err = app.executeOperations(currentManifest)
	if err != nil {
		return err
	}

	jsonManifest, err := json.Marshal(currentManifest)
	if err != nil {
		panic(err)
	}

	if !app.Env.DryRun {
		pterm.Info.Println("Writing manifest to datahub.")
		err = app.M.writeManifestToDatahub(string(jsonManifest))
		if err != nil {
			return err
		}
	}

	if app.Env.EnableManifest {
		if app.Env.EnableJsonOut {
			fmt.Println(string(jsonManifest))
		} else {
			pterm.Println()
			pterm.DefaultParagraph.Println("The following manifest will be stored in the datahub when DRY_RUN is disabled:")
			f := pretty.Pretty(jsonManifest)
			result := pretty.Color(f, nil)
			pterm.Println(string(result))
		}
	}
	if app.Env.DryRun {
		pterm.Success.Println("Dry run deployment finished. To execute the commands on the datahub, set flag --dry-run=false")
	} else {
		pterm.Success.Println("Deployment finished.")
	}

	return nil
}

func (app *App) executeOperations(manifest Manifest) error {
	operations := manifest.Operations
	var cmdOutputs []string

	pterm.Println()
	if app.Env.DryRun {
		message := "Dry run enabled. Showing commands that would be executed without dry run enabled."
		utils.LogPlain(message, app.Env.LogFormat)
		cmdOutputs = append(cmdOutputs, message)
	} else {
		message := "The following commands will be written to datahub using the mim cli:"
		utils.LogPlain(message, app.Env.LogFormat)
		cmdOutputs = append(cmdOutputs, message)
	}
	for _, operation := range operations {
		tmpFileName := "tmp_" + operation.Config.Id + ".json"
		jsonContent, err := json.Marshal(operation.Config.JsonContent)

		if operation.Action != "delete" {
			if err != nil {
				fmt.Println("Failed to marshal config for " + operation.Config.Id + " to json before writing to temp file.")
				return err
			}
			d1 := jsonContent
			err = ioutil.WriteFile(tmpFileName, d1, 0644)
			if err != nil {
				fmt.Println("Failed to write config to temp file")
				return err
			}
		}

		if operation.Config.Type == "content" {
			var args []string
			if operation.Action == "delete" {
				args = []string{"mim", "content", "delete", operation.Config.Id, "-C=false"}
			} else {
				args = []string{"mim", "content", "add", "-f", tmpFileName}
			}

			utils.LogCommand(args, app.Env.LogFormat, "")
			cmdOutputs = append(cmdOutputs, strings.Join(args, " "))

			if !app.Env.DryRun {
				executeCmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(args, " ")))
				output, err := executeCmd.CombinedOutput()
				if err != nil {
					errBody := utils.ErrorDetails{
						File:    operation.ConfigPath,
						Line:    0,
						Col:     0,
						Message: fmt.Sprintf("Failed to write content '%s' to datahub: %s\n", operation.Config.Id, string(output)),
					}
					utils.LogError(errBody, app.Env.LogFormat)
					//pterm.Error.Printf("Failed to write content '%s' to datahub: %s\n", operation.Config.Id, string(output))
					return err
				}
			}
		} else if operation.Config.Type == "job" {
			var jobCmd []string
			if operation.Action == "delete" {
				jobCmd = []string{"mim", "job", "delete", operation.Config.Id, "-C=false"}
			} else {
				// Will handle both add and update
				if operation.HasTransform {
					transformPath := operation.Config.JsonContent["transform"].(map[string]interface{})["Path"].(string)
					transformFullPath := filepath.Join(app.Env.RootPath, "transforms", transformPath)
					jobCmd = []string{"mim", "job", "add", "-f", tmpFileName, "-t", transformFullPath}
				} else {
					jobCmd = []string{"mim", "job", "add", "-f", tmpFileName}
				}
			}
			utils.LogCommand(jobCmd, app.Env.LogFormat, operation.Config.Title)
			cmdOutputs = append(cmdOutputs, strings.Join(jobCmd, " "))

			executeCmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(jobCmd, " ")))
			if !app.Env.DryRun {
				output, err := executeCmd.CombinedOutput()
				if err != nil {
					errBody := utils.ErrorDetails{
						File:    operation.ConfigPath,
						Line:    0,
						Col:     0,
						Message: fmt.Sprintf("Failed to write job to datahub: \n%s\n%s\n", string(output), string(jsonContent)),
					}
					utils.LogError(errBody, app.Env.LogFormat)
					//pterm.Error.Printf("Failed to write job to datahub: %s\n%s\n", string(output), string(jsonContent))
					return err
				}
			}

		}
		if operation.Action != "delete" {
			// Remove temp file
			err := os.Remove(tmpFileName)
			if err != nil {
				pterm.Error.Println("Failed to remove tmp file for ", operation.Config.Id)
				return err
			}
		}
		// Check if required dataset need to be created
		if operation.Config.Type == "job" && operation.Action != "delete" && operation.RequireDataset {
			// Check if dataset exist already
			sinkDataset := determineSinkDataset(operation.Config.JsonContent)
			datasetCmd := []string{"mim", "dataset", "get", sinkDataset, "--json"}
			getDatasetCmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(datasetCmd, " ")))
			err := getDatasetCmd.Run()
			if err != nil {
				// Failed to get dataset. Proceeding to create on datahub.
				pterm.Warning.Printf("Required dataset not available on datahub. Creating dataset '%s' for job '%s'.\n", sinkDataset, operation.Config.Id)
				datasetCmd := []string{"mim", "dataset", "create", sinkDataset}
				createDatasetCmd := exec.Command("/bin/bash", "-c", fmt.Sprintf("%s", strings.Join(datasetCmd, " ")))
				utils.LogCommand(datasetCmd, app.Env.LogFormat, "")
				cmdOutputs = append(cmdOutputs, strings.Join(datasetCmd, " "))

				if !app.Env.DryRun {
					output, err := createDatasetCmd.CombinedOutput()
					if err != nil {
						pterm.Error.Println("Failed to create dataset in datahub: ", string(output))
						return err
					}
				}
			}
		}

	}
	if app.Env.LogFormat == "github" {
		pterm.DefaultBasicText.Println("::set-output name=dry_run_output::", strings.Join(cmdOutputs, "%0A* "))
	}
	return nil
}
