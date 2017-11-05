package main

import (
	"fmt"
	"os"
	"strconv"
	// "strings"

	dnsimple "github.com/dnsimple/dnsimple-go/dnsimple"
)

func main() {
	token := os.Getenv("DNSIMPLE_TOKEN")
	if len(os.Args) < 3 || len(os.Args) > 4 || token == "" {
		fmt.Printf("Usage: dnslink-dnsimple DOMAIN PATH [RECORD||=_dnslink]\n")
		fmt.Printf("Example: dnslink-dnsimple example.com /ipfs/QmFoo\n")
		fmt.Printf("\n")
		fmt.Printf("The DNSIMPLE_TOKEN environment variable is required.\n")
		fmt.Printf("\n")
		os.Exit(1)
	}
	zonename := os.Args[1]
	path := os.Args[2]
	recordname := "_dnslink"
	if len(os.Args) == 4 {
		recordname = os.Args[3]
	}

	client := dnsimple.NewClient(dnsimple.NewOauthTokenCredentials(token))

	whoami, err := client.Identity.Whoami()
	if err != nil {
		fmt.Printf("error in whoami: %s\n", err)
		os.Exit(1)
	}
	accountID := strconv.Itoa(whoami.Data.Account.ID)

	opts := &dnsimple.ZoneRecordListOptions{Name: recordname, Type: "TXT"}
	records, err := client.Zones.ListRecords(accountID, zonename, opts)
	if err != nil {
		fmt.Printf("error in listRecords: %s\n", err)
		os.Exit(1)
	}

	var updatedRecord *dnsimple.ZoneRecord
	if len(records.Data) == 0 {
		record := &dnsimple.ZoneRecord{
			Type:    "TXT",
			Name:    recordname,
			Content: "dnslink=" + path,
			TTL:     120,
		}
		response, err := client.Zones.CreateRecord(accountID, zonename, *record)
		if err != nil {
			fmt.Printf("error in createRecord: %s\n", err)
			os.Exit(1)
		}
		updatedRecord = response.Data
	} else {
		record := records.Data[0]
		record.Content = "dnslink=" + path
		response, err := client.Zones.UpdateRecord(accountID, zonename, record.ID, record)
		if err != nil {
			fmt.Printf("error in updateRecord: %s\n", err)
			os.Exit(1)
		}
		updatedRecord = response.Data
	}

	fmt.Printf("updated TXT %s.%s to %s\n", recordname, zonename, updatedRecord.Content)
}
