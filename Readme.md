# Helm-variable-in-values

**Part of the code in this repository comes from [helm](https://github.com/helm/helm)
and [helm-cm-push](https://github.com/chartmuseum/helm-push). **

## Install

## Usage

### 1. Create viv-chart project

```shell
helm create example
```

### 2. Add your variable-in-values to `vivs/xxx.yaml`

**values.yaml**

```yaml
...

subChart:
  serviceSelector:
    name:

subChart2:
  # realname is <.ReleaseName>-<.Values.subChart2.serviceName>
  serviceName: subChart
```

**vivs/overrideSubChartServiceName.yaml**

```yaml
subChart:
  serviceSelector:
    name: "{{ .Release.Name }}.{{ .Values.subChart2.serviceName }}"
```

### 3. Install

```shell
$ helm viv template --generate-name exmaple/simple-exmaple -f ./values.yaml
```

## Debug

add `--debug` to your command, and then you can see *vivTemp* dir in your chart.

**command**
```shell
$ helm viv template --generate-name exmaple/simple-exmaple -f ./values.yaml --debug
```

**example/simple-example directory**
```shell
$ tree ./example/simple-example

#./example/simple-example
#├── Chart.lock
#├── Chart.yaml
#├── charts
#│   └── ingress-0.1.0.tgz
#├── templates
#│   ├── NOTES.txt
#│   ├── _helpers.tpl
#│   ├── deployment.yaml
#│   ├── hpa.yaml
#│   └── serviceaccount.yaml
#├── values.yaml
#├── vivTemp
#│   ├── _charts_ingressAlias_charts_service_vivs_values.yaml
#│   ├── _charts_ingressAlias_vivs_values.yaml
#│   ├── vivs_autoscaling.yaml
#│   └── vivs_values.yaml
#└── vivs
#    ├── autoscaling.yaml
#    └── values.yaml
```

