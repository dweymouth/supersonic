# supersonic
A lightweight desktop client for Subsonic music servers. This is in REALLY early development and if you decide to use it now, expect to find bugs!

On first startup, the app will prompt you for your Subsonic server connection. The app currently displays an album grid view sorting by
recently added, and double-clicking an album plays that album.

## Build (Ubuntu)

### Install dependencies
* ``sudo snap install --classic go``
* ``sudo apt install libmpv-dev``
* ``sudo apt install gcc libegl1-mesa-dev xorg-dev``

### Build
* ``go build .``
