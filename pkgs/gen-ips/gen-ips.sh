# shellcheck shell=bash
set -euo pipefail

if [[ $# != 1 && $# != 2 ]]; then
	echo >&2 'usage: gen-ips OUTPUT_DIR [GeoLite2-Country.mmdb]'
	exit 1
fi

out_dir=$1
database=${2-}
if [[ -z $database ]]; then
	: "${MAXMIND_ACCOUNT_ID:?Set MAXMIND_ACCOUNT_ID to your MaxMind account ID}"
	: "${MAXMIND_LICENSE_KEY:?Set MAXMIND_LICENSE_KEY to a MaxMind database download license key}"
	tmp_dir=$(mktemp -d)
	trap 'rm -rf "$tmp_dir"' EXIT
	database=$tmp_dir/GeoLite2-Country.mmdb
	# MaxMind's official client handles authentication, downloads and verification.
	# Credentials are supplied at runtime, never written into the Nix store.
	GEOIPUPDATE_ACCOUNT_ID=$MAXMIND_ACCOUNT_ID \
		GEOIPUPDATE_LICENSE_KEY=$MAXMIND_LICENSE_KEY \
		GEOIPUPDATE_EDITION_IDS=GeoLite2-Country \
		geoipupdate --config-file /dev/null --database-directory "$tmp_dir"
fi

convert-ips "$database" "$out_dir"
