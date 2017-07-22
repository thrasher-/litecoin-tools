package main

import (
	"encoding/hex"
	"encoding/json"
	"flag"
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

func SendRPCRequest(method string, req []interface{}) (map[string]interface{}, error) {
	var params []interface{}
	if req != nil {
		params = append(params, req...)
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

func GetBlockHash(block int) (string, error) {
	var request []interface{}
	request = append(request, block)
	result, err := SendRPCRequest("getblockhash", request)
	if err != nil {
		return "", err
	}
	return result["result"].(string), nil
}

func GetBlockHex(block int) (string, error) {
	blockHash, err := GetBlockHash(block)
	if err != nil {
		return "", err
	}

	var request []interface{}
	request = append(request, blockHash)
	request = append(request, false)
	result, err := SendRPCRequest("getblock", request)
	if err != nil {
		return "", err
	}

	hexData := result["result"].(string)
	return hexData, nil
}

func main() {
	var blockNum = flag.Int("block", 0, "block number to hexdump")
	flag.Parse()

	if *blockNum < 0 {
		flag.Usage()
		return
	}

	blockHex, err := GetBlockHex(*blockNum)
	if err != nil {
		log.Fatal(err)
	}

	data, err := hex.DecodeString(blockHex)
	if err != nil {
		log.Fatal(err)
	}

	outputFile := fmt.Sprintf("block%d.raw", *blockNum)
	err = ioutil.WriteFile(outputFile, data, 0644)
	if err != nil {
		log.Fatal(err)
	}

	log.Println("Wrote output file to", outputFile)
}
