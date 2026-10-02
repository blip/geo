package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/oschwald/maxminddb-golang"
	"github.com/sagernet/sing-box/common/srs"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
	"go4.org/netipx"
)

var countries = []string{"GB", "US"}

// Use the location country, not the country where the ISP is registered.
type countryRecord struct {
	Country struct {
		ISOCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
}

func run(database, outDir string) error {
	data, err := os.ReadFile(database)
	if err != nil {
		return err
	}
	db, err := maxminddb.FromBytes(data)
	if err != nil {
		return err
	}
	defer db.Close()
	if db.Metadata.DatabaseType != "GeoLite2-Country" {
		return fmt.Errorf("expected GeoLite2-Country database, got %q", db.Metadata.DatabaseType)
	}
	if err := db.Verify(); err != nil {
		return err
	}

	sets := make(map[string]*netipx.IPSetBuilder)
	for _, country := range countries {
		sets[country] = new(netipx.IPSetBuilder)
	}
	// The MMDB aliases IPv4 into IPv6 transition ranges; emit each real network once.
	networks := db.Networks(maxminddb.SkipAliasedNetworks)
	for networks.Next() {
		var record countryRecord
		network, err := networks.Network(&record)
		if err != nil {
			return err
		}
		builder := sets[record.Country.ISOCode]
		if builder == nil {
			continue
		}
		prefix, ok := netipx.FromStdIPNet(network)
		if !ok {
			return fmt.Errorf("invalid network: %s", network)
		}
		builder.AddPrefix(prefix)
	}
	if err := networks.Err(); err != nil {
		return err
	}
	if err := writeCountries(outDir, sets); err != nil {
		return err
	}
	metadata, err := json.MarshalIndent(map[string]any{
		"database":      db.Metadata.DatabaseType,
		"build_time":    time.Unix(int64(db.Metadata.BuildEpoch), 0).UTC().Format(time.RFC3339),
		"sha256":        fmt.Sprintf("%x", sha256.Sum256(data)),
		"country_field": "country.iso_code",
		"countries":     countries,
		"attribution":   "This product includes GeoLite2 data created by MaxMind, available from https://www.maxmind.com.",
		"license":       "https://creativecommons.org/licenses/by-sa/4.0/",
	}, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(outDir, "ips-metadata.json"), append(metadata, '\n'), 0o644)
}

func writeCountries(outDir string, builders map[string]*netipx.IPSetBuilder) error {
	// Validate all lists before writing anything, so a bad database fails the release.
	lists := make(map[string][]string)
	for _, country := range countries {
		set, err := builders[country].IPSet()
		if err != nil {
			return err
		}
		var ipv4, ipv6 int
		for _, prefix := range set.Prefixes() {
			lists[country] = append(lists[country], prefix.String())
			if prefix.Addr().Is4() {
				ipv4++
			} else {
				ipv6++
			}
		}
		if ipv4 == 0 || ipv6 == 0 {
			return fmt.Errorf("%s must contain both IPv4 and IPv6 networks", country)
		}
		log.Printf("%s: %d IPv4 and %d IPv6 prefixes", country, ipv4, ipv6)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}
	for _, country := range countries {
		cidrs := lists[country]
		var binary bytes.Buffer
		err := srs.Write(&binary, option.PlainRuleSet{
			Rules: []option.HeadlessRule{{
				Type:           C.RuleTypeDefault,
				DefaultOptions: option.DefaultHeadlessRule{IPCIDR: cidrs},
			}},
		}, 2) // Same format as domains-cn.srs, supported by our sing-box 1.10.5.
		if err != nil {
			return err
		}
		base := filepath.Join(outDir, "ips-"+strings.ToLower(country))
		if err := os.WriteFile(base+".srs", binary.Bytes(), 0o644); err != nil {
			return err
		}
		if err := os.WriteFile(base+".txt", []byte(strings.Join(cidrs, "\n")+"\n"), 0o644); err != nil {
			return err
		}
	}
	return nil
}

func main() {
	if len(os.Args) != 3 {
		log.Fatal("usage: convert-ips GeoLite2-Country.mmdb OUTPUT_DIR")
	}
	if err := run(os.Args[1], os.Args[2]); err != nil {
		log.Fatal(err)
	}
}
