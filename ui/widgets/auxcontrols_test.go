package widgets

import (
	"testing"

	"fyne.io/fyne/v2/test"
)

func TestVolumeControlSoftwareVolumeLock(t *testing.T) {
	test.NewTempApp(t)

	c := NewVolumeControl(35)
	var changed []int
	c.OnSetVolume = func(vol int) {
		changed = append(changed, vol)
	}

	c.SetSoftwareVolumeLocked(true, "locked")
	if !c.SoftwareVolumeLocked() {
		t.Fatal("expected software volume to be locked")
	}
	if !c.icon.Disabled() || !c.slider.Disabled() {
		t.Fatal("expected volume controls to be disabled while locked")
	}
	if got := int(c.slider.Value); got != 100 {
		t.Fatalf("locked volume display = %d, want 100", got)
	}

	c.SetVolume(42)
	if got := int(c.slider.Value); got != 100 {
		t.Fatalf("locked volume display after SetVolume = %d, want 100", got)
	}
	c.onChanged(10)
	if len(changed) != 0 {
		t.Fatalf("locked volume change invoked callback: %v", changed)
	}

	c.SetSoftwareVolumeLocked(false, "")
	if c.SoftwareVolumeLocked() {
		t.Fatal("expected software volume to unlock")
	}
	if c.icon.Disabled() || c.slider.Disabled() {
		t.Fatal("expected volume controls to be enabled after unlock")
	}
	if got := int(c.slider.Value); got != 42 {
		t.Fatalf("unlocked volume display = %d, want latent software volume 42", got)
	}
}
