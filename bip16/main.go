package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

var (
	version, block int
	verbose        bool
	RPCHost        string
	RPCPort        int
	RPCUsername    string
	RPCPassword    string
	BIP16target    int64
)

func BuildURL() string {
	return fmt.Sprintf("http://%s:%s@%s:%d", RPCUsername, RPCPassword, RPCHost, RPCPort)
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

func GetBlockHeight() (int, error) {
	result, err := SendRPCRequest("getinfo", nil)
	if err != nil {
		return 0, err
	}

	result = result["result"].(map[string]interface{})
	block := result["blocks"].(float64)
	return int(block), nil
}

func GetBlockHash(block int) (string, error) {
	result, err := SendRPCRequest("getblockhash", block)
	if err != nil {
		return "", err
	}

	return result["result"].(string), nil
}

type BlockInfo struct {
	Hash      string
	BlockTime int64
}

func GetBlockTime(block int) (BlockInfo, error) {
	var blockInfo BlockInfo
	blockHash, err := GetBlockHash(block)
	if err != nil {
		return blockInfo, err
	}

	result, err := SendRPCRequest("getblock", blockHash)
	if err != nil {
		return blockInfo, err
	}

	result = result["result"].(map[string]interface{})
	blockInfo.BlockTime = int64(result["time"].(float64))
	blockInfo.Hash = result["hash"].(string)
	return blockInfo, nil
}

func main() {
	flag.StringVar(&RPCHost, "rpchost", "127.0.0.1", "The RPC host to connect to.")
	flag.IntVar(&RPCPort, "rpcport", 9332, "The RPC port to connect to.")
	flag.StringVar(&RPCUsername, "rpcuser", "user", "The RPC username.")
	flag.StringVar(&RPCPassword, "rpcpass", "pass", "The RPC password.")
	flag.IntVar(&block, "block", 218570, "Block height to start checking from.")
	flag.BoolVar(&verbose, "verbose", false, "Toggle verbose reporting.")
	flag.Int64Var(&BIP16target, "bip16target", 1349049600, "Target timestamp for BIP16 activation.")
	flag.Parse()

	log.Printf("RPC URL: %s", BuildURL())
	currentHeight, err := GetBlockHeight()
	if err != nil {
		log.Fatalf("Failed to retrieve current block height. Err: %s", err)
	}
	log.Printf("Current block height: %d\n", currentHeight)
	log.Printf("Checking for BIP16 target block timestamp >= %d", BIP16target)

	for i := block; i < currentHeight; i++ {
		b, err := GetBlockTime(i)
		if err != nil {
			log.Fatal(err)
		}

		if b.BlockTime >= BIP16target {
			log.Printf("Block: %s height: %d time: %d which has >= BIP16 target timestamp %d", b.Hash, i, b.BlockTime, BIP16target)
			break
		}
		if verbose {
			log.Printf("Block: %s height: %d time: %d\n", b.Hash, i, b.BlockTime)
		}
	}
}
