//go:build windows

#include <windows.h>
#include <shobjidl.h>
#include <initguid.h>
#include <stdio.h>
#include "taskbar_buttons.h"

DEFINE_GUID(IID_ITaskbarList3,
    0xEA1AFB91, 0x9E28, 0x4B86, 0x90, 0xE9, 0x9E, 0x9F, 0x8A, 0x5E, 0xEF, 0xAF);

#define WM_SET_PLAYING_STATE (WM_APP + 1)

static ITaskbarList3 *g_taskbar = NULL;
static ThumbnailCallback g_callback = NULL;
static HWND g_mainHWnd = NULL;
static WNDPROC g_originalProc = NULL;

THUMBBUTTON g_thumbButtons[3];

static HICON g_prevIcon;
static HICON g_nextIcon;
static HICON g_playIcon;
static HICON g_pauseIcon;

static HICON create_icon_from_bgra(const void *bgra, int width, int height) {
    HICON hIcon = NULL;

    // Create mask bitmap (not used, but required)
    HBITMAP hMonoMask = CreateBitmap(width, height, 1, 1, NULL);

    // Create color bitmap from provided pixels
    BITMAPV5HEADER bi;
    ZeroMemory(&bi, sizeof(bi));
    bi.bV5Size = sizeof(bi);
    bi.bV5Width = width;
    bi.bV5Height = -height; // top-down DIB
    bi.bV5Planes = 1;
    bi.bV5BitCount = 32;
    bi.bV5Compression = BI_BITFIELDS;
    bi.bV5RedMask   = 0x00FF0000;
    bi.bV5GreenMask = 0x0000FF00;
    bi.bV5BlueMask  = 0x000000FF;
    bi.bV5AlphaMask = 0xFF000000;

    void *bits = NULL;
    HDC hdc = GetDC(NULL);
    HBITMAP hBmp = CreateDIBSection(hdc, (BITMAPINFO*)&bi, DIB_RGB_COLORS, &bits, NULL, 0);
    ReleaseDC(NULL, hdc);

    if (hBmp && bits) {
        memcpy(bits, bgra, width * height * 4);

        ICONINFO ii;
        ZeroMemory(&ii, sizeof(ii));
        ii.fIcon = TRUE;
        ii.hbmMask = hMonoMask;
        ii.hbmColor = hBmp;

        hIcon = CreateIconIndirect(&ii);

        DeleteObject(hBmp);
        DeleteObject(hMonoMask);
    }
    return hIcon;
}

int initialize_taskbar_icons(const void *bgraPrev, const void *bgraNext, const void *bgraPlay, const void *bgraPause, int width, int height) {
    g_prevIcon = create_icon_from_bgra(bgraPrev, width, height);
    g_nextIcon = create_icon_from_bgra(bgraNext, width, height);
    g_playIcon = create_icon_from_bgra(bgraPlay, width, height);
    g_pauseIcon = create_icon_from_bgra(bgraPause, width, height);

    return 0;
}

LRESULT CALLBACK OverrideWndProc(HWND hwnd, UINT msg, WPARAM wParam, LPARAM lParam) {
    if (msg == WM_COMMAND) {
        int buttonId = LOWORD(wParam);
        if (g_callback) {
            g_callback(buttonId);
            return 0;
        }
    }
    else if (msg == WM_SET_PLAYING_STATE) {
        if (g_taskbar && g_mainHWnd && g_playIcon && g_pauseIcon) {
            int playing = LOWORD(wParam) ? 1 : 0;
            g_thumbButtons[1].hIcon = playing ? g_pauseIcon : g_playIcon;
            wcscpy_s(g_thumbButtons[1].szTip, ARRAYSIZE(g_thumbButtons[0].szTip), playing ? L"Pause" : L"Play");
            g_taskbar->lpVtbl->ThumbBarUpdateButtons(
                g_taskbar,
                g_mainHWnd,
                ARRAYSIZE(g_thumbButtons),
                g_thumbButtons
            );
        }
        return 0;
    }

    return CallWindowProc(g_originalProc, hwnd, msg, wParam, lParam);
}

int set_is_playing(int playing) {
    if (g_mainHWnd) {
        PostMessage(g_mainHWnd, WM_SET_PLAYING_STATE, (WPARAM)playing, 0);
        return 0;
    }
    return -1;
}

int initialize_taskbar_buttons(void *hwndPtr, ThumbnailCallback cb) {
    g_callback = cb;
    g_mainHWnd = (HWND)hwndPtr;

    // subclass the WndProc with our OverrideWndProc
    g_originalProc = (WNDPROC)SetWindowLongPtr(g_mainHWnd, GWLP_WNDPROC, (LONG_PTR)OverrideWndProc);

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
    if (g_prevIcon) {
        g_thumbButtons[0].dwMask |= THB_ICON;
        g_thumbButtons[0].hIcon = g_prevIcon;
    }

    g_thumbButtons[1].dwMask = THB_FLAGS | THB_TOOLTIP;
    g_thumbButtons[1].iId = 2;
    g_thumbButtons[1].dwFlags = THBF_ENABLED;
    wcscpy_s(g_thumbButtons[1].szTip, ARRAYSIZE(g_thumbButtons[1].szTip), L"Play");
    if (g_playIcon) {
        g_thumbButtons[1].dwMask |= THB_ICON;
        g_thumbButtons[1].hIcon = g_playIcon;
    }

    g_thumbButtons[2].dwMask = THB_FLAGS | THB_TOOLTIP;
    g_thumbButtons[2].iId = 3;
    g_thumbButtons[2].dwFlags = THBF_ENABLED;
    wcscpy_s(g_thumbButtons[2].szTip, ARRAYSIZE(g_thumbButtons[2].szTip), L"Next");
    if (g_nextIcon) {
        g_thumbButtons[2].dwMask |= THB_ICON;
        g_thumbButtons[2].hIcon = g_nextIcon;
    }

    g_taskbar->lpVtbl->ThumbBarAddButtons(g_taskbar, g_mainHWnd, ARRAYSIZE(g_thumbButtons), g_thumbButtons);

    return 0;
}