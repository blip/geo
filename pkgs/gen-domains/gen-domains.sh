# shellcheck shell=bash
set -euo pipefail

domain_list_community_dir=$1/data
outDir=$2

declare -A visited
print_file() {
	local name=$1
	local filter_attrs=${2-}

	if [[ -n ${visited[$name]-} ]]; then
		return
	fi
	visited[$name]=1

	while read -r line; do
		line=$(sed -E 's/^[[:space:]]+|[[:space:]]+$|[[:space:]]*#.*//g' <<<"$line")
		if [[ -z $line ]]; then
			continue
		fi
		line=${line,,}
		if [[ $line =~ ^include:[[:space:]]*([^:@[:space:]]+) ]]; then
			print_file "${BASH_REMATCH[1]}" "$filter_attrs"
			continue
		fi

		local type val attrs
		if [[ $line =~ ^([^:@[:space:]]+)(:([^:@[:space:]]+))?([[:space:]]*(@.*))? ]]; then
			if [[ -z ${BASH_REMATCH[3]} ]]; then
				type=domain
				val=${BASH_REMATCH[1]}
				attrs=${BASH_REMATCH[5]}
			else
				type=${BASH_REMATCH[1]}
				if [[ $type = full ]]; then
					type=domain
				fi
				val=${BASH_REMATCH[3]}
				attrs=${BASH_REMATCH[5]}
			fi
		else
			echo >&2 "line with unknown format: $line"
			return 1
		fi

		if [[ -z $filter_attrs ]]; then
			echo "$type:$val"
			continue
		fi
		if [[ -z $attrs ]]; then
			continue
		fi

		for filter_attr in $filter_attrs; do
			for attr in $attrs; do
				if [[ $attr = "$filter_attr" ]]; then
					echo "$type:$val"
					continue 3
				fi
			done
		done
	done <"$domain_list_community_dir/$name"
}

customize_domains=$(
	cat <<'EOF'
FILENAME == "domains.custom" && /[^[:space:]]/ {
  gsub(/^[[:space:]]+|[[:space:]]*#.*|[[:space:]]+$/, "");
  if (tolower($0) == "[add]") { mode="add"; next }
  if (tolower($0) == "[remove]") { mode="remove"; next }

  rule = "domain:" tolower($0)
  if (mode == "remove") { removes[rule] = 1; next }
  print rule
}

FILENAME == "-" {
  if ($0 in removes) { next }
  print
}
EOF
)

{
	curl --disable --fail --silent --show-error --location --parallel \
		https://raw.githubusercontent.com/felixonmars/dnsmasq-china-list/master/accelerated-domains.china.conf \
		https://raw.githubusercontent.com/felixonmars/dnsmasq-china-list/master/google.china.conf \
		https://raw.githubusercontent.com/felixonmars/dnsmasq-china-list/master/apple.china.conf |
		awk -F/ -v IGNORECASE=1 '/^[[:space:]]*server=/ { gsub(/^[[:space:]]+|[[:space:]]+$/, "", $2); print "domain:" tolower($2) }'

	print_file cn
	print_file 'geolocation-!cn' @cn
} | awk "$customize_domains" domains.custom - | sort -u | convert-domains "$outDir"
