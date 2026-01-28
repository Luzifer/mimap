"""
#!/usr/bin/env python3

Original from https://github.com/dgiese/dustcloud/blob/master/dustcloud/build_map.py
Modified to resemble the map inside the Mi Home application

This program is free software: you can redistribute it and/or modify
it under the terms of the GNU General Public License as published by
the Free Software Foundation, either version 3 of the License, or
(at your option) any later version.

This program is distributed in the hope that it will be useful,
but WITHOUT ANY WARRANTY; without even the implied warranty of
MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
GNU General Public License for more details.

"""
import argparse
import io
from PIL import Image, ImageDraw, ImageChops

color_definition = {
    # Original colors
    "black": (0, 0, 0, 255),  # This should be a wall or something
    "blue": (0, 0, 255, 255),  # WTF is blue?
    "grey": (125, 125, 125, 255),  # Original background color
    "lightblue": (57, 121, 200, 255),  # Echoes from the outside world?
    "lightgrey": (250, 250, 250, 255),  # Weird single spot?
    "purple": (255, 0, 255, 255),  # WTF is purple?
    "red": (255, 0, 0, 255),  # Pretty sure a detected magnetic barrier
    "white": (255, 255, 255, 255),  # Floor
    "yellow": (255, 204, 0, 255),  # Could be a collision spot?

    # Our colors
    "background": (24, 102, 181, 255),
    "current_spot": (248, 205, 71, 255),
    "floor": (33, 117, 197, 255),
    "trace": (127, 179, 224, 255),
    "transparent": (0, 0, 0, 0),
    "wall": (98, 202, 255, 255),
}

color_replacement = {
    "black": "wall",
    "blue": "floor",
    "grey": "background",
    "lightgrey": "floor",
    "purple": "floor",
    "red": "floor",
    "white": "floor",
    "yellow": "floor",
}


def build_map(slam_log_data, map_image_data):
    """
    Parses the slam log to get the vacuum path and draws the path into
    the map. Returns the new map as a BytesIO.

    Thanks to CodeKing for the algorithm!
    https://github.com/dgiese/dustcloud/issues/22#issuecomment-367618008
    """
    resize_factor = 4

    map_image = Image.open(io.BytesIO(map_image_data))
    map_image = map_image.convert('RGBA')
    map_image = map_image.resize(
        size=(
            map_image.size[0]*resize_factor,
            map_image.size[0]*resize_factor,
        ),
        resample=Image.Resampling.NEAREST,
    )

    # calculate center of the image
    center_x = map_image.size[0] / 2
    center_y = map_image.size[0] / 2

    # rotate image by -90°
    map_image = map_image.rotate(-90)

    # prepare for drawing
    draw = ImageDraw.Draw(map_image)

    # loop each line of slam log
    prev_pos = None
    for line in slam_log_data.split("\n"):
        # find positions
        if 'estimate' in line:
            d = line.split('estimate')[1].strip()

            # extract x & y
            y, x, z = map(float, d.split(' '))

            # set x & y by center of the image
            # 20 is the factor to fit coordinates in in map
            x = center_x + (x * 20 * resize_factor)
            y = center_y + (y * 20 * resize_factor)

            pos = (x, y)
            if prev_pos:
                draw.line([prev_pos, pos], color_definition["trace"])
            prev_pos = pos

    # draw current position
    def ellipsebb(x, y):
        return x-4, y-4, x+4, y+4
    draw.ellipse(ellipsebb(x, y), color_definition["current_spot"])

    # rotate image back by 90°
    map_image = map_image.rotate(90)

    # crop image
    bgcolor_image = Image.new('RGBA', map_image.size,
                              color_definition["grey"])
    cropbox = ImageChops.subtract_modulo(map_image, bgcolor_image).getbbox(alpha_only=False)
    if cropbox is None:
        raise Exception('unable to determine image bbox')

    map_image = map_image.crop((
        cropbox[0]-20, cropbox[1]-20,
        cropbox[2]+20, cropbox[3]+20
    ))

    # and replace background with transparent pixels
    pixdata = map_image.load()
    for y in range(map_image.size[1]):
        for x in range(map_image.size[0]):
            pixdata[x, y] = get_color_replacement(pixdata[x, y])

    temp = io.BytesIO()
    map_image.save(temp, format="png")
    return temp


def get_color_replacement(in_color):
    color_name = None
    for name, definition in color_definition.items():
        if definition == in_color:
            color_name = name
            break

    if color_name == None or color_name not in color_replacement:
        return in_color

    return color_definition[color_replacement[color_name]]


if __name__ == "__main__":
    parser = argparse.ArgumentParser(description="""
Process the runtime logs of the vacuum (SLAM_fprintf.log, navmap*.ppm from /var/run/shm)
and draw the path into the map. Outputs the map as a PNG image.
    """)

    parser.add_argument(
        "-slam",
        default="SLAM_fprintf.log",
        required=True)
    parser.add_argument(
        "-map",
        required=True)
    parser.add_argument(
        "-out",
        required=False)
    args = parser.parse_args()

    with open(args.slam) as slam_log:
        with open(args.map, 'rb') as mapfile:
            augmented_map = build_map(slam_log.read(), mapfile.read(), )

    out_path = args.out
    if not out_path:
        out_path = args.map[:-4] + ".png"
    if not out_path.endswith(".png"):
        out_path += ".png"

    with open(out_path, 'wb') as out:
        out.write(augmented_map.getvalue())
