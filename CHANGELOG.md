# Change Log

## Unreleased

### Added
- [#145](https://github.com/dweymouth/supersonic/issues/145) Add rating column to tracklist with 5-star rating widget
- [b017995](https://github.com/dweymouth/supersonic/commit/b01799550ded0c6a8f33913827df23818f9a7353) Add Ctrl+W (Cmd+W on Mac) shortcut to close to tray if enabled
- [#95](https://github.com/dweymouth/supersonic/issues/95) Enable click-to-seek behavior in seek bar
- [#40](https://github.com/dweymouth/supersonic/issues/40) Add grid view option to playlists page
- Add tracklist context menu items to bulk set rating and favorite

### Fixed
- [#148](https://github.com/dweymouth/supersonic/issues/148) Fix potential crash when searching for albums
- [#149](https://github.com/dweymouth/supersonic/issues/149) Tracklist columns occasionally misaligned when shrinking window
- [#142](https://github.com/dweymouth/supersonic/issues/142) Disable hyperlinks for track artists that are not also album artists
- [#141](https://github.com/dweymouth/supersonic/issues/141) Fix crash when navigating to "Top Tracks" on artist page not yet fully loaded or invalid artist
- [56b6709](https://github.com/dweymouth/supersonic/commit/56b67097b9259ad16e9799b39df644d323405bbe) Don't pluralize "tracks" for albums and playlists with only one track

## [0.1.0] - 2023-04-22

### Added
- [#107](https://github.com/dweymouth/supersonic/issues/107) Add playback settings dropdown to choose output audio device
- [#119](https://github.com/dweymouth/supersonic/issues/119) Add file path column to tracklist view
- [#117](https://github.com/dweymouth/supersonic/issues/117) Add (optional) system tray menu and close to tray support
- [#55](https://github.com/dweymouth/supersonic/issues/55) Show disc number and disc count for multi-disc albums
- [#115](https://github.com/dweymouth/supersonic/issues/115) Add search bar to artist, genres, and playlists pages
- [#104](https://github.com/dweymouth/supersonic/issues/104) Add alternate (e.g. external) hostname to server connection config
- [#136](https://github.com/dweymouth/supersonic/issues/136) Add "..." button to album page with menu to add album to queue or playlist
- [#70](https://github.com/dweymouth/supersonic/issues/70) Add searchable "All Tracks" page, with button to play random tracks
- [#63](https://github.com/dweymouth/supersonic/issues/63) Add experimental support for setting custom application font

### Fixes
- [198110d](https://github.com/dweymouth/supersonic/commit/198110d1dc412c8ce5c7ec4e4cd2f4206899f0b5) Don't show update available prompt if the found version is the same as the running app version
- [#120](https://github.com/dweymouth/supersonic/issues/120),[#87](https://github.com/dweymouth/supersonic/issues/87) Update Mac build process to support OS versions back to High Sierra (thanks @whorfin!)
- [#125](https://github.com/dweymouth/supersonic/issues/125) Navigating back twice to an albums page with search result clears search state

## [0.1.0-beta] - 2023-04-01

### Added
- [#39](https://github.com/dweymouth/supersonic/issues/39) Add caching of artist images
- [#94](https://github.com/dweymouth/supersonic/issues/94) Add Cmd+[ Cmd+] back/forward shortcuts for Mac (alongside existing Cmd+Left/Right)
- [#96](https://github.com/dweymouth/supersonic/issues/96) Make scrobbling thresholds configurable
- [#21](https://github.com/dweymouth/supersonic/issues/21) Add ability to reorder tracks within a playlist (via context menu)
- [#99](https://github.com/dweymouth/supersonic/issues/99) Add Year (asc + desc) sort orders to album page
- [#102](https://github.com/dweymouth/supersonic/issues/102) Add ReplayGain support (requires files to be tagged on server and transcoding to preserve tags)
- [#106](https://github.com/dweymouth/supersonic/issues/106) Automatically check for updates
- [#12](https://github.com/dweymouth/supersonic/issues/12) Add settings dialog
- [d67dfa0](https://github.com/dweymouth/supersonic/commit/d67dfa07b35aff9e3aff7ca091180d8104a64db4) Add file size column to tracklist view
- [0a11df8](https://github.com/dweymouth/supersonic/commit/0a11df8c07efd0ff4fd540bc0a5bc64946f4d61c) Limit the size of MPV's in-memory audio cache

### Fixed
- [#90](https://github.com/dweymouth/supersonic/issues/90) Wrong covers get loaded for albums if server has different IDs for album and cover art
- [#88](https://github.com/dweymouth/supersonic/issues/88) Incorrect icon size for Mac OS
- [#87](https://github.com/dweymouth/supersonic/issues/87) Fix dependency bundling for Mac OS
- [#45](https://github.com/dweymouth/supersonic/issues/45) Tracklist multi-select on Linux now works properly (Ctrl+click, Shift+click)
- [#109](https://github.com/dweymouth/supersonic/issues/109) Memory leak in image rendering
- [f34d432](https://github.com/dweymouth/supersonic/commit/f34d4329b175e7d1a33a006bec951c2c64e6e978) Don't show tracklist row hover indicator when favorite toggle is hovered
- [98fb148](https://github.com/dweymouth/supersonic/commit/98fb1483e6ba329effb8995a879572138947aee7) Migrate to Fyne 2.3.3 (modest memory improvements, fixes seek slider "wobble", fixes possible issues on M2 Macs)

## [0.0.1-alpha2] - 2023-03-17

### Added
- [#73](https://github.com/dweymouth/supersonic/issues/73) Added "Play Random" button to genre page to play random songs
- [#60](https://github.com/dweymouth/supersonic/issues/60) Add Top Tracks view to artist page
- [#82](https://github.com/dweymouth/supersonic/issues/82) Added "Play Artist Radio" feature
- [#79](https://github.com/dweymouth/supersonic/issues/79) Support legacy unsalted password auth
- [#48](https://github.com/dweymouth/supersonic/issues/48) Send "Now Playing" scrobble to server when beginning new track
- [#76](https://github.com/dweymouth/supersonic/issues/76) Added "Audio Exclusive" playback mode
- [60b7b93](https://github.com/dweymouth/supersonic/commit/60b7b93ee27f22781dbcddc8bd429d0ede8514fb) Add support for deleting playlists and editing playlist metadata

### Fixed
- [#62](https://github.com/dweymouth/supersonic/issues/62) Set a placeholder image if an album cover is not available
- [#66](https://github.com/dweymouth/supersonic/issues/66) Refreshing artist page duplicates list of similar artists 
- [#37](https://github.com/dweymouth/supersonic/issues/37) Infinite scrolling album views stop loading if you scroll with scrollbar immediately to bottom
- [#78](https://github.com/dweymouth/supersonic/issues/78) Favorite button on Artist page always shows unfavorited initially
- [#64](https://github.com/dweymouth/supersonic/issues/64) Improve scrolling smoothness in album grid view

## [0.0.1-alpha1] - 2023-03-05

First release!
