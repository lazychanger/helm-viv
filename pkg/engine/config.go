package engine

import (
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chartutil"
)

type Config struct {
	WorkDir string
	Values  chartutil.Values
	Chart   *chart.Chart
}
