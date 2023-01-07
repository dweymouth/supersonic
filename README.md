# supersonic
A lightweight desktop client for Subsonic music servers. This is in early development and if you decide to use it now, expect to find a few bugs and missing features.

On first startup, the app will prompt you for your Subsonic server connection. The app currently has a searchable albums grid view, an artist discography view, and individual album view with tracklist.

## Features
* Fast, lightweight, native UI
* High-quality gapless audio playback powered by MPV
* Infinite scrolling

## Build (Ubuntu)

### Install dependencies
* ``sudo snap install --classic go``
* ``sudo apt install libmpv-dev``
* ``sudo apt install gcc libegl1-mesa-dev xorg-dev``

### Build
* ``go build .``
* (note that the first build will take some time as it will download and build the UI library)

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
* (note that the first build will take some time as it will download and build the UI library)
