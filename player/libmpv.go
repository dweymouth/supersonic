package player

/*
#include <mpv/client.h>
#include <stdlib.h>
#cgo LDFLAGS: -lmpv

char** newCharArray(int size) {
    return calloc(sizeof(char*), size);
}
void setCharArrayIdx(char** a, int i, char* s) {
    a[i] = s;
}

*/
import "C"
import (
	"errors"
	"fmt"
	"unsafe"
)

type MPVFormat int

const (
	MPVFormatDouble MPVFormat = C.MPV_FORMAT_DOUBLE
	MPVFormatFlag   MPVFormat = C.MPV_FORMAT_FLAG
	MPVFormatString MPVFormat = C.MPV_FORMAT_STRING
	MPVFormatInt64  MPVFormat = C.MPV_FORMAT_INT64
)

type MPVEventID int

const (
	MPVEventNone            MPVEventID = C.MPV_EVENT_NONE
	MPVEventStartFile       MPVEventID = C.MPV_EVENT_START_FILE
	MPVEventEndFile         MPVEventID = C.MPV_EVENT_END_FILE
	MPVEventFileLoaded      MPVEventID = C.MPV_EVENT_FILE_LOADED
	MPVEventIdle            MPVEventID = C.MPV_EVENT_IDLE
	MPVEventAudioReconfig   MPVEventID = C.MPV_EVENT_AUDIO_RECONFIG
	MPVEventSeek            MPVEventID = C.MPV_EVENT_SEEK
	MPVEventPlaybackRestart MPVEventID = C.MPV_EVENT_PLAYBACK_RESTART
	MPVEventPropertyChange  MPVEventID = C.MPV_EVENT_PROPERTY_CHANGE
)

type MPVEvent struct {
	ID       MPVEventID
	ErrCode  int
	Data     unsafe.Pointer
	UserData uint64
}

type libmpv struct {
	handle *C.mpv_handle
}

func CreateMPV() (libmpv, error) {
	handle := C.mpv_create()
	if handle == nil {
		return libmpv{}, errors.New("failed to create mpv instance")
	}
	return libmpv{handle}, nil
}

func (m libmpv) Initialize() error {
	return toMPVError(C.mpv_initialize(m.handle))
}

func (m libmpv) Command(cmd []string) error {
	cArray := C.newCharArray(C.int(len(cmd) + 1))
	if cArray == nil {
		return errors.New("calloc failed")
	}
	defer C.free(unsafe.Pointer(cArray))

	for i, s := range cmd {
		cStr := C.CString(s)
		C.setCharArrayIdx(cArray, C.int(i), cStr)
		defer C.free(unsafe.Pointer(cStr))
	}

	return toMPVError(C.mpv_command(m.handle, cArray))
}

func (m libmpv) SetOption(name string, format MPVFormat, value any) error {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	p := toPointer(format, value)
	return toMPVError(C.mpv_set_option(m.handle, cname, C.mpv_format(format), p))
}

func (m libmpv) SetOptionString(name, value string) error {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	cvalue := C.CString(value)
	defer C.free(unsafe.Pointer(cvalue))
	return toMPVError(C.mpv_set_option_string(m.handle, cname, cvalue))
}

func (m libmpv) SetProperty(name string, format MPVFormat, value any) error {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	p := toPointer(format, value)
	return toMPVError(C.mpv_set_property(m.handle, cname, C.mpv_format(format), p))
}

func (m libmpv) SetPropertyString(name, value string) error {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	cvalue := C.CString(value)
	defer C.free(unsafe.Pointer(cvalue))
	return toMPVError(C.mpv_set_property_string(m.handle, cname, cvalue))
}

func (m libmpv) GetProperty(name string, format MPVFormat) (any, error) {
	cname := C.CString(name)
	defer C.free(unsafe.Pointer(cname))
	switch format {
	case MPVFormatDouble:
		var cdbl C.double
		err := toMPVError(C.mpv_get_property(m.handle, cname, C.mpv_format(format), unsafe.Pointer(&cdbl)))
		if err != nil {
			return nil, err
		}
		return float64(cdbl), nil
	case MPVFormatFlag:
		var cint C.int
		err := toMPVError(C.mpv_get_property(m.handle, cname, C.mpv_format(format), unsafe.Pointer(&cint)))
		if err != nil {
			return nil, err
		}
		return cint == 1, nil
	case MPVFormatInt64:
		var cint64 C.int64_t
		err := toMPVError(C.mpv_get_property(m.handle, cname, C.mpv_format(format), unsafe.Pointer(&cint64)))
		if err != nil {
			return nil, err
		}
		return int64(cint64), nil
	default:
		return nil, errors.New("unsupported mpv format")
	}
}

func (m libmpv) WaitEvent(timeout float64) MPVEvent {
	var cevent *C.mpv_event
	cevent = C.mpv_wait_event(m.handle, C.double(timeout))
	if cevent == nil {
		return MPVEvent{ID: MPVEventNone}
	}

	e := MPVEvent{
		ID:       MPVEventID(cevent.event_id),
		UserData: uint64(cevent.reply_userdata),
		ErrCode:  int(cevent.error),
		Data:     cevent.data,
	}
	return e
}

func (m libmpv) TerminateDestroy() {
	C.mpv_terminate_destroy(m.handle)
}

func toMPVError(errcode C.int) error {
	if errcode == C.MPV_ERROR_SUCCESS {
		return nil
	}
	return fmt.Errorf("mpv error %d: %s", int(errcode), C.GoString(C.mpv_error_string(C.int(errcode))))
}

func toPointer(format MPVFormat, value any) unsafe.Pointer {
	var ptr unsafe.Pointer = nil
	switch format {
	case MPVFormatDouble:
		v := C.double(value.(float64))
		ptr = unsafe.Pointer(&v)
	case MPVFormatInt64:
		i, ok := value.(int64)
		if !ok {
			i = int64(value.(int))
		}
		v := C.int64_t(i)
		ptr = unsafe.Pointer(&v)
	case MPVFormatFlag:
		v := C.int(0)
		if value.(bool) {
			v = C.int(1)
		}
		ptr = unsafe.Pointer(&v)
	case MPVFormatString:
		ptr = unsafe.Pointer(&[]byte(value.(string))[0])
	}
	return ptr
}
