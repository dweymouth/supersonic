#!/bin/sh

PYTHON_PATH=$(otool -L ./Supersonic.app/Contents/Frameworks/libvapoursynth-script.0.dylib | grep 'Python' | awk '{print $1}')
install_name_tool -change "$PYTHON_PATH" "@executable_path/../Frameworks/Python" "./Supersonic.app/Contents/Frameworks/libvapoursynth-script.0.dylib"
cp "$PYTHON_PATH" ./Supersonic.app/Contents/Frameworks
