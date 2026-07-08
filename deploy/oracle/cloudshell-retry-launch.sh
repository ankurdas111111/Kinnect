#!/usr/bin/env bash
# Run this INSIDE OCI Cloud Shell (browser terminal) — it is pre-authenticated,
# so it bypasses local Zscaler/Python cert issues. Keeps launching a
# VM.Standard.A1.Flex until Oracle has Always-Free ARM capacity.
#
# Usage in Cloud Shell:
#   export SUBNET_OCID=ocid1.subnet.oc1...   # public subnet from the VCN wizard
#   export OCPUS=2 MEM=12                     # optional (Always Free max: 4 / 24)
#   bash cloudshell-retry-launch.sh
set -euo pipefail
C="${OCI_TENANCY:?run this inside Cloud Shell (OCI_TENANCY unset)}"
SHAPE=VM.Standard.A1.Flex
OCPUS="${OCPUS:-2}"; MEM="${MEM:-12}"; NAME="${NAME:-kinnect-a1}"

# Resolve the public subnet: either given directly, or auto-found from VCN_OCID.
if [ -z "${SUBNET_OCID:-}" ]; then
  : "${VCN_OCID:?export SUBNET_OCID=ocid1.subnet... OR VCN_OCID=ocid1.vcn...}"
  SUBNET_OCID=$(oci network subnet list -c "$C" --vcn-id "$VCN_OCID" \
    --query 'data[?("prohibit-public-ip-on-vnic"==`false`)].id | [0]' --raw-output)
  : "${SUBNET_OCID:?no public subnet found in $VCN_OCID}"
  echo "resolved public subnet: $SUBNET_OCID"
fi

cat > /tmp/id.pub <<'KEY'
ssh-rsa AAAAB3NzaC1yc2EAAAADAQABAAABAQC+yMwRlPmJ4RwRouSR67lxXrcQmDd+F8MJ9lnwh1ejMu7o/3UoHpYIKH+CO5mvWgIj7dYb/nkiTamebuisl0PTHQX1O/H9MbpCs/GAek5X2upDGn6XcVyQorQcy8d2D/0+dcSLtB2bnbdtWDf/IIoxxRgfmm3T7ipKA50YdgTiFK243cuJiPSfukv+U3w738eYpD1NURCPIU6YHJo53+bDpjQaEZHYFovIKvtFVy8wISxJ6MirXw7GSRP8bBetnTetamncp23zVnezFQvJi7xEvlBURLy98RExFa/qMgWhT2NFNPbM/+P3Bc2IYn0gwlBG/tGFF8viJuC+tpVCKNhV ssh-key-2026-07-07
KEY

IMG=$(oci compute image list -c "$C" --operating-system "Canonical Ubuntu" \
  --operating-system-version "22.04" --shape "$SHAPE" \
  --sort-by TIMECREATED --sort-order DESC --query 'data[0].id' --raw-output)
mapfile -t ADS < <(oci iam availability-domain list -c "$C" \
  --query 'data[].name' --raw-output | tr -d '[]," ' | grep -v '^$')
echo "image=$IMG ; ADs=${ADS[*]} ; ${OCPUS}ocpu/${MEM}gb"

n=0
while true; do n=$((n+1))
  for AD in "${ADS[@]}"; do
    printf '[%d] %s ... ' "$n" "$AD"
    if oci compute instance launch -c "$C" --availability-domain "$AD" \
        --shape "$SHAPE" --shape-config "{\"ocpus\":$OCPUS,\"memoryInGBs\":$MEM}" \
        --image-id "$IMG" --subnet-id "$SUBNET_OCID" --assign-public-ip true \
        --display-name "$NAME" --ssh-authorized-keys-file /tmp/id.pub \
        >/tmp/out 2>/tmp/err; then
      echo "LAUNCHED 🎉"
      oci compute instance list -c "$C" \
        --query "data[?\"display-name\"=='$NAME'].{name:\"display-name\",state:\"lifecycle-state\"}" \
        --output table
      exit 0
    fi
    grep -qiE 'capacity' /tmp/err && echo "no capacity" || { echo "ERROR:"; cat /tmp/err; exit 1; }
  done
  sleep 60
done
