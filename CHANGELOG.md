# Change Log

## [0.5.2] - 2023-09-18

### Fixed
- [#248](https://github.com/dweymouth/supersonic/issues/248) Album lists not populating when connecting to Astiga
- [#230](https://github.com/dweymouth/supersonic/issues/230) MPRIS SetPosition call fails
- [cf09912](https://github.com/dweymouth/supersonic/commit/cf09912474cc910fbbc23195b496206e3de60d36) Album card on artist discography page occasionally showed artist name instead of year
- [ce68593](https://github.com/dweymouth/supersonic/commit/ce685933be4ea7632fc9087269246e40c0d0a284) Fix MPRIS invalid track object path for some Subsonic servers
- [64ec698](https://github.com/dweymouth/supersonic/commit/64ec6982029b7462793fdd32db4aea900f61c6bc) Play count did not increment on Favorites page tracklist
- [a2a56b3](https://github.com/dweymouth/supersonic/commit/a2a56b3d8caff0a0ad0daba756a92bd083838412) Ctrl+F should focus search bar on Artists page
- [5c9d338](https://github.com/dweymouth/supersonic/commit/5c9d338b620c6c03b0a732e41e6d9ba5e28e6f6d) Prevent multiple popup cover images if thumbnail clicked twice

### Changed
- [#223](https://github.com/dweymouth/supersonic/issues/223) Settings dialog opens to last active tab
- [#227](https://github.com/dweymouth/supersonic/issues/227) Add size limit and periodic pruning to on-disc image cache
- [1c27a26](https://github.com/dweymouth/supersonic/commit/1c27a267b9bc8c7c0061e6a88161485432035f33) Added Cmd+, shortcut on Mac to launch settings dialog
- [f2e4805](https://github.com/dweymouth/supersonic/commit/f2e4805b29774db53a1345e5cfde02b0623ec846) Add ContentCreated, UseCount to the MPRIS metadata

## [0.5.1] - 2023-07-17

### Fixed
- [#222](https://github.com/dweymouth/supersonic/issues/222) Regression: play button on Album and Playlist page playing incorrect album
- [#221](https://github.com/dweymouth/supersonic/issues/221) MPRIS reported duration is incorrect
- [#220](https://github.com/dweymouth/supersonic/issues/220) MPRIS SetPosition not implemented

## [0.5.0] - 2023-07-15

### Added
- [#75](https://github.com/dweymouth/supersonic/issues/75) Add MPRIS integration
- [#105](https://github.com/dweymouth/supersonic/issues/105) Add dialog to view album description and Last.fm, Musicbrainz links (thanks @natilou!)
- [#184](https://github.com/dweymouth/supersonic/issues/184) Highlight play icon for currently playing track in tracklist (thanks @adamantike!)
- [#216](https://github.com/dweymouth/supersonic/issues/216) Release Audio Exclusive mode while paused to allow other applications to play audio

### Fixed
- [#42](https://github.com/dweymouth/supersonic/issues/42) Substantial improvement to memory use when navigating quickly through pages
- [#212](https://github.com/dweymouth/supersonic/issues/212) Crash on Wayland with 2x scaling factor

## [0.4.0] - 2023-06-24

### Fixed
- [#196](https://github.com/dweymouth/supersonic/pull/196) Track correctly scrobbles if removed from play queue while playing (thanks @adamantike!)

### Added
- [#207](https://github.com/dweymouth/supersonic/pull/207) Add 15 band graphic equalizer
- [#203](https://github.com/dweymouth/supersonic/pull/203) Add custom theme support
- [fedb9fd](https://github.com/dweymouth/supersonic/commit/fedb9fd70d3374b6cb68f6d434c9e1546ba77678) Add playback status line to bottom of Now Playing page
- [#197](https://github.com/dweymouth/supersonic/pull/197) Add repeat playback modes (thanks @adamantike!)
- [#187](https://github.com/dweymouth/supersonic/pull/187) Add Home button to navigation bar (thanks @natilou!)
- [#191](https://github.com/dweymouth/supersonic/pull/191),[#201](https://github.com/dweymouth/supersonic/pull/201),[#206](https://github.com/dweymouth/supersonic/pull/206) Add ability to download tracks, albums, and playlists (thanks @natilou!)
- [c19edcc](https://github.com/dweymouth/supersonic/commit/c19edccf809988207a6bc257653ddcd5a3b7dae8) Add click-to-set to volume slider (thanks @adamantike!)
- [#193](https://github.com/dweymouth/supersonic/issues/193) Attempting to launch a second instance refocuses the existing window

## [0.3.1] - 2023-06-03

### Fixed
- [#182](https://github.com/dweymouth/supersonic/issues/182) Regression: crash on Funkwhale when navigating to Artist page
- [#178](https://github.com/dweymouth/supersonic/issues/178) Tracklist selection doesn't update properly if sorting with items selected
- [#176](https://github.com/dweymouth/supersonic/issues/176) Regression: searching genres list broken
- [#175](https://github.com/dweymouth/supersonic/issues/175) Incorrect version number shown in Mac system About dialog
- [066be7b](https://github.com/dweymouth/supersonic/commit/066be7bb8d29024b98f4f34419dddce3defe14ab) Favorite button on Artist page showing opposite state
- [8b238fd](https://github.com/dweymouth/supersonic/commit/8b238fd2396205ccb7fd0a7afa08a2cd1b36990e) Missing album icon in cover image placeholder for album grid views

## [0.3.0] - 2023-05-25

### Added
- [#168](https://github.com/dweymouth/supersonic/issues/168) Add multi-server support (add, delete, switch servers)
- [#99](https://github.com/dweymouth/supersonic/issues/99) Add filter options to album grid views
- [#170](https://github.com/dweymouth/supersonic/issues/170) Add ability to sort tracklist by individual columns
- [#151](https://github.com/dweymouth/supersonic/issues/151) Add grid view for artists
- [#166](https://github.com/dweymouth/supersonic/issues/166) Disable back/forward buttons when no navigation is possible (thank you @davidhaymond!)
- [#156](https://github.com/dweymouth/supersonic/issues/156) Allow adjusting volume slider with the scroll wheel
- [3e448ed](https://github.com/dweymouth/supersonic/commit/3e448ed995bd58354ca809c6fa17de875cd40c45) Add context menu to cover thumbnail in bottom panel to rate/favorite/add to playlist current track
- [2083af3](https://github.com/dweymouth/supersonic/commit/2083af3935275f84fb3ab794fe45abcd67100bee) Add shuffle option to tracklist context menu

### Fixed
- [#158](https://github.com/dweymouth/supersonic/issues/158) Crash in "Play artist radio" feature when server returns empty response 
- [#161](https://github.com/dweymouth/supersonic/issues/161) Race condition occasionally causing album search results to be incorrect
- [#163](https://github.com/dweymouth/supersonic/issues/163) Inability to log in on KDE and other systems where a credential storage service is not available
- [#172](https://github.com/dweymouth/supersonic/issues/172) Navigating back to Favorites page twice always resets view to Albums
- [b2dce99](https://github.com/dweymouth/supersonic/commit/b2dce99714a650707b4c2dcda51df47f07a98976) Fixed a rare crash from a race condition with album searching
- [dde8cd0](https://github.com/dweymouth/supersonic/commit/dde8cd02675ab020af2fa58adbf02f324a7a1c3b) Fixed a resource leak of HTTP connections
- [c97edef](https://github.com/dweymouth/supersonic/commit/c97edef403acc92aabe8940675ba9de5152a55a6) Ctrl+A should select all tracks on Favorites page

## [0.2.0] - 2023-05-05

### Added
- [#145](https://github.com/dweymouth/supersonic/issues/145) Add rating column to tracklist with 5-star rating widget
- [#101](https://github.com/dweymouth/supersonic/issues/101) Add light theme, and optional auto-switching between dark and light
- [#40](https://github.com/dweymouth/supersonic/issues/40) Add grid view option to playlists page with playlist cover images
- [#95](https://github.com/dweymouth/supersonic/issues/95) Enable click-to-seek behavior in seek bar
- [#136](https://github.com/dweymouth/supersonic/issues/136) Add context menu to items in grid views (albums and playlists)
- [b017995](https://github.com/dweymouth/supersonic/commit/b01799550ded0c6a8f33913827df23818f9a7353) Add Ctrl+W (Cmd+W on Mac) shortcut to close to tray if enabled
- [5b6b6b4](https://github.com/dweymouth/supersonic/commit/5b6b6b4fe2388701a780e8fcad1690b63ff55a7e),[a9eac50](https://github.com/dweymouth/supersonic/commit/a9eac505eae1ec144d14547133ca36fafc376837) Add tracklist context menu items to bulk set favorite and rating
- [fd01e37](https://github.com/dweymouth/supersonic/commit/fd01e3726057fae85c4721e01852a583dfd73929) Add "meatball menu" actions to playlist page
- [a9263c8](https://github.com/dweymouth/supersonic/commit/a9263c8d4edaa4608381f034f42c35a46548eb98) Add settings option to choose startup page
- [cbc724d](https://github.com/dweymouth/supersonic/commit/cbc724df93c32a28454fc3f8f4be17fd8e16c362) Migrate to Fyne 2.3.4 (few performance and memory improvements)
- [053474f](https://github.com/dweymouth/supersonic/commit/053474f669e6c8df1f9213057a74c65a4bf6dbda) Track title in bottom panel links to track location on Now Playing page
- [7819416](https://github.com/dweymouth/supersonic/commit/78194165f434b5f9d7c9302cb2773bc629df90f0) Clicking thumbnail in bottom panel shows full size cover
- [de36149](https://github.com/dweymouth/supersonic/commit/de36149ba4d9e5db0757507f50232184b2c4345a) Add volume adjustment options to system tray menu

### Fixed
- [#148](https://github.com/dweymouth/supersonic/issues/148) Fix crash when searching for albums on Airsonic servers
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
