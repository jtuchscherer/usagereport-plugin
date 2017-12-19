package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	"code.cloudfoundry.org/cli/cf/terminal"
	"code.cloudfoundry.org/cli/cf/trace"
	"code.cloudfoundry.org/cli/plugin"
	"github.com/jtuchscherer/usagereport-plugin/apihelper"
	"github.com/jtuchscherer/usagereport-plugin/models"
)

//UsageReportCmd the plugin
type UsageReportCmd struct {
	apiHelper         apihelper.CFAPIHelper
	ui                terminal.UI
	servicePlanFilter string
}

// contains CLI flag values
type flagVal struct {
	OrgName string
	Format  string
}

func ParseFlags(args []string) flagVal {
	flagSet := flag.NewFlagSet(args[0], flag.ContinueOnError)

	// Create flags
	orgName := flagSet.String("o", "", "-o orgName")
	format := flagSet.String("f", "format", "-f <csv>")

	err := flagSet.Parse(args[1:])
	if err != nil {

	}

	return flagVal{
		OrgName: string(*orgName),
		Format:  string(*format),
	}
}

//GetMetadata returns metatada
func (cmd *UsageReportCmd) GetMetadata() plugin.PluginMetadata {
	return plugin.PluginMetadata{
		Name: "usage-report",
		Version: plugin.VersionType{
			Major: 1,
			Minor: 5,
			Build: 0,
		},
		Commands: []plugin.Command{
			{
				Name:     "usage-report",
				HelpText: "Report AI and memory usage for orgs and spaces",
				UsageDetails: plugin.Usage{
					Usage: "cf usage-report [-o orgName] [-f <csv>]",
					Options: map[string]string{
						"o": "organization",
						"f": "format",
					},
				},
			},
		},
	}
}

//UsageReportCommand doer
func (cmd *UsageReportCmd) UsageReportCommand(args []string) {
	if args[0] != "usage-report" {
		return
	}

	traceLogger := trace.NewLogger(os.Stdout, true, os.Getenv("CF_TRACE"), "")
	cmd.ui = terminal.NewUI(os.Stdin, os.Stdout, terminal.NewTeePrinter(os.Stdout), traceLogger)

	flagVals := ParseFlags(args)

	var orgs []models.Org
	var err error
	var report models.Report

	cmd.setServiceFilterString()

	if flagVals.OrgName != "" {
		org, err := cmd.getOrg(flagVals.OrgName)
		if nil != err {
			fmt.Println(err)
			os.Exit(1)
		}
		orgs = append(orgs, org)
	} else {
		orgs, err = cmd.getOrgs()
		if nil != err {
			fmt.Println(err)
			os.Exit(1)
		}
	}

	report.Orgs = orgs

	if flagVals.Format == "csv" {
		fmt.Println(report.CSV())
	} else {
		fmt.Println(report.String())
	}
}

func (cmd *UsageReportCmd) getOrgs() ([]models.Org, error) {
	rawOrgs, err := cmd.apiHelper.GetOrgs()
	if nil != err {
		return nil, err
	}

	var orgs = []models.Org{}

	for _, o := range rawOrgs {
		orgDetails, err := cmd.getOrgDetails(o)
		if err != nil {
			return nil, err
		}
		orgs = append(orgs, orgDetails)
	}
	return orgs, nil
}

func (cmd *UsageReportCmd) getOrg(name string) (models.Org, error) {
	rawOrg, err := cmd.apiHelper.GetOrg(name)
	if nil != err {
		return models.Org{}, err
	}

	return cmd.getOrgDetails(rawOrg)
}

func (cmd *UsageReportCmd) getOrgDetails(o apihelper.Organization) (models.Org, error) {
	usage, err := cmd.apiHelper.GetOrgMemoryUsage(o)
	if nil != err {
		return models.Org{}, err
	}
	quota, err := cmd.apiHelper.GetQuotaMemoryLimit(o.QuotaURL)
	if nil != err {
		return models.Org{}, err
	}
	spaces, err := cmd.getSpaces(o.SpacesURL)
	if nil != err {
		return models.Org{}, err
	}

	return models.Org{
		Name:        o.Name,
		MemoryQuota: int(quota),
		MemoryUsage: int(usage),
		Spaces:      spaces,
	}, nil
}

