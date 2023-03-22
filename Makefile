icon_path = ./res/appicon-500.png
icon_path_mac = ./res/appicon-macos.png
app_name = Supersonic

build:
	go build

# dylibbundler doesn't seem to pick up on the Python framework dependency,
# so the last 3 cmds move it over manually. This is a bit fragile though
# since it assumes a specific location and version of the dependency
package_macos:
	fyne package -os darwin -name $(app_name) -icon $(icon_path_mac)
	dylibbundler -od -b -x ./Supersonic.app/Contents/MacOS/supersonic -d ./Supersonic.app/Contents/Frameworks/ -p @executable_path/../Frameworks/
	install_name_tool -change "/usr/local/opt/python@3.11/Frameworks/Python.framework/Versions/3.11/Python" "@executable_path/../Frameworks/Python" "./Supersonic.app/Contents/Frameworks/libvapoursynth-script.0.dylib"
	cp /usr/local/opt/python@3.11/Frameworks/Python.framework/Versions/3.11/Python ./Supersonic.app/Contents/Frameworks
	codesign --force --deep --preserve-metadata=entitlements,requirements,flags,runtime --sign - "./Supersonic.app/Contents/MacOS/supersonic"

package_windows:
	fyne package -os windows -name $(app_name) -icon $(icon_path)

package_linux:
	fyne package -os linux -name $(app_name) -icon $(icon_path)
