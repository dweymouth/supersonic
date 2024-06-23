//go:build linux && !wayland

#include <stdio.h>
#include <stdlib.h>
#include <X11/Xlib.h>
#include <X11/Xatom.h>
#include <unistd.h>

#include "xresize.h"

// thanks ChatGPT ;)
int find_windows_by_pid(Display *display, Window root, pid_t pid, Window* out, int n) {
    Window *children;
    unsigned int nchildren;
    if (!XQueryTree(display, root, &root, &root, &children, &nchildren)) {
        return 0;
    }

    for (unsigned int i = 0; i < nchildren; i++) {
        Atom pidAtom = XInternAtom(display, "_NET_WM_PID", True);
        if (pidAtom != None) {
            Atom type;
            int format;
            unsigned long nitems, bytes_after;
            unsigned char *prop_pid = NULL;

            if (XGetWindowProperty(display, children[i], pidAtom, 0, 1, False, XA_CARDINAL, 
                                   &type, &format, &nitems, &bytes_after, &prop_pid) == Success && prop_pid) {
                if (pid == *((pid_t *)prop_pid)) {
                    out[n++] = children[i];
                }
                XFree(prop_pid);
            }
        }

        n = find_windows_by_pid(display, children[i], pid, out, n);
    }

    XFree(children);
    return n;
}

void send_resize_event(Display *display, Window window, int width, int height) {
   XResizeWindow(display, window, width, height);
   XFlush(display);
}

int send_resize_to_pid(int pid, int w, int h) {
    Display *display = XOpenDisplay(NULL);
    if (!display) {
        return 1;
    }
    int ret = 0;
    Window windows[128];
    Window root = DefaultRootWindow(display);

    int n = find_windows_by_pid(display, root, pid, &windows, 0);
    if (n > 0) {
	    for (int i = 0; i < n; i++) {
        	send_resize_event(display, windows[i], w, h);
	    }
    } else {
        ret = 1;
    }

    XCloseDisplay(display);
    return ret;
}
