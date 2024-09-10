#!/bin/bash
if (ls supersonic);
then
mkdir -p Supersonic.AppDir/usr/lib
mkdir Supersonic.AppDir/usr/bin
cp /usr/lib/x86_64-linux-gnu/libpostproc.so.57 Supersonic.AppDir/usr/lib
cp /usr/lib/x86_64-linux-gnu/libsrt-gnutls.so.1.5 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libx264.so.164 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libcodec2.so.1.2 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libdav1d.so.7 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libvpx.so.9 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libmfx.so.1 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libavdevice.so.60 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libswresample.so.4 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libavfilter.so.9 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libswscale.so.7 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libavformat.so.60 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libavutil.so.58 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libavcodec.so.60 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libuchardet.so.0 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libsixel.so.1 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/liblua5.2.so.0 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libplacebo.so.338 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libmujs.so.3 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libcdio_cdda.so.2 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libcdio_paranoia.so.2 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libmpv.so.2 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libjpeg.so.8 Supersonic.AppDir/usr/lib/
cp /usr/lib/x86_64-linux-gnu/libass.so.9 Supersonic.AppDir/usr/lib/
printf '%s\n' '#!/bin/bash' 'SELF=$(readlink -f "$0")' 'HERE=${SELF%/*}' 'EXEC="${HERE}/usr/bin/supersonic"' 'export LD_LIBRARY_PATH="${HERE}/usr/lib:/usr/lib"' 'exec "${EXEC}";' > Supersonic.AppDir/AppRun
printf '%s\n' '[Desktop Entry]' 'Name=Supersonic' 'Exec=supersonic' 'Icon=ico' 'Type=Application' 'Comment=A lightweight cross-platform desktop client for self-hosted music servers' 'Categories=AudioVideo;' > Supersonic.AppDir/"supersonic.desktop"
chmod +x Supersonic.AppDir/AppRun
chmod +x Supersonic.AppDir/supersonic.desktop
wget https://github.com/dweymouth/supersonic/blob/main/res/appicon.png?raw=true -O Supersonic.AppDir/ico.png
cp supersonic Supersonic.AppDir/usr/bin/
chmod +x Supersonic.AppDir/usr/bin/supersonic
wget https://github.com/AppImage/appimagetool/releases/download/continuous/appimagetool-x86_64.AppImage
chmod +x appimagetool-x86_64.AppImage
./appimagetool-x86_64.AppImage Supersonic.AppDir/
echo "Script finished"
else
echo "executable not found!"
fi
