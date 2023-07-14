#!/bin/sh
find * -type d -exec asset-pack -margin 0 -o {}.png {} \;
asset-pack -margin 0 -o ../media.png *.png
rm *.png*