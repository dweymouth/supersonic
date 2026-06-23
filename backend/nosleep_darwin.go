//go:build darwin

package backend

/*
#cgo LDFLAGS: -framework CoreFoundation -framework IOKit
#include <CoreFoundation/CoreFoundation.h>
#include <IOKit/pwr_mgt/IOPMLib.h>

static IOPMAssertionID supersonicSleepAssertion = kIOPMNullAssertionID;

static void supersonic_set_system_sleep_disabled(int disable) {
	if (disable) {
		if (supersonicSleepAssertion != kIOPMNullAssertionID) {
			return;
		}
		CFStringRef reason = CFSTR("Supersonic playback");
		IOPMAssertionCreateWithName(kIOPMAssertionTypeNoIdleSleep,
		                            kIOPMAssertionLevelOn,
		                            reason,
		                            &supersonicSleepAssertion);
		return;
	}
	if (supersonicSleepAssertion != kIOPMNullAssertionID) {
		IOPMAssertionRelease(supersonicSleepAssertion);
		supersonicSleepAssertion = kIOPMNullAssertionID;
	}
}
*/
import "C"

func SetSystemSleepDisabled(disable bool) {
	if disable {
		C.supersonic_set_system_sleep_disabled(1)
		return
	}
	C.supersonic_set_system_sleep_disabled(0)
}
