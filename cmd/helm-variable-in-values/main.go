package main

import (
	"context"
	"fmt"
	"github.com/lazychanger/helm-variable-in-values/cmd/helm-variable-in-values/utils"
	"github.com/lazychanger/helm-variable-in-values/common"
	vivEngine "github.com/lazychanger/helm-variable-in-values/pkg/engine"
	pkgUtils "github.com/lazychanger/helm-variable-in-values/pkg/utils"
	"github.com/pkg/errors"
	"github.com/spf13/cobra"
	"helm.sh/helm/v3/pkg/registry"

	"helm.sh/helm/v3/pkg/action"
	"helm.sh/helm/v3/pkg/chart"
	"helm.sh/helm/v3/pkg/chart/loader"
	"helm.sh/helm/v3/pkg/chartutil"
	"helm.sh/helm/v3/pkg/cli"
	"helm.sh/helm/v3/pkg/cli/values"
	"helm.sh/helm/v3/pkg/downloader"
	"helm.sh/helm/v3/pkg/getter"
	kubefake "helm.sh/helm/v3/pkg/kube/fake"
	"helm.sh/helm/v3/pkg/release"
	"helm.sh/helm/v3/pkg/storage/driver"

	"io"
	"io/ioutil"
	"k8s.io/client-go/discovery"
	"log"
	"os"
	"os/exec"
	"os/signal"
	"sigs.k8s.io/yaml"
	"strings"
	"syscall"
)

var (
	usage = `Helm plugin to use variable in values

Examples:
  $ helm viv install releaseName repo/chart -n namespace   
  $ helm viv upgrade releaseName repo/chart -n namespace  
`
	settings     = cli.New()
	cliFlags     = new(utils.Flags)
	actionConfig = new(action.Configuration)
	version      = common.GetVersion()
	helmbin      = "helm"
)

func init() {
	log.SetFlags(log.Lshortfile)

	cliFlags = utils.ParseFlags(os.Args)
	settings.Debug = utils.BoolDefaultValue(cliFlags.GetBool("debug"), settings.Debug)
	settings.SetNamespace(utils.StringDefaultValue(cliFlags.GetString("n", "namespace"), settings.Namespace()))
	settings.KubeConfig = utils.StringDefaultValue(cliFlags.GetString("kubeconfig"), settings.KubeConfig)
	settings.KubeContext = utils.StringDefaultValue(cliFlags.GetString("kube-context"), settings.KubeContext)
	settings.KubeToken = utils.StringDefaultValue(cliFlags.GetString("kube-token"), settings.KubeToken)
	settings.KubeAsUser = utils.StringDefaultValue(cliFlags.GetString("kube-as-user"), settings.KubeAsUser)
	settings.KubeAsGroups = utils.StringSliceDefaultValue(cliFlags.GetStringSlice("kube-as-groups"), settings.KubeAsGroups)
	settings.KubeAPIServer = utils.StringDefaultValue(cliFlags.GetString("kube-apiserver"), settings.KubeAPIServer)
	settings.KubeCaFile = utils.StringDefaultValue(cliFlags.GetString("kube-ca-file"), settings.KubeCaFile)
	settings.KubeTLSServerName = utils.StringDefaultValue(cliFlags.GetString("kube-tls-server-name"), settings.KubeTLSServerName)
	settings.KubeInsecureSkipTLSVerify = utils.BoolDefaultValue(cliFlags.GetBool("kube-insecure-skip-tls-verify"), settings.KubeInsecureSkipTLSVerify)
	settings.RegistryConfig = utils.StringDefaultValue(cliFlags.GetString("registry-config"), settings.RegistryConfig)
	settings.RepositoryCache = utils.StringDefaultValue(cliFlags.GetString("registry-cache"), settings.RepositoryCache)
	settings.BurstLimit = utils.IntDefaultValue(cliFlags.GetInt("burst-limit"), settings.BurstLimit)
	settings.RepositoryConfig = utils.StringDefaultValue(cliFlags.GetString("repository-config"), settings.RepositoryConfig)

	_helmbin := os.Getenv("HELM_VIV_HELMBIN")
	if _helmbin != "" {
		helmbin = _helmbin
	}
}

