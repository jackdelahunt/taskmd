package main

import (
	"fmt"
	pipepinev1 "github.com/tektoncd/pipeline/pkg/apis/pipeline/v1"
	"log"
	"os"
	"sigs.k8s.io/yaml"
	"strings"
)

type TaskBundle struct {
	path string
	task pipepinev1.Task
}

func main() {
	logError := log.New(os.Stderr, "", 0)
	if len(os.Args) <= 1 {
		logError.Fatal("usage: taskmd [task-files]")
	}

	err := run(logError)
	if err != nil {
		logError.Fatal(err.Error())
	}
}

func run(logError *log.Logger) error {
	paths := os.Args[1:]

	tasks, err := LoadAllTasks(paths)
	if err != nil {
		return err
	}

	err = GenerateMarkdownToDirectory(tasks)
	if err != nil {
		return err
	}

	return nil
}

func LoadAllTasks(paths []string) ([]TaskBundle, error) {
	tasks := make([]TaskBundle, 0)

	for _, path := range paths {
		bytes, err := os.ReadFile(path)
		if err != nil {
			return nil, err
		}

		var task pipepinev1.Task
		err = yaml.Unmarshal(bytes, &task)
		if err != nil {
			return nil, err
		}

		tasks = append(tasks, TaskBundle{
			path: path,
			task: task,
		})
	}

	return tasks, nil
}

func GenerateMarkdownToDirectory(tasks []TaskBundle) error {
	err := os.RemoveAll("taskmd.out")
	if err != nil {
		return err
	}

	err = os.MkdirAll("taskmd.out", os.ModePerm)
	if err != nil {
		return err
	}

	for _, taskBundle := range tasks {
		stringBuilder := strings.Builder{}
		err = generateHeader(&taskBundle, &stringBuilder)
		if err != nil {
			return err
		}

		err = generateDescription(&taskBundle, &stringBuilder)
		if err != nil {
			return err
		}

		err = generateInputs(&taskBundle, &stringBuilder)
		if err != nil {
			return err
		}

		err = generateWorkspaces(&taskBundle, &stringBuilder)
		if err != nil {
			return err
		}

		err = generateResults(&taskBundle, &stringBuilder)
		if err != nil {
			return err
		}

		err = writeGenerateMarkdown(&taskBundle, &stringBuilder)
		if err != nil {
			return err
		}
	}

	return nil
}

func writeGenerateMarkdown(taskBundle *TaskBundle, stringBuilder *strings.Builder) error {
	err := os.MkdirAll(fmt.Sprintf("taskmd.out/%v", taskBundle.task.Name), os.ModePerm)
	if err != nil {
		return err
	}

	readmeFile, err := os.Create(fmt.Sprintf("taskmd.out/%v/README.md", taskBundle.task.Name))
	if err != nil {
		return err
	}

	_, err = readmeFile.WriteString(stringBuilder.String())
	if err != nil {
		return err
	}

	stat, err := os.Stat(taskBundle.path)
	if err != nil {
		return err
	}

	bytes, err := os.ReadFile(taskBundle.path)
	if err != nil {
		return err
	}

	taskFile, err := os.Create(fmt.Sprintf("taskmd.out/%v/%v", taskBundle.task.Name, stat.Name()))
	if err != nil {
		return err
	}

	_, err = taskFile.Write(bytes)
	if err != nil {
		return err
	}

	return nil
}

func generateResults(taskBundle *TaskBundle, stringBuilder *strings.Builder) error {
	_, err := stringBuilder.WriteString("## Results\n")
	if err != nil {
		return err
	}

	for _, result := range taskBundle.task.Spec.Results {
		_, err := stringBuilder.WriteString(fmt.Sprintf("* **%v**: %v\n", result.Name, result.Description))
		if err != nil {
			return err
		}
	}

	return nil
}

func generateWorkspaces(taskBundle *TaskBundle, stringBuilder *strings.Builder) error {
	_, err := stringBuilder.WriteString("## Workspaces\n")
	if err != nil {
		return err
	}

	for _, workspace := range taskBundle.task.Spec.Workspaces {

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

func generateInputs(taskBundle *TaskBundle, stringBuilder *strings.Builder) error {
	_, err := stringBuilder.WriteString("## Parameters\n")
	if err != nil {
		return err
	}

	for _, param := range taskBundle.task.Spec.Params {

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

func generateDescription(taskBundle *TaskBundle, stringBuilder *strings.Builder) error {
	_, err := stringBuilder.WriteString(taskBundle.task.Spec.Description)
	if err != nil {
		return err
	}

	_, err = stringBuilder.WriteString("\n\n")
	if err != nil {
		return err
	}

	return nil
}

func generateHeader(taskBundle *TaskBundle, stringBuilder *strings.Builder) error {
	_, err := stringBuilder.WriteString(fmt.Sprintf("# `%v`\n\n", taskBundle.task.Name))
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
