#pragma once

#ifdef __cplusplus
extern "C" {
#endif

typedef void (*ThumbnailCallback)(int buttonId);

int initialize_taskbar_buttons(void *hwndPtr, ThumbnailCallback cb);

#ifdef __cplusplus
}
#endif
