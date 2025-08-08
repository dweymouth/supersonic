//go:build windows

#include <windows.h>
#include <shobjidl.h>
#include <initguid.h>
#include <stdio.h>
#include "taskbar_buttons.h"

DEFINE_GUID(IID_ITaskbarList3,
    0xEA1AFB91, 0x9E28, 0x4B86, 0x90, 0xE9, 0x9E, 0x9F, 0x8A, 0x5E, 0xEF, 0xAF);

static ITaskbarList3 *g_taskbar = NULL;
static ThumbnailCallback g_callback = NULL;
static HWND g_mainHWnd = NULL;
static WNDPROC g_originalProc = NULL;

THUMBBUTTON g_thumbButtons[3];

LRESULT CALLBACK OverrideWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    if (msg == WM_COMMAND) {
        int buttonId = LOWORD(wParam);
        if (g_callback) {
            g_callback(buttonId);
            return 0;
        }
    }

    return CallWindowProc(g_originalProc, hwnd, msg, wParam, lParam);
}

void hook_window_proc(HWND hwnd) {
    g_originalProc = (WNDPROC)SetWindowLongPtr(hwnd, GWLP_WNDPROC, (LONG_PTR)OverrideWndProc);
}

int initialize_taskbar_buttons(void *hwndPtr, ThumbnailCallback cb) {
    g_callback = cb;
    g_mainHWnd = (HWND)hwndPtr;
    hook_window_proc(g_mainHWnd);

    CoInitialize(NULL);
    CoCreateInstance(&CLSID_TaskbarList, NULL, CLSCTX_INPROC_SERVER, &IID_ITaskbarList3, (void**)&g_taskbar);

    if (g_taskbar) {
        g_taskbar->lpVtbl->HrInit(g_taskbar);
    }

    if (!g_taskbar || !g_mainHWnd) return -1;

    ZeroMemory(g_thumbButtons, sizeof(g_thumbButtons));

    g_thumbButtons[0].dwMask = THB_FLAGS | THB_TOOLTIP;
    g_thumbButtons[0].iId = 1;
    g_thumbButtons[0].dwFlags = THBF_ENABLED;
    wcscpy_s(g_thumbButtons[0].szTip, ARRAYSIZE(g_thumbButtons[0].szTip), L"Previous");

    g_thumbButtons[1].dwMask = THB_FLAGS | THB_TOOLTIP;
    g_thumbButtons[1].iId = 2;
    g_thumbButtons[1].dwFlags = THBF_ENABLED;
    wcscpy_s(g_thumbButtons[1].szTip, ARRAYSIZE(g_thumbButtons[1].szTip), L"Play");

    g_thumbButtons[2].dwMask = THB_FLAGS | THB_TOOLTIP;
    g_thumbButtons[2].iId = 3;
    g_thumbButtons[2].dwFlags = THBF_ENABLED;
    wcscpy_s(g_thumbButtons[2].szTip, ARRAYSIZE(g_thumbButtons[2].szTip), L"Next");

    g_taskbar->lpVtbl->ThumbBarAddButtons(g_taskbar, g_mainHWnd, ARRAYSIZE(g_thumbButtons), g_thumbButtons);

    return 0;
}