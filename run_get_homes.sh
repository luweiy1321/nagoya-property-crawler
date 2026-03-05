#!/bin/bash
export PATH="/opt/homebrew/opt/postgresql@16/bin:$PATH"
cd /Users/lw/nagoya-property-crawler
go run get_homes_card.go > homes_card_output.txt 2>&1
cat homes_card_output.txt | tail -100