func main() {

	// run when each command's execute method is called
	cobra.OnInitialize(func() {
		helmDriver := os.Getenv("HELM_DRIVER")
		if err := actionConfig.Init(settings.RESTClientGetter(), settings.Namespace(), helmDriver, debug); err != nil {
			log.Fatal(err)
		}
		if helmDriver == "memory" {
			loadReleasesInMemory(actionConfig)
		}
	})

	if err := (&cobra.Command{
		Use:                "helm viv",
		Short:              "Helm plugin to use variable in values",
		Long:               usage,
		SilenceUsage:       false,
		DisableFlagParsing: true,
		Version:            version.Version,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Usage()
			}
			switch args[0] {
			case "help":
				return cmd.Help()
			case "version":
				return pkgUtils.Tmpl(cmd.OutOrStdout(), cmd.VersionTemplate(), map[string]string{
					"Name":    cmd.Name(),
					"Version": cmd.Version,
				})
			case "install", "upgrade", "lint", "template":
				e, err := buildVIVEngine(args)
				if err != nil {
					return err
				}

				if !settings.Debug {
					defer e.Clear()
				}
				for _, f := range e.RenderToTemp() {
					args = append(args, "-f", f)
				}

				break
			}

			return proxyHelmCmd(args)
		},
	}).Execute(); err != nil {
		log.Print(err.Error())
		os.Exit(1)
	}
}

func buildVIVEngine(args []string) (*vivEngine.Engine, error) {
	registryClient, err := registry.NewClient(
		registry.ClientOptDebug(settings.Debug),
		registry.ClientOptEnableCache(true),
		registry.ClientOptWriter(os.Stdout),
		registry.ClientOptCredentialsFile(settings.RegistryConfig),
	)

	if err != nil {
		return nil, err
	}
	actionConfig.RegistryClient = registryClient

	client := action.NewInstall(actionConfig)
	client.DryRun = true
	client.ReleaseName = "release-name"
	client.Replace = true // Skip the name check
	client.ClientOnly = false
	client.APIVersions = chartutil.VersionSet(nil)
	client.IncludeCRDs = false

	client.ChartPathOptions.Version = utils.StringDefaultValue(cliFlags.GetString("version"), client.ChartPathOptions.Version)
	client.ChartPathOptions.Verify = utils.BoolDefaultValue(cliFlags.GetBool("verify"), client.ChartPathOptions.Verify)
	client.ChartPathOptions.Keyring = utils.StringDefaultValue(cliFlags.GetString("keyring"), client.ChartPathOptions.Keyring)
	client.ChartPathOptions.RepoURL = utils.StringDefaultValue(cliFlags.GetString("repo-url"), client.ChartPathOptions.RepoURL)
	client.ChartPathOptions.Username = utils.StringDefaultValue(cliFlags.GetString("username"), client.ChartPathOptions.Username)
	client.ChartPathOptions.Password = utils.StringDefaultValue(cliFlags.GetString("password"), client.ChartPathOptions.Password)
	client.ChartPathOptions.CertFile = utils.StringDefaultValue(cliFlags.GetString("cert-file"), client.ChartPathOptions.CertFile)
	client.ChartPathOptions.KeyFile = utils.StringDefaultValue(cliFlags.GetString("key-file"), client.ChartPathOptions.KeyFile)
	client.ChartPathOptions.InsecureSkipTLSverify = utils.BoolDefaultValue(cliFlags.GetBool("insecure-skip-tls-verify"), client.ChartPathOptions.InsecureSkipTLSverify)
	client.ChartPathOptions.KeyFile = utils.StringDefaultValue(cliFlags.GetString("key-file"), client.ChartPathOptions.KeyFile)
	client.ChartPathOptions.CaFile = utils.StringDefaultValue(cliFlags.GetString("ca-file"), client.ChartPathOptions.CaFile)
	client.ChartPathOptions.PassCredentialsAll = utils.BoolDefaultValue(cliFlags.GetBool("pass-credentials"), client.ChartPathOptions.PassCredentialsAll)

	valueOpts := &values.Options{}
	valueOpts.ValueFiles = cliFlags.GetStringSlice("f", "values")
	valueOpts.Values = cliFlags.GetStringSlice("set")
	valueOpts.FileValues = cliFlags.GetStringSlice("set-file")
	valueOpts.StringValues = cliFlags.GetStringSlice("set-string")
	valueOpts.JSONValues = cliFlags.GetStringSlice("set-json")

	chartRequested, workdir, err := buildChart(clearFlags(args[1:]), client, os.Stdout)
	if err != nil {
		return nil, err
	}

	values, err := buildValuesRender(chartRequested, client, valueOpts, actionConfig, os.Stdout)
	if err != nil {
		return nil, err
	}

	e := vivEngine.NewEngine(&vivEngine.Config{
		WorkDir: strings.TrimRight(workdir, "/"),
		Values:  values,
		Chart:   chartRequested,
	})

	return e, nil
}

