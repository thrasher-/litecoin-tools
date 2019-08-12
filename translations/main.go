package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"io"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

var (
	Translations map[string]string
	Exclusions   []string
	Replacements int
)

func BuildTranslationsAndExclusions() {
	Translations = make(map[string]string)
	Translations["BITCOIN"] = "LITECOIN"
	Translations["Bitcoin"] = "Litecoin"
	Translations["bitcoin"] = "litecoin"
	Translations["Bitcion"] = "Litecion"
	Translations["BTC"] = "LTC"
	Translations["btc"] = "ltc"
	Translations["بيتكوين"] = "Litecoin"
	Translations["Біткойн"] = "Litecoin"
	Translations["біткойн"] = "litecoin"
	Translations["биткойн"] = "Litecoin"
	Translations["Биткойн"] = "Litecoin"
	Translations["Bitconi"] = "Liteconi"
	Translations["Bitcoini"] = "Litecoini"
	Translations["הביטקוין"] = "לייטקוין"
	Translations["ביטקוין"] = "ללייטקוין"
	Translations["비트코인"] = "라이트코인" // KR: bitcoins -> litecoins
	Translations["بیت‌کوین"] = "Litecoin"
	Translations["بیت کوین"] = "litecoin"
	Translations["बिटकोइन"] = "Litecoin"
	Translations["比特币"] = "莱特币"
	Translations["Bitmon"] = "Litecoin"
	Translations["Bitmono"] = "Litecoin"
	Translations["bitmona"] = "litecoin"
	Translations["비트코인을"] = "라이트코인을" // KR: Bitcoin -> Litecoin
	Translations["비트코인들을"] = "라이트코인들을" // KR: BITCOINS -> LITECOINS
	Translations["БИТКОЙНЫ"] = "ЛАЙТКОИНЫ" // RU: BITCOINS -> LITECOINS
	Translations["ЛАЙТКОИНЫ"] = "лайткоин" // RU: Bitcoin -> Litecoin
	Exclusions = append(Exclusions, []string{"The Bitcoin Core Developers", "BitcoinGUI", "bitcoin-core", ".cpp"}...)
}

func ContainsTranslationString(input []byte) bool {
	inputStr := string(input)
	for x, _ := range Translations {
		if strings.Contains(inputStr, x) {
			return true
		}
	}
	return false
}

func ContainsExclusionString(input []byte) bool {
	inputStr := string(input)
	for _, x := range Exclusions {
		if strings.Contains(inputStr, x) {
			return true
		}
	}
	return false
}

func ProcessAndModifyLine(input []byte) []byte {
	if ContainsExclusionString(input) {
		return input
	}

	if !ContainsTranslationString(input) {
		return input
	}

	output := []byte{}
	for x, y := range Translations {
		if strings.Contains(string(input), x) {
			Replacements++
			output = bytes.Replace(input, []byte(x), []byte(y), -1)
		}
	}
	return output
}

func ProcessFile(file string) error {
	f, err := os.Open(file)
	if err != nil {
		return err
	}

	log.Printf("Processing %s..", file)
	r := bufio.NewReaderSize(f, 4*1024)
	var outputFile []byte
	line, prefix, err := r.ReadLine()

	for err == nil && !prefix {
		result := ProcessAndModifyLine(line)
		outputFile = append(outputFile, result...)
		outputFile = append(outputFile, []byte(GetOSNewLine())...)
		line, prefix, err = r.ReadLine()
	}

	if prefix {
		return errors.New("Buffer size is too small.")
	}

	if err != io.EOF {
		return err
	}

	if !strings.Contains(file, "bitcoin_de.ts") { // uglyyyy
		outputFile = outputFile[:len(outputFile)-len(GetOSNewLine())]
		err = ioutil.WriteFile(file, outputFile, 0644)
	}
	if err != nil {
		return err
	}

	return nil
}

func GetOSNewLine() string {
	switch runtime.GOOS {
	case "Windows":
		return "\r\n"
	case "darwin":
		return "\r"
	default:
		return "\n"
	}
}

func GetOSPathSlash() string {
	if runtime.GOOS == "windows" {
		return "\\"
	}
	return "/"
}

func main() {
	var srcDir string
	flag.StringVar(&srcDir, "srcdir", "", "The source dir of the locale files.")
	flag.Parse()

	if srcDir == "" {
		log.Fatal("A source directory of the locale files must be specified.")
	}

	files, err := ioutil.ReadDir(srcDir)
	if err != nil {
		log.Fatal(err)
	}

	BuildTranslationsAndExclusions()

	for _, file := range files {
		if filepath.Ext(file.Name()) == ".qm" {
			continue
		}

		filePath := srcDir + GetOSPathSlash() + file.Name()
		err := ProcessFile(filePath)
		if err != nil {
			log.Printf("\nError processing file %s. Error: %s", filePath, err)
			continue
		}
		log.Println("OK")
	}

	log.Printf("Done! Replaced %d occurences.\n", Replacements)
}
