package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

var (
	output              Output
	slack               Slack
	config              Config
	ip                  string
	endpointErrorState  map[string]int
	knownErrorEndpoints []string
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

func IsSiteExluded(host, protocol string) bool {
	for _, x := range config.Websites {
		if host == x.Host && strings.Contains(protocol, x.Exclusions) && x.Exclusions != "" {
			return true
		}
	}
	return false
}

func CheckContentMatch(endpoint, content string) bool {
	for _, x := range config.Websites {
		for _, y := range x.ContentMatch {
			ep := y.Subdomains + "." + x.Host
			if (ep == endpoint && y.StringCheck != "" && strings.Contains(content, y.StringCheck)) || y.StringCheck == "" {
				return true
			}
		}
	}
	return false
}

func TestSites(name string, subdomains []string) []Site {
	log.Printf("Testing %s site..\n", name)
	tm := time.Now()
	errCounter := 0
	var siteList []Site
	httpPrefixes := []string{"http://", "https://"}

	for _, x := range subdomains {
		var site Site
		for _, y := range httpPrefixes {
			url := fmt.Sprintf("%s%s.%s", y, x, name)
			site.Name = StripHTTPPrefix(url)
			var result SiteProtocol

			if IsSiteExluded(name, y) {
				result.Status = "NA"
				result.Error = ""
			} else {
				tm2 := time.Now()
				content, contentSize, httpCode, err := SendHTTPGetRequest(url, false)
				result.ContentSize = contentSize
				result.HTTPCode = httpCode
				result.RespTime = time.Since(tm2).String()
				result.Status = GetOnlineOffline(true) // default to online

				var contentMatch bool
				if content == nil {
					contentMatch = CheckContentMatch(site.Name, "")
				} else {
					contentMatch = CheckContentMatch(site.Name, content.(string))
				}

				if err != nil || !contentMatch {
					result.Status = GetOnlineOffline(false)

					if !contentMatch {
						err = fmt.Errorf("%s content match failed", url)
					}

					result.Error = err.Error()
					site.NeedsAttention = true
					log.Printf("%s FAIL.\t\t Test took %s. Error: %s\n", url, time.Since(tm2).String(), err)
				} else {
					log.Println(url, "content match PASSED!")
					log.Printf("%s OK.\t\t Test took %s\n", url, time.Since(tm2).String())
				}
			}

			if y == "http://" {
				site.Protocol.HTTP = result
			} else {
				site.Protocol.HTTPS = result
			}
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
			endpointErrorState[x.Name]++
			CheckExistingErrorState(x.Name, x.Error)
			log.Printf("DNS seeder %s needs attention. Error: %s Failure counter: %d Is known: %v\n", x.Name, x.Error,
				endpointErrorState[x.Name], IsKnownErrorEndpoint(x.Name))
			health = "Needs attention."
		} else {
			endpointErrorState[x.Name] = 0
		}
	}

	for _, x := range result.Websites {
		if x.Protocol.HTTP.Error != "" {
			endpointName := "http://" + x.Name
			endpointErrorState[endpointName]++
			CheckExistingErrorState(endpointName, x.Protocol.HTTP.Error)
			log.Printf("Website %s needs attention. Error: %s Failure counter: %d Is known: %v\n", x.Name, x.Protocol.HTTP.Error,
				endpointErrorState[endpointName], IsKnownErrorEndpoint(endpointName))
			health = "Needs attention."
		} else {
			endpointErrorState["http://"+x.Name] = 0
		}
		if x.Protocol.HTTPS.Error != "" {
			endpointName := "https://" + x.Name
			endpointErrorState[endpointName]++
			CheckExistingErrorState(endpointName, x.Protocol.HTTPS.Error)
			log.Printf("Website %s needs attention. Error: %s Failure counter: %d Is known: %v\n", x.Name, x.Protocol.HTTPS.Error,
				endpointErrorState[endpointName], IsKnownErrorEndpoint(endpointName))
			health = "Needs attention."
		} else {
			endpointErrorState["https://"+x.Name] = 0
		}
	}
	return health
}

func (o *Output) Update(seeders []DNSSeeder, sites []Site) {
	o.mux.Lock()
	o.DNSSeeders = seeders
	o.Websites = sites
	o.LastUpdated = time.Now().Unix()
	o.Status = GetOverallStatus(o)
	o.mux.Unlock()
}

func (o *Output) UpdateBlockInfo(bi BlockInfo) {
	o.mux.Lock()
	o.Block = bi
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
		if oldOuput.Websites[x].NeedsAttention != newOutput.Websites[x].NeedsAttention {
			if oldOuput.Websites[x].Protocol.HTTP.Error == "" && newOutput.Websites[x].Protocol.HTTP.Error != "" {
				ReportStateChange("http://"+oldOuput.Websites[x].Name, false, newOutput.Websites[x].Protocol.HTTP.Error)
			} else if oldOuput.Websites[x].Protocol.HTTP.Error != "" && newOutput.Websites[x].Protocol.HTTP.Error == "" {
				ReportStateChange("http://"+oldOuput.Websites[x].Name, true, "")
			}

			if oldOuput.Websites[x].Protocol.HTTPS.Error == "" && newOutput.Websites[x].Protocol.HTTPS.Error != "" {
				ReportStateChange("https://"+oldOuput.Websites[x].Name, false, newOutput.Websites[x].Protocol.HTTPS.Error)
			} else if oldOuput.Websites[x].Protocol.HTTPS.Error != "" && newOutput.Websites[x].Protocol.HTTPS.Error == "" {
				ReportStateChange("https://"+oldOuput.Websites[x].Name, true, "")
			}
		}
	}
}

