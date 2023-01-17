# supersonic
A lightweight desktop client for Subsonic music servers. This is in early development, but currently usable for searching, browsing, and playing by albums. If you decide to use it now, expect to find a few minor bugs and missing features.

On first startup, the app will prompt you for your Subsonic server connection. The app currently has a searchable albums grid view, an artist discography view, list views of all artists and genres, and individual album view with tracklist.
<p align="center">
<img src="https://raw.github.com/dweymouth/supersonic/main/res/screenshots/albums-view.png" scale="50%"/><br/>
Slightly outdated screenshots of Supersonic running against the Navidrome demo server<br/>
<img src="https://raw.github.com/dweymouth/supersonic/main/res/screenshots/album-view.png" scale="50%"/>
</p>

## Features
* [x] Fast, lightweight, native UI
* [x] High-quality gapless audio playback powered by MPV
* [x] Infinite scrolling
* [x] Scrobble plays to server
* [x] Browse, search, and play albums
* [x] Album view with tracklist
* [x] Browse by genre
* [x] Browse by artist
* [ ] View and edit play queue (coming soon)
* [ ] Set/unset favorite and browse by favorites (coming soon)
* [ ] Browse, create, and edit playlists (coming soon)
* [ ] Artist view with biography, image, similar artists (coming soon)
* [ ] Shuffle and repeat playback modes (planned)
* [ ] Set and view five-star rating (planned)
* [ ] Browse by folders (planned)
* [ ] Multi-server support (planned)
* [ ] Download album or playlist (planned)
* [ ] ReplayGain audio normalization support (planned, depending on files being ReplayGain tagged on server)
* [ ] Cast to uPnP/DLNA devices (tentatively planned)
* [ ] Offline mode (eventually planned)
* [ ] iOS/Android support (eventually planned)
* [ ] Lyrics support (eventually planned)

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
