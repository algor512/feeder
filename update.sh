#!/bin/sh
set -e

BASEDIR=$(dirname "$0")

DATE_FROM=$(date -d "-3 days" +%s)
"$BASEDIR/bin/parse_feeds" -config "$BASEDIR/config.rec" \
						   -status "$BASEDIR/data/status.txt" > "$BASEDIR/data/new_records.tsv"
cat "$BASEDIR/data/new_records.tsv" "$BASEDIR/data/records.tsv" \
	| sort -t "$(printf "\t")" -k2,2 -k3,3 -u \
	| awk -v dlim=$DATE_FROM 'BEGIN { FS=OFS="\t" }; $1>=dlim' \
	| sort -t "$(printf "\t")" -n -k1,1 -r > "$BASEDIR/data/new_new_records.tsv"
mv "$BASEDIR/data/new_new_records.tsv" "$BASEDIR/data/records.tsv"
rm "$BASEDIR/data/new_records.tsv"
"$BASEDIR/bin/update_page" -records "$BASEDIR/data/records.tsv" \
						   -status "$BASEDIR/data/status.txt" \
						   -template "$BASEDIR/template.html" > "$BASEDIR/html/index.html"
