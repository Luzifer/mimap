[![Go Report Card](https://goreportcard.com/badge/github.com/Luzifer/mimap)](https://goreportcard.com/report/github.com/Luzifer/mimap)
![](https://badges.fyi/github/license/Luzifer/mimap)
![](https://badges.fyi/github/downloads/Luzifer/mimap)
![](https://badges.fyi/github/latest-release/Luzifer/mimap)

# Luzifer / mimap

`mimap` is a small web application to receive map updates from a script running on opened Xiaomi Mi Vacuum robots decoupled from the Xiaomi cloud. The image generator is derived from the [dustcloud](https://github.com/dgiese/dustcloud/) image generator patched to resemble the map images used inside the Mi Home application.

## Usage

- Start the `luzifer/mimap:latest` image with exposed port and an attached volume:
    ```
    docker run --rm -ti -v /tmp/mimap:/data -p 3000:3000 luzifer/mimap:latest
    ```
- Adjust the `upload.sh` script to upload the files to the `mimap` instance
- Put the modified `upload.sh` script onto the vacuum
- Execute the script with
    ```
    bash upload.sh
    ```
- The map should be generated by the `mimap` instance and be available at http://localhost:3000/map.png

