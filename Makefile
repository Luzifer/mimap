default:

test: test-generate
	sha512sum -c <test/checksums.sha512

test-generate: test/navmap.ppm
	uv run ./build_map.py -slam test/slam.log -map test/navmap.ppm -out test/map.png

test/navmap.ppm:
	gzip -cd $@.gz >$@
