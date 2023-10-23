<img src="res/appicon-128.png" alt="Supersonic logo" title="Supersonic" align="left" height="60px"/>
<a href='https://flathub.org/apps/details/io.github.dweymouth.supersonic'><img height="50px" align="right" alt='Download on Flathub' src='https://flathub.org/assets/badges/flathub-badge-en.png'/></a></div>
<a href='https://ko-fi.com/dweymouth' target='_blank'><img height='40' align='right' src='https://az743702.vo.msecnd.net/cdn/kofi3.png?v=0' border='0' alt='Buy Me a Coffee at ko-fi.com'/>

# Supersonic

[![License](https://img.shields.io/github/license/dweymouth/supersonic)](https://github.com/dweymouth/supersonic/blob/main/LICENSE)
[![Last Release](https://img.shields.io/github/v/release/dweymouth/supersonic?logo=github&label=latest&style=flat)](https://github.com/dweymouth/supersonic/releases)
[![Downloads](https://img.shields.io/github/downloads/dweymouth/supersonic/total?logo=github&style=flat)](https://github.com/dweymouth/supersonic/releases/latest)
[![Go Report Card](https://goreportcard.com/badge/github.com/dweymouth/supersonic)](https://goreportcard.com/report/github.com/dweymouth/supersonic)

A lightweight cross-platform desktop client for Subsonic music servers (Navidrome, Gonic, Airsonic, etc).

[Jump to installation instructions](https://github.com/dweymouth/supersonic#installation)

## Screenshots

Screenshots of Supersonic running against the Navidrome [demo server](https://www.navidrome.org/demo/)

<a href="https://raw.githubusercontent.com/dweymouth/supersonic/main/res/screenshots/AlbumsView.png"><img src="https://raw.github.com/dweymouth/supersonic/main/res/screenshots/AlbumsView.png" width="49.5%"/></a>
<a href="https://raw.githubusercontent.com/dweymouth/supersonic/main/res/screenshots/AlbumView.png"><img src="https://raw.github.com/dweymouth/supersonic/main/res/screenshots/AlbumView.png" width="49.5%"/></a>
<a href="https://raw.githubusercontent.com/dweymouth/supersonic/main/res/screenshots/ArtistView.png"><img src="https://raw.github.com/dweymouth/supersonic/main/res/screenshots/ArtistView.png" width="49.5%"/></a>
<a href="https://raw.githubusercontent.com/dweymouth/supersonic/main/res/screenshots/FavoriteSongsView.png"><img src="https://raw.github.com/dweymouth/supersonic/main/res/screenshots/FavoriteSongsView.png" width="49.5%"/></a>

## Features
* [x] Fast, lightweight, native UI with infinite scrolling
* [x] Light and Dark themes, with optional auto theme switching
* [x] High-quality gapless audio playback powered by MPV, with optional audio exclusive mode
* [x] ReplayGain support (depends on files being tagged on server)
* [x] MPRIS and Mac OS media center integration for media key and desktop control
* [x] Built-in 15-band graphic equalizer
* [x] Scrobble plays to server, with configurable criteria
* [x] Add and switch between multiple servers
* [x] Primary and alternate server hostnames, e.g. for internal and external URLs
* [x] Browse by albums, artists, genres, playlists
* [x] Set filters in albums browsing view
* [x] Album and playlist views with tracklist and cover image
* [x] Artist view with biography, image, similar artists, and discography
* [x] Play "artist radio" (mix of songs from given artist and similar artists, depends on your server's support)
* [x] Create, play, and update playlists
* [x] Sort tracklist views by column and configure visible tracklist columns
* [x] Set/unset favorite and browse by favorite albums, artists, and songs
* [x] Set and view track rating (0-5 stars)
* [x] Download songs, albums or playlists
* [x] View and edit play queue (add and remove tracks; reorder support coming soon)
* [x] Shuffle and repeat playback modes (partial; shuffle album, playlist, artist radio, random songs; repeat one/all)
* [ ] Browse by folders (planned)
* [ ] Cast to uPnP/DLNA devices (likely planned)
* [ ] Offline mode (eventually planned)
* [ ] Lyrics support (eventually planned)
* [ ] iOS/Android support (maybe eventually planned)

## Installation

On Linux, Supersonic is [available as a Flatpak](https://flathub.org/apps/details/io.github.dweymouth.supersonic)! (Thank you @anarcat!) If you prefer to directly install the release build, or build from source, read below.

If you are running Windows, Mac OS, or a Debian-based Linux distro, download the latest [release](https://github.com/dweymouth/supersonic/releases) for your operating system. Tou can also download the latest build from the `main` branch via the [Actions](https://github.com/dweymouth/supersonic/actions) tab for Intel Macs and Debian-based Linux, to get unreleased features and bug fixes (you must be signed in to Github to do this). If you prefer to build from source, or are not running one of these OSes, then see the build instructions for your platform below.

**Apple Silicon (M1/M2) Macs:** You will have to remove the "quarantine bit" that Mac will automatically set, being an application downloaded from the internet. After copying the .app bundle to your Applications folder, in the terminal run `sudo xattr -r -d com.apple.quarantine /Applications/Supersonic.app`

**If you are on Linux** you must have libmpv installed on your system. On apt-based systems, run `sudo apt install libmpv1` if it is not already installed. The Windows and Mac release builds bundle the mpv dependencies.

## Build instructions (Linux)

### Ubuntu dependencies
* ``sudo snap install --classic go``, and make sure the Go bin directory is in your `$PATH`
* ``sudo apt install libmpv-dev gcc libegl1-mesa-dev xorg-dev``

### Fedora dependencies
* ``sudo dnf install golang mpv-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libglvnd-devel libXxf86vm-devel``

### Build
* clone the repo, CD into the repo root, and run ``go build .``
* (note that the first build will take some time as it will download and build the UI library)

### Generate installable .tar.xz bundle
* install the ``fyne`` packaging tool ``go install fyne.io/fyne/v2/cmd/fyne@latest``
* run ``make package_linux``

## Build instructions (Arch Linux)

Supersonic is available in the AUR and can be built either manually with `makepkg` or with an AUR helper like [yay](https://github.com/Jguer/yay). (Please contact package maintainer @dusnm for any issues with the AUR package.)

### Build manually
* Make sure you have the base-devel package group installed on your system
  - ``sudo pacman -S --needed base-devel``
* Clone the AUR repository and navigate into the cloned directory
  - ``git clone https://aur.archlinux.org/supersonic-desktop.git && cd supersonic-desktop``
* Build the package with makepkg
  - ``makepkg -si``

### Build with an AUR helper
* Invoke your favorite AUR helper to automatically build the package
  - ``yay -S supersonic-desktop``

## Build instructions (Mac OS)

### Install dependencies
* install go, and make sure the Go bin directory is in your `$PATH`
  - ``brew install go``
  - ``export PATH="/Users/<yourname>/go/bin:$PATH"``
* install the ``fyne`` packaging tool ``go install fyne.io/fyne/v2/cmd/fyne@latest``
* install Xcode command-line tools (``xcode-select --install``)
* install libmpv (``brew install mpv``)
* install dylibbundler (``brew install dylibbundler``) - needed only the bundledeps step, see below

### Build
* Make sure header and library include paths include the dir in which homebrew installs headers/dylibs (may differ dep. on OS/Homebrew version)
  - ``export C_INCLUDE_PATH=/opt/homebrew/include:$C_INCLUDE_PATH``
  - ``export LIBRARY_PATH=/opt/homebrew/lib:$LIBRARY_PATH``

* clone the repo, CD into the repo root, and run ``go build .``
* (note that the first build will take some time as it will download and build the UI library)
* run ``make package_macos`` to generate the .app bundle
* **If** you are on Mac OS **High Sierra** through **Catalina**, run ``make bundledeps_macos_highsierra`` and you are done! Otherwise, continue reaading.
* At this point, the Supersonic.app bundle can be copied to Applications and it will run on your machine, but it depends on the brew installation of mpv
* To copy the dependencies into the app bundle, and make it truly portable, run ``make bundledeps_macos``

## Build instructions (Windows)

### Install dependencies
* install go (https://go.dev/doc/install)
* install MSYS2 and the required packages for the Fyne toolkit (follow instructions for Windows at https://developer.fyne.io/started/)
* install libmpv (in the MSYS2 terminal, ``pacman -S mingw-w64-x86_64-mpv``)

### Build
* in the MSYS2 terminal: clone the repo, CD into the repo root, and run ``go build .``
* (note that the first build will take some time as it will download and build the UI library)
* **Note**: The .exe dynamically links to MSYS2 libmpv dependency dlls and must be started from the MSYS2 terminal, or all dependency DLLS must be copied to the same folder as the .exe
* -> If you obtain a statically built mpv-2.dll (containing all its dependencies), and rename it to libmpv-2.dll, you can place just that DLL in the same directory as the EXE, and it should run
* Improvements to Windows build process will be forthcoming
