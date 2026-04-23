#!/bin/sh

if [ "$#" -ne 2 ]; then
  echo "Usage: $0 <circuits_dir> <vehicles_dir>" >&2
  exit 1
fi

circuits_dir=$1
vehicles_dir=$2

circuits_version=$(grep -h lastModified "${circuits_dir}"/*.json | sort -u | tail -n 1 | sed 's/^ *//' | sed 's/,$//')
vehicles_version=$(grep -h lastModified "${vehicles_dir}"/*.json | sort -u | tail -n 1 | sed 's/^ *//' | sed 's/,$//')

cat <<EOD
{
  "circuits": {
    $circuits_version
  },
  "vehicles": {
    $vehicles_version
  }
}
EOD
