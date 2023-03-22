icon_path = ./res/appicon-500.png
icon_path_mac = ./res/appicon-macos.png
app_name = Supersonic

build:
	go build

package_macos:
	fyne package -os darwin -name $(app_name) -icon $(icon_path_mac)
	dylibbundler -od -b -x ./Supersonic.app/Contents/MacOS/supersonic -d ./Supersonic.app/Contents/Frameworks/ -p @executable_path/../Frameworks/

package_windows:
	fyne package -os windows -name $(app_name) -icon $(icon_path)

package_linux:
	fyne package -os linux -name $(app_name) -icon $(icon_path)
