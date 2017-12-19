package apihelper

import (
	"errors"
	"fmt"
	"net/url"
	"strconv"
	"strings"

	"code.cloudfoundry.org/cli/plugin"
	"github.com/krujos/cfcurl"
)

var (
	errOrgNotFound = errors.New("organization not found")
)

//Organization representation
type Organization struct {
	URL       string
	Name      string
	QuotaURL  string
	SpacesURL string
}

//Space representation
type Space struct {
	Name                 string
	AppsURL              string
	ServiceInstancessURL string
}

//App representation
type App struct {
	Instances float64
	RAM       float64
	Running   bool
}

//App representation
type ServiceInstance struct {
	Name string
}

//Service representation
type Service struct {
	Label           string
	ServicePlansURL string
}

//ServicePlan representation
type ServicePlan struct {
	GUID string
	Name string
}

//CFAPIHelper to wrap cf curl results
type CFAPIHelper interface {
	GetOrgs() ([]Organization, error)
	GetOrg(string) (Organization, error)
	GetQuotaMemoryLimit(string) (float64, error)
	GetOrgMemoryUsage(Organization) (float64, error)
	GetOrgSpaces(string) ([]Space, error)
	GetSpaceApps(string) ([]App, error)
	GetServices([]string) ([]Service, error)
	GetServicePlans(string) ([]ServicePlan, error)
	GetSpaceServiceInstances(string) ([]ServiceInstance, error)
}

//APIHelper implementation
type APIHelper struct {
	cli plugin.CliConnection
}

func New(cli plugin.CliConnection) CFAPIHelper {
	return &APIHelper{cli}
}

//GetOrgs returns a struct that represents critical fields in the JSON
func (api *APIHelper) GetOrgs() ([]Organization, error) {
	orgsJSON, err := cfcurl.Curl(api.cli, "/v2/organizations")
	if nil != err {
		return nil, err
	}
	pages := int(orgsJSON["total_pages"].(float64))
	orgs := []Organization{}
	for i := 1; i <= pages; i++ {
		if 1 != i {
			orgsJSON, err = cfcurl.Curl(api.cli, "/v2/organizations?page="+strconv.Itoa(i))
		}
		for _, o := range orgsJSON["resources"].([]interface{}) {
			theOrg := o.(map[string]interface{})
			entity := theOrg["entity"].(map[string]interface{})
			metadata := theOrg["metadata"].(map[string]interface{})
			orgs = append(orgs,
				Organization{
					Name:      entity["name"].(string),
					URL:       metadata["url"].(string),
					QuotaURL:  entity["quota_definition_url"].(string),
					SpacesURL: entity["spaces_url"].(string),
				})
		}
	}
	return orgs, nil
}

//GetOrg returns a struct that represents critical fields in the JSON
func (api *APIHelper) GetOrg(name string) (Organization, error) {
	query := fmt.Sprintf("name:%s", name)
	path := fmt.Sprintf("/v2/organizations?q=%s&inline-relations-depth=1", url.QueryEscape(query))
	orgsJSON, err := cfcurl.Curl(api.cli, path)
	if nil != err {
		return Organization{}, err
	}

	results := int(orgsJSON["total_results"].(float64))
	if results == 0 {
		return Organization{}, errOrgNotFound
	}

	orgResource := orgsJSON["resources"].([]interface{})[0]
	org := api.orgResourceToOrg(orgResource)

	return org, nil
}

func (api *APIHelper) orgResourceToOrg(o interface{}) Organization {
	theOrg := o.(map[string]interface{})
	entity := theOrg["entity"].(map[string]interface{})
	metadata := theOrg["metadata"].(map[string]interface{})
	return Organization{
		Name:      entity["name"].(string),
		URL:       metadata["url"].(string),
		QuotaURL:  entity["quota_definition_url"].(string),
		SpacesURL: entity["spaces_url"].(string),
	}
}

//GetQuotaMemoryLimit retruns the amount of memory (in MB) that the org is allowed
func (api *APIHelper) GetQuotaMemoryLimit(quotaURL string) (float64, error) {
	quotaJSON, err := cfcurl.Curl(api.cli, quotaURL)
	if nil != err {
		return 0, err
	}
	return quotaJSON["entity"].(map[string]interface{})["memory_limit"].(float64), nil
}

//GetOrgMemoryUsage returns the amount of memory (in MB) that the org is consuming
func (api *APIHelper) GetOrgMemoryUsage(org Organization) (float64, error) {
	usageJSON, err := cfcurl.Curl(api.cli, org.URL+"/memory_usage")
	if nil != err {
		return 0, err
	}
	return usageJSON["memory_usage_in_mb"].(float64), nil
}

