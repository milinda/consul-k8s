// Script to move generated CRD yaml into consul-helm and modify it to match
// the expected consul-helm format.
package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"strings"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run ./... <path-to-consul-helm>")
		os.Exit(1)
	}

	helmRepoPath := os.Args[1]
	if !filepath.IsAbs(helmRepoPath) {
		var err error
		// NOTE: Must add ../.. to a relative path because this program is in
		// hack/crds-to-consul-helm.
		helmRepoPath, err = filepath.Abs(filepath.Join("../..", helmRepoPath))
		if err != nil {
			fmt.Printf("Error: %s\n", err)
			os.Exit(1)
		}
	}
	fmt.Printf("Using consul-helm repo path: %s\n", helmRepoPath)

	if err := realMain(helmRepoPath); err != nil {
		fmt.Printf("Error: %s\n", err)
		os.Exit(1)
	}
	os.Exit(0)
}

func realMain(helmPathAbs string) error {
	return filepath.Walk("../../config/crd/bases", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() || filepath.Ext(path) != ".yaml" {
			return nil
		}

		printf("processing %s", filepath.Base(path))

		contentBytes, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}
		contents := string(contentBytes)

		// Strip leading newline.
		contents = strings.TrimPrefix(contents, "\n")

		// Add {{- if .Values.controller.enabled }} {{- end }} wrapper.
		contents = fmt.Sprintf("{{- if .Values.controller.enabled }}\n%s{{- end }}\n", contents)

		// Add labels, this is hacky because we're relying on the line number
		// but it means we don't need to regex or yaml parse.
		splitOnNewlines := strings.Split(contents, "\n")
		labelLines := []string{
			`  labels:`,
			`    app: {{ template "consul.name" . }}`,
			`    chart: {{ template "consul.chart" . }}`,
			`    heritage: {{ .Release.Service }}`,
			`    release: {{ .Release.Name }}`,
			`    component: crd`,
		}
		withLabels := append(splitOnNewlines[0:9], append(labelLines, splitOnNewlines[9:]...)...)
		contents = strings.Join(withLabels, "\n")

		// Construct the destination filename.
		filenameSplit := strings.Split(info.Name(), "_")
		crdName := filenameSplit[1]
		destinationPath := filepath.Join(helmPathAbs, "templates", fmt.Sprintf("crd-%s", crdName))

		// Write it.
		printf("writing to %s", destinationPath)
		return ioutil.WriteFile(destinationPath, []byte(contents), 0644)
	})
}

func printf(format string, args ...interface{}) {
	fmt.Println(fmt.Sprintf(format, args...))
}
