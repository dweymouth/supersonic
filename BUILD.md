# Building Supersonic from source

To build Supersonic from source, first clone the Git repo or download the source code zip from the latest release. Then follow the instructions for your platform below.

## Build instructions (Linux)

### Ubuntu dependencies
* ``sudo snap install --classic go``
* Make sure the Go bin directory is in your `$PATH` (`export PATH=~/go/bin:$PATH`)
* ``sudo apt install libmpv-dev gcc libegl1-mesa-dev xorg-dev``

### Fedora dependencies
* ``sudo dnf install golang mpv-devel libX11-devel libXcursor-devel libXrandr-devel libXinerama-devel libXi-devel libglvnd-devel libXxf86vm-devel``

### Build
* clone the repo, CD into the repo root, and run ``make build``
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

#### Homebrew
* install libmpv (``brew install mpv``)
* install dylibbundler (``brew install dylibbundler``) - needed only the bundledeps step, see below

#### Macports
* install mpv with the libmpv variant (``sudo port install mpv +libmpv``)
* install dylibbundler (``sudo port install dylibbundler``) - needed only the bundledeps step, see below

### Build

#### Building the application
* clone the repo, CD into the repo root, and run ``make build``
* (note that the first build will take some time as it will download and build the UI library)
* run ``make package_macos`` to generate the .app bundle
* **If** you are on Mac OS **High Sierra** through **Catalina**, run ``make bundledeps_macos_highsierra`` and you are done! Otherwise, continue reaading.
* At this point, the Supersonic.app bundle can be copied to Applications and it will run on your machine, but it depends on the brew installation of mpv
* To copy the dependencies into the app bundle, and make it truly portable, run the appopriate command for your package manager:
  - **Homebrew**: ``make bundledeps_macos_homebrew``
  - **Macports**: ``make bundledeps_macos_macports``

## Build instructions (Windows)

### Install dependencies
* install go (https://go.dev/doc/install)
* install MSYS2 and the required packages for the Fyne toolkit (follow instructions for Windows at https://developer.fyne.io/started/)
* install libmpv (in the MSYS2 terminal, ``pacman -S mingw-w64-x86_64-mpv``)

### Build
* in the MSYS2 terminal: clone the repo, CD into the repo root, and run ``go build``
* (note that the first build will take some time as it will download and build the UI library)
* **Note**: The .exe dynamically links to MSYS2 libmpv dependency dlls and must be started from the MSYS2 terminal, or all dependency DLLS must be copied to the same folder as the .exe
* -> If you obtain a statically built mpv-2.dll (containing all its dependencies), and rename it to libmpv-2.dll, you can place just that DLL in the same directory as the EXE, and it should run
* Improvements to Windows build process will be forthcoming

## Build instructions (dev container)

See `.devcontainer/DEVCONTAINER.md` for more info