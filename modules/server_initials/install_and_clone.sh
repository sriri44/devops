#!/bin/bash

set -e

sudo apt install git -y
sudo apt install nginx -y
sudo apt install docker -y

echo -n "Enter the git clone URL: "
read git_clone_url
git clone "$git_clone_url" 