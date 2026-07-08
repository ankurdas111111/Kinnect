#!/usr/bin/env bash
# Keep launching a VM.Standard.A1.Flex until Oracle has capacity.
# Loops over every availability domain, retrying every 60s.
#
# Prereqs:
#   1) brew install oci-cli   &&   oci setup config   (upload the public API key in the console)
#   2) Pre-create a VCN + PUBLIC subnet once (VCN Wizard), grab its subnet OCID.
#
# Required env vars:
#   export COMPARTMENT_OCID=ocid1.tenancy.oc1..xxxx     # your root/tenancy OCID
#   export SUBNET_OCID=ocid1.subnet.oc1..xxxx           # the PUBLIC subnet OCID
#   export SSH_KEY_FILE=$HOME/.ssh/oracle-kinnect.pub   # your SSH *public* key
# Optional:
#   export OCPUS=2 MEM=12 DISPLAY_NAME=kinnect-a1
set -euo pipefail
: "${COMPARTMENT_OCID:?set COMPARTMENT_OCID}"
: "${SUBNET_OCID:?set SUBNET_OCID}"
SSH_KEY_FILE="${SSH_KEY_FILE:-$HOME/.ssh/oracle-kinnect.pub}"
SHAPE="VM.Standard.A1.Flex"
OCPUS="${OCPUS:-2}"
MEM="${MEM:-12}"
DISPLAY_NAME="${DISPLAY_NAME:-kinnect-a1}"

echo "==> Resolving latest Ubuntu 22.04 (aarch64) image for $SHAPE"
IMAGE_OCID=$(oci compute image list \
  --compartment-id "$COMPARTMENT_OCID" \
  --operating-system "Canonical Ubuntu" \
  --operating-system-version "22.04" \
  --shape "$SHAPE" \
  --sort-by TIMECREATED --sort-order DESC \
  --query 'data[0].id' --raw-output)
echo "    image: $IMAGE_OCID"

echo "==> Listing availability domains"
mapfile -t ADS < <(oci iam availability-domain list \
  --compartment-id "$COMPARTMENT_OCID" \
  --query 'data[].name' --raw-output | tr -d '[]," ' | grep -v '^$')
echo "    ADs: ${ADS[*]}"

attempt=0
while true; do
  attempt=$((attempt+1))
  for AD in "${ADS[@]}"; do
    printf '[try %d] AD=%s ... ' "$attempt" "$AD"
    if oci compute instance launch \
        --availability-domain "$AD" \
        --compartment-id "$COMPARTMENT_OCID" \
        --shape "$SHAPE" \
        --shape-config "{\"ocpus\":$OCPUS,\"memoryInGBs\":$MEM}" \
        --image-id "$IMAGE_OCID" \
        --subnet-id "$SUBNET_OCID" \
        --assign-public-ip true \
        --display-name "$DISPLAY_NAME" \
        --ssh-authorized-keys-file "$SSH_KEY_FILE" \
        >/tmp/oci_launch_out 2>/tmp/oci_launch_err; then
      echo "LAUNCHED 🎉"
      echo "Instance is provisioning. Get its public IP from the console (or 'oci compute instance list')."
      exit 0
    fi
    if grep -qiE "out of host capacity|outofcapacity|capacity" /tmp/oci_launch_err; then
      echo "no capacity"
    else
      echo "ERROR (not capacity):"; cat /tmp/oci_launch_err; exit 1
    fi
  done
  sleep 60
done
