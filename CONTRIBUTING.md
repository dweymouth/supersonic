# Contributing

When contributing to this repository, please first discuss the change you wish to make via issue, discussion topic, or any other method with the owners of this repository before making a change.

## Development Environment

To set up your development environment, you need the following things:

* The build dependencies for the Fyne toolkit on your OS (see README.md)
* Libmpv and development headers installed (e.g. via apt, brew, or MinGW pacman)
* A Subsonic server to connect to for testing (your own server or the Navidrome demo server)

Alternately you can use a Dev Container.  See `.devcontainer/DEVCONTAINER.md` for more info

## Pull Request Process

Before opening a pull request, please ensure the following things:

1. You have discussed your proposed changes with the project maintainer(s)
2. Your PR references any issues that will be closed/fixed by your contribution
3. You have thorougly tested the change

At this time, the `main` branch is intended to be 'shippable' at any time, meaning only bug fixes and complete, tested features should be merged to main. Experimental or partially-implemented new features can be merged to `develop` or to a specific feature branch.

## Project Structure

As this application is in early development, major refactorings and re-organization of the code structure may occur frequently as needed. As a rough guide to newcomers, the high-level project structure is as follows:

* `backend` - To the extent possible all backend logic should reside in this package
  - `playbackmanager.go` - Subsonic-aware high level playback APIs. Manages play queue and communication between the UI and various player backends.
  - `player` - Interface and package for different player backends, currently MPV (local), and Jukebox, and in the future, DLNA and Chromecast
* `res` - Application resource files
* `sharedutil` - Utility functions shared between the backend and UI layers
* `ui` - The UI and UI business logic
  - `browsing` - The individual "pages" or views (album, artist, playlists, etc), plus the browsing component which shows the pages and manages navigation history. Pages are responsible for connecting the callbacks for their constituent widgets. Simple business logic may be implemented directly in the page's code, but more involved logic should be extracted to the Controller.
  - `controller` - UI business logic, currently mostly related to dialog handling
  - `dialogs` - Any UI components to be displayed in a popup dialog. Keep business logic in `controller`
  - `layouts` - Custom Fyne layouts
  - `util` - Util functions not needed in backend
  - `widgets` - Individual custom widgets (tracklist, album grid, etc). Code should be related to UI presentation only, no business logic. Widgets should expose callbacks to signal an intended user action rather than directly implementing business logic (e.g. OnPlayAlbum, etc)
