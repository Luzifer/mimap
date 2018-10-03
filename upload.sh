#!/bin/bash
set -euo pipefail

function cleanup() {
	rm -f \
		/tmp/navmap.ppm \
		/tmp/slam.log
}

function get_archived_map_dir() {
	LOGDIR=/mnt/data/rockrobo/rrlog/

	# Query archived maps, take the newest one
	LATEST=$(find "${LOGDIR}" -name 'navmap*.0*.gz' | sort | tail -n1)

	# Exit if no navmap was found in the last 24h
	[ -n "${LATEST}" ] || exit 0

	LATEST_DIR=$(dirname ${LATEST})
	echo "${LATEST_DIR}"
}

function main() {
	if [ -f "$(ls /var/run/shm/navmap*.ppm)" ]; then
		map_file="$(ls /var/run/shm/navmap*.ppm)"
		slam_file="/var/run/shm/SLAM_fprintf.log"

		# Map is written at full minute, lets wait a moment
		sleep 5

		cp "${map_file}" /tmp/navmap.ppm
		cp "${slam_file}" /tmp/slam.log
	else
		map_dir=$(get_archived_map_dir)
		map_file="$(ls -1 "${map_dir}"/navmap*.ppm.0*.gz)"
		slam_file="$(ls -1 "${map_dir}"/SLAM_fprintf.log.0*.gz)"

		gzip -d -c "${map_file}" >/tmp/navmap.ppm
		gzip -d -c "${slam_file}" >/tmp/slam.log
	fi

	trap cleanup EXIT

	if [ -f /mnt/data/map_upload.sums ]; then
		# Check whether the data changed
		shasum -c /mnt/data/map_upload.sums && exit 0 || true
	fi

	curl -sSfL \
		-F "map=@/tmp/navmap.ppm" \
		-F "sum_map=$(sum /tmp/navmap.ppm)" \
		-F "slam=@/tmp/slam.log" \
		-F "sum_slam=$(sum /tmp/slam.log)" \
		http://10.229.2.2:1240/upload

	shasum /tmp/navmap.ppm /tmp/slam.log >/mnt/data/map_upload.sums
}

function sum() {
	shasum "$1" | cut -d ' ' -f 1
}

main
