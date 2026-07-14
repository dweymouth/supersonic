#!/bin/sh

VAPOURSYNTH_LIB=./Supersonic.app/Contents/Frameworks/libvapoursynth-script.0.dylib

# Check if VapourSynth library exists before processing
if [ ! -f "$VAPOURSYNTH_LIB" ]; then
    echo "VapourSynth library not found, skipping Python dependency copy"
    exit 0
fi

PYTHON_PATH=$(otool -L "$VAPOURSYNTH_LIB" | grep 'Python' | awk '{print $1}')
if [ -n "$PYTHON_PATH" ]; then
    install_name_tool -change "$PYTHON_PATH" "@executable_path/../Frameworks/Python" "$VAPOURSYNTH_LIB"
    cp "$PYTHON_PATH" ./Supersonic.app/Contents/Frameworks
fi
