#pragma once

#ifdef __cplusplus
extern "C" {
#endif

typedef void (*ThumbnailCallback)(int buttonId);

// sets the tool tip strings for prev, next, play, and pause
// should be called before initialize_taskbar_buttons or they will not have tool tips
void set_tooltips_utf8(const char* prev, const char* next, const char* play, const char* pause);

// sets the icons that will be used for the buttons.
// the arguments are pointers to BGRA pixel data, and the dimensions (w, h) of all 4 images.
int initialize_taskbar_icons(const void *bgraPrev, const void *bgraNext, const void *bgraPlay, const void *bgraPause, int width, int height);

// adds the buttons to the window and registers the given callback to receive the button press events.
int initialize_taskbar_buttons(void *hwndPtr, ThumbnailCallback cb);

// sets whether the player is playing. controls which icon/tooltip the center button uses.
// safe to call from any thread
int set_is_playing(int playing);

#ifdef __cplusplus
}
#endif
