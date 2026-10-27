package constant

import (
	"encoding/json"
	"net/netip"
	"strings"
	"testing"
)

func TestMappedHostIsOmittedFromJSONUnlessSet(t *testing.T) {
	m := Metadata{Host: "example.com", DstIP: netip.MustParseAddr("192.0.2.1")}

	plain, err := json.Marshal(&m)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(plain), "mappedHost") {
		t.Fatalf("a connection without a reverse-map guess must serialize exactly as before: %s", plain)
	}

	m.MappedHost = true
	marked, err := json.Marshal(&m)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(marked), `"mappedHost":true`) {
		t.Fatalf("a marked connection must expose the marker: %s", marked)
	}

	if !m.Clone().MappedHost {
		t.Fatal("Clone must carry the marker")
	}
}
