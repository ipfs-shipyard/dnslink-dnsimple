package main

import (
	"fmt"
	"os"
	"errors"
	"io/ioutil"
	"strings"
	"flag"
	logging "log"

	dns "github.com/dnsimple/dnsimple-go/dnsimple"
)

var log *logging.Logger

type Args struct{
	Token   string
	Domain  string
	RecName string
	Link    string
	Verbose bool
	TTL     int
}

const (
	DnslinkPrefix = "dnslink="
	DefaultTTL    = 60
	TXT           = "TXT"
)

var Usage = `
USAGE
		dnslink-dnsimple -t <api-token> -d <domain-name> [-r <record-name>] -l <dnslink-value>

OPTIONS
		-t, --token <string>    dnsimple api token (required)
		-l, --link  <string>    dnslink value, eg. ipfs path (required)
		-d, --domain <string>   dnsimple domain name (required)
		-r, --record <string>   domain record name
		-v, --verbose           show logging output
		-h, --help              show this documentation
		--ttl <int>             set the ttl of the record (default: 60)

EXAMPLES
		dnslink-dnsimple -t $(cat dnsimple-token) -d domain.net -r _dnslink -l /ipns/ipfs.io
`

func parseArgs() (Args, error) {
	var a Args
	flag.StringVar(&a.Token, "t", "", "")
	flag.StringVar(&a.Token, "token", "", "")
	flag.StringVar(&a.Domain, "d", "", "")
	flag.StringVar(&a.Domain, "domain", "", "")
	flag.StringVar(&a.RecName, "r", "", "")
	flag.StringVar(&a.RecName, "record", "", "")
	flag.StringVar(&a.Link, "l", "", "")
	flag.StringVar(&a.Link, "link", "", "")
	flag.BoolVar(&a.Verbose, "v", false, "")
	flag.BoolVar(&a.Verbose, "verbose", false, "")
	flag.IntVar(&a.TTL, "ttl", DefaultTTL, "")

  flag.Usage = func() {
    fmt.Fprintf(os.Stderr, Usage)
  }
	flag.Parse()
	if a.Token == "" || a.Domain == "" || a.Link == "" {
		return a, errors.New("token, record, and link arguments required")
	}
	return a, nil
}

func main() {
	args, err := parseArgs()
	if err != nil {
    fmt.Fprintln(os.Stderr, "error:", err)
    fmt.Fprintln(os.Stderr, Usage)
    os.Exit(-1)
	}

	if args.Verbose {
		log = logging.New(os.Stderr, "", 0)
	} else {
		log = logging.New(ioutil.Discard, "", 0)
	}

  if err := errMain(args); err != nil {
    fmt.Fprintln(os.Stderr, "error:", sanitizeErr(args.Token, err))
    os.Exit(-1)
  }
}

func sanitizeErr(token string, err error) string {
	return strings.Replace(fmt.Sprintf("%s", err), token, "", -1)
}

func errMain(args Args) error {
	client := dns.NewClient(dns.NewOauthTokenCredentials(args.Token))

	// get the account responsible for zone, and the dnslink record if there is one.
	acc, oldR, err := findAccountAndRecord(client, args)
	if err != nil {
		return err
	}

	// Create or update the _dnslink record.
	var newR *dns.ZoneRecord
	if oldR == nil {
		newR, err = createRecord(client, args, acc)
	} else { // got an old record to update
		newR, err = updateRecord(client, args, acc, oldR)
	}
	if err != nil {
		return err
	}

	fmt.Printf("updated TXT %s.%s to %s\n", newR.Name, args.Domain, newR.Content)
	return nil
}

// findAccountAndRecord finds the right account that can update the desired record
// if we find the specific record to update, return it too.
func findAccountAndRecord(c *dns.Client, args Args) (acc string, rec *dns.ZoneRecord, err error) {
	acc, recs, err := findAccountForZone(c, args)
	if err != nil {
		return "", nil, err
	}

	// find record to replace, if any
	var oldR *dns.ZoneRecord
	for _, r := range recs {
		if strings.HasPrefix(r.Content, "dnslink") {
			oldR = &r
		}
	}
	if oldR == nil {
		log.Println("existing dnslink record: not found")
	} else {
		log.Println("existing dnslink record:", oldR)
	}

	return acc, oldR, nil
}

func findAccountForZone(c *dns.Client, args Args) (acc string, recs []dns.ZoneRecord, err error) {
	// Loop over all accounts to find the one containing the relevant zone.
	accopts := &dns.ListOptions{}
	accounts, err := c.Accounts.ListAccounts(accopts)
	if err != nil {
		return "", nil, err
	}
	log.Printf("found %d accounts for token: %s\n", len(accounts.Data), accounts.Data)

	for _, a := range accounts.Data {
		acc := fmt.Sprintf("%d", a.ID)
		zropts := &dns.ZoneRecordListOptions{Name: args.RecName, Type: TXT}
		recs, err := c.Zones.ListRecords(acc, args.Domain, zropts)
		if err != nil {
			log.Printf("error listing records of account %s: %s", acc, err)
			continue
		}

		records := recs.Data
		log.Printf("found domain %s in account %s with %d records\n",
				args.Domain, acc, len(records))
		return acc, records, nil
	}

	return "", nil, fmt.Errorf("did not find account for: %s", args.Domain)
}


func createRecord(c *dns.Client, args Args, acc string) (newR *dns.ZoneRecord, err error) {
	newR = newRecord(args)
	log.Println("will CreateRecord:", newR)
	res, err := c.Zones.CreateRecord(acc, args.Domain, *newR)
	if err != nil {
		return nil, fmt.Errorf("CreateRecord: %v", err)
	}
	log.Println("did CreateRecord:", res.Data)
	return res.Data, nil
}

func updateRecord(c *dns.Client, args Args, acc string, oldR *dns.ZoneRecord) (newR *dns.ZoneRecord, err error) {
	// just update value
	oldR.Content = DnslinkPrefix + args.Link

	// we only want to change the value.
	// looking at the API, it should only update what we ask it to update.
	// (i was getting "regions not available in your plan")
	newR = newRecord(args)
	newR.ID = oldR.ID
	newR.ZoneID = oldR.ZoneID
	newR.ParentID = oldR.ParentID

	log.Println("will UpdateRecord:", newR)
	res, err := c.Zones.UpdateRecord(acc, args.Domain, newR.ID, *newR)
	if err != nil {
		return nil, fmt.Errorf("UpdateRecord: %v", err)
	}
	log.Println("did UpdateRecord:", res.Data)
	return res.Data, nil
}

func newRecord(args Args) *dns.ZoneRecord {
	return &dns.ZoneRecord{
		Type:    TXT,
		Name:    args.RecName,
		Content: DnslinkPrefix + args.Link,
		TTL:     args.TTL,
	}
}
