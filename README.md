# dnslink-dnsimple

Update dnslink TXT records in DNSimple

## Usage

```
> dnslink-dnsimple -h
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
```
