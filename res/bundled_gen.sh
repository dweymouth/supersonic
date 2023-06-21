#!/bin/sh

fyne bundle -package res -prefix Res appicon-256.png > bundled.go
fyne bundle -append -prefix Res icons/freepik/disc.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/disc-invert.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/headphones.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/headphones-invert.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/heart-filled.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/heart-filled-invert.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/heart-outline.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/heart-outline-invert.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/musicnotes.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/musicnotes-invert.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/people.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/people-invert.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/playbutton.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/playlist.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/playlist-invert.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/podcast.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/podcast-invert.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/theatermasks.png >> bundled.go
fyne bundle -append -prefix Res icons/freepik/theatermasks-invert.png >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/shuffle.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/shuffle-invert.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/star-outline.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/star-filled.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/grid.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/list.svg >> bundled.go
fyne bundle -append -prefix Res icons/publicdomain/filter.svg >> bundled.go

fyne bundle -append -prefix Res themes/default.toml >> bundled.go

fyne bundle -append -prefix Res ../LICENSE >> bundled.go
fyne bundle -append -prefix Res licenses/BSDLICENSE >> bundled.go
fyne bundle -append -prefix Res licenses/MITLICENSE >> bundled.go
