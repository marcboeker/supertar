package main

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"io/ioutil"
	"os"
	"path/filepath"
	"regexp"
)

var (
	regex = regexp.MustCompile(`[^a-zA-Z0-9]*`)
)

func main() {
	if len(os.Args) < 3 {
		panic("please specify src and dest")
	}

	src := os.Args[1]
	dest := os.Args[2]

	fileMap := map[string]string{}

	buf := bytes.NewBuffer(nil)
	buf.WriteString("package server\n\nimport \"encoding/base64\"\n\n")

	walkFunc := func(path string, info os.FileInfo, err error) error {
		if !info.Mode().IsRegular() {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return err
		}

		filename := regex.ReplaceAllString(path, "")
		buf.WriteString(
			fmt.Sprintf("var _asset%s, _ = base64.StdEncoding.DecodeString(\"%s\")\n", filename, base64.StdEncoding.EncodeToString(data)),
		)

		fileMap[path] = filename

		return nil
	}

	filepath.Walk(src, walkFunc)

	buf.WriteString("var fileMap = map[string][]byte{\n")
	for k, v := range fileMap {
		fmt.Fprintf(buf, "\t\"/%s\": _asset%s,\n", k, v)
	}
	buf.WriteString("}")

	ioutil.WriteFile(dest, buf.Bytes(), 0700)
}
