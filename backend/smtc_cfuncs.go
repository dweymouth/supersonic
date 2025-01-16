//go:build windows

package backend

/*
void btn_callback_cgo(int in) {
	void btnCallback(int);
	btnCallback(in);
}

void seek_callback_cgo(int in) {
	void seekCallback(int);
	seekCallback(in);
}
*/
import "C"
