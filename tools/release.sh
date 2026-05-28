#!/bin/sh

set -e

TMP_DIR="./_tmp"
CREATED_RCLONE_CONFIG=false

usage() {
    echo "Usage: $0 <action>"
    echo ""
    echo "Actions:"
    echo "  vehicles   release vehicle inventory data"
    echo "  circuits   release circuit inventory data"
    echo "  version    release version.json with current inventory versions"
    echo "  all        release vehicles, circuits, and version"
    echo 
    echo "Required environment variables:"
    echo "  VEHICLE_INVENTORY_PATH    vehicle inventory path"
    echo "  CIRCUIT_INVENTORY_PATH    circuit inventory path"
    echo "  R2_REMOTE                 rclone remote and bucket path (e.g. r2:mybucket)"
    echo "  CLOUDFLARE_API_TOKEN      Cloudflare API token (R2 write and CDN purge access)"
    echo "  CLOUDFLARE_ZONE_ID        Cloudflare zone ID for CDN cache purging"
    echo "  CLOUDFLARE_CDN_BASEURL    Cloudflare CDN base URL (e.g. https://static.example.com)"
    echo
    echo "Optional rclone configuration:"
    echo "  RCLONE_CONFIG             path to an existing rclone config file"
    echo "                            default: ~/.config/rclone/rclone.conf"
    echo
    echo "CI specific environment variables (no local rclone.conf file):"
    echo "  CLOUDFLARE_ACCOUNT_ID     Cloudflare account ID"
    echo "  S3_ACCESS_KEY_ID          Cloudflare R2 (S3) access key id"
    echo "  S3_SECRET_ACCESS_KEY      Cloudflare R2 (S3) secret access key"
}

validate_env_vehicle() {
    if [ -z "${VEHICLE_INVENTORY_PATH}" ]; then
        echo "Error: VEHICLE_INVENTORY_PATH is not set"
        exit 1
    fi
}

validate_env_circuit() {
    if [ -z "${CIRCUIT_INVENTORY_PATH}" ]; then
        echo "Error: CIRCUIT_INVENTORY_PATH is not set"
        exit 1
    fi
}

validate_env_cf() {
    if [ -z "${CLOUDFLARE_ZONE_ID}" ]; then
        echo "Error: CLOUDFLARE_ZONE_ID is not set"
        exit 1
    fi

    if [ -z "${CLOUDFLARE_API_TOKEN}" ]; then
        echo "Error: CLOUDFLARE_API_TOKEN is not set"
        exit 1
    fi

    if [ -z "${CLOUDFLARE_CDN_BASEURL}" ]; then
        echo "Error: CLOUDFLARE_CDN_BASEURL is not set"
        exit 1
    fi
}

validate_env_r2() {
    if [ -z "${S3_ACCESS_KEY_ID}" ]; then
        echo "Error: S3_ACCESS_KEY_ID is not set"
        exit 1
    fi

    if [ -z "${S3_SECRET_ACCESS_KEY}" ]; then
        echo "Error: S3_SECRET_ACCESS_KEY is not set"
        exit 1
    fi

    if [ -z "${CLOUDFLARE_API_TOKEN}" ]; then
        echo "Error: CLOUDFLARE_API_TOKEN is not set"
        exit 1
    fi

    if [ -z "${CLOUDFLARE_ACCOUNT_ID}" ]; then
        echo "Error: CLOUDFLARE_ACCOUNT_ID is not set"
        exit 1
    fi
}

setup() {
    mkdir -p "${TMP_DIR}"

    if [ -n "${RCLONE_CONFIG}" ]; then
        # Already set externally, use as-is
        :
    elif [ -f "$HOME/.config/rclone/rclone.conf" ]; then
        export RCLONE_CONFIG="$HOME/.config/rclone/rclone.conf"
    else
        export RCLONE_CONFIG="${TMP_DIR}/rclone.conf"
        write_config
        CREATED_RCLONE_CONFIG=true
    fi
}

write_config() {
    validate_env_r2

    cat <<EOD > "${RCLONE_CONFIG}"
[r2]
type = s3
provider = Cloudflare
access_key_id = ${S3_ACCESS_KEY_ID}
secret_access_key = ${S3_SECRET_ACCESS_KEY}
region = auto
endpoint = https://${CLOUDFLARE_ACCOUNT_ID}.r2.cloudflarestorage.com
disable_checksum = false
no_check_bucket = true
EOD
}

cleanup() {
    set +e
    rm -f "${CIRCUIT_INVENTORY_PATH}/manifest.json"
    rm -f "${VEHICLE_INVENTORY_PATH}/manifest.json"
    rm -rf "${TMP_DIR}"
}

