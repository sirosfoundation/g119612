package main

import (
	"crypto/x509"
	"encoding/base64"
	"flag"
	"fmt"
	"os"

	"github.com/sirosfoundation/g119612/pkg/etsi119612"
)

var Version = "1.0.0"

var (
	urlVar = ""
	x5cVar = ""
)

func init() {
	flag.StringVar(&urlVar, "url", "", "URL of a trust status list")
	flag.StringVar(&x5cVar, "x5c", "", "base64 encoded certificate (single line)")
}

func Usage(cmd string) {
	fmt.Printf(`
Usage: %s
	show --url <url>
	validate --url <url> --x5c <base64 encoded certificate>

`, cmd)
}

func main() {
	validateCmd := flag.NewFlagSet("validate", flag.ExitOnError)
	validateUrl := validateCmd.String("url", "", "source url")
	validateX5C := validateCmd.String("x5c", "", "base64 encoded certificate")

	showCmd := flag.NewFlagSet("show", flag.ExitOnError)
	showUrl := showCmd.String("url", "", "source url")

	if len(os.Args) < 2 {
		Usage(os.Args[0])
		os.Exit(1)
	}

	switch os.Args[1] {
	case "validate":
		validateCmd.Parse(os.Args[2:])
		tsl, err := etsi119612.FetchTSL(*validateUrl)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}

		data, err := base64.StdEncoding.DecodeString(*validateX5C)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		cert, err := x509.ParseCertificate(data)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}

		pool := tsl.ToCertPool(etsi119612.PolicyAll)
		_, err = cert.Verify(x509.VerifyOptions{Roots: pool})
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		fmt.Print("OK!\n")
	case "show":
		showCmd.Parse(os.Args[2:])
                fmt.Printf("fetching %s\n",*showUrl)
		tsl, err := etsi119612.FetchTSL(*showUrl)
		if err != nil {
			fmt.Printf("error: %v\n", err)
			return
		}
		fmt.Printf("%s\n", tsl)
		for _, tsp := range tsl.StatusList.TslTrustServiceProviderList.TslTrustServiceProvider {
			name_en := etsi119612.FindByLanguage(tsp.TslTSPInformation.TSPName, "en", "Unknown tsp")
			s_count := len(tsp.TslTSPServices.TslTSPService)
			plural := ""
			if s_count > 1 {
				plural = "s"
			}

			fmt.Printf("  - \"%s\" (%d service%s)\n", name_en, s_count, plural)
		}
	}

}
