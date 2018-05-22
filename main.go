package main

import (
	"fmt"
	"os"
	"strings"

	dnsimple "github.com/dnsimple/dnsimple-go/dnsimple"
)

func printError(token string, err string) {
	err = strings.Replace(err, token, "", -1)
	fmt.Println(err)
}

func main() {
	token := os.Getenv("DNSIMPLE_TOKEN")
	if len(os.Args) < 3 || len(os.Args) > 4 || token == "" {
		fmt.Printf("Usage: dnslink-dnsimple DOMAIN PATH [RECORD_NAME]\n")
		fmt.Printf("Example: dnslink-dnsimple example.com /ipfs/QmFoo\n")
		fmt.Printf("\n")
		fmt.Printf("The DNSIMPLE_TOKEN environment variable is required.\n")
		fmt.Printf("\n")
		os.Exit(1)
	}
	zonename := os.Args[1]
	path := os.Args[2]
	recordname := ""
	if len(os.Args) == 4 {
		recordname = os.Args[3]
	}

	client := dnsimple.NewClient(dnsimple.NewOauthTokenCredentials(token))

	// Loop over all accounts to find the one containing the relevant zone.
	accopts := &dnsimple.ListOptions{}
	accounts, err := client.Accounts.ListAccounts(accopts)
	if err != nil {
		printError(token, fmt.Sprintf("error in listAccounts: %s", err))
		os.Exit(1)
	}
	var account string
	var records []dnsimple.ZoneRecord
	for _, a := range accounts.Data {
		acc := string(a.ID)
		zropts := &dnsimple.ZoneRecordListOptions{Name: recordname, Type: "TXT"}
		recs, err := client.Zones.ListRecords(acc, zonename, zropts)
		if err != nil {
			continue
		}
		account = acc
		records = recs.Data
	}

	// Create or update the _dnslink record.
	var updatedRecord *dnsimple.ZoneRecord
	if len(records) == 0 {
		record := &dnsimple.ZoneRecord{
			Type:    "TXT",
			Name:    recordname,
			Content: "dnslink=" + path,
			TTL:     120,
		}
		response, err := client.Zones.CreateRecord(account, zonename, *record)
		if err != nil {
			printError(token, fmt.Sprintf("error in createRecord: %s", err))
			os.Exit(1)
		}
		updatedRecord = response.Data
	} else {
		record := records[0]
		record.Content = "dnslink=" + path
		response, err := client.Zones.UpdateRecord(account, zonename, record.ID, record)
		if err != nil {
			printError(token, fmt.Sprintf("error in updateRecord: %s", err))
			os.Exit(1)
		}
		updatedRecord = response.Data
	}

	fmt.Printf("updated TXT %s.%s to %s\n", recordname, zonename, updatedRecord.Content)
}
