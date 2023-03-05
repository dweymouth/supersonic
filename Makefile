icon_path = ./res/appicon-500.png

build:
	go build

package_macos:
	fyne package -os darwin -icon $(icon_path)
	mv ./supersonic.app Supersonic.app
	dylibbundler -od -b -x ./Supersonic.app/Contents/MacOS/supersonic -d ./Supersonic.app/Contents/Frameworks/ -p @executable_path/../Frameworks/

package_windows:
	fyne package -os windows -icon $(icon_path)
