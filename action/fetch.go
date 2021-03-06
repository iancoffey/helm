package action

import (
	"os"
	"path/filepath"

	"github.com/helm/helm/chart"
	"github.com/helm/helm/dependency"
	"github.com/helm/helm/log"
	helm "github.com/helm/helm/util"
)

// Fetch gets a chart from the source repo and copies to the workdir.
//
// - chartName is the source
// - lname is the local name for that chart (chart-name); if blank, it is set to the chart.
// - homedir is the home directory for the user
func Fetch(chartName, lname, homedir string) {

	r := mustConfig(homedir).Repos
	repository, chartName := r.RepoChart(chartName)

	if lname == "" {
		lname = chartName
	}

	fetch(chartName, lname, homedir, repository)

	chartFilePath := filepath.Join(homedir, helm.WorkspaceChartPath, lname, "Chart.yaml")
	cfile, err := chart.LoadChartfile(chartFilePath)
	if err != nil {
		log.Die("Source is not a valid chart. Missing Chart.yaml: %s", err)
	}

	deps, err := dependency.Resolve(cfile, filepath.Join(homedir, helm.WorkspaceChartPath))
	if err != nil {
		log.Warn("Could not check dependencies: %s", err)
		return
	}

	if len(deps) > 0 {
		log.Warn("Unsatisfied dependencies:")
		for _, d := range deps {
			log.Msg("\t%s %s", d.Name, d.Version)
		}
	}

	log.Info("Fetched chart into workspace %s", filepath.Join(homedir, helm.WorkspaceChartPath, lname))
	log.Info("Done")
}

func fetch(chartName, lname, homedir, chartpath string) {
	src := filepath.Join(homedir, helm.CachePath, chartpath, chartName)
	dest := filepath.Join(homedir, helm.WorkspaceChartPath, lname)

	if fi, err := os.Stat(src); err != nil {
		log.Die("Chart %s not found in %s", lname, src)
	} else if !fi.IsDir() {
		log.Die("Malformed chart %s: Chart must be in a directory.", chartName)
	}

	if err := os.MkdirAll(dest, 0755); err != nil {
		log.Die("Could not create %q: %s", dest, err)
	}

	log.Debug("Fetching %s to %s", src, dest)
	if err := helm.CopyDir(src, dest); err != nil {
		log.Die("Failed copying %s to %s", src, dest)
	}

	if err := updateChartfile(src, dest, lname); err != nil {
		log.Die("Failed to update Chart.yaml: %s", err)
	}
}

func updateChartfile(src, dest, lname string) error {
	sc, err := chart.LoadChartfile(filepath.Join(src, "Chart.yaml"))
	if err != nil {
		return err
	}

	dc, err := chart.LoadChartfile(filepath.Join(dest, "Chart.yaml"))
	if err != nil {
		return err
	}

	dc.Name = lname
	dc.From = &chart.Dependency{
		Name:    sc.Name,
		Version: sc.Version,
		Repo:    chart.RepoName(src),
	}

	return dc.Save(filepath.Join(dest, "Chart.yaml"))
}
