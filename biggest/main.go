package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

const (
	RPC_PORT     = 9332
	RPC_USERNAME = "user"
	RPC_PASSWORD = "pass"
	RPC_HOST     = "127.0.0.1"
)

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
	Hash    string
	Size    int
	TXCount int
}

func GetBlock(block int) (BlockInfo, error) {
	bi := BlockInfo{}
	blockHash, err := GetBlockHash(block)
	if err != nil {
		return bi, err
	}

	result, err := SendRPCRequest("getblock", blockHash)
	if err != nil {
		return bi, err
	}

	result = result["result"].(map[string]interface{})
	bi.Hash = blockHash
	bi.Size = int(result["size"].(float64))
	bi.TXCount = len(result["tx"].([]interface{}))
	return bi, nil
}

type BiggestBlockInfo struct {
	BiggestBlock struct {
		BlockHeight int
		BlockHash   string
		BlockSize   int
	}
	BiggestBlockTX struct {
		BlockHeight int
		BlockHash   string
		TXCount     int
	}
}

func main() {
	currentHeight, err := GetBlockHeight()
	if err != nil {
		log.Fatal(err)
	}

	log.Printf("Current block height: %d\n", currentHeight)
	log.Println("Checking for biggest block size and largest tx amount within a block.. (this may take a few minutes)")
	bbi := BiggestBlockInfo{}

	for i := 0; i < currentHeight; i++ {
		bi, err := GetBlock(i)
		if err != nil {
			log.Fatal(err)
		}

		if bi.Size > bbi.BiggestBlock.BlockSize {
			bbi.BiggestBlock.BlockHash = bi.Hash
			bbi.BiggestBlock.BlockHeight = i
			bbi.BiggestBlock.BlockSize = bi.Size
		}

		if bi.TXCount > bbi.BiggestBlockTX.TXCount {
			bbi.BiggestBlockTX.BlockHash = bi.Hash
			bbi.BiggestBlockTX.BlockHeight = i
			bbi.BiggestBlockTX.TXCount = bi.TXCount
		}

		if i%50000 == 0 && i > 0 {
			progress := (float64(i) / float64(currentHeight)) * 100 / 1
			log.Printf("%.2f%% complete. %d/%d", progress, i, currentHeight)
		}
	}

	log.Printf("Biggest block is %v\n", bbi.BiggestBlock)
	log.Printf("Biggest tx block is %v\n", bbi.BiggestBlockTX)
}
