#!/bin/bash
cat <<EOF | sudo tee /etc/sysctl.d/99-kubernetes-cri.conf
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.ip_forward = 1
net.bridge.bridge-nf-call-ip6tables = 1
EOF

# install criu: 
sudo apt install criu -y

# install cri-o
sudo apt update && sudo apt -y full-upgrade
CRIO_VERSION=1.28
OS=xUbuntu_20.04
echo "deb https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/$OS/ /"|sudo tee /etc/apt/sources.list.d/devel:kubic:libcontainers:stable.list
echo "deb http://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable:/cri-o:/$CRIO_VERSION/$OS/ /"|sudo tee /etc/apt/sources.list.d/devel:kubic:libcontainers:stable:cri-o:$CRIO_VERSION.list
curl -L https://download.opensuse.org/repositories/devel:kubic:libcontainers:stable:cri-o:$CRIO_VERSION/$OS/Release.key | sudo apt-key add -
curl -L https://download.opensuse.org/repositories/devel:/kubic:/libcontainers:/stable/$OS/Release.key | sudo apt-key add -
sudo apt update
sudo apt install cri-o cri-o-runc -y
sudo systemctl daemon-reload
sudo systemctl enable crio
sudo systemctl start crio

# enable criu within crio settings: 
sudo sed -i -e 's/^# \(enable_criu_support = \)false/\1true/' -e 's/^# \(drop_infra_ctr = \)true/\1false/' /etc/crio/crio.conf

# enable the crio pull image from insecure registry:
CONFIG_FILE="/etc/crio/crio.conf"
sudo sed -i '/^\# insecure_registries = \[/a\insecure_registries = ["localhost:30010"]' "$CONFIG_FILE"
sudo grep -A 1 "insecure_registries" "$CONFIG_FILE"
sudo systemctl restart crio


# install kubectl,kubeadm
sudo apt-get update
apt-get install -y kubelet kubeadm kubectl
apt-mark hold kubelet kubeadm kubectl

echo 'source <(kubectl completion bash)' >> ~/.bashrc
echo 'alias k=kubectl' >>~/.bashrc
echo 'complete -F __start_kubectl k' >>~/.bashrc
# kubeadm join command: copy the command written in joincommand.txt and paste here

# Before doing the following, please add the worker node to the cluster first

### To enable criu migration, you need to enable cgroup v1 in the worker and master nodes:
### if you use the ec2, don't do the reboot instance in the aws console, use the following command
echo 'GRUB_CMDLINE_LINUX_DEFAULT="${GRUB_CMDLINE_LINUX_DEFAULT} systemd.unified_cgroup_hierarchy=0"' | sudo tee /etc/default/grub.d/70-cgroup-unified.cfg
sudo update-grub
sudo reboot