#pragma once

#include <stdint.h>

// Decode-step return codes (av_player_decode_step)
#define AVPLAYER_DECODE_OK         0   // decoded and buffered a frame
#define AVPLAYER_DECODE_RING_FULL  1   // ring buffer full; caller should sleep briefly
#define AVPLAYER_DECODE_EOF        2   // current track exhausted (ring buffer still draining)
#define AVPLAYER_DECODE_NEXT_READY 3   // next track pre-opened and ready; swapped in
#define AVPLAYER_DECODE_STOPPED    4   // playback stopped (ring buffer drained after EOF)
#define AVPLAYER_DECODE_ERROR     -1   // unrecoverable error

// Player states
#define AVPLAYER_STATE_STOPPED  0
#define AVPLAYER_STATE_PLAYING  1
#define AVPLAYER_STATE_PAUSED   2

// Output sample rate and format (fixed for gapless compatibility across tracks)
#define AVPLAYER_SAMPLE_RATE        48000
#define AVPLAYER_SAMPLE_RATE_STR    "48000"
#define AVPLAYER_CHANNELS           2
// Ring buffer capacity in frames (~4 seconds)
#define AVPLAYER_RING_FRAMES    (AVPLAYER_SAMPLE_RATE * 4)

typedef struct av_player av_player_t;

// Audio device info (for enumeration)
typedef struct {
    char name[256];
    char description[256];
} av_device_info_t;

// Media info populated after a track opens
typedef struct {
    char   codec[64];
    int    sample_rate;
    int    channels;
    int    bitrate;        // bits/sec; updated per-packet for VBR
} av_media_info_t;

// ---- Lifecycle -------------------------------------------------------

// Allocate a new player.  Must be followed by av_player_init().
av_player_t *av_player_create(void);

// Initialise the miniaudio device.
// device_name: NULL or "" for default device.
// exclusive: non-zero to request exclusive access.
// Returns 0 on success, non-zero on failure.
int av_player_init(av_player_t *p, const char *device_name, int exclusive);

// Destroy the player and free all resources.
void av_player_destroy(av_player_t *p);

// ---- Playback --------------------------------------------------------

// Open a URL and prepare for playback.  eq_filter is a libavfilter graph
// fragment string (may be NULL or "").  rg_gain_db is the ReplayGain volume
// adjustment in dB to apply (0.0 = no change).  rg_prevent_clip: non-zero
// means clamp output to 0 dBFS.
// Returns 0 on success.
int av_player_open(av_player_t *p, const char *url, double start_time,
                   const char *eq_filter, double rg_gain_db, int rg_prevent_clip);

// Pre-open the next URL for gapless playback.  Call while current track is
// playing; the decode loop will seamlessly switch when current track ends.
// Pass NULL/empty url to clear any queued next track.
int av_player_open_next(av_player_t *p, const char *url,
                        const char *eq_filter, double rg_gain_db, int rg_prevent_clip);

// Stop playback and discard any buffered audio.
void av_player_stop(av_player_t *p);

// Pause/resume (low-latency; does not stop the miniaudio device).
void av_player_pause(av_player_t *p);
void av_player_resume(av_player_t *p);

// Seek to an absolute time (seconds).  Returns 0 on success.
int av_player_seek(av_player_t *p, double seconds);

// Set playback volume (0.0–1.0).
void av_player_set_volume(av_player_t *p, float volume);

// ---- Filters ---------------------------------------------------------

// Rebuild the audio filter graph with new EQ / ReplayGain settings.
// Takes effect on the current track; safe to call while playing.
int av_player_set_filters(av_player_t *p,
                          const char *eq_filter,
                          double rg_gain_db,
                          int rg_prevent_clip);

// ---- Status ----------------------------------------------------------

int    av_player_get_state(av_player_t *p);
double av_player_get_position(av_player_t *p);
double av_player_get_duration(av_player_t *p);

// Number of frames still buffered in the ring (proxy for "is audio draining")
int av_player_buffered_frames(av_player_t *p);

// ---- Peaks -----------------------------------------------------------

// Populate peak/RMS values from the last processed frame (dBFS, -inf..0).
// Values are invalid (0.0) when not playing.
void av_player_get_peaks(av_player_t *p,
                         double *l_peak, double *r_peak,
                         double *l_rms,  double *r_rms);

// Enable/disable peak measurement (astats filter in graph).
void av_player_set_peaks_enabled(av_player_t *p, int enabled);

// ---- Media info ------------------------------------------------------

void av_player_get_media_info(av_player_t *p, av_media_info_t *info);

// ---- Device management -----------------------------------------------

// Fill devices[0..max_devices-1].  Returns the number of devices filled.
int av_player_list_devices(av_device_info_t *devices, int max_devices);

// Change the output device.  NULL or "" selects the system default.
// Returns 0 on success.
int av_player_set_device(av_player_t *p, const char *device_name);

// Set exclusive mode on/off.  Returns 0 on success.
int av_player_set_exclusive(av_player_t *p, int exclusive);

// ---- Decode loop (called from Go goroutine) --------------------------

// Perform one decode step: read → decode → filter → write ring buffer.
// See AVPLAYER_DECODE_* return codes above.
// This is the only function that should be called repeatedly in a loop.
int av_player_decode_step(av_player_t *p);