purge_cdn() {
    purge_list="$1"

    [ -s "${purge_list}" ] || return 0

    echo "Purging CDN cache..."
    batches_file="${TMP_DIR}/purge_batches.json"
    jq -Rcn '[inputs] | range(0; length; 30) as $i | .[$i:$i+30] | {"files": .}' \
        "${purge_list}" > "${batches_file}"

    while IFS= read -r batch; do
        curl_exit=0
        curl_output=$(curl -sf -X POST "https://api.cloudflare.com/client/v4/zones/${CLOUDFLARE_ZONE_ID}/purge_cache" \
            -H "Authorization: Bearer ${CLOUDFLARE_API_TOKEN}" \
            -H "Content-Type: application/json" \
            --data "${batch}" 2>&1) || curl_exit=$?
        if [ ${curl_exit} -ne 0 ]; then
            echo "Error: CDN cache purge failed (curl exit code ${curl_exit}): ${curl_output}"
            exit 1
        fi
    done < "${batches_file}"
}

release_vehicles() {
    validate_env_vehicle
    validate_env_cf

    echo "Generating manifest for vehicle inventory..."
    go run tools/vehicle_inventory/*.go manifest "${VEHICLE_INVENTORY_PATH}" > "${VEHICLE_INVENTORY_PATH}/manifest.json"

    differ_file="${TMP_DIR}/vehicles_differ.txt"
    purge_file="${TMP_DIR}/vehicles_purge.txt"

    echo "Checking vehicle inventory for updates"
    if rclone check "${VEHICLE_INVENTORY_PATH}/" "${R2_REMOTE}/gt7/data/vehicles/" \
        --config "${RCLONE_CONFIG}" \
        --exclude ".DS_Store" \
        --differ "${differ_file}" \
        --missing-on-src "${purge_file}" 2>/dev/null; then
        echo "Vehicle data is up to date"
        return
    fi

    awk -v base="${CLOUDFLARE_CDN_BASEURL}/gt7/data/vehicles" \
        '{print base "/" $0}' "${differ_file}" "${purge_file}" >> "${TMP_DIR}/cdn_purge.txt" 2>/dev/null || true

    echo "Publishing updated vehicle inventory"
    rclone sync "${VEHICLE_INVENTORY_PATH}/" "${R2_REMOTE}/gt7/data/vehicles/" \
        --config "${RCLONE_CONFIG}" \
        --exclude ".DS_Store" \
        --stats-one-line --stats 1h
}

release_circuits() {
    validate_env_circuit
    validate_env_cf

    echo "Generating manifest for circuit inventory..."
    go run tools/circuit_inventory/main.go manifest "${CIRCUIT_INVENTORY_PATH}" > "${CIRCUIT_INVENTORY_PATH}/manifest.json"

    differ_file="${TMP_DIR}/circuits_differ.txt"
    purge_file="${TMP_DIR}/circuits_purge.txt"

    echo "Checking circuit inventory for updates"
    if rclone check "${CIRCUIT_INVENTORY_PATH}/" "${R2_REMOTE}/gt7/data/circuits/" \
        --config "${RCLONE_CONFIG}" \
        --exclude ".DS_Store" \
        --differ "${differ_file}" \
        --missing-on-src "${purge_file}" 2>/dev/null; then
        echo "Circuit data is up to date"
        return
    fi

    awk -v base="${CLOUDFLARE_CDN_BASEURL}/gt7/data/circuits" \
        '{print base "/" $0}' "${differ_file}" "${purge_file}" >> "${TMP_DIR}/cdn_purge.txt" 2>/dev/null || true

    echo "Publishing updated circuit inventory"
    rclone sync "${CIRCUIT_INVENTORY_PATH}/" "${R2_REMOTE}/gt7/data/circuits/" \
        --config "${RCLONE_CONFIG}" \
        --exclude ".DS_Store" \
        --stats-one-line --stats 1h
}

release_version() {
    validate_env_vehicle
    validate_env_circuit
    validate_env_cf

    echo "Checking version.json for updates"
    tools/inventory_version.sh "${CIRCUIT_INVENTORY_PATH}" "${VEHICLE_INVENTORY_PATH}" > "${TMP_DIR}/version.json"

    if rclone check "${TMP_DIR}/version.json" "${R2_REMOTE}/gt7/data/" \
        --config "${RCLONE_CONFIG}" 2>/dev/null; then
        echo "Version data is up to date"
        return
    fi

    echo "Publishing updated version.json"
    rclone copy "${TMP_DIR}/version.json" "${R2_REMOTE}/gt7/data/" \
        --config "${RCLONE_CONFIG}" \
        --stats-one-line --stats 1h

    echo "${CLOUDFLARE_CDN_BASEURL}/gt7/data/version.json" >> "${TMP_DIR}/cdn_purge.txt"
}


if [ -z "${R2_REMOTE}" ]; then
    echo "Error: R2_REMOTE is not set"
    echo
    usage
    exit 1
fi

if [ $# -lt 1 ]; then
    usage
    exit 1
fi

trap cleanup EXIT

action=$1

setup

case $action in
    vehicles)
        release_vehicles
        ;;
    circuits)
        release_circuits
        ;;
    version)
        release_version
        ;;
    all)
        release_vehicles
        release_circuits
        release_version
        ;;
    *)
        echo "Error: unknown action '$action'"
        echo
        usage
        exit 1
        ;;
esac

purge_cdn "${TMP_DIR}/cdn_purge.txt"
