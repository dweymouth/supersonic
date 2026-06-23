//go:build !mpv

package localav_test

import (
	"os"
	"strings"
	"testing"
)

func TestExclusiveModeDoesNotBypassSoftwareVolume(t *testing.T) {
	src, err := os.ReadFile("av_player.c")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	code := string(src)

	if strings.Contains(code, "p->exclusive_requested || p->output_format != ma_format_f32") {
		t.Fatal("exclusive mode must not bypass software volume; only bit-perfect mode may do that")
	}
	if !strings.Contains(code, "p->bitperfect_active || p->output_format != ma_format_f32") {
		t.Fatal("expected effective bit-perfect mode to be the no-DSP/no-volume bypass condition")
	}
	if !strings.Contains(code, "if (out_cfg->bitperfect && out_cfg->bitperfect_candidate)") {
		t.Fatal("expected exact decoded output format to be gated by bit-perfect mode")
	}
	if !strings.Contains(code, "cfg->exclusive = p->exclusive_requested || p->bitperfect_requested") ||
		!strings.Contains(code, "cfg->bitperfect = p->bitperfect_requested") {
		t.Fatal("expected exclusive/hog ownership and bit-perfect policy to remain separate")
	}
}

func TestSoftwareVolumeLockUsesEffectiveBitPerfectStatus(t *testing.T) {
	src, err := os.ReadFile("player.go")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	code := string(src)

	if !strings.Contains(code, "info.SoftwareVolumeLocked = info.BitPerfectActive") {
		t.Fatal("software volume must lock from effective bit-perfect status, not requested settings")
	}
}

func TestCoreAudioExclusiveModeAttemptsSystemVolumeFull(t *testing.T) {
	src, err := os.ReadFile("av_player.c")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	code := string(src)

	volumeIdx := strings.Index(code, "coreaudio_set_system_volume_full(hog_device)")
	hogIdx := strings.Index(code, "coreaudio_set_hog_mode(hog_device, 1)")
	if volumeIdx < 0 {
		t.Fatal("expected CoreAudio exclusive mode to attempt setting system output volume to 100%")
	}
	if hogIdx < 0 {
		t.Fatal("expected CoreAudio exclusive mode to request hog mode")
	}
	if volumeIdx > hogIdx {
		t.Fatal("expected system output volume to be set before entering CoreAudio hog mode")
	}
}

func TestCoreAudioBitPerfectUsesIOProcAndNonMixablePhysicalFormats(t *testing.T) {
	src, err := os.ReadFile("av_player.c")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	code := string(src)

	for _, want := range []string{
		"AudioDeviceCreateIOProcID",
		"AudioDeviceStart",
		"CoreAudioBitPerfectIOProc",
		"kAudioStreamPropertyAvailablePhysicalFormats",
		"kAudioFormatFlagIsNonMixable",
		"kAudioFormatFlagIsBigEndian",
		"choice.mixable",
		"no matching non-mixable integer CoreAudio format",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected CoreAudio bit-perfect path to contain %q", want)
		}
	}
}

func TestCoreAudioIOProcRestoresPhysicalFormatOnErrors(t *testing.T) {
	src, err := os.ReadFile("av_player.c")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	code := string(src)

	if !strings.Contains(code, "p->coreaudio_io_stream = choice.stream_id") {
		t.Fatal("expected CoreAudio stream to be tracked before changing the physical format")
	}
	if strings.Count(code, "player_release_coreaudio_ioproc(p)") < 4 {
		t.Fatal("expected IOProc cleanup to run on physical-format setup/start failures and normal release")
	}
}

