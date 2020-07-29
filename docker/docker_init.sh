#!/bin/bash
if test "$(grep "geodb_enabled: true" mika.yaml)"; then
  echo "Geo database: Enabled"
  if ! test -f "geo_data/IP2LOCATION-LITE-ASN.CSV"; then
    echo "Updating geo database (Fresh)"
    ./mika updategeo
  fi
  if test "$(find "geo_data/IP2LOCATION-LITE-ASN.CSV" -mmin +10080)"; then
    echo "Updating geo database (Old)"
    ./mika updategeo
  fi
else
  echo "Geo database: Disabled"
fi
echo "Starting mika..."
exec "$@"