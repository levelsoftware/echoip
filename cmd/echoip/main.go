package main

import (
	"flag"
	"log"
	"strings"

	"os"

	"github.com/levelsoftware/echoip/cache"
	"github.com/levelsoftware/echoip/http"
	"github.com/levelsoftware/echoip/iputil"
	"github.com/levelsoftware/echoip/iputil/geo"
	"github.com/levelsoftware/echoip/iputil/ipstack"
	parser "github.com/levelsoftware/echoip/iputil/paser"
	ipstackApi "github.com/qioalice/ipstack"
)

type multiValueFlag []string

func (f *multiValueFlag) String() string {
	return strings.Join([]string(*f), ", ")
}

func (f *multiValueFlag) Set(v string) error {
	*f = append(*f, v)
	return nil
}

func init() {
	log.SetPrefix("echoip: ")
	log.SetFlags(log.Lshortfile)
}

func main() {
	var ipstackApiKey string
	flag.StringVar(&ipstackApiKey, "S", "", "IP Stack API Key")

	service := flag.String("d", "geoip", "Which database to use, 'ipstack' or 'geoip'")
	ipStackEnableSecurityModule := flag.Bool("x", false, "Enable security module for IP Stack ( must have security module, aka. non-free account. )")
	ipStackUseHttps := flag.Bool("h", false, "Use HTTPS for IP Stack ( only non-free accounts )")

	countryFile := flag.String("f", "", "Path to GeoIP country database")
	cityFile := flag.String("c", "", "Path to GeoIP city database")
	asnFile := flag.String("a", "", "Path to GeoIP ASN database")
	listen := flag.String("l", ":8080", "Listening address")
	reverseLookup := flag.Bool("r", false, "Perform reverse hostname lookups")
	portLookup := flag.Bool("p", false, "Enable port lookup")
	template := flag.String("t", "html", "Path to template dir")
	profile := flag.Bool("P", false, "Enables profiling handlers")
	sponsor := flag.Bool("s", false, "Show sponsor logo")

	var headers multiValueFlag
	flag.Var(&headers, "H", "Header to trust for remote IP, if present (e.g. X-Real-IP)")

	var redisCacheUrl string
	flag.StringVar(&redisCacheUrl, "C", "", "Redis cache URL ( redis://localhost:6379?password=hello&protocol=3 )")

	flag.Parse()

	if len(flag.Args()) != 0 {
		flag.Usage()
		return
	}

	var parser parser.Parser
	if *service == "geoip" {
		log.Print("Using GeoIP for IP database")
		geo, err := geo.Open(*countryFile, *cityFile, *asnFile)
		if err != nil {
			log.Fatal(err)
		}
		parser = &geo
	}

	if *service == "ipstack" {
		log.Print("Using GeoIP for IP database")
		if *ipStackEnableSecurityModule {
			log.Print("Enable Security Module ( Requires Professional Plus account )")
		}
		enableSecurity := ipstackApi.ParamEnableSecurity(*ipStackEnableSecurityModule)
		apiKey := ipstackApi.ParamToken(ipstackApiKey)
		useHttps := ipstackApi.ParamUseHTTPS(*ipStackUseHttps)
		if *ipStackUseHttps {
			log.Print("Use IP Stack HTTPS API ( Requires non-free account )")
		}
		if err := ipstackApi.Init(apiKey, enableSecurity, useHttps); err != nil {
			log.Fatal(err)
		}
		ips := ipstack.IPStack{}
		parser = &ips
	}

	cache, err := cache.RedisCache(redisCacheUrl)
	if err != nil {
		log.Fatal(err)
	}

	server := http.New(parser, &cache, *profile)
	server.IPHeaders = headers
	if _, err := os.Stat(*template); err == nil {
		server.Template = *template
	} else {
		log.Printf("Not configuring default handler: Template not found: %s", *template)
	}
	if *reverseLookup {
		log.Println("Enabling reverse lookup")
		server.LookupAddr = iputil.LookupAddr
	}
	if *portLookup {
		log.Println("Enabling port lookup")
		server.LookupPort = iputil.LookupPort
	}
	if *sponsor {
		log.Println("Enabling sponsor logo")
		server.Sponsor = *sponsor
	}
	if len(headers) > 0 {
		log.Printf("Trusting remote IP from header(s): %s", headers.String())
	}
	if *profile {
		log.Printf("Enabling profiling handlers")
	}
	log.Printf("Listening on http://%s", *listen)
	if err := server.ListenAndServe(*listen); err != nil {
		log.Fatal(err)
	}
}
