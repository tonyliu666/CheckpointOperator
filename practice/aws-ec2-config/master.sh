#!/bin/bash
set -e
IFNAME=$1
ADDRESS="$(ip -4 addr show $IFNAME | grep "inet" | head -1 |awk '{print $2}' | cut -d/ -f1)"
sed -e "s/^.*${HOSTNAME}.*/${ADDRESS} ${HOSTNAME} ${HOSTNAME}.local/" -i /etc/hosts

# update DNS
sed -e '/^.*ubuntu-bionic.*/d' -i /etc/hosts
sed -i -e 's/#DNS=/DNS=8.8.8.8/' /etc/systemd/resolved.conf
service systemd-resolved restart


cat <<EOF | sudo tee /etc/sysctl.d/99-kubernetes-cri.conf
net.bridge.bridge-nf-call-iptables = 1
net.ipv4.ip_forward = 1
net.bridge.bridge-nf-call-ip6tables = 1
EOF
# setup hosts:
set -e
IFNAME=$1
ADDRESS="$(ip -4 addr show $IFNAME | grep "inet" | head -1 |awk '{print $2}' | cut -d/ -f1)"
sed -e "s/^.*${HOSTNAME}.*/${ADDRESS} ${HOSTNAME} ${HOSTNAME}.local/" -i /etc/hosts

# remove ubuntu-bionic entry
sed -e '/^.*ubuntu-bionic.*/d' -i /etc/hosts

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
# install kubectl,kubeadm
sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl gpg
curl -fsSL https://pkgs.k8s.io/core:/stable:/v1.28/deb/Release.key | sudo gpg --dearmor -o /etc/apt/keyrings/kubernetes-apt-keyring.gpg
echo 'deb [signed-by=/etc/apt/keyrings/kubernetes-apt-keyring.gpg] https://pkgs.k8s.io/core:/stable:/v1.28/deb/ /' | sudo tee /etc/apt/sources.list.d/kubernetes.list
sudo apt-get update
sudo apt-get install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl
echo 'source <(kubectl completion bash)' >> ~/.bashrc
echo 'alias k=kubectl' >>~/.bashrc
echo 'complete -F __start_kubectl k' >>~/.bashrc

# enable criu within crio settings: 
sudo sed -i -e 's/^# \(enable_criu_support = \)false/\1true/' -e 's/^# \(drop_infra_ctr = \)true/\1false/' /etc/crio/crio.conf

# enable the crio pull image from insecure registry:
CONFIG_FILE="/etc/crio/crio.conf"
sudo sed -i '/^\# insecure_registries = \[/a\insecure_registries = ["localhost:30010"]' "$CONFIG_FILE"
sudo grep -A 1 "insecure_registries" "$CONFIG_FILE"
sudo systemctl restart crio
# kubeadm settings
kubeadm init --config=/vagrant/kubernetes/kubeadm-config.yaml --upload-certs | tee kubeadm-init.out
mkdir -p $HOME/.kube
sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
sudo chown $(id -u):$(id -g) $HOME/.kube/config

kubeadm token create --print-join-command | tee joincommand.txt

### To enable criu migration, you need to enable cgroup v1 in the worker and master nodes:
### if you use the ec2, don't do the reboot instance in the aws console, use the following command
echo 'GRUB_CMDLINE_LINUX_DEFAULT="${GRUB_CMDLINE_LINUX_DEFAULT} systemd.unified_cgroup_hierarchy=0"' | sudo tee /etc/default/grub.d/70-cgroup-unified.cfg
sudo update-grub
sudo reboot