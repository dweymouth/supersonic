#!/bin/sh

fyne bundle -package res -prefix Res appicon-256.png > bundled.go
fyne bundle -append -prefix Res icons/freepik/playbutton.png >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/disc.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/headphones.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/heart-filled.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/heart-outline.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/musicnotes.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/people.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/playlist.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/shuffle.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/shuffle-invert.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/star-outline.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/star-filled.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/theatermasks.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/grid.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/list.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/filter.svg >> bundled.go
fyne bundle -append -prefix Res icons/remix_design/repeat.svg >> bundled.go
fyne bundle -append -prefix Res icons/remix_design/repeatone.svg >> bundled.go

fyne bundle -append -prefix Res themes/default.toml >> bundled.go

fyne bundle -append -prefix Res ../LICENSE >> bundled.go
fyne bundle -append -prefix Res licenses/BSDLICENSE >> bundled.go
fyne bundle -append -prefix Res licenses/MITLICENSE >> bundled.go
