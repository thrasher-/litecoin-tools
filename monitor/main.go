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
	ip     string
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
			log.Printf("DNS seeder %s needs attention. Error: %s", x.Name, x.Error)
			health = "Needs attention."
		}
	}

	for _, x := range result.Websites {
		if x.Protocol.HTTP.Error != "" {
			log.Printf("Website %s needs attention. Error: %s", x.Name, x.Protocol.HTTP.Error)
			health = "Needs attention."
		}
		if x.Protocol.HTTPS.Error != "" {
			log.Printf("Website %s needs attention. Error: %s", x.Name, x.Protocol.HTTPS.Error)
			health = "Needs attention."
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
	var err error
	config, err = LoadConfig()
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Loaded config.")
	config.CheckDelay = time.Minute * config.CheckDelay
	log.Println("Check delay set to", config.CheckDelay.Minutes(), "minute(s).")

	go SlackConnect(config.Slack.Token, config.Slack.Channel)

	go BlockMonitor()

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
			time.Sleep(config.CheckDelay)
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
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.Write(data)
	})

	log.Printf("Starting HTTP server on port %s\n", config.HTTPServer)
	log.Fatal(http.ListenAndServe(config.HTTPServer, nil))
}
