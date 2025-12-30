#!/bin/sh
./server_initials/install_and_clone.sh
./point-hostinger-domain/main
cd Devops-AI-Agent/modules/setup_nginx && ./main \
  --country=US \
  --state="New York" \
  --city="New York City" \
  --org="Bouncy Castles, Inc." \
  --unit="Ministry of Water Slides" \
  --common=default \
  --email=nandakishorep212@gmail.com
