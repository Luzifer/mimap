default:

test-build-map: test-generate-map
	sha512sum -c <test/checksums.sha512

test-generate-map: test/navmap.ppm
	uv run --locked ./build_map.py -slam test/slam.log -map test/navmap.ppm -out test/map.png

test/navmap.ppm:
	gzip -cd $@.gz >$@
