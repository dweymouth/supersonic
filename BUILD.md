# Building Supersonic from source

To build Supersonic from source, first clone the Git repo or download the source code zip from the latest release. Then follow the instructions for your platform below.

## Build tasks

Supersonic uses `mise.toml` for local build tasks. Install [mise](https://mise.jdx.dev/getting-started.html), then run `mise install` once from the repo root to install the pinned Go toolchain.

Common tasks:

* ``mise run build`` - build the app binary
* ``mise run launch`` - launch the app from source
* ``mise run all`` - build and package everything needed for the current platform
* ``mise run build:debug`` - build with localav/CoreAudio debug logging
* ``mise run launch:debug`` - launch with localav/CoreAudio debug logging

## Build instructions (Linux)

### Ubuntu dependencies
* ``sudo apt install libavformat-dev libavcodec-dev libavfilter-dev libavutil-dev libswresample-dev libswscale-dev libwavpack-dev gcc libegl1-mesa-dev xorg-dev``

### Debian dependencies
* ``sudo apt install libavformat-dev libavcodec-dev libavfilter-dev libavutil-dev libswresample-dev libswscale-dev libwavpack-dev gcc libegl1-mesa-dev xorg-dev``

### Fedora dependencies
* ``sudo dnf install ffmpeg-devel wavpack-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libglvnd-devel libXxf86vm-devel``

### Build
* clone the repo, CD into the repo root, and run ``mise run build``
* (note that the first build will take some time as it will download and build the UI library)

### Generate installable .tar.xz bundle
* run ``mise run package:linux``. The task installs the ``fyne`` packaging tool if it is missing.

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
* install Xcode command-line tools (``xcode-select --install``)

#### Homebrew
* install pkg-config metadata support, FFmpeg, and WavPack (``brew install pkgconf ffmpeg wavpack``)
* install dylibbundler (``brew install dylibbundler``) - needed only for the bundledeps step, see below

#### Macports
* install pkg-config metadata support, FFmpeg, and WavPack (``sudo port install pkgconfig ffmpeg wavpack``)
* install dylibbundler (``sudo port install dylibbundler``) - needed only for the bundledeps step, see below

### Build

#### Building the application
* clone the repo, CD into the repo root, and run ``mise run build``
* (note that the first build will take some time as it will download and build the UI library)
* run ``mise run package:macos`` to generate the .app bundle. The task installs the ``fyne`` packaging tool if it is missing.
* At this point, the Supersonic.app bundle can be copied to Applications and it will run on your machine, but it depends on the package-manager FFmpeg installation.
* To copy the dependencies into the app bundle, and make it truly portable, run the appropriate command for your package manager:
  - **Homebrew**: ``mise run bundledeps:macos:homebrew``
  - **Macports**: ``mise run bundledeps:macos:macports``

## Build instructions (Windows)

### Install dependencies
* install mise and run ``mise install`` from the repo root
* install MSYS2 and the required packages for the Fyne toolkit (follow instructions for Windows at https://developer.fyne.io/started/)
* install FFmpeg and WavPack (in the MSYS2 terminal, ``pacman -S mingw-w64-x86_64-ffmpeg mingw-w64-x86_64-wavpack``)

### Build
* in the MSYS2 terminal: clone the repo, CD into the repo root, and run ``mise run build``
* (note that the first build will take some time as it will download and build the UI library)
* **Note**: The .exe dynamically links to MSYS2 FFmpeg dependency DLLs and must be started from the MSYS2 terminal, or all dependency DLLs must be copied to the same folder as the .exe.
* Improvements to Windows build process will be forthcoming

## Build instructions (dev container)

See `.devcontainer/DEVCONTAINER.md` for more info
