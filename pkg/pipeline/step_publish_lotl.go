package pipeline

import (
	"fmt"

	"github.com/sirosfoundation/g119612/pkg/etsi119602"
)

// PublishLoTL is an alias for PublishLoTE. The unified PublishLoTE function
// publishes both LoTEs and LoTLs from context.
// See PublishLoTE for full documentation.
var PublishLoTL = PublishLoTE

type lotlFormat struct {
	json bool
	xml  bool
}

func parseLotlFormat(args []string) lotlFormat {
	for _, arg := range args {
		switch arg {
		case "json-only":
			return lotlFormat{json: true, xml: false}
		case "xml-only":
			return lotlFormat{json: false, xml: true}
		}
	}
	return lotlFormat{json: true, xml: true}
}

func lotlFilename(lotl *etsi119602.ListOfTrustedLists, index int) string {
	if lotl.SchemeInformation.Territory != "" {
		return fmt.Sprintf("list_of_trusted_lists-%s", lotl.SchemeInformation.Territory)
	}
	if index == 0 {
		return "list_of_trusted_lists"
	}
	return fmt.Sprintf("list_of_trusted_lists-%d", index)
}

// filterSignerArgs removes format flags from the args to pass to signer creation.
func filterSignerArgs(args []string) []string {
	var filtered []string
	for _, arg := range args {
		switch arg {
		case "json-only", "xml-only":
			continue
		default:
			filtered = append(filtered, arg)
		}
	}
	return filtered
}

func init() {
	RegisterFunction("publish-lotl", PublishLoTL)
}
