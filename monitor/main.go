package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	output Output
	slack  Slack
	config Config
)

func TestSeeders(name string, seeders []string) []DNSSeeder {
	log.Printf("Testing %s seeders..\n", name)
	tm := time.Now()
	errCounter := 0
	var dnsList []DNSSeeder
	for _, x := range seeders {
		seeder := DNSSeeder{Name: x, Type: name}
		tm2 := time.Now()
		result, err := net.LookupHost(x)
		if err != nil {
			errCounter++
			seeder.Status = GetOnlineOffline(false)
			seeder.Error = err.Error()
			log.Printf("%s FAIL.\t\t Test took %s. Error: %s\n", x, time.Since(tm2).String(), err)
		} else {
			seeder.Status = GetOnlineOffline(true)
			seeder.NodeCount = len(result)
			log.Printf("%s OK\t\t %d hosts returned. Test took %s\n", x, len(result), time.Since(tm2).String())
		}
		dnsList = append(dnsList, seeder)
	}
	log.Printf("%d/%d %s DNS seeders online. Total test duration took %s\n", len(seeders)-errCounter, len(seeders), name, time.Since(tm).String())
	return dnsList
}

func TestSites(name string, subdomains []string) []Site {
	log.Printf("Testing %s site..\n", name)
	tm := time.Now()
	errCounter := 0
	var siteList []Site
	//TO-DO: httpPrefixes := []string{"http://", "https://"}

	for _, x := range subdomains {
		url := fmt.Sprintf("http://%s.%s", x, name)
		tm2 := time.Now()
		_, contentSize, httpCode, err := SendHTTPGetRequest(url, false)
		url = StripHTTPPrefix(url)
		site := Site{Name: url}
		if err != nil {
			errCounter++
			site.Status = GetOnlineOffline(false)
			site.HTTPCode = httpCode
			site.ContentSize = contentSize
			site.Error = err.Error()
			log.Printf("%s FAIL.\t\t Test took %s. Error: %s\n", url, time.Since(tm2).String(), err)
		} else {
			site.Status = GetOnlineOffline(true)
			site.HTTPCode = httpCode
			site.ContentSize = contentSize
			site.RespTime = time.Since(tm2).String()
			log.Printf("%s OK.\t\t Test took %s\n", url, time.Since(tm2).String())
		}
		siteList = append(siteList, site)
	}
	log.Printf("%d/%d %s site components online. Total test duration took %s\n", len(subdomains)-errCounter, len(subdomains), name, time.Since(tm).String())
	return siteList
}

func GetOverallStatus(result *Output) string {
	health := "OK"
	for _, x := range result.DNSSeeders {
		if x.Error != "" {
			log.Printf("DNS seeder %s needs attention. Error: %s", x.Name, x.Error)
			health = "Needs attention."
		}
	}

	for _, x := range result.Websites {
		if x.Error != "" {
			log.Printf("Website %s needs attention. Error: %s", x.Name, x.Error)
			health = "Needs attention."
		}
	}
	return health
}

func (o *Output) Update(seeders []DNSSeeder, sites []Site, block BlockInfo) {
	o.mux.Lock()
	o.DNSSeeders = seeders
	o.Websites = sites
	o.LastUpdated = time.Now().Unix()
	o.Block = block
	o.Status = GetOverallStatus(o)
	o.mux.Unlock()
}

func (o *Output) UpdateBlockTime() {
	o.mux.Lock()
	o.Block.TimeElapsed = GetSecondsElapsed(o.Block.BlockTime)
	o.Block.Status = TimeSinceLastBlock(o.Block.BlockTime)
	o.mux.Unlock()
}

func (o Output) Get() Output {
	o.mux.Lock()
	defer o.mux.Unlock()
	return o
}

func CheckState(oldOuput, newOutput Output) {
	for x := range oldOuput.DNSSeeders {
		if oldOuput.DNSSeeders[x].Error != newOutput.DNSSeeders[x].Error {
			if oldOuput.DNSSeeders[x].Error == "" && newOutput.DNSSeeders[x].Error != "" {
				ReportStateChange(oldOuput.DNSSeeders[x].Name, false, newOutput.DNSSeeders[x].Error)
			} else if oldOuput.DNSSeeders[x].Error != "" && newOutput.DNSSeeders[x].Error == "" {
				ReportStateChange(oldOuput.DNSSeeders[x].Name, true, "")
			}
		}
	}

	for x := range oldOuput.Websites {
		if oldOuput.Websites[x].Error != newOutput.Websites[x].Error {
			if oldOuput.Websites[x].Error == "" && newOutput.Websites[x].Error != "" {
				ReportStateChange(oldOuput.Websites[x].Name, false, newOutput.Websites[x].Error)
			} else if oldOuput.Websites[x].Error != "" && newOutput.Websites[x].Error == "" {
				ReportStateChange(oldOuput.Websites[x].Name, true, "")
			}
		}
	}
}

func ReportStateChange(endpoint string, nowOnline bool, err string) {
	var result string
	if nowOnline {
		result = fmt.Sprintf("%s has transitioned from OFFLINE to ONLINE.", endpoint)
	} else {
		result = fmt.Sprintf("%s has transitioned from ONLINE to OFFLINE. Error %s", endpoint, err)
	}
	if slack.Connected {
		slack.SendMessage(slack.Channel, result)
	}
}

func main() {
	//ready := make(chan bool)

	config, err := LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Loaded config")

	go SlackConnect(config.Slack.Token, config.Slack.Channel)

	go func() {
		for {
			var seeders []DNSSeeder
			var sites []Site

			for _, x := range config.DNSSeeders {
				result := TestSeeders(x.Type, strings.Split(x.Hosts, ","))
				seeders = append(seeders, result...)
			}

			for _, x := range config.Websites {
				result := TestSites(x.Host, strings.Split(x.Subdomains, ","))
				sites = append(sites, result...)
			}

			block, err := TestBlockHeight()
			if err != nil {
				block.Status = err.Error()
			} else {
				log.Printf("Block height: %d with time: %d: %s.\n", block.BlockHeight, block.BlockTime, TimeSinceLastBlock(block.BlockTime))
			}

			oldOutput := output
			output.Update(seeders, sites, block)
			newOutput := output.Get()
			CheckState(oldOutput, newOutput)
			//	ready <- true
			time.Sleep(time.Minute)
		}
	}()
	//<-ready

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		output.UpdateBlockTime()
		data, err := json.MarshalIndent(output.Get(), "", "\t")
		if err != nil {
			panic(err)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	log.Printf("Starting HTTP server on port %s\n", config.HTTPServer)
	log.Fatal(http.ListenAndServe(config.HTTPServer, nil))
}
