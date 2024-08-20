# Change Log

## [0.13.1]

### Added
- New translations: French, Polish, Portuguese (Brazillian), Romanian, Spanish
- [#116](https://github.com/dweymouth/supersonic/issues/116) Add tool tips to buttons and truncated labels
- [#462](https://github.com/dweymouth/supersonic/issues/462) Remember the "skip duplicate tracks" setting when adding tracks to playlists
- [#454](https://github.com/dweymouth/supersonic/pull/454) Switch sorting on All Tracks page to recently added

### Fixed
- [#441](https://github.com/dweymouth/supersonic/issues/441) Custom themes didn't work on Windows
- Fixed "jumpy" scrolling on MacOS
- [#452](https://github.com/dweymouth/supersonic/issues/452) Window sometimes misdrawing on startup for Linux (even better fix than last time)
- [#440](https://github.com/dweymouth/supersonic/issues/440) Crash when rendering certain bitmap fonts
- Fixed some more memory leaks from communicating with MPV
- [#446](https://github.com/dweymouth/supersonic/issues/446) Missing space in error dialog text for sharing
- Play queue sometimes not saving on exit
- Pop up play queue crashing on Windows

## [0.13.0] - 2024-07-28

### Added
- [#65](https://github.com/dweymouth/supersonic/issues/65) Add support for translations, and add Chinese and Italian translations
- [#428](https://github.com/dweymouth/supersonic/issues/428) Add a peak/RMS meter visualization
- [#424](https://github.com/dweymouth/supersonic/issues/424) Add track info dialog and context menu item
- [#409](https://github.com/dweymouth/supersonic/issues/409) Add support for Composer (new tracklist column, and row in Track Info dialog)
- [#432](https://github.com/dweymouth/supersonic/issues/432) Use artist sortName for sorting artist grid by name, if present
- [#415](https://github.com/dweymouth/supersonic/issues/415) Add button to sort artist discography by name or year (asc or desc)
- [#317](https://github.com/dweymouth/supersonic/issues/317) Prevent Windows from sleeping while music playing
- Add a new button below the volume control to show a pop-up play queue
- Add a config file option to disable SSL/TLS validation (useful for self-signed certificates)

### Fixed
- [#435](https://github.com/dweymouth/supersonic/issues/435) Japanese and possibly other scripts not truncating properly in grid views
- [#420](https://github.com/dweymouth/supersonic/issues/420) Crash if navigating away from Artist page before cover image loaded
- [#411](https://github.com/dweymouth/supersonic/issues/411) Regression in not detecting dark/light mode for Linux
- [#417](https://github.com/dweymouth/supersonic/issues/417) Artist page not loading artist image for servers that don't support artist largeImageURL
- Memory leak when querying certain MPV properties
- [#421](https://github.com/dweymouth/supersonic/issues/421) Fixed handling of multiple instances of the same track in the play queue
- [#302](https://github.com/dweymouth/supersonic/issues/302) Improve metadata in Linux .desktop file
- Window occasionally misrendered into smaller space on opening for Linux over xwayland (more reliable fix than last release)

## [0.12.0] - 2024-07-01

### Added
- [#363](https://github.com/dweymouth/supersonic/issues/363) Enable drag-and-drop reordering of tracks in the play queue and playlists
- [#403](https://github.com/dweymouth/supersonic/pull/403) Add command-line options to control playback
- Add option to show album years in grid views
- Include radio station results in Quick Search
- Better stringification of play times longer than 1 hour
- Add fallback logic for populating related tracks and artist top tracks if server returns none

### Fixed
- [#397](https://github.com/dweymouth/supersonic/issues/397) Add "Play next/later" options to the related tracks list on the Now Playing page
- [#405](https://github.com/dweymouth/supersonic/pull/405) Change wording of the Add/Edit server form to be less confusing (thanks @mintsoft!)
- Window occasionally misrendered into smaller space on opening for Linux over xwayland
- Don't crash if server returns nil saved play queue but no Subsonic error

## [0.11.0] - 2024-06-05

### Added / Changed
- Updated to Fyne 2.5, adds/fixes the following
  - [#57](https://github.com/dweymouth/supersonic/issues/57) Unable to show CJK characters without setting custom app font
  - [#329](https://github.com/dweymouth/supersonic/issues/329) Adds support for Ctrl+{backspace/delete} to remove words in text inputs
- [#356](https://github.com/dweymouth/supersonic/issues/356), [#372](https://github.com/dweymouth/supersonic/issues/372) Adds synced lyrics support, and Jellyfin lyrics support
  - LrcLib.net is used as a fallback lyrics source and can be disabled in the config file
- [#344](https://github.com/dweymouth/supersonic/issues/344) Improve UX of Add to Playlist dialog (thanks @natilou!)
- [#276](https://github.com/dweymouth/supersonic/issues/276) Show track thumbnails in tracklist views
- [#328](https://github.com/dweymouth/supersonic/issues/328) Adds support for internet radio stations for Subsonic servers
- [#286](https://github.com/dweymouth/supersonic/issues/286) Adds "option button" to right of current track title to bring up action menu
- [#371](https://github.com/dweymouth/supersonic/issues/371) Adds a portable mode
- [#387](https://github.com/dweymouth/supersonic/issues/387) Improves performance and behavior of Random albums sort with upcoming Navidrome releases
- Adds dynamic gradient background to Now Playing page

### Fixed
- [#165](https://github.com/dweymouth/supersonic/issues/165) Last track occasionally missing in album view depending on window size
- [#383](https://github.com/dweymouth/supersonic/issues/383) Album filter button disappearing when restoring pages from history
- [#391](https://github.com/dweymouth/supersonic/issues/391)  Artist radio button sometimes plays radio for wrong artist
- [#378](https://github.com/dweymouth/supersonic/issues/378) Clicking Home button doesn't automatically refresh page


## [0.10.1] - 2024-04-21

### Fixed
- [#369](https://github.com/dweymouth/supersonic/issues/369) Crashing when trying to add a new server
- [#370](https://github.com/dweymouth/supersonic/issues/370) Lyrics don't refresh when playing next song
- What's New dialog wasn't showing when launching updated version
- Crash on exit when saving play queue to server and no track playing

## [0.10.0] - 2024-04-17

### Added
- [#128](https://github.com/dweymouth/supersonic/issues/128) Support mouse back and forward buttons for navigation
- [#238](https://github.com/dweymouth/supersonic/issues/238) Redesigned Now Playing page with large cover art and lyrics
- [#303](https://github.com/dweymouth/supersonic/issues/303) Add context menu entry to create sharing URLs (Subsonic servers only)
- [#337](https://github.com/dweymouth/supersonic/issues/337) Add option to load/save play queue to Subsonic servers instead of locally
- [#350](https://github.com/dweymouth/supersonic/issues/350) Add option to shuffle artist's discography by albums or tracks
- [#351](https://github.com/dweymouth/supersonic/issues/351) Show original and reissue date for albums if available (OpenSubsonic servers)
- [#352](https://github.com/dweymouth/supersonic/pull/352) Add "Song radio" option to tracklist context menus
- [#361](https://github.com/dweymouth/supersonic/issues/361) Add "Play next" option to context menus
- [#366](https://github.com/dweymouth/supersonic/pull/366) Add sorting options to Artists page

### Fixed
- [#330](https://github.com/dweymouth/supersonic/issues/330) Can't reorder track in play queue more than once
- [#345](https://github.com/dweymouth/supersonic/issues/345) Playlist's comment text doesn't wrap
- [#346](https://github.com/dweymouth/supersonic/issues/346) Pressing down arrow on tracklist with few tracks could cause some to be hidden
- [#364](https://github.com/dweymouth/supersonic/issues/364) "Move down" option in play queue context menu was moving to bottom

## [0.9.1] - 2024-02-25

### Added
- [#327](https://github.com/dweymouth/supersonic/issues/327) Show visual loading cue when searching and no results yet
- [#22](https://github.com/dweymouth/supersonic/issues/22) Arrow key scrolling and focus keyboard traversal to grid and list views
- Ctrl+Q keyboard shortcut to quit app on Windows and Linux

### Fixed
- A few MPRIS bugs (thanks @adamantike!)
- [#319](https://github.com/dweymouth/supersonic/issues/319) Show modal dialog when connecting to server to block UI interaction which could cause crash
- [#321](https://github.com/dweymouth/supersonic/issues/321) High CPU use when certain dialogs were shown
- [#326](https://github.com/dweymouth/supersonic/issues/326) Make tracklist widget more efficient
- Lower CPU use when text entry fields are focused

## [0.9.0] - 2024-01-27

### Added
- [#33](https://github.com/dweymouth/supersonic/issues/33) Allow reordering of tracks in the play queue
- [#218](https://github.com/dweymouth/supersonic/issues/218) Highlight the icon of the current page's navigation button (thanks @natilou!)
- [#273](https://github.com/dweymouth/supersonic/issues/273) Show release type in album page header (for OpenSubsonic servers)
- [#309](https://github.com/dweymouth/supersonic/issues/309) Setting to save and reload play queue on exit/startup
- [#315](https://github.com/dweymouth/supersonic/issues/315) Use most recent playlist as default in "Add to playlist" dialog
- [#316](https://github.com/dweymouth/supersonic/issues/316) Option to show desktop notifications on track change
- Added icons to context menu items

### Fixed
- [#313](https://github.com/dweymouth/supersonic/issues/313) OpenGL startup error on some hardware

## [0.8.2] - 2023-12-16

### Fixed
- [#295](https://github.com/dweymouth/supersonic/issues/295) Occasional crash when showing album info dialog, especially on Jellyfin
- [#294](https://github.com/dweymouth/supersonic/issues/294) Unable to connect to Airsonic servers not supporting latest Subsonic API
- [#293](https://github.com/dweymouth/supersonic/issues/293) Long album titles overflow the bounds of info dialog
- [#292](https://github.com/dweymouth/supersonic/issues/292) ReplayGain "prevent clipping" setting was reversed

## [0.8.1] - 2023-12-06

### Fixed
- [#284](https://github.com/dweymouth/supersonic/issues/284) Artist Radio on Jellyfin not generating a fresh mix if clicked a second time
- [#283](https://github.com/dweymouth/supersonic/issues/283) On Jellyfin, a long artist biography could overflow the page header
- [#288](https://github.com/dweymouth/supersonic/issues/288) Systray icon missing on Linux since 0.7.0

## [0.8.0] - 2023-11-15

### Added
- [#98](https://github.com/dweymouth/supersonic/issues/98) Add support for connecting to Jellyfin servers
- [#157](https://github.com/dweymouth/supersonic/issues/157) Add "Auto" ReplayGain option to auto-choose between Track and Album mode
- [#179](https://github.com/dweymouth/supersonic/issues/179) Add experimental setting for changing UI scaling

### Fixed
- [#282](https://github.com/dweymouth/supersonic/issues/282) Crash when repeatedly searching the All Tracks page quickly 
- [#275](https://github.com/dweymouth/supersonic/issues/275) What's New dialog sometimes continuing to re-show on subsequent launches
- [#277](https://github.com/dweymouth/supersonic/issues/277) Config settings sometimes not being saved due to abnormal exits
- Don't crash with zero-track albums or tracks with no artists
- Slightly improved the time it takes to check server connectivity

## [0.7.0] - 2023-11-07

### Added
- [#263](https://github.com/dweymouth/supersonic/issues/263) "Quick Search" popup dialog to search entire library from any page
- [#245](https://github.com/dweymouth/supersonic/issues/245) UI Refresh and misc. internal improvements from moving to Fyne 2.4
- New [custom theme](https://github.com/dweymouth/supersonic/wiki/Custom-Themes) colors - Hyperlink and PageHeader
- Added support for displaying WEBP images (Thanks @adamantike!)
- Added playback setting to force disable server transcoding

### Fixed
- [#270](https://github.com/dweymouth/supersonic/issues/270) Crash when removing from the currently playing track to the end of the play queue
- UI hang if playback slow to begin after clicking on green play button on cover images

## [0.6.0] - 2023-10-23

### Added
- [#259](https://github.com/dweymouth/supersonic/pull/259) Support for multi-valued artists and genres through the new OpenSubsonic API extensions
- [#233](https://github.com/dweymouth/supersonic/issues/233) Mac OS media center integration incl. media key support (Thanks @zackslash!)
- [#77](https://github.com/dweymouth/supersonic/issues/77) Builds off latest `main` branch available via Github Actions for Ubuntu and Intel Macs
- [#252](https://github.com/dweymouth/supersonic/issues/252) Sorting now enabled on genres and playlists list views
- [#261](https://github.com/dweymouth/supersonic/issues/261) Sortable tracklist columns provide a visual indication on hover
- [#249](https://github.com/dweymouth/supersonic/issues/249) "Rescan library" option added to settings menu to trigger a library scan on the server

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