func loadReleasesInMemory(actionConfig *action.Configuration) {
	filePaths := strings.Split(os.Getenv("HELM_MEMORY_DRIVER_DATA"), ":")
	if len(filePaths) == 0 {
		return
	}

	store := actionConfig.Releases
	mem, ok := store.Driver.(*driver.Memory)
	if !ok {
		// For an unexpected reason we are not dealing with the memory storage driver.
		return
	}

	actionConfig.KubeClient = &kubefake.PrintingKubeClient{Out: io.Discard}

	for _, path := range filePaths {
		b, err := ioutil.ReadFile(path)
		if err != nil {
			log.Fatal("Unable to read memory driver data", err)
		}

		releases := []*release.Release{}
		if err := yaml.Unmarshal(b, &releases); err != nil {
			log.Fatal("Unable to unmarshal memory driver data: ", err)
		}

		for _, rel := range releases {
			if err := store.Create(rel); err != nil {
				log.Fatal(err)
			}
		}
	}
	// Must reset namespace to the proper one
	mem.SetNamespace(settings.Namespace())
}

func clearFlags(args []string) []string {
	flagStartIdx := len(args)
	for idx, arg := range args {
		if strings.HasPrefix(arg, "-") {
			flagStartIdx = idx
			break
		}

	}
	return args[0:flagStartIdx]
}

func proxyHelmCmd(args []string) error {

	log.Printf("exec: %s %s", helmbin, strings.Join(args, " "))
	cmd := exec.Command(helmbin, args...)
	cmd.Env = os.Environ()
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Stdin = os.Stdin

	return cmd.Run()
}

func debug(format string, v ...interface{}) {
	if settings.Debug {
		format = fmt.Sprintf("[debug] %s\n", format)
		log.Output(2, fmt.Sprintf(format, v...))
	}
}

func warning(format string, v ...interface{}) {
	format = fmt.Sprintf("WARNING: %s\n", format)
	fmt.Fprintf(os.Stderr, format, v...)
}

func buildChart(args []string, client *action.Install, out io.Writer) (*chart.Chart, string, error) {

	name, chart, err := client.NameAndChart(args)
	if err != nil {
		return nil, "", err
	}
	client.ReleaseName = name

	cp, err := client.ChartPathOptions.LocateChart(chart, settings)
	if err != nil {
		return nil, "", err
	}

	debug("CHART PATH: %s\n", cp)
	chartRequested, err := loader.Load(cp)
	if err != nil {
		return nil, "", err
	}

	if err := checkIfInstallable(chartRequested); err != nil {
		return nil, "", err
	}

	if chartRequested.Metadata.Deprecated {
		warning("This chart is deprecated")
	}

	if req := chartRequested.Metadata.Dependencies; req != nil {
		// If CheckDependencies returns an error, we have unfulfilled dependencies.
		// As of Helm 2.4.0, this is treated as a stopping condition:
		// https://github.com/helm/helm/issues/2209
		if err := action.CheckDependencies(chartRequested, req); err != nil {
			err = errors.Wrap(err, "An error occurred while checking for chart dependencies. You may need to run `helm dependency build` to fetch missing dependencies")
			if client.DependencyUpdate {
				man := &downloader.Manager{
					Out:              out,
					ChartPath:        cp,
					Keyring:          client.ChartPathOptions.Keyring,
					SkipUpdate:       false,
					Getters:          getter.All(settings),
					RepositoryConfig: settings.RepositoryConfig,
					RepositoryCache:  settings.RepositoryCache,
					Debug:            settings.Debug,
				}
				if err := man.Update(); err != nil {
					return nil, "", err
				}
				// Reload the chart with the updated Chart.lock file.
				if chartRequested, err = loader.Load(cp); err != nil {
					return nil, "", errors.Wrap(err, "failed reloading chart after repo update")
				}

				return chartRequested, cp, nil
			} else {
				return nil, "", err
			}
		}
	}

	return chartRequested, cp, nil
}

