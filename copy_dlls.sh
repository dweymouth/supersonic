#!/bin/bash
cd /x/supersonic
ldd supersonic.exe | grep mingw64 | cut -d' ' -f3 | xargs -I '{}' cp '{}' "/x/[Tools]/supersonic/"
ldd /mingw64/bin/libmpv-2.dll | grep mingw64 | cut -d' ' -f3 | xargs -I '{}' cp '{}' "/x/[Tools]/supersonic/"
