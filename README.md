# supersonic
A lightweight desktop client for Subsonic music servers. This is in early development and if you decide to use it now, expect to find some bugs!

On first startup, the app will prompt you for your Subsonic server connection. The app currently displays a searchable album grid view that is also sortable by the standard Subsonic API sort orders. Double-clicking an album plays that album.

## Build (Ubuntu)

### Install dependencies
* ``sudo snap install --classic go``
* ``sudo apt install libmpv-dev``
* ``sudo apt install gcc libegl1-mesa-dev xorg-dev``

### Build
* ``go build .``

## Build (Mac OS)

### Install dependencies
* install go
* install Xcode command-line tools (``xcode-select --install``)
* install libmpv (``brew install mpv``)

### Build
* Make sure header and library include paths include the dir in which homebrew installs headers/dylibs (may differ dep. on OS/Homebrew version)
  - ``export C_INCLUDE_PATH=/opt/homebrew/include:$C_INCLUDE_PATH``
  - ``export LIBRARY_PATH=/opt/homebrew/lib:$LIBRARY_PATH``

* ``go build .``
