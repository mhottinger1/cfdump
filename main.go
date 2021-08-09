package main

import (
	"encoding/csv"
	"fmt"
	"github.com/cloudfoundry-community/go-cfclient"
	"gopkg.in/yaml.v2"
	"net/url"
	"os"
	"strconv"
	"time"
)

type Config struct {
	Endpoint struct {
		Host string `yaml:"host"`
		Username string `yaml:"username"`
		Password string `yaml:"password"`
	} `yaml:"endpoint"`
}

func processError(err error) {
    fmt.Println(err)
    os.Exit(2)
}

func readFile(cfg *Config) {
    f, err := os.Open("config.yml")
    if err != nil {
        processError(err)
    }
    defer f.Close()

    decoder := yaml.NewDecoder(f)
    err = decoder.Decode(cfg)
    if err != nil {
        processError(err)
    }
}

func writeDataToCsv(allApps [][]string) {
	t := time.Now()
	csvfile, err := os.Create("cf-app-dump-" + t.Format("20060102150405") + ".csv")

	if err != nil {
		fmt.Println("Failed creating file: %s", err)
	}

	csvwriter := csv.NewWriter(csvfile)
	csvwriter.Write([]string{"Foundation", "App Name", "Organization", "Space", "Instances", "State", "Metadata"})
	for _, row := range allApps {
		_ = csvwriter.Write(row)
	}

	csvwriter.Flush()
	csvfile.Close()
}

func processAppsOnePageAtATime(client *cfclient.Client) [][]string {
	page := 1
	pageSize := 50

	q := url.Values{}
	q.Add("results-per-page", strconv.Itoa(pageSize))
	var allApps [][]string

	for {
		// get the current page of apps
		q.Set("page", strconv.Itoa(page))
		apps, err := client.ListAppsByQuery(q)
		if err != nil {
			fmt.Printf("Error getting apps by query: %s", err)
			return allApps
		}

		// do something with each app
		for _, a := range apps {
			space, _ := client.GetSpaceByGuid(a.SpaceGuid)
			org, _ := client.GetOrgByGuid(space.OrganizationGuid)
			v3app, _ := client.GetV3AppByGUID(a.Guid)

			// Convert map to slice of values.
			annotations := []string{}
			for _, value := range v3app.Metadata.Annotations {
				annotations = append(annotations, value)
			}

			row := []string{
				client.Config.ApiAddress,
				a.Name,
				org.Name,
				space.Name,
				strconv.Itoa(a.Instances),
				a.State,
			}
			row = append(row, annotations...)
			fmt.Println(row)
			allApps = append(allApps, row)
		}

		// if we hit an empty page or partial page, that means we're done
		if len(apps) < pageSize {
			break
		}

		// next page
		page++
	}

	return allApps
}

func main() {
	var cfg Config
	readFile(&cfg)
        
	c := &cfclient.Config {
		ApiAddress: cfg.Endpoint.Host,
		Username:   cfg.Endpoint.Username,
		Password:   cfg.Endpoint.Password,
	}
	client, _ := cfclient.NewClient(c)
	allApps := processAppsOnePageAtATime(client)

	if len(allApps) == 0 {
		fmt.Println("Nothing to write, done.")
		os.Exit(0)
	} else {
		writeDataToCsv(allApps)
	}
}
