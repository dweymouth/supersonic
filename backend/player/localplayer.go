package player

// AudioDevice describes a system audio output device.
type AudioDevice struct {
	// Name is the identifier passed to SetAudioDevice.
	Name string
	// Description is the human-friendly label shown in the UI.
	Description string
	// UID is the host API's stable device identifier when available.
	UID string
	// Transport describes the device connection type, e.g. Built-in, USB, HDMI.
	Transport string
	// ExclusiveReady reports whether the device exposes a non-mixable output format.
	ExclusiveReady bool
	// Lossy reports whether the transport is known to be lossy, e.g. Bluetooth or AirPlay.
	Lossy bool
	// PhysicalFormatCount is the number of output physical formats reported by the host API.
	PhysicalFormatCount int
	// ExclusiveFormatCount is the number of non-mixable physical formats.
	ExclusiveFormatCount int
	// MinSampleRate and MaxSampleRate describe the output device's physical rate range.
	MinSampleRate int
	MaxSampleRate int
	// MaxBitDepth is the largest PCM container depth exposed by the device.
	MaxBitDepth int
	// Channels is the maximum channel count exposed by the device formats.
	Channels int
	// ActiveFormat is the current host physical format, if available.
	ActiveFormat string
	// CapabilitySummary is a compact user-facing diagnostic summary.
	CapabilitySummary string
}

// MediaInfo holds information about the currently playing stream.
type MediaInfo struct {
	// Codec name (e.g. "flac", "mp3", "aac").
	Codec string
	// The decoded PCM sample format before app-side output conversion.
	Format string
	// Samplerate in Hz.
	Samplerate int
	// Number of audio channels.
	ChannelCount int
	// Bitrate in bits per second (instantaneous for VBR codecs).
	Bitrate int
	// The sample format delivered to the host audio API.
	OutputFormat string
	// Output sample rate in Hz.
	OutputSamplerate int
	// Output channel count.
	OutputChannelCount int
	// Whether exclusive/hog mode was requested by the user.
	ExclusiveRequested bool
	// Whether the host audio API granted exclusive/hog ownership.
	ExclusiveActive bool
	// Whether strict bit-perfect output was requested by the user.
	BitPerfectRequested bool
	// Whether the current localav path avoids app-side DSP, remixing, and resampling.
	BitPerfectActive bool
	// Human-readable reason when bit-perfect output is unavailable.
	BitPerfectReason string
	// Whether changing software volume would break the current output contract.
	SoftwareVolumeLocked bool
	// PlaybackPath is the effective backend path, e.g. CoreAudioBitPerfectIOProc.
	PlaybackPath string
	// SignalStatus is a compact status label for the current signal path.
	SignalStatus string
	// Device details for the selected output.
	DeviceName      string
	DeviceUID       string
	DeviceTransport string
	// Signal-path descriptions.
	SourceFormat string
	DecodePath   string
	OutputPath   string
	DACFormat    string
	// OutputMixable reports whether the active host format is mixable.
	OutputMixable bool
	// Device capability details for diagnostics.
	DevicePhysicalFormats  int
	DeviceExclusiveFormats int
	DeviceMinSampleRate    int
	DeviceMaxSampleRate    int
	DeviceMaxBitDepth      int
	DeviceChannels         int
	// DSD / DoP details. DSDRate is the native one-bit DSD clock in Hz;
	// DoPCarrierRate is the PCM wrapper rate required by the DAC.
	SourceIsDSD    bool
	DSDRate        int
	DoPCarrierRate int
}

// LocalPlayer is implemented by any local audio playback engine
// (currently mpv and localav).  All methods used by the UI and backend
// that go beyond the core BasePlayer / URLPlayer interfaces are listed here.
type LocalPlayer interface {
	URLPlayer
	ReplayGainPlayer

	SetEqualizer(Equalizer) error
	Equalizer() Equalizer

	SetAudioExclusive(bool) error
	SetAudioBitPerfect(bool) error
	SetPauseFade(bool)

	ListAudioDevices() ([]AudioDevice, error)
	SetAudioDevice(string) error

	GetMediaInfo() (MediaInfo, error)

	SetPeaksEnabled(bool) error
	GetPeaks() (float64, float64, float64, float64)

	ObserveIcyRadioTitle(func(string))
	UnobserveIcyRadioTitle()
}
