//go:build darwin

#import <AppKit/AppKit.h>

void setWindowDarkMode(void* windowPtr, int mode) {
    if (windowPtr == NULL) return;

    NSWindow* window = (__bridge NSWindow*)windowPtr;

    if (mode == 1) {
        // Forces the window chrome and borders into Dark Mode
        window.appearance = [NSAppearance appearanceNamed:NSAppearanceNameVibrantDark];
    } else if (mode == 2) {
        // Forces the window chrome and borders into Light Mode
        window.appearance = [NSAppearance appearanceNamed:NSAppearanceNameVibrantLight];
    } else {
        // Reverts the window back to standard/system default behavior
        window.appearance = nil; // Fallback to inherited system style
    }
}
