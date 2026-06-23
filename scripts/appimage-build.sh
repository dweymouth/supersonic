#!/bin/bash -e

if [[ ! -x supersonic ]]; then
  echo "executable not found!"
  exit 1
fi

rm -rf Supersonic.AppDir/
mkdir -p Supersonic.AppDir/usr/lib
mkdir -p Supersonic.AppDir/usr/bin

# Add binary dependencies except those on the AppImage exclude list.
EXCLUDE=$(wget -nv -O - https://github.com/AppImageCommunity/pkg2appimage/raw/refs/heads/master/excludelist | grep -v "^#")
ldd ./supersonic | awk '/=> \\/.*x86_64-linux-gnu/ { print $1, $3 }' | sort | while read -r file path; do
  if [[ "$EXCLUDE" == *"$file"* ]]; then
    echo "exclude $file"
  else
    echo "add $path"
    cp "$path" "Supersonic.AppDir/usr/lib/$file"
  fi
done

cat > Supersonic.AppDir/AppRun <<'EOF'
#!/bin/bash
SELF=$(readlink -f "$0")
HERE=${SELF%/*}
EXEC="${HERE}/usr/bin/supersonic"
export LD_LIBRARY_PATH="/usr/lib64:/lib64:/usr/lib/x86_64-linux-gnu:/usr/lib:${HERE}/usr/lib/"
exec "${EXEC}"
EOF

cat > Supersonic.AppDir/supersonic.desktop <<'EOF'
[Desktop Entry]
Name=Supersonic
Exec=supersonic
Icon=ico
Type=Application
Comment=A lightweight cross-platform desktop client for self-hosted music servers
Categories=AudioVideo;
EOF

chmod +x Supersonic.AppDir/AppRun
chmod +x Supersonic.AppDir/supersonic.desktop
wget -nv https://raw.githubusercontent.com/dweymouth/supersonic/main/res/appicon.png -O Supersonic.AppDir/ico.png
cp supersonic Supersonic.AppDir/usr/bin/
chmod +x Supersonic.AppDir/usr/bin/supersonic

wget -nv -nc https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
chmod +x appimagetool-x86_64.AppImage
./appimagetool-x86_64.AppImage Supersonic.AppDir/

echo "Script finished"
