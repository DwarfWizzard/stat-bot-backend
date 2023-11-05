#!/bin/bash

service ssh start
echo "PermitRootLogin yes" >> /etc/ssh/sshd_config
echo "root:1234" | chpasswd
service ssh restart