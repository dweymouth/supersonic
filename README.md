# supersonic
A lightweight desktop client for Subsonic music servers. This is in early development and if you decide to use it now, expect to find a few bugs and missing features.

On first startup, the app will prompt you for your Subsonic server connection. The app currently has a searchable albums grid view, an artist discography view, and individual album view with tracklist.

## Features
* [x] Fast, lightweight, native UI
* [x] High-quality gapless audio playback powered by MPV
* [x] Infinite scrolling
* [x] Scrobble plays to server
* [x] Browse, search, and play albums
* [x] Album view with tracklist
* [x] Browse by genre (partial; full support coming soon)
* [x] Browse by artist (partial; full support coming soon)
* [ ] View and edit play queue (coming soon)
* [ ] Set/unset favorite and browse by favorites (coming soon)
* [ ] Browse, create, and edit playlists (coming soon)
* [ ] Shuffle and repeat playback modes (planned)
* [ ] Set and view five-star rating (planned)
* [ ] Browse by folders (planned)
* [ ] Multi-server support (planned)
* [ ] Download album or playlist (planned)
* [ ] Cast to uPnP/DLNA devices (tentatively planned)


## Build instructions (Ubuntu)

### Install dependencies
* ``sudo snap install --classic go``
* ``sudo apt install libmpv-dev``
* ``sudo apt install gcc libegl1-mesa-dev xorg-dev``

### Build
* clone the repo, CD into the repo root, and run ``go build .``
* (note that the first build will take some time as it will download and build the UI library)

## Build instructions (Mac OS)

### Install dependencies
* install go
* install Xcode command-line tools (``xcode-select --install``)
* install libmpv (``brew install mpv``)

### Build
* Make sure header and library include paths include the dir in which homebrew installs headers/dylibs (may differ dep. on OS/Homebrew version)
  - ``export C_INCLUDE_PATH=/opt/homebrew/include:$C_INCLUDE_PATH``
  - ``export LIBRARY_PATH=/opt/homebrew/lib:$LIBRARY_PATH``

* clone the repo, CD into the repo root, and run ``go build .``
* (note that the first build will take some time as it will download and build the UI library)

## Build instructions (Windows)

### Install dependencies
* install go (https://go.dev/doc/install)
* install MSYS2 and the required packages for the Fyne toolkit (follow instructions for Windows at https://developer.fyne.io/started/)
* install libmpv (in the MSYS2 terminal, ``pacman -S mingw-w64-x86_64-mpv``)

### Build
* in the MSYS2 terminal: clone the repo, CD into the repo root, and run ``go build .``
* (note that the first build will take some time as it will download and build the UI library)
