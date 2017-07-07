package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	RPC_PORT     = 9332
	RPC_USERNAME = "user"
	RPC_PASSWORD = "pass"
	RPC_HOST     = "127.0.0.1"
)

var (
	MainnetSeeders          = []string{"seed-a.litecoin.loshan.co.uk", "dnsseed.thrasher.io", "dnsseed.litecointools.com", "dnsseed.litecoinpool.org", "dnsseed.koin-project.com"}
	TestnetSeeders          = []string{"testnet-seed.litecointools.com", "seed-b.litecoin.loshan.co.uk", "dnsseed-testnet.thrasher.io"}
	LitecoinOrgSites        = []string{"http://www.litecoin.org", "http://litecoin.org", "http://download.litecoin.org", "http://blog.litecoin.org"}
	LitecoinComSites        = []string{"http://www.litecoin.com", "http://litecoin.com"}
	LitecoinCoreSites       = []string{"https://www.litecoincore.org", "https://litecoincore.org"}
	LitecoinFoundationSites = []string{"https://www.litecoin-foundation.org", "https://litecoin-foundation.org"}
	LitecoreSites           = []string{"https://insight.litecore.io"}
)

type BlockInfo struct {
	BlockHeight int64  `json:"block_height"`
	BlockTime   int64  `json:"block_time"`
	TimeElapsed int64  `json:"time_elapsed"`
	Status      string `json:"status"`
}

type DNSSeeder struct {
	Name      string `json:"name"`
	Type      string `json:"type"`
	NodeCount int    `json:"node_count"`
	Status    string `json:"status"`
	Error     string `json:"error"`
}

type Site struct {
	Name        string `json:"name"`
	Status      string `json:"status"`
	HTTPCode    int    `json:"http_code"`
	ContentSize int    `json:"content_size"`
	RespTime    string `json:"response_time"`
	Error       string `json:"error"`
}

type Output struct {
	DNSSeeders  []DNSSeeder `json:"dns_seeders"`
	Websites    []Site      `json:"websites"`
	Block       BlockInfo   `json:"network"`
	Status      string      `json:"status"`
	LastUpdated int64       `json:"last_updated"`
	mux         sync.Mutex
}

func StripHTTPPrefix(url string) string {
	if strings.Contains(url, "http://") {
		return strings.TrimPrefix(url, "http://")
	} else {
		return strings.TrimPrefix(url, "https://")
	}
}

func GetOnlineOffline(online bool) string {
	if !online {
		return "OFFLINE"
	}
	return "ONLINE"
}

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

func TestSites(name string, sites []string) []Site {
	log.Printf("Testing %s site..\n", name)
	tm := time.Now()
	errCounter := 0
	var siteList []Site
	for _, x := range sites {
		tm2 := time.Now()
		_, contentSize, httpCode, err := SendHTTPGetRequest(x, false)
		x = StripHTTPPrefix(x)
		site := Site{Name: x}
		if err != nil {
			errCounter++
			site.Status = GetOnlineOffline(false)
			site.HTTPCode = httpCode
			site.ContentSize = contentSize
			site.Error = err.Error()
			log.Printf("%s FAIL.\t\t Test took %s. Error: %s\n", x, time.Since(tm2).String(), err)
		} else {
			site.Status = GetOnlineOffline(true)
			site.HTTPCode = httpCode
			site.ContentSize = contentSize
			site.RespTime = time.Since(tm2).String()
			log.Printf("%s OK.\t\t Test took %s\n", x, time.Since(tm2).String())
		}
		siteList = append(siteList, site)
	}
	log.Printf("%d/%d %s site components online. Total test duration took %s\n", len(sites)-errCounter, len(sites), name, time.Since(tm).String())
	return siteList
}

func SendHTTPGetRequest(url string, jsonDecode bool) (result interface{}, contentSize, httpCode int, err error) {
	res, err := http.Get(url)

	if err != nil {
		return
	}

	httpCode = res.StatusCode
	if httpCode != 200 {
		log.Printf("HTTP status code: %d\n", httpCode)
		err = errors.New("Status code was not 200.")
		return
	}

	contents, err := ioutil.ReadAll(res.Body)
	contentSize = len(contents)

	if err != nil {
		return
	}

	defer res.Body.Close()

	if jsonDecode {
		err := JSONDecode(contents, &result)

		if err != nil {
			return result, contentSize, httpCode, err
		}
	} else {
		result = string(contents)
	}

	return
}

func BuildURL() string {
	return fmt.Sprintf("http://%s:%s@%s:%d", RPC_USERNAME, RPC_PASSWORD, RPC_HOST, RPC_PORT)
}

