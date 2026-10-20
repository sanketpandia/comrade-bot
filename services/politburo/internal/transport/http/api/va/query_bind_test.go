package va

import (
	"net/url"
	"testing"

	"github.com/oapi-codegen/runtime"
	politburoapi "infinite-experiment/politburo/internal/api/generated/politburo"
)

func TestGetVAConfigQueryTypeBinding(t *testing.T) {
	u, err := url.Parse("http://localhost/api/v1/va/config?configType=BASIC")
	if err != nil {
		t.Fatal(err)
	}
	var params politburoapi.GetVAConfigParams
	err = runtime.BindQueryParameterWithOptions("form", true, true, "configType", u.Query(), &params.ConfigType, runtime.BindQueryParameterOptions{Type: "string", Format: ""})
	if err != nil {
		t.Fatalf("bind: %v", err)
	}
	if params.ConfigType != politburoapi.BASIC {
		t.Fatalf("configType = %q", params.ConfigType)
	}
}
