#!/bin/sh

fyne bundle -package res -prefix Res appicon-256.png > bundled.go
fyne bundle -append -prefix Res icons/coreui/playlist-add-next.svg >> bundled.go
fyne bundle -append -prefix Res icons/freepik/playbutton.png >> bundled.go
fyne bundle -append -prefix Res icons/majesticons/library.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/cast.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/disc.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/headphones.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/heart-filled.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/heart-outline.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/infinity.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/musicnotes.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/oscilloscope.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/people.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/playlist.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/playqueue.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/sidebar.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/star-outline.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/star-filled.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/theatermasks.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/grid.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/list.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/filter.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/save.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/saveas.svg >> bundled.go
fyne bundle -append -prefix Res icons/remix_design/broadcast.svg >> bundled.go
fyne bundle -append -prefix Res icons/remix_design/repeat.svg >> bundled.go
fyne bundle -append -prefix Res icons/remix_design/repeatone.svg >> bundled.go
fyne bundle -append -prefix Res icons/remix_design/shuffle.svg >> bundled.go
fyne bundle -append -prefix Res icons/remix_design/share.svg >> bundled.go
fyne bundle -append -prefix Res icons/remix_design/updownarrow.svg >> bundled.go

fyne bundle -append -prefix Res themes/default.toml >> bundled.go

fyne bundle -append -prefix Res LICENSE >> bundled.go
fyne bundle -append -prefix Res licenses/BSDLICENSE >> bundled.go
fyne bundle -append -prefix Res licenses/MITLICENSE >> bundled.go