func TestBitPerfectUnavailableFallsBackVisibly(t *testing.T) {
	src, err := os.ReadFile("av_player.c")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	code := string(src)

	for _, forbidden := range []string{
		"out_cfg.bitperfect && !p->bitperfect_active",
		"cfg.bitperfect && !p->bitperfect_active",
		"next_cfg.bitperfect && !p->bitperfect_active",
	} {
		if strings.Contains(code, forbidden) {
			t.Fatalf("bit-perfect unavailable must not be a fatal playback error; found %q", forbidden)
		}
	}
	for _, want := range []string{
		"output_config_make_app_pcm_fallback",
		"bit-perfect output unavailable; falling back to app PCM",
		"cfg->bitperfect_candidate = 0",
		"strncpy(p->signal_status, \"Not Bit-Perfect\"",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected visible bit-perfect fallback path to contain %q", want)
		}
	}
}

func TestDSDDoPUsesCarrierPathAndMarkerAlternation(t *testing.T) {
	src, err := os.ReadFile("av_player.c")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	code := string(src)

	for _, want := range []string{
		"CoreAudioDoPIOProc",
		"dop_pack_s32",
		"0x05",
		"0xfa",
		"p->dop_marker_phase ^= 1",
		"decoder_clear_dop_state(next)",
		"no matching non-mixable CoreAudio DoP carrier format",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected DSD DoP path to contain %q", want)
		}
	}
}

func TestDSDCarrierRateUsesDSDByteRate(t *testing.T) {
	src, err := os.ReadFile("av_player.c")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	code := string(src)

	if !strings.Contains(code, "return d->codec_ctx->sample_rate / 2;") {
		t.Fatal("expected DoP carrier rate to be derived from FFmpeg's DSD byte rate")
	}
	if !strings.Contains(code, "return d->wv_dop_carrier_rate;") ||
		!strings.Contains(code, "decoder_dsd_byte_rate(d)") {
		t.Fatal("expected WavPack DSD carrier rate to be derived from libwavpack's DSD byte rate")
	}
	if strings.Contains(code, "d->codec_ctx->sample_rate / 16") {
		t.Fatal("DSD DoP carrier must not divide FFmpeg's DSD byte rate by 16")
	}
}

func TestUnsupportedDSDDoesNotReportBitPerfectWithoutRawDoP(t *testing.T) {
	src, err := os.ReadFile("av_player.c")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	code := string(src)

	if !strings.Contains(code, "decoder_is_dsd(d)") ||
		!strings.Contains(code, "decoder_supports_raw_dop(d)") {
		t.Fatal("expected DSD sources to be checked for raw DoP support before bit-perfect status is reported")
	}
	if !strings.Contains(code, "DSD DoP requires raw DSF/DFF or native WavPack DSD input") ||
		!strings.Contains(code, "WavPack DSD native decode unavailable") {
		t.Fatal("expected unsupported DSD sources to fail visibly instead of claiming bit-perfect output")
	}
}

func TestWavPackDSDUsesNativeDoPPath(t *testing.T) {
	src, err := os.ReadFile("av_player.c")
	if err != nil {
		t.Fatalf("ReadFile: %v", err)
	}
	code := string(src)

	for _, want := range []string{
		"AV_CODEC_ID_WAVPACK",
		"wavpack_packet_is_dsd",
		"0x80000000u",
		"source_is_wavpack_dsd",
		"<wavpack/wavpack.h>",
		"OPEN_DSD_NATIVE",
		"WavpackOpenFileInput",
		"WavpackUnpackSamples",
		"WavpackGetNativeSampleRate",
		"WavpackSeekSample64",
		"libwavpack native DSD ->",
	} {
		if !strings.Contains(code, want) {
			t.Fatalf("expected WavPack DSD native DoP path to contain %q", want)
		}
	}

	for _, path := range []string{"cgo_darwin.go", "cgo_linux.go", "cgo_windows.go"} {
		src, err := os.ReadFile(path)
		if err != nil {
			t.Fatalf("ReadFile(%s): %v", path, err)
		}
		if !strings.Contains(string(src), "wavpack") {
			t.Fatalf("expected %s to link libwavpack", path)
		}
	}
}