//GetOrgSpaces returns the spaces in an org.
func (api *APIHelper) GetOrgSpaces(spacesURL string) ([]Space, error) {
	nextURL := spacesURL
	spaces := []Space{}
	for nextURL != "" {
		spacesJSON, err := cfcurl.Curl(api.cli, nextURL)
		if nil != err {
			return nil, err
		}
		for _, s := range spacesJSON["resources"].([]interface{}) {
			theSpace := s.(map[string]interface{})
			entity := theSpace["entity"].(map[string]interface{})
			spaces = append(spaces,
				Space{
					AppsURL:              entity["apps_url"].(string),
					ServiceInstancessURL: entity["service_instances_url"].(string),
					Name:                 entity["name"].(string),
				})
		}
		if next, ok := spacesJSON["next_url"].(string); ok {
			nextURL = next
		} else {
			nextURL = ""
		}
	}
	return spaces, nil
}

//GetSpaceApps returns the apps in a space
func (api *APIHelper) GetSpaceApps(appsURL string) ([]App, error) {
	nextURL := appsURL
	apps := []App{}
	for nextURL != "" {
		appsJSON, err := cfcurl.Curl(api.cli, nextURL)
		if nil != err {
			return nil, err
		}
		for _, a := range appsJSON["resources"].([]interface{}) {
			theApp := a.(map[string]interface{})
			entity := theApp["entity"].(map[string]interface{})
			apps = append(apps,
				App{
					Instances: entity["instances"].(float64),
					RAM:       entity["memory"].(float64),
					Running:   "STARTED" == entity["state"].(string),
				})
		}
		if next, ok := appsJSON["next_url"].(string); ok {
			nextURL = next
		} else {
			nextURL = ""
		}
	}
	return apps, nil
}

//GetSpaceServiceInstances returns the apps in a space
func (api *APIHelper) GetSpaceServiceInstances(serviceInstancesURL string) ([]ServiceInstance, error) {
	serviceInstances, err := processPagedResults(api.cli, serviceInstancesURL, func(metadata map[string]interface{}, entity map[string]interface{}) interface{} {
		return ServiceInstance{
			Name: entity["name"].(string),
		}
	})

	retVal := make([]ServiceInstance, len(serviceInstances))
	for i := range serviceInstances {
		retVal[i] = serviceInstances[i].(ServiceInstance)
	}

	if nil != err {
		return nil, err
	}

	return retVal, nil
}

//GetServices returns a struct that represents critical fields in the JSON
func (api *APIHelper) GetServices(desiredLabels []string) ([]Service, error) {
	queryParam := fmt.Sprintf("?q=label%%20IN%%20%s", strings.Join(desiredLabels, ","))
	url := "/v2/services" + queryParam

	services, err := processPagedResults(api.cli, url, func(metadata map[string]interface{}, entity map[string]interface{}) interface{} {
		return Service{
			Label:           entity["label"].(string),
			ServicePlansURL: entity["service_plans_url"].(string),
		}
	})

	retVal := make([]Service, len(services))
	for i := range services {
		retVal[i] = services[i].(Service)
	}

	if nil != err {
		return nil, err
	}

	return retVal, nil
}

//GetServices returns a struct that represents critical fields in the JSON
func (api *APIHelper) GetServicePlans(plansURL string) ([]ServicePlan, error) {
	serviceplans, err := processPagedResults(api.cli, plansURL, func(metadata map[string]interface{}, entity map[string]interface{}) interface{} {
		return ServicePlan{
			GUID: metadata["guid"].(string),
			Name: entity["name"].(string),
		}
	})

	retVal := make([]ServicePlan, len(serviceplans))
	for i := range serviceplans {
		retVal[i] = serviceplans[i].(ServicePlan)
	}

	if nil != err {
		return nil, err
	}

	return retVal, nil
}

//Function type to simplify processing paged results
type process func(metadata map[string]interface{}, entity map[string]interface{}) interface{}

func processPagedResults(cli plugin.CliConnection, url string, fn process) ([]interface{}, error) {

	theJSON, err := cfcurl.Curl(cli, url)

	if nil != err {
		return nil, err
	}

	pages := int(theJSON["total_pages"].(float64))
	var objects []interface{}
	for i := 1; i <= pages; i++ {
		if 1 != i {
			theJSON, err = cfcurl.Curl(cli, url+"?page="+strconv.Itoa(i))
		}
		for _, o := range theJSON["resources"].([]interface{}) {
			theObj := o.(map[string]interface{})
			entity := theObj["entity"].(map[string]interface{})
			metadata := theObj["metadata"].(map[string]interface{})
			objects = append(objects, fn(metadata, entity))
		}

	}

	return objects, nil
}
