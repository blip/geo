package main

import (
	"bytes"
	"net/netip"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/sagernet/sing-box/common/srs"
	"go4.org/netipx"
)

func testBuilders() map[string]*netipx.IPSetBuilder {
	builders := make(map[string]*netipx.IPSetBuilder)
	for _, country := range countries {
		builders[country] = new(netipx.IPSetBuilder)
		for _, cidr := range []string{"192.0.2.0/25", "192.0.2.128/25", "2001:db8::/32"} {
			builders[country].AddPrefix(netip.MustParsePrefix(cidr))
		}
	}
	return builders
}

func TestBinaryAndTextCoverage(t *testing.T) {
	out := t.TempDir()
	if err := writeCountries(out, testBuilders()); err != nil {
		t.Fatal(err)
	}
	want := []string{"192.0.2.0/24", "2001:db8::/32"}
	for _, country := range countries {
		base := filepath.Join(out, "ips-"+strings.ToLower(country))
		text, err := os.ReadFile(base + ".txt")
		if err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(strings.Fields(string(text)), want) {
			t.Fatalf("incorrect merged coverage: %s", text)
		}
		binary, err := os.ReadFile(base + ".srs")
		if err != nil {
			t.Fatal(err)
		}
		decoded, err := srs.Read(bytes.NewReader(binary), true)
		if err != nil {
			t.Fatal(err)
		}
		if decoded.Version != 2 || len(decoded.Options.Rules) != 1 {
			t.Fatalf("unexpected SRS format: %+v", decoded)
		}
		if !reflect.DeepEqual([]string(decoded.Options.Rules[0].DefaultOptions.IPCIDR), want) {
			t.Fatal("binary coverage differs from text coverage")
		}
	}
}

func TestIncompleteDatabaseDoesNotReplaceLists(t *testing.T) {
	out := t.TempDir()
	name := filepath.Join(out, "ips-gb.srs")
	if err := os.WriteFile(name, []byte("previous list"), 0o644); err != nil {
		t.Fatal(err)
	}
	builders := testBuilders()
	builders["US"] = new(netipx.IPSetBuilder)
	if err := writeCountries(out, builders); err == nil {
		t.Fatal("expected an error for a missing country")
	}
	data, err := os.ReadFile(name)
	if err != nil || string(data) != "previous list" {
		t.Fatal("invalid input replaced the previous list")
	}
}

func TestInvalidDatabase(t *testing.T) {
	dir := t.TempDir()
	name := filepath.Join(dir, "invalid.mmdb")
	if err := os.WriteFile(name, []byte("not a MaxMind database"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := run(name, filepath.Join(dir, "output")); err == nil {
		t.Fatal("expected invalid database to fail")
	}
}
