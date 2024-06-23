//go:build linux && !wayland

int send_resize_to_pid(int pid, int w, int h);
