package main

import (
	"io"
	"io/ioutil"
	"os"
	"path"
	"strings"
)

const templatesDir = "./templates"

func main() {
	files, _ := ioutil.ReadDir(templatesDir)

	out, _ := os.Create(path.Join(templatesDir, "templates.go"))
	out.Write([]byte("package templates\n\nvar Get = map[string]string{\n"))
	for _, fileInfo := range files {
		if strings.HasSuffix(fileInfo.Name(), ".html") {
			out.Write([]byte(strings.TrimSuffix(fileInfo.Name(), ".html") + ": `"))
			file, _ := os.Open(path.Join(templatesDir, fileInfo.Name()))
			io.Copy(out, file)
			out.Write([]byte("`,\n"))
		}
	}
	out.Write([]byte("}\n"))
}