func (cmd *UsageReportCmd) getSpaces(spaceURL string) ([]models.Space, error) {
	rawSpaces, err := cmd.apiHelper.GetOrgSpaces(spaceURL)
	if nil != err {
		return nil, err
	}
	var spaces = []models.Space{}
	for _, s := range rawSpaces {
		apps, err := cmd.getApps(s.AppsURL)
		if nil != err {
			return nil, err
		}

		serviceInstances, err := cmd.getServiceInstances(s.ServiceInstancessURL)
		if nil != err {
			return nil, err
		}

		spaces = append(spaces,
			models.Space{
				Apps:             apps,
				ServiceInstances: serviceInstances,
				Name:             s.Name,
			},
		)
	}
	return spaces, nil
}

func (cmd *UsageReportCmd) getApps(appsURL string) ([]models.App, error) {
	rawApps, err := cmd.apiHelper.GetSpaceApps(appsURL)
	if nil != err {
		return nil, err
	}
	var apps = []models.App{}
	for _, a := range rawApps {
		apps = append(apps, models.App{
			Instances: int(a.Instances),
			Ram:       int(a.RAM),
			Running:   a.Running,
		})
	}
	return apps, nil
}

func (cmd *UsageReportCmd) getServiceInstances(serviceInstancesURL string) ([]models.ServiceInstance, error) {
	url := serviceInstancesURL + cmd.servicePlanFilter

	rawServiceInstances, err := cmd.apiHelper.GetSpaceServiceInstances(url)
	if nil != err {
		return nil, err
	}
	var serviceInstances = []models.ServiceInstance{}
	for _, si := range rawServiceInstances {
		serviceInstances = append(serviceInstances, models.ServiceInstance{
			Name: si.Name,
		})
	}
	return serviceInstances, nil
}

func (cmd *UsageReportCmd) getServices(desiredLabels []string) ([]models.Service, error) {
	rawServices, err := cmd.apiHelper.GetServices(desiredLabels)
	if nil != err {
		return nil, err
	}

	var services = []models.Service{}

	for _, s := range rawServices {

		serviceplans, err := cmd.getServicePlans(s.ServicePlansURL)
		if nil != err {
			return nil, err
		}

		services = append(services, models.Service{
			Label: s.Label,
			Plans: serviceplans,
		})
	}
	return services, nil
}

func (cmd *UsageReportCmd) getServicePlans(servicePlansURL string) ([]models.ServicePlan, error) {
	rawServicePlans, err := cmd.apiHelper.GetServicePlans(servicePlansURL)
	if nil != err {
		return nil, err
	}

	var serviceplans = []models.ServicePlan{}

	for _, sp := range rawServicePlans {

		serviceplans = append(serviceplans, models.ServicePlan{
			GUID: sp.GUID,
			Name: sp.Name,
		})
	}
	return serviceplans, nil
}

func (cmd *UsageReportCmd) setServiceFilterString() {

	servicePlanFilterExpr := "?q=service_plan_guid%%20IN%%20%s"

	desiredPlans := []string{"p-redis", "p-mysql", "p-rabbitmq"}

	services, err := cmd.getServices(desiredPlans)
	if err != nil {
		cmd.ui.Failed("Sorry, could not retrieve service listing")
		return
	}

	var planGUIDs []string
	for _, service := range services {
		for _, plan := range service.Plans {
			planGUIDs = append(planGUIDs, plan.GUID)
		}
	}

	cmd.servicePlanFilter = fmt.Sprintf(servicePlanFilterExpr, strings.Join(planGUIDs, ","))
}

//Run runs the plugin
func (cmd *UsageReportCmd) Run(cli plugin.CliConnection, args []string) {
	if args[0] == "usage-report" {
		cmd.apiHelper = apihelper.New(cli)
		cmd.UsageReportCommand(args)
	}
}

func main() {
	plugin.Start(new(UsageReportCmd))
}
