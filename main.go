package main

import (
	"encoding/json"
	"fmt"
	pipepinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"log"
	"os"
	"sigs.k8s.io/kustomize/api/krusty"
	"sigs.k8s.io/kustomize/api/resmap"
	"sigs.k8s.io/kustomize/api/resource"
	"sigs.k8s.io/kustomize/kyaml/filesys"
	"strings"
)

func main() {
	logError := log.New(os.Stderr, "", 0)
	if len(os.Args) != 2 {
		logError.Fatal("usage: taskmd [kustomize-dir]")
	}

	err := run(logError)
	if err != nil {
		logError.Fatal(err.Error())
	}
}

func run(logError *log.Logger) error {
	kustomizePath := os.Args[1]

	resourceMap, err := KustomizeBuild(kustomizePath)
	if err != nil {
		logError.Fatal(err.Error())
	}

	tasks, err := GetAllTasksFromResourceMap(resourceMap)
	if err != nil {
		logError.Fatal(err.Error())
	}

	err = GenerateMarkdownToDirectory(tasks)
	if err != nil {
		return err
	}

	return nil
}

func KustomizeBuild(path string) (resmap.ResMap, error) {
	options := krusty.MakeDefaultOptions()
	k := krusty.MakeKustomizer(options)
	fs := filesys.FileSystemOrOnDisk{
		FileSystem: nil,
	}

	resourceMap, err := k.Run(fs, path)
	if err != nil {
		return nil, err
	}

	return resourceMap, nil
}

func GetAllTasksFromResourceMap(resourceMap resmap.ResMap) ([]pipepinev1.Task, error) {
	tasks := make([]pipepinev1.Task, 0)

	for _, res := range resourceMap.Resources() {
		if res.GetKind() != "Task" {
			return tasks, nil
		}

		var task pipepinev1.Task

		err := ResourceToType(res, &task)
		if err != nil {
			return tasks, err
		}

		tasks = append(tasks, task)
	}

	return tasks, nil
}

func ResourceToType[T any](resource *resource.Resource, t *T) error {
	bytes, err := resource.MarshalJSON()
	if err != nil {
		return err
	}

	err = json.Unmarshal(bytes, &t)
	if err != nil {
		return err
	}

	return nil
}

func GenerateMarkdownToDirectory(tasks []pipepinev1.Task) error {
	err := os.MkdirAll("taskmd.out", os.ModePerm)
	if err != nil {
		return err
	}

	for _, task := range tasks {
		stringBuilder := strings.Builder{}
		err = generateHeader(&task, &stringBuilder)
		if err != nil {
			return err
		}

		err = generateDescription(&task, &stringBuilder)
		if err != nil {
			return err
		}

		err = generateInputs(&task, &stringBuilder)
		if err != nil {
			return err
		}

		err = generateWorkspaces(&task, &stringBuilder)
		if err != nil {
			return err
		}

		err = generateResults(&task, &stringBuilder)
		if err != nil {
			return err
		}

		err = writeGenerateMarkdown(&task, &stringBuilder)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeGenerateMarkdown(task *pipepinev1.Task, stringBuilder *strings.Builder) error {
	file, err := os.Create(fmt.Sprintf("taskmd.out/%v.md", task.Name))
	if err != nil {
		return err
	}

	_, err = file.WriteString(stringBuilder.String())
	if err != nil {
		return err
	}

	return nil
}

func generateResults(task *pipepinev1.Task, stringBuilder *strings.Builder) error {
	_, err := stringBuilder.WriteString("## Results\n")
	if err != nil {
		return err
	}

	for _, result := range task.Spec.Results {
		_, err := stringBuilder.WriteString(fmt.Sprintf("* **%v**: %v\n", result.Name, result.Description))
		if err != nil {
			return err
		}
	}

	return nil
}

func generateWorkspaces(task *pipepinev1.Task, stringBuilder *strings.Builder) error {
	_, err := stringBuilder.WriteString("## Workspaces\n")
	if err != nil {
		return err
	}

	for _, workspace := range task.Spec.Workspaces {

		var formatString string
		if workspace.Optional {
			formatString = "* **%v** (optional): %v\n"
		} else {
			formatString = "* **%v**: %v\n"
		}

		_, err := stringBuilder.WriteString(fmt.Sprintf(formatString, workspace.Name, workspace.Description))
		if err != nil {
			return err
		}
	}

	_, err = stringBuilder.WriteString("\n")
	if err != nil {
		return err
	}

	return nil
}

func generateInputs(task *pipepinev1.Task, stringBuilder *strings.Builder) error {
	_, err := stringBuilder.WriteString("## Parameters\n")
	if err != nil {
		return err
	}

	for _, param := range task.Spec.Params {

		if param.Default != nil {
			paramString, err := stringifyParam(param.Default)
			if err != nil {
				return err
			}

			_, err = stringBuilder.WriteString(fmt.Sprintf("* **%v**: %v `(Default: %v)`\n", param.Name, param.Description, paramString))
			if err != nil {
				return err
			}
		} else {
			_, err := stringBuilder.WriteString(fmt.Sprintf("* **%v**: %v\n", param.Name, param.Description))
			if err != nil {
				return err
			}
		}

	}

	_, err = stringBuilder.WriteString("\n")
	if err != nil {
		return err
	}

	return nil
}

func generateDescription(task *pipepinev1.Task, stringBuilder *strings.Builder) error {
	_, err := stringBuilder.WriteString(task.Spec.Description)
	if err != nil {
		return err
	}

	_, err = stringBuilder.WriteString("\n\n")
	if err != nil {
		return err
	}

	return nil
}

func generateHeader(task *pipepinev1.Task, stringBuilder *strings.Builder) error {
	_, err := stringBuilder.WriteString(fmt.Sprintf("# `%v`\n\n", task.Name))
	if err != nil {
		return err
	}

	return nil
}

func stringifyParam(param *pipepinev1.ParamValue) (string, error) {
	if param == nil {
		return "", fmt.Errorf("cannot stringify nil param value")
	}

	switch param.Type {
	case "string":
		return param.StringVal, nil
	case "array":
		var stringBuilder strings.Builder
		_, err := stringBuilder.WriteString("[")
		if err != nil {
			return "", err
		}

		var formatString string

		for i, s := range param.ArrayVal {
			if i == len(param.ArrayVal)-1 {
				formatString = "%v, "
			} else {
				formatString = "%v"
			}

			_, err := stringBuilder.WriteString(fmt.Sprintf(formatString, s))
			if err != nil {
				return "", err
			}
		}

		_, err = stringBuilder.WriteString("]")
		if err != nil {
			return "", err
		}
	case "object":
		return "{}", nil
	}

	return "", nil
}
