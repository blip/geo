package main

import (
	"bufio"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/sagernet/sing-box/common/srs"
	C "github.com/sagernet/sing-box/constant"
	"github.com/sagernet/sing-box/option"
)

type Rule struct {
	Type  string
	Value string
}

func (rule Rule) String() string {
	return fmt.Sprintf("%s:%s", rule.Type, rule.Value)
}

type DomainRules map[string]struct{}

func (m DomainRules) isRuleRedundant(rule Rule) bool {
	switch rule.Type {
	case "domain", "full":
		parts := strings.Split(rule.Value, ".")
		i := 0
		if rule.Type == "domain" {
			i = 1
		}
		for ; i < len(parts); i++ {
			domain := strings.Join(parts[i:], ".")
			if _, ok := m[domain]; ok {
				log.Printf("%s is made redundant by domain:%s\n", rule, domain)
				return true
			}
		}
		return false
	default:
		return false
	}
}

func run(r io.Reader, outDir string) error {
	domainRules := make(DomainRules)

	var rules []Rule
	scanner := bufio.NewScanner(r)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		if len(parts) != 2 {
			return errors.New(`line is not of format "type:value": ` + line)
		}
		typ := parts[0]
		val := parts[1]

		rules = append(rules, Rule{
			Type:  typ,
			Value: val,
		})
		if typ == "domain" {
			domainRules[val] = struct{}{}
		}
	}

	dedupedRules := make([]Rule, 0, len(rules))
	var domainValues []string
	var regexValues []string
	for _, rule := range rules {
		if domainRules.isRuleRedundant(rule) {
			continue
		}
		switch rule.Type {
		case "domain":
			domainValues = append(domainValues, rule.Value)
		case "regexp":
			regexValues = append(regexValues, rule.Value)
		default:
			return errors.New("unknown rule type: " + rule.String())
		}
		dedupedRules = append(dedupedRules, rule)
	}
	err := writeText(filepath.Join(outDir, "domains-cn"), dedupedRules)
	if err != nil {
		return err
	}
	return writeSrs(filepath.Join(outDir, "domains-cn.srs"), domainValues, regexValues)
}
func writeText(name string, rules []Rule) error {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()
	for _, rule := range rules {
		_, err = fmt.Fprintln(f, rule)
		if err != nil {
			return err
		}
	}
	return nil
}
func writeSrs(name string, domainValues, regexValues []string) (err error) {
	f, err := os.Create(name)
	if err != nil {
		return err
	}
	defer func() {
		cerr := f.Close()
		if err == nil {
			err = cerr
		}
	}()
	return srs.Write(f, option.PlainRuleSet{
		Rules: []option.HeadlessRule{{
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultHeadlessRule{
				DomainSuffix: domainValues,
			},
		}, {
			Type: C.RuleTypeDefault,
			DefaultOptions: option.DefaultHeadlessRule{
				DomainRegex: regexValues,
			},
		}},
	}, 2)
}

func main() {
	err := run(os.Stdin, os.Args[1])
	if err != nil {
		log.Fatal(err)
	}
}