func IsKnownErrorEndpoint(endpoint string) bool {
	for x := range knownErrorEndpoints {
		if knownErrorEndpoints[x] == endpoint {
			return true
		}
	}
	return false
}

func RemoveKnownErrorEndpoint(endpoint string) {
	for i, v := range knownErrorEndpoints {
		if v == endpoint {
			knownErrorEndpoints = append(knownErrorEndpoints[:i], knownErrorEndpoints[i+1:]...)
		}
	}
}

func CheckExistingErrorState(endpoint, err string) {
	var result string
	if endpointErrorState[endpoint] == config.ErrorTransitionThreshold && !IsKnownErrorEndpoint(endpoint) {
		result = fmt.Sprintf("%s has transitioned from ONLINE to OFFLINE. Error %s", endpoint, err)
		knownErrorEndpoints = append(knownErrorEndpoints, endpoint)
		if slack.Connected {
			slack.SendMessage(slack.Channel, result)
		}
	}
}

func ReportStateChange(endpoint string, nowOnline bool, err string) {
	if nowOnline && endpointErrorState[endpoint] >= config.ErrorTransitionThreshold || IsKnownErrorEndpoint(endpoint) {
		endpointErrorState[endpoint] = 0
		RemoveKnownErrorEndpoint(endpoint)
		result := fmt.Sprintf("%s has transitioned from OFFLINE to ONLINE.", endpoint)
		if slack.Connected {
			slack.SendMessage(slack.Channel, result)
		}
	}
}

func HandleInterrupt() {
	c := make(chan os.Signal, 1)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		sig := <-c
		log.Printf("Captured %v.", sig)
		Shutdown()
	}()
}

func Shutdown() {
	log.Println("Shutting down")
	config.KnownErrorEndpoints = strings.Join(knownErrorEndpoints, ",")
	err := SaveConfig(config)

	if err != nil {
		log.Println("Failed to save config")
	} else {
		log.Println("Saved config file successfully")
	}

	log.Println("Exiting.")
	os.Exit(1)
}

func main() {
	//ready := make(chan bool)
	go HandleInterrupt()

	var err error
	config, err = LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Loaded config.")
	log.Println("Check delay set to", (config.CheckDelay * time.Minute).Minutes(), "minute(s).")
	log.Printf("Error transition threshold set to %d.\n", config.ErrorTransitionThreshold)

	knownErrorEndpoints = strings.Split(config.KnownErrorEndpoints, ",")
	log.Printf("Ignoring known error endpoints until resolution: %s\n", knownErrorEndpoints)

	go SlackConnect(config.Slack.Token, config.Slack.Channel)

	go BlockMonitor()

	endpointErrorState = make(map[string]int)

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

			oldOutput := output
			output.Update(seeders, sites)
			newOutput := output.Get()
			CheckState(oldOutput, newOutput)
			//	ready <- true
			time.Sleep(time.Minute * config.CheckDelay)
		}
	}()
	//<-ready

	ip, err = GetExternalIP()
	if err != nil {
		log.Printf("Unable to get IP. Error %s", err)
	} else {
		log.Println("External IP: ", ip)
	}

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		output.UpdateBlockTime()
		data, err := json.MarshalIndent(output.Get(), "", "\t")
		if err != nil {
			panic(err)
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	log.Printf("Starting HTTP server on port %s\n", config.HTTPServer)
	log.Fatal(http.ListenAndServe(config.HTTPServer, nil))
}
