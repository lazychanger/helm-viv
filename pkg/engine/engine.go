package engine

import (
	"bytes"
	"fmt"
	"github.com/lazychanger/helm-variable-in-values/pkg/utils"
	"github.com/pkg/errors"
	"helm.sh/helm/v3/pkg/chart"
	"log"
	"os"
	"path"
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

	e.vivFileDirs = []string{dst}

	defer func() {
		if err := recover(); err != nil {
			log.Println(err)
		}
	}()

	outputFiles, err := e.eachChart(dst, e.cfg.Chart, "")
	if err != nil {
		log.Println(err)
		return []string{}
	}
	return outputFiles
}

func (e *Engine) RenderToTemp() []string {
	return e.RenderTo("vivTemp")
}

func (e *Engine) eachChart(dst string, chart *chart.Chart, node string) ([]string, error) {
	outputFiles := make([]string, 0)

	for _, file := range chart.Raw {
		if !strings.HasPrefix(file.Name, "vivs") {
			continue
		}

		vivfilePath := path.Join(chart.ChartFullPath()[len(e.cfg.Chart.ChartFullPath()):], file.Name)
		vivfileFullpath := path.Join(dst, strings.ReplaceAll(vivfilePath, "/", "_"))

		if err := e.render(vivfileFullpath, node, e.cfg.Values, file.Data); err != nil {
			return outputFiles, errors.Wrap(err, fmt.Sprintf("vivfile generate failed. %s", vivfilePath))
		}

		outputFiles = append(outputFiles, vivfileFullpath)
	}

	for _, d := range chart.Dependencies() {

		subChartOutputFiles, err := e.eachChart(dst, d, fmt.Sprintf("%s.%s", node, d.Name()))

		if err != nil {
			return outputFiles, errors.Wrap(err, fmt.Sprintf("subchart generate failed. %s", d.Name()))
		}

		outputFiles = append(outputFiles, subChartOutputFiles...)
	}

	return outputFiles, nil
}

func (e *Engine) render(output string, root string, values map[string]interface{}, template []byte) error {

	if err := os.MkdirAll(path.Dir(output), os.ModePerm); err != nil {
		return err
	}

	f, err := os.OpenFile(output, os.O_RDWR|os.O_CREATE|os.O_TRUNC, 0777)
	if err != nil {
		return err
	}

	buf := &bytes.Buffer{}

	if err := utils.Tmpl(buf, string(template), values); err != nil {
		return err
	}

	newData, err := addRootNode(root, buf.Bytes())
	if err != nil {
		return err
	}

	if _, err := f.Write(newData); err != nil {
		return err
	}

	return nil
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