func buildValuesRender(chartRequested *chart.Chart, client *action.Install, valueOpts *values.Options, cfg *action.Configuration, out io.Writer) (chartutil.Values, error) {
	debug("Original chart version: %q", client.Version)
	if client.Version == "" && client.Devel {
		debug("setting version to >0.0.0-0")
		client.Version = ">0.0.0-0"
	}

	p := getter.All(settings)
	vals, err := valueOpts.MergeValues(p)
	if err != nil {
		return nil, err
	}

	client.Namespace = settings.Namespace()

	// Create context and prepare the handle of SIGTERM
	ctx := context.Background()
	ctx, cancel := context.WithCancel(ctx)

	// Set up channel on which to send signal notifications.
	// We must use a buffered channel or risk missing the signal
	// if we're not ready to receive when the signal is sent.
	cSignal := make(chan os.Signal, 2)
	signal.Notify(cSignal, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-cSignal
		fmt.Fprintf(out, "Release %s has been cancelled.\n", client.ReleaseName)
		cancel()
	}()

	if err := chartutil.ProcessDependencies(chartRequested, vals); err != nil {
		return nil, err
	}

	isUpgrade := client.IsUpgrade && client.DryRun

	options := chartutil.ReleaseOptions{
		Name:      client.ReleaseName,
		Namespace: client.Namespace,
		Revision:  1,
		IsInstall: !isUpgrade,
		IsUpgrade: isUpgrade,
	}
	caps, err := GetCapabilities(cfg)
	if err != nil {
		return nil, err
	}

	return chartutil.ToRenderValues(chartRequested, vals, options, caps)
}

func GetCapabilities(cfg *action.Configuration) (*chartutil.Capabilities, error) {
	if cfg.Capabilities != nil {
		return cfg.Capabilities, nil
	}
	dc, err := cfg.RESTClientGetter.ToDiscoveryClient()
	if err != nil {
		return nil, errors.Wrap(err, "could not get Kubernetes discovery client")
	}
	// force a discovery cache invalidation to always fetch the latest server version/capabilities.
	dc.Invalidate()
	kubeVersion, err := dc.ServerVersion()
	if err != nil {
		return nil, errors.Wrap(err, "could not get server version from Kubernetes")
	}
	// Issue #6361:
	// Client-Go emits an error when an API service is registered but unimplemented.
	// We trap that error here and print a warning. But since the discovery client continues
	// building the API object, it is correctly populated with all valid APIs.
	// See https://github.com/kubernetes/kubernetes/issues/72051#issuecomment-521157642
	apiVersions, err := action.GetVersionSet(dc)
	if err != nil {
		if discovery.IsGroupDiscoveryFailedError(err) {
			cfg.Log("WARNING: The Kubernetes server has an orphaned API service. Server reports: %s", err)
			cfg.Log("WARNING: To fix this, kubectl delete apiservice <service-name>")
		} else {
			return nil, errors.Wrap(err, "could not get apiVersions from Kubernetes")
		}
	}

	cfg.Capabilities = &chartutil.Capabilities{
		APIVersions: apiVersions,
		KubeVersion: chartutil.KubeVersion{
			Version: kubeVersion.GitVersion,
			Major:   kubeVersion.Major,
			Minor:   kubeVersion.Minor,
		},
		HelmVersion: chartutil.DefaultCapabilities.HelmVersion,
	}
	return cfg.Capabilities, nil
}

// checkIfInstallable validates if a chart can be installed
//
// Application chart type is only installable
func checkIfInstallable(ch *chart.Chart) error {
	switch ch.Metadata.Type {
	case "", "application":
		return nil
	}
	return errors.Errorf("%s charts are not installable", ch.Metadata.Type)
}

func createRelease(i *action.Install, cfg *action.Configuration, chrt *chart.Chart, rawVals map[string]interface{}) *release.Release {
	ts := cfg.Now()
	return &release.Release{
		Name:      i.ReleaseName,
		Namespace: i.Namespace,
		Chart:     chrt,
		Config:    rawVals,
		Info: &release.Info{
			FirstDeployed: ts,
			LastDeployed:  ts,
			Status:        release.StatusUnknown,
		},
		Version: 1,
	}
}
