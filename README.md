# img1b

Package img1b is a fork of the standard Go image package modified for 1-bit images.
Images are kept packed so they take up to 8 times less memory and may be processed
faster.

It provides an Image type that is in most aspects very similar to image.Paletted with
palette limited to two colors.

img1b.Image implements image.PalettedImage and can be encoded with image/png and
probably other encoders but that is not very efficient due to all the bit shuffling
involved. Subpackage img1b/png is a modified png codec that processes whole rows
which is several times faster.

