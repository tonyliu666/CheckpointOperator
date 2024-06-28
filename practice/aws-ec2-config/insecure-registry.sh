#!/bin/bash 
PRIVATE_IP=$(ip -o -4 addr show enX0 | awk '{print $4}' | cut -d/ -f1)

# Define the insecure registry address with the port 30010
INSECURE_REGISTRY="${PRIVATE_IP}:30010"

# CRIO configuration file
CONFIG_FILE="/etc/crio/crio.conf"

# Check if the insecure_registries field exists in the configuration file
if grep -q "insecure_registries" "$CONFIG_FILE"; then
  # Update the existing insecure_registries field
  sudo sed -i "s|insecure_registries = \[.*\]|insecure_registries = [\"${INSECURE_REGISTRY}\"]|g" "$CONFIG_FILE"
else
  # Add the insecure_registries field
  sudo sed -i "/\[registries\]/a insecure_registries = [\"${INSECURE_REGISTRY}\"]" "$CONFIG_FILE"
fi
sudo systemctl restart crio
echo "Insecure registry set to ${INSECURE_REGISTRY} and CRI-O restarted."