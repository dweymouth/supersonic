package mpv

// #include <mpv/client.h>
// int mpv_get_peaks(mpv_handle* handle, double* lPeak, double* rPeak, double* lRMS, double* rRMS);
import "C"

import (
	"github.com/supersonic-app/go-mpv"
)

// rather than querying peaks through m.mpv.GetProperty which necessitates
// converting the MPV node to a Go map, we can do so in C with no Go allocations
func (m *Player) getPeaks() (float64, float64, float64, float64, error) {
	var lPeak, rPeak, lRMS, rRMS C.double
	ret := int(C.mpv_get_peaks((*C.mpv_handle)(m.mpv.MPVHandle()), &lPeak, &rPeak, &lRMS, &rRMS))
	if err := mpv.NewError(ret); err != nil {
		return 0, 0, 0, 0, err
	}
	return float64(lPeak), float64(rPeak), float64(lRMS), float64(rRMS), nil
}