func JSONDecode(data []byte, to interface{}) error {
	err := json.Unmarshal(data, &to)

	if err != nil {
		return err
	}

	return nil
}

func SendRPCRequest(method, req interface{}) (map[string]interface{}, error) {
	var params []interface{}
	if req != nil {
		params = append(params, req)
	} else {
		params = nil
	}

	data, err := json.Marshal(map[string]interface{}{
		"method": method,
		"id":     1,
		"params": params,
	})

	if err != nil {
		return nil, err
	}

	resp, err := http.Post(BuildURL(), "application/json", strings.NewReader(string(data)))
	if err != nil {
		return nil, err
	}

	defer resp.Body.Close()
	body, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	result := make(map[string]interface{})
	err = json.Unmarshal(body, &result)
	if err != nil {
		return nil, err
	}

	if result["error"] != nil {
		errorMsg := result["error"].(map[string]interface{})
		return nil, fmt.Errorf("Error code: %v, message: %v\n", errorMsg["code"], errorMsg["message"])
	}
	return result, nil
}

func GetBlockHeight() (int64, error) {
	result, err := SendRPCRequest("getinfo", nil)
	if err != nil {
		return 0, err
	}
	result = result["result"].(map[string]interface{})
	block := result["blocks"].(float64)
	return int64(block), nil
}

func GetBlockHash(block int64) (string, error) {
	result, err := SendRPCRequest("getblockhash", block)
	if err != nil {
		return "", err
	}
	return result["result"].(string), nil
}

func GetBlockTime(block string) (int64, error) {
	result, err := SendRPCRequest("getblock", block)
	if err != nil {
		return 0, err
	}

	result = result["result"].(map[string]interface{})
	blockTime := result["time"].(float64)
	return int64(blockTime), nil
}

func GetSecondsElapsed(timestamp int64) int64 {
	tm := time.Unix(timestamp, 0)
	return int64(time.Since(tm).Seconds())
}

func TestBlockHeight() (BlockInfo, error) {
	var blockInfo BlockInfo
	blockHeight, err := GetBlockHeight()
	if err != nil {
		return blockInfo, err
	}

	blockHash, err := GetBlockHash(blockHeight)
	if err != nil {
		return blockInfo, err
	}

	blockTime, err := GetBlockTime(blockHash)
	if err != nil {
		return blockInfo, err
	}

	blockInfo.BlockHeight = blockHeight
	blockInfo.BlockTime = blockTime
	blockInfo.TimeElapsed = GetSecondsElapsed(blockTime)
	blockInfo.Status = TimeSinceLastBlock(blockTime)
	return blockInfo, nil
}

func TimeSinceLastBlock(blockTime int64) string {
	seconds := GetSecondsElapsed(blockTime)
	if seconds >= int64(60*2.5) && seconds < 60*10 {
		return "Block not found within 2.5 minutes."
	}
	if seconds >= 60*10 && seconds < 60*30 {
		return "Block not found within 10 minutes."
	}
	if seconds > 60*30 {
		return "POTENTIAL ISSUE: Block not found within 30 minutes."
	}
	return "OK"
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

func (o *Output) Get() *Output {
	o.mux.Lock()
	defer o.mux.Unlock()
	return o
}

func main() {
	var output Output
	ready := make(chan bool)

	go func() {
		for {
			var seeders []DNSSeeder
			result := TestSeeders("mainnet", MainnetSeeders)
			seeders = append(seeders, result...)

			result = TestSeeders("testnet", TestnetSeeders)
			seeders = append(seeders, result...)

			var sites []Site
			siteResult := TestSites("litecoin.org", LitecoinOrgSites)
			sites = append(sites, siteResult...)

			siteResult = TestSites("litecoin.com", LitecoinComSites)
			sites = append(sites, siteResult...)

			siteResult = TestSites("litecoincore.org", LitecoinCoreSites)
			sites = append(sites, siteResult...)

			siteResult = TestSites("litecoin-foundation.org", LitecoinFoundationSites)
			sites = append(sites, siteResult...)

			siteResult = TestSites("litecore.io", LitecoreSites)
			sites = append(sites, siteResult...)

			block, err := TestBlockHeight()
			if err != nil {
				log.Println(err)
			} else {
				log.Printf("Block height: %d with time: %d: %s.\n", block.BlockHeight, block.BlockTime, TimeSinceLastBlock(block.BlockTime))
			}

			output.Update(seeders, sites, block)
			ready <- true
			time.Sleep(time.Second * 30)
		}
	}()
	<-ready

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

	log.Println("Starting HTTP server on port 8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
