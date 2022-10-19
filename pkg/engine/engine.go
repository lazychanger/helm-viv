package engine

import (
	"fmt"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/engine"
	"log"
	"os"
	"path"
	"regexp"
	"strings"
)

type Engine struct {
	cfg *Config

	vivFileDirs []string
}

func NewEngine(cfg *Config) *Engine {
	return &Engine{
		cfg: cfg,
	}
}

func (e *Engine) RenderTo(dst string) []string {
	if !path.IsAbs(dst) {
		dst = path.Join(e.cfg.WorkDir, e.cfg.Chart.ChartFullPath(), dst)
	}

	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		log.Panicln(err)
	}

	e.vivFileDirs = []string{dst}

	outputFiles, err := e.eachChart(e.cfg.Chart, "")
	if err != nil {
		log.Println(err)
		return []string{}
	}

	e.cfg.Chart.Templates = append(e.cfg.Chart.Templates, outputFiles...)

	tmpls, err := engine.Render(e.cfg.Chart, e.cfg.Values)
	if err != nil {
		return []string{}
	}

	outputRealFilepath := make([]string, len(outputFiles))

	for i, f := range outputFiles {
		filename := path.Join(e.cfg.Chart.Name(), f.Name)
		realfilepath := path.Join(dst, strings.ReplaceAll(filename, "/", "_"))

		newdata, err := addRootNode(getNode(f.Name), []byte(tmpls[filename]))
		if err != nil {
			log.Println(tmpls[filename])
			panic(errors.Wrap(err, fmt.Sprintf("file: %s", filename)))
		}
		writeFile(realfilepath, newdata)

		outputRealFilepath[i] = realfilepath
	}

	return outputRealFilepath
}

func (e *Engine) RenderToTemp() []string {
	return e.RenderTo("vivTemp")
}

func (e *Engine) eachChart(ch *chart.Chart, node string) ([]*chart.File, error) {

	renderFiles := make([]*chart.File, 0)

	for _, f := range ch.Raw {
		if !strings.HasPrefix(f.Name, "vivs/") || f.Name == "" || len(f.Data) == 0 {
			continue
		}
		f.Name = path.Join(ch.ChartFullPath()[len(e.cfg.Chart.Name()):], f.Name)
		renderFiles = append(renderFiles, f)
	}

	for _, d := range ch.Dependencies() {

		subChartOutputFiles, err := e.eachChart(d, fmt.Sprintf("%s.%s", node, d.Name()))

		if err != nil {
			return renderFiles, errors.Wrap(err, fmt.Sprintf("subchart generate failed. %s", d.Name()))
		}

		renderFiles = append(renderFiles, subChartOutputFiles...)
	}

	return renderFiles, nil
}

func (e *Engine) Clear() {
	if e.vivFileDirs != nil && len(e.vivFileDirs) > 0 {
		for _, dir := range e.vivFileDirs {
			_ = os.RemoveAll(dir)
		}
	}
}

func addRootNode(root string, data []byte) ([]byte, error) {
	current, _ := newTree(nil)
	nodes := strings.Split(root, ".")
	for i := 0; i < len(nodes); i++ {
		if nodes[i] == "" {
			continue
		}
		current = current.CreateChildAndSelect(nodes[i])
	}

	if err := current.UnmarshalWithYAML(data); err != nil {
		return data, err
	}

	return current.Top().MarshalWithYAML()
}

func writeFile(filepath string, data []byte) {
	f, err := os.OpenFile(filepath, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		log.Panic(err)
	}
	defer f.Close()
	_, err = f.Write(data)
	if err != nil {
		log.Panic(err)
	}
}

var partten = "charts/([a-zA-Z]+[a-zA-Z0-9]+)"

func getNode(name string) string {
	r := regexp.MustCompile(partten)

	nodes := make([]string, 1)
	for _, match := range r.FindAllStringSubmatch(name, -1) {
		nodes = append(nodes, match[1])
	}

	return strings.Join(nodes, ".")
}
