// av_player.c — FFmpeg + miniaudio audio player
// One translation unit: defines miniaudio implementation here.

// Use FFmpeg for all decoding; disable miniaudio decoders
#define MA_NO_DECODING
#define MA_NO_ENCODING
#define MA_IMPLEMENTATION
#include "miniaudio.h"

#include "av_player.h"

#include <libavcodec/avcodec.h>
#include <libavformat/avformat.h>
#include <libavfilter/avfilter.h>
#include <libavfilter/buffersink.h>
#include <libavfilter/buffersrc.h>
#include <libavutil/opt.h>
#include <libavutil/samplefmt.h>
#include <libavutil/channel_layout.h>
#include <libavutil/dict.h>
#include <libswresample/swresample.h>
#include <wavpack/wavpack.h>

#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <math.h>
#include <stdatomic.h>
#include <inttypes.h>

#if defined(__APPLE__)
#include <AudioToolbox/AudioHardwareService.h>
#include <CoreAudio/CoreAudio.h>
#include <CoreFoundation/CoreFoundation.h>
#include <unistd.h>
#endif

#if defined(SUPERSONIC_AUDIO_DEBUG)
#define AV_DEBUG(...) do { \
    fprintf(stderr, "[localav] "); \
    fprintf(stderr, __VA_ARGS__); \
    fputc('\n', stderr); \
    fflush(stderr); \
} while (0)
#else
#define AV_DEBUG(...) do { } while (0)
#endif

#define AVPLAYER_MAX_DOP_CHANNELS 8

static int handle_eof_and_gapless(av_player_t *p);

// --------------------------------------------------------------------------
// Internal types
// --------------------------------------------------------------------------

typedef struct decoder {
    AVFormatContext *fmt_ctx;
    AVCodecContext  *codec_ctx;
    AVFilterGraph   *filter_graph;
    AVFilterContext *buffersrc_ctx;
    AVFilterContext *buffersink_ctx;
    AVPacket        *pkt;
    AVFrame         *frame;
    AVFrame         *filt_frame;
    AVFrame         *pending_frame; // filtered frame that didn't fit in ring — retry next step
    AVPacket        *dop_pending_pkt;
    uint8_t          dop_pending_byte[AVPLAYER_MAX_DOP_CHANNELS];
    uint8_t          dop_pending_valid[AVPLAYER_MAX_DOP_CHANNELS];
    int              audio_stream_idx;
    int              source_is_wavpack_dsd;
    WavpackContext  *wv_ctx;
    int              wv_dsd;
    int              wv_dsd_lsb_first;
    int              wv_channels;
    int              wv_dsd_byte_rate;
    int              wv_dsd_bit_rate;
    int              wv_dop_carrier_rate;
    int              wv_average_bitrate;
    long long        wv_samples_unpacked;
    char             wv_error[160];
    double           duration;      // seconds; 0 if unknown
    // ReplayGain tag values read from file metadata (defaults: gain=0.0, peak=1.0)
    double           rg_track_gain_db;
    double           rg_album_gain_db;
    float            rg_track_peak;
    float            rg_album_peak;
    char             url[2048];
} decoder_t;

// EQ bank: one set of up to AVPLAYER_MAX_EQ_BANDS ma_peak2 biquad filters
typedef struct {
    ma_peak2  filters[AVPLAYER_MAX_EQ_BANDS];
    int       num_bands;
    int       initialized;
    float     preamp;  // linear gain (1.0 = 0 dB)
} eq_bank_t;

// SPSC ring buffer — producer: decode goroutine, consumer: miniaudio callback
typedef struct {
    uint8_t        *buf;         // interleaved PCM frames
    int             cap;         // in frames
    int             channels;
    int             bytes_per_frame;
    ma_format       format;
    int             sample_rate;
    atomic_int      write_idx;   // next frame to write (mod cap)
    atomic_int      read_idx;    // next frame to read  (mod cap)
    atomic_int      fill;        // frames available to read
    _Atomic long long frames_written_total; // cumulative frames written since last av_player_open
} ring_buf_t;

// Bitrate history ring: maps ring-write frame positions to per-packet bitrates,
// so that GetMediaInfo can return the bitrate matching the currently playing audio
// rather than the most recently decoded packet (which is ~ring-buffer-duration ahead).
// Producer: decode goroutine (sole writer). Consumer: Go thread via av_player_get_media_info.
#define BITRATE_HISTORY_SIZE 512  // covers ~4s at typical packet rates

typedef struct {
    long long frame_pos;  // ring.frames_written_total at time of capture
    int       bitrate;    // bits per second
} bitrate_entry_t;

typedef struct {
    bitrate_entry_t entries[BITRATE_HISTORY_SIZE];
    _Atomic int     write_idx;  // next slot to write (mod BITRATE_HISTORY_SIZE)
} bitrate_history_t;

typedef struct {
    int                 sample_rate;
    int                 channels;
    enum AVSampleFormat av_format;
    ma_format           ma_format;
    int                 bit_depth;
    int                 exclusive;
    int                 bitperfect;
    int                 bitperfect_candidate;
    int                 dop;
    int                 dsd_bit_rate;
    int                 dop_carrier_rate;
    char                reason[160];
} output_config_t;

struct av_player {
    // Current decoder
    decoder_t       *dec;
    // Pre-fetched next decoder (for gapless)
    decoder_t       *next_dec;

    ring_buf_t       ring;

    ma_device        device;
    ma_context       ma_ctx;
    int              ma_ctx_init;  // bool: ma_ctx was initialised
    int              device_init;  // bool: device was initialised
    char             device_name[256];
    int              exclusive_requested;
    int              exclusive_active;
    int              bitperfect_requested;
    ma_format        output_format;
    int              output_sample_rate;
    int              output_channels;
    enum AVSampleFormat output_av_format;
    int              output_bit_depth;
    int              bitperfect_active;
    char             bitperfect_reason[160];
    char             playback_path[64];
    char             signal_status[64];
    char             device_uid[256];
    char             device_transport[64];
    char             device_display_name[256];
    char             dac_format[160];
    int              output_mixable;
    int              device_physical_formats;
    int              device_exclusive_formats;
    int              device_min_sample_rate;
    int              device_max_sample_rate;
    int              device_max_bit_depth;
    int              device_channels;
    int              source_is_dsd;
    int              dsd_rate;
    int              dop_carrier_rate;
    int              dop_active;
    int              dop_marker_phase;

#if defined(__APPLE__)
    AudioObjectID    coreaudio_hog_device;
    AudioObjectID    coreaudio_io_device;
    AudioObjectID    coreaudio_io_stream;
    AudioDeviceIOProcID coreaudio_io_proc;
    AudioStreamBasicDescription coreaudio_original_format;
    int              coreaudio_original_format_valid;
    int              coreaudio_io_active;
#endif

    atomic_int       state;        // AVPLAYER_STATE_*
    atomic_int       seek_pending; // 1 = a seek is in progress
    double           seek_target;  // seconds (guarded by seek_pending)

    // Time tracking (updated by decode loop from packet/frame pts)
    _Atomic double   time_pos;
    _Atomic double   duration;

    float            volume;       // 0.0–1.0 (applied in miniaudio callback)

    // Peaks (updated each filtered frame; read from Go at ~60 Hz)
    _Atomic double   l_peak;
    _Atomic double   r_peak;
    _Atomic double   l_rms;
    _Atomic double   r_rms;
    int              peaks_enabled; // bool

    // Per-packet bitrate for VBR (updated in decode loop)
    _Atomic int      cur_bitrate;

    // Media info (written once when track opens)
    av_media_info_t  media_info;

    // EOF flag set by decode loop; cleared when ring drains
    atomic_int       eof_reached;      // 1 = decoder exhausted
    atomic_int       next_consumed;    // 1 = next track has been swapped in

    // Gapless track-change delay: AVPLAYER_DECODE_NEXT_READY is held back until
    // frames_played_total reaches track_change_threshold, meaning the new track's
    // audio has actually started playing out of the ring buffer.
    _Atomic long long frames_played_total;   // incremented in ma_data_callback
    atomic_int        pending_track_change;  // 1 = waiting to fire NEXT_READY
    long long         track_change_threshold; // non-atomic: only accessed by decode goroutine
    // Position clock: position = position_offset + (frames_played_total - position_clock_ref) / rate
    // Reset on av_player_open, av_player_seek, and when NEXT_READY fires.
    _Atomic double    position_offset;    // seconds at last clock reset
    _Atomic long long position_clock_ref; // frames_played_total at last clock reset

    // Protects pointer swaps of dec/next_dec between the decode goroutine
    // (gapless swap in av_player_decode_step) and the playback manager
    // goroutine (av_player_open_next).  Never held during slow operations
    // like decoder_free — only during pointer reads/writes.
    ma_mutex         decoder_lock;

    // Protects the active decoder's filter graph (filter_graph, buffersrc_ctx,
    // buffersink_ctx) against concurrent rebuilds from av_player_set_filters /
    // av_player_set_peaks_enabled while av_player_decode_step is using those
    // pointers.  Held briefly around each individual filter API call.
    ma_mutex         filter_lock;

    // EQ double-buffer: updated from Go thread, read in audio callback.
    // eq_active_idx (0 or 1) selects which bank the callback reads.
    // The other bank is safe to write to between swaps.
    eq_bank_t        eq_banks[2];
    atomic_int       eq_active_idx;

    // Bitrate history for display sync (see bitrate_history_t above)
    bitrate_history_t bitrate_hist;

    // ReplayGain output-stage gain (applied in miniaudio callback as a linear multiplier).
    // rg_gain is the active gain; a pending switch fires when frames_played_total
    // reaches rg_switch_threshold (set at gapless track boundary).
    _Atomic float      rg_gain;             // current linear gain (1.0 = no adjustment)
    _Atomic float      rg_gain_pending;     // gain for the next track
    _Atomic long long  rg_switch_threshold; // switch when frames_played_total >= this
    atomic_int         rg_switch_pending;   // 1 = a switch is waiting
    // Settings written from Go; read by decode loop at gapless swap.
    int    rg_mode;        // 0=off, 1=track, 2=album
    int    rg_prevent_clip;
    double rg_preamp_db;   // additional dB offset (from ReplayGainOptions.PreampGain)

    // ICY metadata: last seen StreamTitle, for change detection.
    // Only accessed from the decode goroutine and av_player_open (after decode stops).
    char   last_icy_title[512];
};

// --------------------------------------------------------------------------
// Ring buffer helpers
// --------------------------------------------------------------------------

static int ring_init(ring_buf_t *r, int sample_rate, int channels, ma_format format) {
    if (sample_rate <= 0) sample_rate = AVPLAYER_DEFAULT_SAMPLE_RATE;
    if (channels <= 0) channels = AVPLAYER_DEFAULT_CHANNELS;
    if (format == ma_format_unknown) format = ma_format_f32;

    r->sample_rate = sample_rate;
    r->channels = channels;
    r->format = format;
    r->bytes_per_frame = (int)ma_get_bytes_per_frame(format, (ma_uint32)channels);
    if (r->bytes_per_frame <= 0) return -1;

    r->cap = sample_rate * AVPLAYER_RING_SECONDS;
    r->buf = (uint8_t *)malloc((size_t)r->cap * (size_t)r->bytes_per_frame);
    if (!r->buf) return -1;
    atomic_store(&r->write_idx, 0);
    atomic_store(&r->read_idx,  0);
    atomic_store(&r->fill,      0);
    atomic_store_explicit((_Atomic long long *)&r->frames_written_total, 0LL, memory_order_relaxed);
    return 0;
}

static void ring_free(ring_buf_t *r) {
    free(r->buf);
    r->buf = NULL;
    r->cap = 0;
    r->channels = 0;
    r->bytes_per_frame = 0;
    r->format = ma_format_unknown;
    r->sample_rate = 0;
}

static int ring_reinit(ring_buf_t *r, int sample_rate, int channels, ma_format format) {
    if (r->buf &&
        r->sample_rate == sample_rate &&
        r->channels == channels &&
        r->format == format) {
        atomic_store(&r->write_idx, 0);
        atomic_store(&r->read_idx,  0);
        atomic_store(&r->fill,      0);
        atomic_store_explicit((_Atomic long long *)&r->frames_written_total, 0LL, memory_order_relaxed);
        return 0;
    }
    ring_free(r);
    return ring_init(r, sample_rate, channels, format);
}

static void ring_clear(ring_buf_t *r) {
    atomic_store(&r->write_idx, 0);
    atomic_store(&r->read_idx,  0);
    atomic_store(&r->fill,      0);
}

// Returns free frames available for writing
static int ring_space(const ring_buf_t *r) {
    return r->cap - atomic_load(&r->fill);
}

// Returns frames available to read
static int ring_avail(const ring_buf_t *r) {
    return atomic_load(&r->fill);
}

// Write exactly n_frames of interleaved PCM in r->format.
// Returns n_frames on success, 0 if there is not enough space (all-or-nothing).
// Never clips or drops samples — the caller must retry with the same data.
static int ring_write(ring_buf_t *r, const void *src, int n_frames) {
    if (ring_space(r) < n_frames) return 0;

    int wi  = atomic_load(&r->write_idx);
    int cap = r->cap;
    int bpf = r->bytes_per_frame;

    // How many frames fit before the wrap-around point?
    int first  = cap - wi;             // frames from wi to end of buffer
    if (first > n_frames) first = n_frames;
    int second = n_frames - first;     // frames that wrap to start of buffer

    memcpy(&r->buf[wi * bpf], src, (size_t)first * (size_t)bpf);
    if (second > 0) {
        memcpy(&r->buf[0], ((const uint8_t *)src) + (size_t)first * (size_t)bpf,
               (size_t)second * (size_t)bpf);
    }

    atomic_store(&r->write_idx, (wi + n_frames) % cap);
    atomic_fetch_add(&r->fill, n_frames);
    atomic_fetch_add_explicit((_Atomic long long *)&r->frames_written_total,
                              (long long)n_frames, memory_order_relaxed);
    return n_frames;
}

// Read up to n_frames into dst.  Returns number of frames actually read.
static int ring_read(ring_buf_t *r, void *dst, int n_frames) {
    int avail   = ring_avail(r);
    int to_read = (n_frames < avail) ? n_frames : avail;
    if (to_read == 0) return 0;

    int ri  = atomic_load(&r->read_idx);
    int cap = r->cap;
    int bpf = r->bytes_per_frame;

    int first  = cap - ri;
    if (first > to_read) first = to_read;
    int second = to_read - first;

    memcpy(dst, &r->buf[ri * bpf], (size_t)first * (size_t)bpf);
    if (second > 0) {
        memcpy(((uint8_t *)dst) + (size_t)first * (size_t)bpf, &r->buf[0],
               (size_t)second * (size_t)bpf);
    }

    atomic_store(&r->read_idx, (ri + to_read) % cap);
    atomic_fetch_sub(&r->fill, to_read);
    return to_read;
}

// --------------------------------------------------------------------------
// Output format helpers
// --------------------------------------------------------------------------

static ma_format ma_format_from_av_sample_fmt(enum AVSampleFormat fmt) {
    switch (av_get_packed_sample_fmt(fmt)) {
    case AV_SAMPLE_FMT_U8:
        return ma_format_u8;
    case AV_SAMPLE_FMT_S16:
        return ma_format_s16;
    case AV_SAMPLE_FMT_S32:
        return ma_format_s32;
    case AV_SAMPLE_FMT_FLT:
        return ma_format_f32;
    default:
        return ma_format_unknown;
    }
}

static enum AVSampleFormat av_sample_fmt_from_ma_format(ma_format format) {
    switch (format) {
    case ma_format_u8:
        return AV_SAMPLE_FMT_U8;
    case ma_format_s16:
        return AV_SAMPLE_FMT_S16;
    case ma_format_s32:
        return AV_SAMPLE_FMT_S32;
    case ma_format_f32:
        return AV_SAMPLE_FMT_FLT;
    default:
        return AV_SAMPLE_FMT_NONE;
    }
}

static const char *ma_format_label(ma_format format) {
    switch (format) {
    case ma_format_u8:
        return "u8";
    case ma_format_s16:
        return "s16";
    case ma_format_s24:
        return "s24";
    case ma_format_s32:
        return "s32";
    case ma_format_f32:
        return "f32";
    default:
        return "unknown";
    }
}

static int ma_format_bit_depth(ma_format format) {
    switch (format) {
    case ma_format_u8:
        return 8;
    case ma_format_s16:
        return 16;
    case ma_format_s24:
        return 24;
    case ma_format_s32:
    case ma_format_f32:
        return 32;
    default:
        return 0;
    }
}

static int source_bit_depth_from_decoder(const decoder_t *d) {
    if (!d || !d->codec_ctx) return 0;
    if (d->codec_ctx->bits_per_raw_sample > 0) {
        return d->codec_ctx->bits_per_raw_sample;
    }
    if (d->codec_ctx->bits_per_coded_sample > 0) {
        return d->codec_ctx->bits_per_coded_sample;
    }
    enum AVSampleFormat packed = av_get_packed_sample_fmt(d->codec_ctx->sample_fmt);
    switch (packed) {
    case AV_SAMPLE_FMT_U8:
        return 8;
    case AV_SAMPLE_FMT_S16:
        return 16;
    case AV_SAMPLE_FMT_S32:
    case AV_SAMPLE_FMT_FLT:
        return 32;
    case AV_SAMPLE_FMT_DBL:
    case AV_SAMPLE_FMT_S64:
        return 64;
    default:
        return 0;
    }
}

static uint32_t read_le32(const uint8_t *p) {
    return ((uint32_t)p[0]) |
           ((uint32_t)p[1] << 8) |
           ((uint32_t)p[2] << 16) |
           ((uint32_t)p[3] << 24);
}

static int wavpack_packet_is_dsd(const AVPacket *pkt) {
    const uint32_t wavpack_dsd_flag = 0x80000000u;
    if (!pkt || pkt->size < 32 || memcmp(pkt->data, "wvpk", 4) != 0) return 0;
    return (read_le32(pkt->data + 24) & wavpack_dsd_flag) != 0;
}

static void decoder_probe_wavpack_dsd(decoder_t *d) {
    if (!d || !d->fmt_ctx || !d->codec_ctx ||
        d->codec_ctx->codec_id != AV_CODEC_ID_WAVPACK ||
        !d->fmt_ctx->pb ||
        (d->fmt_ctx->pb->seekable & AVIO_SEEKABLE_NORMAL) == 0) {
        return;
    }

    AVPacket *probe = av_packet_alloc();
    if (!probe) return;
    for (int i = 0; i < 16; i++) {
        int ret = av_read_frame(d->fmt_ctx, probe);
        if (ret < 0) break;
        if (probe->stream_index == d->audio_stream_idx && wavpack_packet_is_dsd(probe)) {
            d->source_is_wavpack_dsd = 1;
            av_packet_unref(probe);
            break;
        }
        av_packet_unref(probe);
    }
    av_packet_free(&probe);
    av_seek_frame(d->fmt_ctx, d->audio_stream_idx, 0, AVSEEK_FLAG_BACKWARD);
    if (d->codec_ctx) {
        avcodec_flush_buffers(d->codec_ctx);
    }
}

static void decoder_close_wavpack(decoder_t *d) {
    if (!d || !d->wv_ctx) return;
    d->wv_ctx = WavpackCloseFile(d->wv_ctx);
    d->wv_dsd = 0;
    d->wv_samples_unpacked = 0;
}

static void decoder_open_wavpack_dsd(decoder_t *d) {
    if (!d || !d->source_is_wavpack_dsd || !d->url[0]) return;

    char error[256] = {0};
    d->wv_ctx = WavpackOpenFileInput(d->url, error, OPEN_DSD_NATIVE, 0);
    if (!d->wv_ctx) {
        snprintf(d->wv_error, sizeof(d->wv_error),
                 "%s", error[0] ? error : "libwavpack could not open source");
        AV_DEBUG("WavPack DSD native open failed for %s: %s", d->url, d->wv_error);
        return;
    }

    int qmode = WavpackGetQualifyMode(d->wv_ctx);
    int channels = WavpackGetNumChannels(d->wv_ctx);
    int byte_rate = (int)WavpackGetSampleRate(d->wv_ctx);
    int native_rate = (int)WavpackGetNativeSampleRate(d->wv_ctx);
    if (native_rate <= 0 && byte_rate > 0) native_rate = byte_rate * 8;
    if ((qmode & QMODE_DSD_AUDIO) == 0 ||
        channels <= 0 || channels > AVPLAYER_MAX_DOP_CHANNELS ||
        byte_rate <= 0 || native_rate <= 0 ||
        WavpackGetBytesPerSample(d->wv_ctx) != 1 ||
        WavpackGetBitsPerSample(d->wv_ctx) != 8) {
        snprintf(d->wv_error, sizeof(d->wv_error),
                 "unsupported WavPack DSD stream for native DoP");
        decoder_close_wavpack(d);
        return;
    }

    d->wv_dsd = 1;
    d->wv_dsd_lsb_first = (qmode & QMODE_DSD_LSB_FIRST) != 0;
    d->wv_channels = channels;
    d->wv_dsd_byte_rate = byte_rate;
    d->wv_dsd_bit_rate = native_rate;
    d->wv_dop_carrier_rate = byte_rate / 2;
    d->wv_average_bitrate = (int)WavpackGetAverageBitrate(d->wv_ctx, 1);
    d->wv_samples_unpacked = 0;

    int64_t samples = WavpackGetNumSamples64(d->wv_ctx);
    if (samples > 0 && byte_rate > 0) {
        d->duration = (double)samples / (double)byte_rate;
    }
    d->codec_ctx->sample_rate = byte_rate;
    AVChannelLayout layout;
    av_channel_layout_default(&layout, channels);
    av_channel_layout_uninit(&d->codec_ctx->ch_layout);
    av_channel_layout_copy(&d->codec_ctx->ch_layout, &layout);
    av_channel_layout_uninit(&layout);
    AV_DEBUG("WavPack DSD native path ready channels=%d dsd_rate=%d carrier=%d qmode=0x%x",
             channels, d->wv_dsd_bit_rate, d->wv_dop_carrier_rate, qmode);
}

static int decoder_is_dsd(const decoder_t *d) {
    if (!d || !d->codec_ctx || !d->codec_ctx->codec) return 0;
    if (d->source_is_wavpack_dsd) return 1;
    switch (d->codec_ctx->codec_id) {
    case AV_CODEC_ID_DSD_LSBF:
    case AV_CODEC_ID_DSD_MSBF:
    case AV_CODEC_ID_DSD_LSBF_PLANAR:
    case AV_CODEC_ID_DSD_MSBF_PLANAR:
        return 1;
    default:
        break;
    }
    const char *name = d->codec_ctx->codec->name;
    return name &&
           (strstr(name, "dsd") != NULL ||
            strcmp(name, "dsf") == 0 ||
            strcmp(name, "dff") == 0);
}

static int decoder_supports_raw_dop(const decoder_t *d) {
    if (!d || !d->codec_ctx) return 0;
    if (d->wv_dsd) return 1;
    switch (d->codec_ctx->codec_id) {
    case AV_CODEC_ID_DSD_LSBF:
    case AV_CODEC_ID_DSD_MSBF:
    case AV_CODEC_ID_DSD_LSBF_PLANAR:
    case AV_CODEC_ID_DSD_MSBF_PLANAR:
        return 1;
    default:
        return 0;
    }
}

static int decoder_channel_count(const decoder_t *d) {
    if (!d) return 0;
    if (d->wv_dsd && d->wv_channels > 0) return d->wv_channels;
    if (d->codec_ctx && d->codec_ctx->ch_layout.nb_channels > 0) {
        return d->codec_ctx->ch_layout.nb_channels;
    }
    return 0;
}

static int decoder_dsd_is_planar(const decoder_t *d) {
    if (!d || !d->codec_ctx) return 0;
    if (d->wv_dsd) return 0;
    return d->codec_ctx->codec_id == AV_CODEC_ID_DSD_LSBF_PLANAR ||
           d->codec_ctx->codec_id == AV_CODEC_ID_DSD_MSBF_PLANAR;
}

static int decoder_dsd_is_lsb_first(const decoder_t *d) {
    if (!d || !d->codec_ctx) return 0;
    if (d->wv_dsd) return d->wv_dsd_lsb_first;
    return d->codec_ctx->codec_id == AV_CODEC_ID_DSD_LSBF ||
           d->codec_ctx->codec_id == AV_CODEC_ID_DSD_LSBF_PLANAR;
}

static int decoder_dsd_bit_rate(const decoder_t *d) {
    if (d && d->wv_dsd) return d->wv_dsd_bit_rate;
    if (!d || !d->codec_ctx || d->codec_ctx->sample_rate <= 0) return 0;
    // FFmpeg's DSF/DFF demuxers expose the DSD byte rate as sample_rate.
    return d->codec_ctx->sample_rate * 8;
}

static int decoder_dsd_byte_rate(const decoder_t *d) {
    if (d && d->wv_dsd) return d->wv_dsd_byte_rate;
    if (!d || !d->codec_ctx || d->codec_ctx->sample_rate <= 0) return 0;
    return d->codec_ctx->sample_rate;
}

static int decoder_dop_carrier_rate(const decoder_t *d) {
    if (d && d->wv_dsd) return d->wv_dop_carrier_rate;
    if (!d || !d->codec_ctx || d->codec_ctx->sample_rate <= 0) return 0;
    // DoP carries 16 one-bit DSD samples in each PCM carrier frame.
    return d->codec_ctx->sample_rate / 2;
}

static void decoder_clear_dop_state(decoder_t *d) {
    if (!d) return;
    if (d->dop_pending_pkt) {
        av_packet_unref(d->dop_pending_pkt);
    }
    memset(d->dop_pending_byte, 0, sizeof(d->dop_pending_byte));
    memset(d->dop_pending_valid, 0, sizeof(d->dop_pending_valid));
}

static void set_bitperfect_reason(av_player_t *p, const char *reason) {
    if (!reason) reason = "";
    strncpy(p->bitperfect_reason, reason, sizeof(p->bitperfect_reason) - 1);
    p->bitperfect_reason[sizeof(p->bitperfect_reason) - 1] = '\0';
}

static void choose_output_config(av_player_t *p, const decoder_t *d, output_config_t *cfg) {
    memset(cfg, 0, sizeof(*cfg));
    cfg->sample_rate = AVPLAYER_DEFAULT_SAMPLE_RATE;
    cfg->channels = AVPLAYER_DEFAULT_CHANNELS;
    cfg->av_format = AV_SAMPLE_FMT_FLT;
    cfg->ma_format = ma_format_f32;
    cfg->bit_depth = 32;
    cfg->exclusive = p->exclusive_requested || p->bitperfect_requested;
    cfg->bitperfect = p->bitperfect_requested;
    cfg->bitperfect_candidate = 0;
    strncpy(cfg->reason, cfg->exclusive ? "bit-perfect off" : "exclusive mode off",
            sizeof(cfg->reason) - 1);

    if (!cfg->bitperfect || !d || !d->codec_ctx) {
        return;
    }

    cfg->sample_rate = d->codec_ctx->sample_rate > 0 ? d->codec_ctx->sample_rate : AVPLAYER_DEFAULT_SAMPLE_RATE;
    cfg->channels = decoder_channel_count(d) > 0 ? decoder_channel_count(d) : AVPLAYER_DEFAULT_CHANNELS;
    cfg->av_format = av_get_packed_sample_fmt(d->codec_ctx->sample_fmt);
    cfg->ma_format = ma_format_from_av_sample_fmt(cfg->av_format);
    cfg->bit_depth = source_bit_depth_from_decoder(d);

    if (cfg->channels <= 0) {
        cfg->channels = AVPLAYER_DEFAULT_CHANNELS;
        strncpy(cfg->reason, "unknown channel layout", sizeof(cfg->reason) - 1);
        return;
    }
    if (decoder_is_dsd(d)) {
        cfg->dsd_bit_rate = decoder_dsd_bit_rate(d);
        cfg->dop_carrier_rate = decoder_dop_carrier_rate(d);
        if (!decoder_supports_raw_dop(d)) {
            cfg->bitperfect_candidate = 0;
            if (d->source_is_wavpack_dsd && d->wv_error[0]) {
                snprintf(cfg->reason, sizeof(cfg->reason),
                         "WavPack DSD native decode unavailable: %.110s", d->wv_error);
            } else {
                snprintf(cfg->reason, sizeof(cfg->reason),
                         "DSD DoP requires raw DSF/DFF or native WavPack DSD input");
            }
            cfg->av_format = AV_SAMPLE_FMT_FLT;
            cfg->ma_format = ma_format_f32;
            cfg->bit_depth = 32;
            return;
        }
        if (cfg->channels > AVPLAYER_MAX_DOP_CHANNELS) {
            cfg->bitperfect_candidate = 0;
            snprintf(cfg->reason, sizeof(cfg->reason),
                     "DoP supports up to %d channels in this backend", AVPLAYER_MAX_DOP_CHANNELS);
            cfg->av_format = AV_SAMPLE_FMT_FLT;
            cfg->ma_format = ma_format_f32;
            cfg->bit_depth = 32;
            return;
        }
        int dsd_byte_rate = decoder_dsd_byte_rate(d);
        if (cfg->dop_carrier_rate <= 0 || dsd_byte_rate <= 0 || (dsd_byte_rate % 2) != 0) {
            cfg->bitperfect_candidate = 0;
            snprintf(cfg->reason, sizeof(cfg->reason), "invalid DSD rate for DoP carrier");
            cfg->av_format = AV_SAMPLE_FMT_FLT;
            cfg->ma_format = ma_format_f32;
            cfg->bit_depth = 32;
            return;
        }
        cfg->sample_rate = cfg->dop_carrier_rate;
        cfg->av_format = AV_SAMPLE_FMT_S32;
        cfg->ma_format = ma_format_s32;
        cfg->bit_depth = 32;
        cfg->dop = 1;
        cfg->bitperfect_candidate = 1;
        cfg->reason[0] = '\0';
        return;
    }
    if (cfg->ma_format == ma_format_unknown) {
        snprintf(cfg->reason, sizeof(cfg->reason), "unsupported decoded sample format %s",
                 av_get_sample_fmt_name(d->codec_ctx->sample_fmt));
        cfg->av_format = AV_SAMPLE_FMT_FLT;
        cfg->ma_format = ma_format_f32;
        cfg->bit_depth = 32;
        return;
    }

    cfg->bitperfect_candidate = 1;
    cfg->reason[0] = '\0';
}

// --------------------------------------------------------------------------
// Decoder helpers
// --------------------------------------------------------------------------

static decoder_t *decoder_alloc(void) {
    decoder_t *d = (decoder_t *)calloc(1, sizeof(decoder_t));
    if (!d) return NULL;
    d->pkt        = av_packet_alloc();
    d->frame      = av_frame_alloc();
    d->filt_frame = av_frame_alloc();
    d->dop_pending_pkt = av_packet_alloc();
    if (!d->pkt || !d->frame || !d->filt_frame || !d->dop_pending_pkt) {
        av_packet_free(&d->pkt);
        av_packet_free(&d->dop_pending_pkt);
        av_frame_free(&d->frame);
        av_frame_free(&d->filt_frame);
        free(d);
        return NULL;
    }
    return d;
}

static void decoder_free(decoder_t *d) {
    if (!d) return;
    decoder_close_wavpack(d);
    if (d->filter_graph) avfilter_graph_free(&d->filter_graph);
    if (d->codec_ctx)    avcodec_free_context(&d->codec_ctx);
    if (d->fmt_ctx)      avformat_close_input(&d->fmt_ctx);
    av_packet_free(&d->pkt);
    av_packet_free(&d->dop_pending_pkt);
    av_frame_free(&d->frame);
    av_frame_free(&d->filt_frame);
    av_frame_free(&d->pending_frame);  // safe if NULL
    free(d);
}

// Build filter graph: abuffer → optional conversion → abuffersink
// ReplayGain and EQ are both applied in the miniaudio callback.
static int decoder_build_filter_graph(decoder_t *d, const output_config_t *out_cfg)
{
    if (d->filter_graph) {
        avfilter_graph_free(&d->filter_graph);
        d->filter_graph     = NULL;
        d->buffersrc_ctx    = NULL;
        d->buffersink_ctx   = NULL;
    }

    AVFilterGraph *graph = avfilter_graph_alloc();
    if (!graph) return AVERROR(ENOMEM);

    // --- abuffer source ---
    char src_args[256];
    AVRational tb = d->fmt_ctx->streams[d->audio_stream_idx]->time_base;
    snprintf(src_args, sizeof(src_args),
             "sample_rate=%d:sample_fmt=%s:channel_layout=0x%" PRIx64 ":time_base=%d/%d",
             d->codec_ctx->sample_rate,
             av_get_sample_fmt_name(d->codec_ctx->sample_fmt),
             (uint64_t)d->codec_ctx->ch_layout.u.mask,
             tb.num, tb.den);

    AVFilterContext *src_ctx = NULL;
    int ret = avfilter_graph_create_filter(&src_ctx,
                                           avfilter_get_by_name("abuffer"),
                                           "in", src_args, NULL, graph);
    if (ret < 0) goto fail;

    // --- abuffersink ---
    AVFilterContext *sink_ctx = NULL;
    ret = avfilter_graph_create_filter(&sink_ctx,
                                       avfilter_get_by_name("abuffersink"),
                                       "out", NULL, NULL, graph);
    if (ret < 0) goto fail;

    // Output format constraints are enforced via the aformat filter in the chain
    // below rather than via deprecated av_opt_set_int_list on the sink.

    // --- Build filter chain string ---
    // ReplayGain and EQ are applied in the miniaudio callback. Unless strict
    // bit-perfect mode is requested, normalize to a stable 48 kHz float stereo
    // stream so app volume/EQ/ReplayGain remain available even with hog mode.
    char chain[256];
    const char *fmt_name = av_get_sample_fmt_name(out_cfg->av_format);
    uint64_t layout_mask = d->codec_ctx->ch_layout.u.mask;
    if (layout_mask == 0) {
        AVChannelLayout tmp;
        av_channel_layout_default(&tmp, out_cfg->channels);
        layout_mask = tmp.u.mask;
        av_channel_layout_uninit(&tmp);
    }
    if (out_cfg->bitperfect && out_cfg->bitperfect_candidate) {
        snprintf(chain, sizeof(chain),
                 "aformat=sample_fmts=%s:sample_rates=%d:channel_layouts=0x%" PRIx64,
                 fmt_name ? fmt_name : "flt",
                 out_cfg->sample_rate,
                 layout_mask);
    } else {
        snprintf(chain, sizeof(chain),
                 "aresample=" AVPLAYER_DEFAULT_SAMPLE_RATE_STR
                 ",aformat=sample_fmts=flt:channel_layouts=stereo");
    }

    // Link: in → chain → out
    AVFilterInOut *outputs = avfilter_inout_alloc();
    AVFilterInOut *inputs  = avfilter_inout_alloc();
    if (!outputs || !inputs) {
        avfilter_inout_free(&outputs);
        avfilter_inout_free(&inputs);
        ret = AVERROR(ENOMEM);
        goto fail;
    }
    outputs->name       = av_strdup("in");
    outputs->filter_ctx = src_ctx;
    outputs->pad_idx    = 0;
    outputs->next       = NULL;

    inputs->name        = av_strdup("out");
    inputs->filter_ctx  = sink_ctx;
    inputs->pad_idx     = 0;
    inputs->next        = NULL;

    ret = avfilter_graph_parse_ptr(graph, chain, &inputs, &outputs, NULL);
    avfilter_inout_free(&outputs);
    avfilter_inout_free(&inputs);
    if (ret < 0) goto fail;

    ret = avfilter_graph_config(graph, NULL);
    if (ret < 0) goto fail;

    d->filter_graph   = graph;
    d->buffersrc_ctx  = src_ctx;
    d->buffersink_ctx = sink_ctx;
    return 0;

fail:
    avfilter_graph_free(&graph);
    return ret;
}

// Look up a metadata key in container-level and stream-level dicts.
// Tries uppercase first, then lowercase.  Returns the entry or NULL.
static AVDictionaryEntry *rg_dict_get(AVFormatContext *fmt_ctx, int stream_idx,
                                       const char *key_upper, const char *key_lower)
{
    AVDictionary *dicts[2] = {
        fmt_ctx->metadata,
        fmt_ctx->streams[stream_idx]->metadata
    };
    for (int i = 0; i < 2; i++) {
        if (!dicts[i]) continue;
        AVDictionaryEntry *e = av_dict_get(dicts[i], key_upper, NULL, 0);
        if (!e) e = av_dict_get(dicts[i], key_lower, NULL, 0);
        if (e) return e;
    }
    return NULL;
}

// Read all four ReplayGain tags from d->fmt_ctx into d->rg_* fields.
// Defaults: gain = 0.0 dB, peak = 1.0 (linear).
static void decoder_read_rg_tags(decoder_t *d) {
    d->rg_track_gain_db = 0.0;
    d->rg_album_gain_db = 0.0;
    d->rg_track_peak    = 1.0f;
    d->rg_album_peak    = 1.0f;

    AVDictionaryEntry *e;
    // Gain values: "−6.23 dB" — atof stops at the non-numeric ' ' or 'd'
    e = rg_dict_get(d->fmt_ctx, d->audio_stream_idx,
                    "REPLAYGAIN_TRACK_GAIN", "replaygain_track_gain");
    if (e) d->rg_track_gain_db = atof(e->value);

    e = rg_dict_get(d->fmt_ctx, d->audio_stream_idx,
                    "REPLAYGAIN_ALBUM_GAIN", "replaygain_album_gain");
    if (e) d->rg_album_gain_db = atof(e->value);

    // Peak values: linear 0..1 (e.g. "0.988312")
    e = rg_dict_get(d->fmt_ctx, d->audio_stream_idx,
                    "REPLAYGAIN_TRACK_PEAK", "replaygain_track_peak");
    if (e) d->rg_track_peak = (float)atof(e->value);

    e = rg_dict_get(d->fmt_ctx, d->audio_stream_idx,
                    "REPLAYGAIN_ALBUM_PEAK", "replaygain_album_peak");
    if (e) d->rg_album_peak = (float)atof(e->value);
}

// Compute the linear RG gain multiplier for the given decoder and settings.
// rg_mode: 0=off, 1=track, 2=album (falls back to track if album tag absent).
static float compute_rg_gain(const decoder_t *d, int rg_mode, int prevent_clip, double preamp_db) {
    if (!d || rg_mode == 0) return 1.0f;

    double gain_db;
    float  peak;
    if (rg_mode == 2 && d->rg_album_gain_db != 0.0) {
        gain_db = d->rg_album_gain_db;
        peak    = d->rg_album_peak;
    } else {
        gain_db = d->rg_track_gain_db;
        peak    = d->rg_track_peak;
    }

    gain_db += preamp_db;
    float linear = (float)pow(10.0, gain_db / 20.0);

    if (prevent_clip && peak > 0.0f) {
        float max_gain = 1.0f / peak;
        if (linear > max_gain) linear = max_gain;
    }
    return linear;
}

// Open a URL; populate dec->fmt_ctx, codec_ctx, audio_stream_idx, duration.
static int decoder_open(decoder_t *d, const char *url) {
    // Tune for network streams (generous probe)
    AVDictionary *opts = NULL;
    av_dict_set(&opts, "timeout", "10000000", 0);    // 10s connect timeout
    av_dict_set(&opts, "reconnect", "1", 0);
    av_dict_set(&opts, "reconnect_streamed", "1", 0);
    av_dict_set(&opts, "reconnect_delay_max", "5", 0);
    av_dict_set(&opts, "user_agent", "supersonic/1.0", 0);

    int ret = avformat_open_input(&d->fmt_ctx, url, NULL, &opts);
    av_dict_free(&opts);
    if (ret < 0) return ret;

    ret = avformat_find_stream_info(d->fmt_ctx, NULL);
    if (ret < 0) { avformat_close_input(&d->fmt_ctx); return ret; }

    const AVCodec *codec = NULL;
    int stream_idx = av_find_best_stream(d->fmt_ctx, AVMEDIA_TYPE_AUDIO, -1, -1, &codec, 0);
    if (stream_idx < 0) {
        avformat_close_input(&d->fmt_ctx);
        return stream_idx;
    }
    d->audio_stream_idx = stream_idx;

    d->codec_ctx = avcodec_alloc_context3(codec);
    if (!d->codec_ctx) { avformat_close_input(&d->fmt_ctx); return AVERROR(ENOMEM); }

    ret = avcodec_parameters_to_context(d->codec_ctx, d->fmt_ctx->streams[stream_idx]->codecpar);
    if (ret < 0) goto fail;

    ret = avcodec_open2(d->codec_ctx, codec, NULL);
    if (ret < 0) goto fail;

    // Ensure we have a valid channel layout
    if (!d->codec_ctx->ch_layout.nb_channels) {
        av_channel_layout_default(&d->codec_ctx->ch_layout, 2);
    }
    if (!d->codec_ctx->ch_layout.u.mask) {
        // Convert from nb_channels if mask is 0
        AVChannelLayout tmp = AV_CHANNEL_LAYOUT_STEREO;
        if (d->codec_ctx->ch_layout.nb_channels == 1) {
            tmp = (AVChannelLayout)AV_CHANNEL_LAYOUT_MONO;
        }
        av_channel_layout_copy(&d->codec_ctx->ch_layout, &tmp);
    }

    double dur_secs = 0.0;
    if (d->fmt_ctx->duration > 0) {
        dur_secs = (double)d->fmt_ctx->duration / AV_TIME_BASE;
    }
    d->duration = dur_secs;

    strncpy(d->url, url, sizeof(d->url) - 1);
    decoder_probe_wavpack_dsd(d);
    decoder_open_wavpack_dsd(d);
    decoder_read_rg_tags(d);
    return 0;

fail:
    avcodec_free_context(&d->codec_ctx);
    avformat_close_input(&d->fmt_ctx);
    return ret;
}

// --------------------------------------------------------------------------
// Audio render callbacks
// --------------------------------------------------------------------------

static void player_render_audio(av_player_t *p, void *output, ma_uint32 frame_count) {
    int channels = p->ring.channels > 0 ? p->ring.channels : AVPLAYER_DEFAULT_CHANNELS;
    int bytes_per_frame = p->ring.bytes_per_frame > 0
                          ? p->ring.bytes_per_frame
                          : (int)ma_get_bytes_per_frame(p->output_format, (ma_uint32)channels);

    if (atomic_load(&p->state) != AVPLAYER_STATE_PLAYING) {
        // Silence output when paused or stopped
        memset(output, 0, (size_t)frame_count * (size_t)bytes_per_frame);
        return;
    }

    int got = ring_read(&p->ring, output, (int)frame_count);
    if (got > 0)
        atomic_fetch_add_explicit((_Atomic long long *)&p->frames_played_total,
                                  (long long)got, memory_order_relaxed);

    // Compute peaks from pre-volume PCM (represents the demuxed/decoded signal).
    if (p->peaks_enabled && got > 0 && p->output_format == ma_format_f32) {
        float *out = (float *)output;
        float l_max = 0.0f, r_max = 0.0f;
        float l_sq = 0.0f, r_sq = 0.0f;
        for (int i = 0; i < got; i++) {
            float l = out[i * channels + 0];
            float r = channels > 1 ? out[i * channels + 1] : l;
            float al = l < 0.0f ? -l : l;
            float ar = r < 0.0f ? -r : r;
            if (al > l_max) l_max = al;
            if (ar > r_max) r_max = ar;
            l_sq += l * l;
            r_sq += r * r;
        }
        double lp = (l_max > 0.0f) ? 20.0 * log10((double)l_max) : -INFINITY;
        double rp = (r_max > 0.0f) ? 20.0 * log10((double)r_max) : -INFINITY;
        double lr = (l_sq > 0.0f) ? 10.0 * log10((double)(l_sq / got)) : -INFINITY;
        double rr = (r_sq > 0.0f) ? 10.0 * log10((double)(r_sq / got)) : -INFINITY;
        atomic_store_explicit((_Atomic double *)&p->l_peak, lp, memory_order_relaxed);
        atomic_store_explicit((_Atomic double *)&p->r_peak, rp, memory_order_relaxed);
        atomic_store_explicit((_Atomic double *)&p->l_rms,  lr, memory_order_relaxed);
        atomic_store_explicit((_Atomic double *)&p->r_rms,  rr, memory_order_relaxed);
    } else if (got == 0) {
        atomic_store_explicit((_Atomic double *)&p->l_peak, -INFINITY, memory_order_relaxed);
        atomic_store_explicit((_Atomic double *)&p->r_peak, -INFINITY, memory_order_relaxed);
        atomic_store_explicit((_Atomic double *)&p->l_rms,  -INFINITY, memory_order_relaxed);
        atomic_store_explicit((_Atomic double *)&p->r_rms,  -INFINITY, memory_order_relaxed);
    }

    // Effective bit-perfect mode is a strict, no-DSP path. Also return early
    // for integer PCM because the EQ/RG/volume code below operates on float.
    if (p->bitperfect_active || p->output_format != ma_format_f32) {
        if (got < (int)frame_count) {
            memset(((uint8_t *)output) + (size_t)got * (size_t)bytes_per_frame, 0,
                   (size_t)(frame_count - got) * (size_t)bytes_per_frame);
        }
        return;
    }

    float *out = (float *)output;

    // Apply EQ preamp + bands (after peaks, before volume — peak meter reflects raw decoded signal)
    if (got > 0) {
        int eq_idx = atomic_load_explicit(&p->eq_active_idx, memory_order_acquire);
        eq_bank_t *eq = &p->eq_banks[eq_idx];
        if (eq->preamp != 1.0f && eq->preamp > 0.0f) {
            for (int i = 0; i < got * channels; i++)
                out[i] *= eq->preamp;
        }
        if (eq->initialized && eq->num_bands > 0) {
            for (int b = 0; b < eq->num_bands; b++) {
                ma_peak2_process_pcm_frames(&eq->filters[b], out, out, (ma_uint64)got);
            }
        }
    }

    // Switch RG gain at track boundary if pending
    if (atomic_load_explicit(&p->rg_switch_pending, memory_order_acquire)) {
        long long fp = atomic_load_explicit(
            (_Atomic long long *)&p->frames_played_total, memory_order_relaxed);
        if (fp >= atomic_load_explicit(&p->rg_switch_threshold, memory_order_relaxed)) {
            atomic_store_explicit(&p->rg_gain,
                atomic_load_explicit(&p->rg_gain_pending, memory_order_relaxed),
                memory_order_relaxed);
            atomic_store_explicit(&p->rg_switch_pending, 0, memory_order_relaxed);
        }
    }

    // Apply ReplayGain
    if (got > 0) {
        float rg = atomic_load_explicit(&p->rg_gain, memory_order_relaxed);
        if (rg != 1.0f && rg > 0.0f) {
            for (int i = 0; i < got * channels; i++)
                out[i] *= rg;
        }
    }

    // Apply volume
    float vol = p->volume;
    if (vol < 0.9999f || vol > 1.0001f) {
        for (int i = 0; i < got * channels; i++) {
            out[i] *= vol;
        }
    }
    // Fill remainder with silence
    if (got < (int)frame_count) {
        memset(out + got * channels, 0,
               (size_t)(frame_count - got) * (size_t)channels * sizeof(float));
    }
}

static void ma_data_callback(ma_device *device, void *output, const void *input, ma_uint32 frame_count) {
    (void)input;
    av_player_t *p = (av_player_t *)device->pUserData;
    player_render_audio(p, output, frame_count);
}

// --------------------------------------------------------------------------
// Device management helpers
// --------------------------------------------------------------------------

#if defined(__APPLE__)
#ifndef kAudioObjectPropertyElementMain
#define kAudioObjectPropertyElementMain 0
#endif

typedef struct {
    AudioObjectID stream_id;
    AudioStreamBasicDescription asbd;
    int score;
    int bit_depth;
    int mixable;
    char label[160];
} coreaudio_format_choice_t;

static void player_release_coreaudio_hog(av_player_t *p);

static int coreaudio_default_output_device(AudioObjectID *device_id) {
    AudioObjectPropertyAddress addr = {
        kAudioHardwarePropertyDefaultOutputDevice,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    UInt32 size = sizeof(*device_id);
    OSStatus status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, device_id);
    return status == noErr && *device_id != kAudioObjectUnknown ? 0 : -1;
}

static int coreaudio_device_for_uid(const char *uid, AudioObjectID *device_id) {
    if (!uid || uid[0] == '\0') {
        return coreaudio_default_output_device(device_id);
    }

    CFStringRef uid_str = CFStringCreateWithCString(kCFAllocatorDefault, uid, kCFStringEncodingUTF8);
    if (!uid_str) return -1;

    AudioValueTranslation translation;
    translation.mInputData = &uid_str;
    translation.mInputDataSize = sizeof(uid_str);
    translation.mOutputData = device_id;
    translation.mOutputDataSize = sizeof(*device_id);

    AudioObjectPropertyAddress addr = {
        kAudioHardwarePropertyDeviceForUID,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    UInt32 size = sizeof(translation);
    OSStatus status = AudioObjectGetPropertyData(kAudioObjectSystemObject, &addr, 0, NULL, &size, &translation);
    CFRelease(uid_str);
    return status == noErr && *device_id != kAudioObjectUnknown ? 0 : -1;
}

static const char *coreaudio_transport_label(UInt32 transport) {
    switch (transport) {
    case kAudioDeviceTransportTypeBuiltIn:
        return "Built-in";
    case kAudioDeviceTransportTypeUSB:
        return "USB";
    case kAudioDeviceTransportTypeHDMI:
        return "HDMI";
    case kAudioDeviceTransportTypeDisplayPort:
        return "DisplayPort";
    case kAudioDeviceTransportTypeBluetooth:
    case kAudioDeviceTransportTypeBluetoothLE:
        return "Bluetooth";
    case kAudioDeviceTransportTypeAirPlay:
        return "AirPlay";
    case kAudioDeviceTransportTypeAVB:
        return "AVB";
    case kAudioDeviceTransportTypeThunderbolt:
        return "Thunderbolt";
    case kAudioDeviceTransportTypeVirtual:
        return "Virtual";
    case kAudioDeviceTransportTypeAggregate:
        return "Aggregate";
    default:
        return "Unknown";
    }
}

static int coreaudio_get_transport(AudioObjectID device_id, char *buf, size_t buflen) {
    if (!buf || buflen == 0) return -1;
    UInt32 transport = 0;
    UInt32 size = sizeof(transport);
    AudioObjectPropertyAddress addr = {
        kAudioDevicePropertyTransportType,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    if (AudioObjectGetPropertyData(device_id, &addr, 0, NULL, &size, &transport) != noErr) {
        strncpy(buf, "Unknown", buflen - 1);
        buf[buflen - 1] = '\0';
        return -1;
    }
    strncpy(buf, coreaudio_transport_label(transport), buflen - 1);
    buf[buflen - 1] = '\0';
    return 0;
}

static void coreaudio_fourcc(UInt32 value, char out[5]) {
    out[0] = (char)((value >> 24) & 0xff);
    out[1] = (char)((value >> 16) & 0xff);
    out[2] = (char)((value >> 8) & 0xff);
    out[3] = (char)(value & 0xff);
    out[4] = '\0';
    for (int i = 0; i < 4; i++) {
        if (out[i] < 32 || out[i] > 126) out[i] = '?';
    }
}

static int coreaudio_asbd_bit_depth(const AudioStreamBasicDescription *asbd) {
    if (!asbd) return 0;
    if (asbd->mBitsPerChannel > 0) return (int)asbd->mBitsPerChannel;
    if (asbd->mBytesPerFrame > 0 && asbd->mChannelsPerFrame > 0) {
        return (int)(asbd->mBytesPerFrame * 8 / asbd->mChannelsPerFrame);
    }
    return 0;
}

static int coreaudio_format_mixable(const AudioStreamBasicDescription *asbd) {
    return (asbd->mFormatFlags & kAudioFormatFlagIsNonMixable) == 0;
}

static void coreaudio_format_label(const AudioStreamBasicDescription *asbd, char *buf, size_t buflen) {
    if (!asbd || !buf || buflen == 0) return;
    char fourcc[5];
    coreaudio_fourcc(asbd->mFormatID, fourcc);
    const char *kind = fourcc;
    if (asbd->mFormatID == kAudioFormatLinearPCM) {
        if (asbd->mFormatFlags & kAudioFormatFlagIsFloat) {
            kind = "Float";
        } else if (asbd->mFormatFlags & kAudioFormatFlagIsSignedInteger) {
            kind = "SInt";
        } else {
            kind = "UInt";
        }
    }
    snprintf(buf, buflen, "%s%d %s %.0f Hz %uch",
             kind,
             coreaudio_asbd_bit_depth(asbd),
             coreaudio_format_mixable(asbd) ? "Mixable" : "Non-Mixable",
             asbd->mSampleRate,
             (unsigned)asbd->mChannelsPerFrame);
}

static void coreaudio_fill_device_caps(AudioObjectID device_id, av_device_info_t *info) {
    if (!info || device_id == kAudioObjectUnknown) return;

    coreaudio_get_transport(device_id, info->transport, sizeof(info->transport));
    info->lossy = strcmp(info->transport, "Bluetooth") == 0 ||
                  strcmp(info->transport, "AirPlay") == 0;

    AudioObjectPropertyAddress streams_addr = {
        kAudioDevicePropertyStreams,
        kAudioDevicePropertyScopeOutput,
        kAudioObjectPropertyElementMain
    };
    UInt32 streams_size = 0;
    if (AudioObjectGetPropertyDataSize(device_id, &streams_addr, 0, NULL, &streams_size) != noErr ||
        streams_size == 0) {
        snprintf(info->capability_summary, sizeof(info->capability_summary),
                 "%s output", info->transport[0] ? info->transport : "Unknown");
        return;
    }

    UInt32 stream_count = streams_size / sizeof(AudioObjectID);
    AudioObjectID *streams = (AudioObjectID *)calloc(stream_count, sizeof(AudioObjectID));
    if (!streams) return;
    if (AudioObjectGetPropertyData(device_id, &streams_addr, 0, NULL, &streams_size, streams) != noErr) {
        free(streams);
        return;
    }

    int min_rate = 0;
    int max_rate = 0;
    int max_bits = 0;
    int max_channels = 0;
    int physical_count = 0;
    int exclusive_count = 0;

    for (UInt32 i = 0; i < stream_count; i++) {
        AudioObjectPropertyAddress cur_addr = {
            kAudioStreamPropertyPhysicalFormat,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        AudioStreamBasicDescription current;
        UInt32 current_size = sizeof(current);
        if (info->active_format[0] == '\0' &&
            AudioObjectGetPropertyData(streams[i], &cur_addr, 0, NULL, &current_size, &current) == noErr) {
            coreaudio_format_label(&current, info->active_format, sizeof(info->active_format));
        }

        AudioObjectPropertyAddress formats_addr = {
            kAudioStreamPropertyAvailablePhysicalFormats,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        UInt32 formats_size = 0;
        if (AudioObjectGetPropertyDataSize(streams[i], &formats_addr, 0, NULL, &formats_size) != noErr ||
            formats_size == 0) {
            continue;
        }
        UInt32 format_count = formats_size / sizeof(AudioStreamRangedDescription);
        AudioStreamRangedDescription *formats =
            (AudioStreamRangedDescription *)calloc(format_count, sizeof(AudioStreamRangedDescription));
        if (!formats) continue;
        if (AudioObjectGetPropertyData(streams[i], &formats_addr, 0, NULL, &formats_size, formats) != noErr) {
            free(formats);
            continue;
        }

        for (UInt32 j = 0; j < format_count; j++) {
            AudioStreamBasicDescription *asbd = &formats[j].mFormat;
            if (asbd->mFormatID != kAudioFormatLinearPCM) continue;
            physical_count++;
            int lo = (int)floor(formats[j].mSampleRateRange.mMinimum + 0.5);
            int hi = (int)floor(formats[j].mSampleRateRange.mMaximum + 0.5);
            if (lo <= 0) lo = (int)floor(asbd->mSampleRate + 0.5);
            if (hi <= 0) hi = (int)floor(asbd->mSampleRate + 0.5);
            if (lo > 0 && (min_rate == 0 || lo < min_rate)) min_rate = lo;
            if (hi > max_rate) max_rate = hi;
            int bits = coreaudio_asbd_bit_depth(asbd);
            if (bits > max_bits) max_bits = bits;
            if ((int)asbd->mChannelsPerFrame > max_channels) max_channels = (int)asbd->mChannelsPerFrame;
            if (!coreaudio_format_mixable(asbd)) exclusive_count++;
        }
        free(formats);
    }

    free(streams);

    info->physical_format_count = physical_count;
    info->exclusive_format_count = exclusive_count;
    info->exclusive_ready = exclusive_count > 0 && !info->lossy;
    info->min_sample_rate = min_rate;
    info->max_sample_rate = max_rate;
    info->max_bit_depth = max_bits;
    info->channels = max_channels;
    snprintf(info->capability_summary, sizeof(info->capability_summary),
             "%s%s up to %d-bit / %.1f kHz",
             info->exclusive_ready ? "Exclusive Ready, " : "",
             info->transport[0] ? info->transport : "Unknown",
             max_bits,
             max_rate > 0 ? (double)max_rate / 1000.0 : 0.0);
}

static int coreaudio_score_format(const AudioStreamBasicDescription *asbd,
                                  int target_rate,
                                  int target_channels,
                                  int target_bits,
                                  int dop) {
    if (!asbd || asbd->mFormatID != kAudioFormatLinearPCM) return -1;
    if ((int)asbd->mChannelsPerFrame != target_channels) return -1;
    if ((int)floor(asbd->mSampleRate + 0.5) != target_rate) return -1;
    if ((asbd->mFormatFlags & kAudioFormatFlagIsNonInterleaved) != 0) return -1;
    if ((asbd->mFormatFlags & kAudioFormatFlagIsBigEndian) != 0) return -1;
    if ((asbd->mFormatFlags & kAudioFormatFlagIsFloat) != 0) return -1;
    if ((asbd->mFormatFlags & kAudioFormatFlagIsSignedInteger) == 0) return -1;

    int bits = coreaudio_asbd_bit_depth(asbd);
    if (dop) {
        if (bits != 24 && bits != 32) return -1;
        if (asbd->mBytesPerFrame != (UInt32)(4 * target_channels)) return -1;
        if (bits == 24 && (asbd->mFormatFlags & kAudioFormatFlagIsAlignedHigh) == 0) return -1;
    } else {
        if (bits <= 0 || bits != target_bits) return -1;
        if (bits % 8 != 0) return -1;
        if (asbd->mBytesPerFrame != (UInt32)((bits / 8) * target_channels)) return -1;
    }

    int score = coreaudio_format_mixable(asbd) ? 100 : 1000;
    if (dop && bits == 24) score += 20;
    score += 80;
    return score;
}

static int coreaudio_choose_physical_format(AudioObjectID device_id,
                                            int target_rate,
                                            int target_channels,
                                            int target_bits,
                                            int dop,
                                            coreaudio_format_choice_t *choice) {
    if (!choice) return -1;
    memset(choice, 0, sizeof(*choice));
    choice->stream_id = kAudioObjectUnknown;
    choice->score = -1;

    AudioObjectPropertyAddress streams_addr = {
        kAudioDevicePropertyStreams,
        kAudioDevicePropertyScopeOutput,
        kAudioObjectPropertyElementMain
    };
    UInt32 streams_size = 0;
    if (AudioObjectGetPropertyDataSize(device_id, &streams_addr, 0, NULL, &streams_size) != noErr ||
        streams_size == 0) {
        return -1;
    }
    UInt32 stream_count = streams_size / sizeof(AudioObjectID);
    AudioObjectID *streams = (AudioObjectID *)calloc(stream_count, sizeof(AudioObjectID));
    if (!streams) return -1;
    if (AudioObjectGetPropertyData(device_id, &streams_addr, 0, NULL, &streams_size, streams) != noErr) {
        free(streams);
        return -1;
    }

    for (UInt32 i = 0; i < stream_count; i++) {
        AudioObjectPropertyAddress formats_addr = {
            kAudioStreamPropertyAvailablePhysicalFormats,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        UInt32 formats_size = 0;
        if (AudioObjectGetPropertyDataSize(streams[i], &formats_addr, 0, NULL, &formats_size) != noErr ||
            formats_size == 0) {
            continue;
        }
        UInt32 format_count = formats_size / sizeof(AudioStreamRangedDescription);
        AudioStreamRangedDescription *formats =
            (AudioStreamRangedDescription *)calloc(format_count, sizeof(AudioStreamRangedDescription));
        if (!formats) continue;
        if (AudioObjectGetPropertyData(streams[i], &formats_addr, 0, NULL, &formats_size, formats) != noErr) {
            free(formats);
            continue;
        }
        for (UInt32 j = 0; j < format_count; j++) {
            AudioStreamBasicDescription candidate = formats[j].mFormat;
            int lo = (int)floor(formats[j].mSampleRateRange.mMinimum + 0.5);
            int hi = (int)floor(formats[j].mSampleRateRange.mMaximum + 0.5);
            if (lo > 0 && hi > 0 && (target_rate < lo || target_rate > hi)) {
                continue;
            }
            candidate.mSampleRate = (Float64)target_rate;
            int score = coreaudio_score_format(&candidate, target_rate, target_channels, target_bits, dop);
            if (score > choice->score) {
                choice->stream_id = streams[i];
                choice->asbd = candidate;
                choice->score = score;
                choice->bit_depth = coreaudio_asbd_bit_depth(&candidate);
                choice->mixable = coreaudio_format_mixable(&candidate);
                coreaudio_format_label(&candidate, choice->label, sizeof(choice->label));
            }
        }
        free(formats);
    }

    free(streams);
    return choice->score >= 0 ? 0 : -1;
}

static int coreaudio_set_hog_mode(AudioObjectID device_id, int enabled) {
    pid_t hog_pid = enabled ? getpid() : (pid_t)-1;
    AudioObjectPropertyAddress addr = {
        kAudioDevicePropertyHogMode,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    OSStatus status = AudioObjectSetPropertyData(device_id, &addr, 0, NULL, sizeof(hog_pid), &hog_pid);
    return status == noErr ? 0 : -1;
}

static int coreaudio_set_system_volume_full(AudioObjectID device_id) {
    Float32 volume = 1.0f;
    Boolean settable = false;
    AudioObjectPropertyAddress service_addr = {
        kAudioHardwareServiceDeviceProperty_VirtualMainVolume,
        kAudioDevicePropertyScopeOutput,
        kAudioObjectPropertyElementMain
    };
    if (AudioObjectHasProperty(device_id, &service_addr) &&
        AudioObjectIsPropertySettable(device_id, &service_addr, &settable) == noErr &&
        settable &&
        AudioObjectSetPropertyData(device_id, &service_addr, 0, NULL, sizeof(volume), &volume) == noErr) {
        return 0;
    }

    settable = false;
    AudioObjectPropertyAddress scalar_addr = {
        kAudioDevicePropertyVolumeScalar,
        kAudioDevicePropertyScopeOutput,
        kAudioObjectPropertyElementMain
    };
    if (AudioObjectHasProperty(device_id, &scalar_addr) &&
        AudioObjectIsPropertySettable(device_id, &scalar_addr, &settable) == noErr &&
        settable &&
        AudioObjectSetPropertyData(device_id, &scalar_addr, 0, NULL, sizeof(volume), &volume) == noErr) {
        return 0;
    }

    return -1;
}

static OSStatus coreaudio_io_proc(AudioObjectID inDevice,
                                  const AudioTimeStamp *inNow,
                                  const AudioBufferList *inInputData,
                                  const AudioTimeStamp *inInputTime,
                                  AudioBufferList *outOutputData,
                                  const AudioTimeStamp *inOutputTime,
                                  void *inClientData) {
    (void)inDevice;
    (void)inNow;
    (void)inInputData;
    (void)inInputTime;
    (void)inOutputTime;
    av_player_t *p = (av_player_t *)inClientData;
    if (!p || !outOutputData || outOutputData->mNumberBuffers == 0) {
        return noErr;
    }

    int bytes_per_frame = p->ring.bytes_per_frame > 0 ? p->ring.bytes_per_frame : 0;
    if (bytes_per_frame <= 0) {
        for (UInt32 i = 0; i < outOutputData->mNumberBuffers; i++) {
            if (outOutputData->mBuffers[i].mData && outOutputData->mBuffers[i].mDataByteSize > 0) {
                memset(outOutputData->mBuffers[i].mData, 0, outOutputData->mBuffers[i].mDataByteSize);
            }
        }
        return noErr;
    }

    AudioBuffer *primary = &outOutputData->mBuffers[0];
    ma_uint32 frame_count = (ma_uint32)(primary->mDataByteSize / (UInt32)bytes_per_frame);
    player_render_audio(p, primary->mData, frame_count);

    for (UInt32 i = 1; i < outOutputData->mNumberBuffers; i++) {
        if (outOutputData->mBuffers[i].mData && outOutputData->mBuffers[i].mDataByteSize > 0) {
            memset(outOutputData->mBuffers[i].mData, 0, outOutputData->mBuffers[i].mDataByteSize);
        }
    }
    return noErr;
}

static void player_release_coreaudio_ioproc(av_player_t *p) {
    if (p->coreaudio_io_active) {
        (void)AudioDeviceStop(p->coreaudio_io_device, p->coreaudio_io_proc);
        if (p->coreaudio_io_proc) {
            (void)AudioDeviceDestroyIOProcID(p->coreaudio_io_device, p->coreaudio_io_proc);
        }
    }
    if (p->coreaudio_original_format_valid && p->coreaudio_io_stream != kAudioObjectUnknown) {
        AudioObjectPropertyAddress fmt_addr = {
            kAudioStreamPropertyPhysicalFormat,
            kAudioObjectPropertyScopeGlobal,
            kAudioObjectPropertyElementMain
        };
        (void)AudioObjectSetPropertyData(p->coreaudio_io_stream, &fmt_addr, 0, NULL,
                                         sizeof(p->coreaudio_original_format),
                                         &p->coreaudio_original_format);
    }
    p->coreaudio_io_active = 0;
    p->coreaudio_io_proc = NULL;
    p->coreaudio_io_device = kAudioObjectUnknown;
    p->coreaudio_io_stream = kAudioObjectUnknown;
    p->coreaudio_original_format_valid = 0;
}

static int player_init_coreaudio_ioproc(av_player_t *p,
                                        const char *uid,
                                        const char *display_name,
                                        const output_config_t *out_cfg) {
    AudioObjectID device_id = kAudioObjectUnknown;
    if (coreaudio_device_for_uid(uid, &device_id) != 0) {
        set_bitperfect_reason(p, "CoreAudio device lookup failed");
        return -1;
    }

    av_device_info_t caps;
    memset(&caps, 0, sizeof(caps));
    coreaudio_fill_device_caps(device_id, &caps);

    int target_bits = ma_format_bit_depth(out_cfg->ma_format);
    if (target_bits <= 0) {
        set_bitperfect_reason(p, "unsupported bit-perfect output format");
        return -1;
    }

    coreaudio_format_choice_t choice;
    if (coreaudio_choose_physical_format(device_id,
                                         out_cfg->sample_rate,
                                         out_cfg->channels,
                                         target_bits,
                                         out_cfg->dop,
                                         &choice) != 0 ||
        choice.mixable) {
        set_bitperfect_reason(p, out_cfg->dop
                              ? "no matching non-mixable CoreAudio DoP carrier format"
                              : "no matching non-mixable integer CoreAudio format");
        return -1;
    }

    AudioObjectPropertyAddress fmt_addr = {
        kAudioStreamPropertyPhysicalFormat,
        kAudioObjectPropertyScopeGlobal,
        kAudioObjectPropertyElementMain
    };
    UInt32 fmt_size = sizeof(p->coreaudio_original_format);
    if (AudioObjectGetPropertyData(choice.stream_id, &fmt_addr, 0, NULL,
                                   &fmt_size, &p->coreaudio_original_format) == noErr) {
        p->coreaudio_original_format_valid = 1;
    } else {
        p->coreaudio_original_format_valid = 0;
    }

    if (coreaudio_set_system_volume_full(device_id) == 0) {
        AV_DEBUG("CoreAudio system volume set to 100%% for uid=%s", uid ? uid : "<default>");
    } else {
        AV_DEBUG("CoreAudio system volume not writable for uid=%s", uid ? uid : "<default>");
    }

    if (coreaudio_set_hog_mode(device_id, 1) != 0) {
        set_bitperfect_reason(p, "CoreAudio hog mode unavailable");
        p->coreaudio_original_format_valid = 0;
        return -1;
    }
    p->coreaudio_hog_device = device_id;

    p->coreaudio_io_stream = choice.stream_id;
    if (AudioObjectSetPropertyData(choice.stream_id, &fmt_addr, 0, NULL,
                                   sizeof(choice.asbd), &choice.asbd) != noErr) {
        set_bitperfect_reason(p, "failed to set CoreAudio physical format");
        player_release_coreaudio_ioproc(p);
        player_release_coreaudio_hog(p);
        return -1;
    }

    AudioDeviceIOProcID proc = NULL;
    if (AudioDeviceCreateIOProcID(device_id, coreaudio_io_proc, p, &proc) != noErr) {
        set_bitperfect_reason(p, "failed to create CoreAudio IOProc");
        player_release_coreaudio_ioproc(p);
        player_release_coreaudio_hog(p);
        return -1;
    }

    if (AudioDeviceStart(device_id, proc) != noErr) {
        (void)AudioDeviceDestroyIOProcID(device_id, proc);
        set_bitperfect_reason(p, "failed to start CoreAudio IOProc");
        player_release_coreaudio_ioproc(p);
        player_release_coreaudio_hog(p);
        return -1;
    }

    p->coreaudio_io_device = device_id;
    p->coreaudio_io_proc = proc;
    p->coreaudio_io_active = 1;
    p->exclusive_active = 1;
    p->output_mixable = choice.mixable;
    strncpy(p->dac_format, choice.label, sizeof(p->dac_format) - 1);
    strncpy(p->playback_path,
            out_cfg->dop ? "CoreAudioDoPIOProc" : "CoreAudioBitPerfectIOProc",
            sizeof(p->playback_path) - 1);
    strncpy(p->signal_status,
            out_cfg->dop ? "Bit-Perfect DoP" : "Bit-Perfect",
            sizeof(p->signal_status) - 1);
    if (uid && uid[0]) {
        strncpy(p->device_uid, uid, sizeof(p->device_uid) - 1);
    }
    if (display_name && display_name[0]) {
        strncpy(p->device_display_name, display_name, sizeof(p->device_display_name) - 1);
    }
    strncpy(p->device_transport, caps.transport, sizeof(p->device_transport) - 1);
    p->device_physical_formats = caps.physical_format_count;
    p->device_exclusive_formats = caps.exclusive_format_count;
    p->device_min_sample_rate = caps.min_sample_rate;
    p->device_max_sample_rate = caps.max_sample_rate;
    p->device_max_bit_depth = caps.max_bit_depth;
    p->device_channels = caps.channels;

    AV_DEBUG("CoreAudio IOProc active format=%s", choice.label);
    return 0;
}

static void player_release_coreaudio_hog(av_player_t *p) {
    if (p->coreaudio_hog_device != kAudioObjectUnknown) {
        (void)coreaudio_set_hog_mode(p->coreaudio_hog_device, 0);
        p->coreaudio_hog_device = kAudioObjectUnknown;
    }
}
#endif

static void player_release_exclusive(av_player_t *p) {
#if defined(__APPLE__)
    player_release_coreaudio_ioproc(p);
    player_release_coreaudio_hog(p);
#endif
    p->exclusive_active = 0;
}

static int player_init_device(av_player_t *p, const char *device_name, const output_config_t *out_cfg) {
    player_release_exclusive(p);
    if (p->device_init) {
        ma_device_uninit(&p->device);
        p->device_init = 0;
    }
    p->output_mixable = 1;
    p->dop_active = 0;
    p->device_uid[0] = '\0';
    p->device_transport[0] = '\0';
    p->device_display_name[0] = '\0';
    p->device_physical_formats = 0;
    p->device_exclusive_formats = 0;
    p->device_min_sample_rate = 0;
    p->device_max_sample_rate = 0;
    p->device_max_bit_depth = 0;
    p->device_channels = 0;
    snprintf(p->dac_format, sizeof(p->dac_format), "%s Mixable %d Hz %dch",
             ma_format_label(out_cfg->ma_format), out_cfg->sample_rate, out_cfg->channels);
    strncpy(p->playback_path, "SharedMiniaudio", sizeof(p->playback_path) - 1);
    strncpy(p->signal_status, out_cfg->exclusive ? "Exclusive Mode" : "Shared Output",
            sizeof(p->signal_status) - 1);

    AV_DEBUG("init device=%s exclusive=%d bitperfect=%d format=%s rate=%d channels=%d",
             (device_name && device_name[0] != '\0') ? device_name : "<default>",
             out_cfg->exclusive,
             out_cfg->bitperfect,
             ma_get_format_name(out_cfg->ma_format),
             out_cfg->sample_rate,
             out_cfg->channels);

    ma_device_config cfg = ma_device_config_init(ma_device_type_playback);
    cfg.playback.format   = out_cfg->ma_format;
    cfg.playback.channels = (ma_uint32)out_cfg->channels;
    cfg.sampleRate        = (ma_uint32)out_cfg->sample_rate;
    cfg.dataCallback      = ma_data_callback;
    cfg.pUserData         = p;
    cfg.periodSizeInMilliseconds = 10;

#if defined(__APPLE__)
    cfg.coreaudio.allowNominalSampleRateChange = out_cfg->exclusive ? MA_TRUE : MA_FALSE;
#else
    if (out_cfg->exclusive) {
        cfg.playback.shareMode = ma_share_mode_exclusive;
    }
#endif

#if defined(_WIN32)
    if (out_cfg->exclusive) {
        cfg.wasapi.noAutoConvertSRC = MA_TRUE;
        cfg.wasapi.noDefaultQualitySRC = MA_TRUE;
        cfg.wasapi.usage = ma_wasapi_usage_pro_audio;
    }
#endif

#if defined(__linux__)
    if (out_cfg->exclusive) {
        cfg.alsa.noAutoFormat = MA_TRUE;
        cfg.alsa.noAutoChannels = MA_TRUE;
        cfg.alsa.noAutoResample = MA_TRUE;
    }
#endif

#if defined(__APPLE__)
    char selected_coreaudio_uid[256] = {0};
    char selected_device_display_name[256] = {0};
#endif

    if (device_name && device_name[0] != '\0') {
        // Find device by name
        ma_device_info *playback_infos;
        ma_uint32        playback_count;
        if (ma_context_get_devices(&p->ma_ctx,
                                    &playback_infos, &playback_count,
                                    NULL, NULL) == MA_SUCCESS) {
            for (ma_uint32 i = 0; i < playback_count; i++) {
                if (strcmp(playback_infos[i].name, device_name) == 0) {
                    cfg.playback.pDeviceID = &playback_infos[i].id;
#if defined(__APPLE__)
                    strncpy(selected_coreaudio_uid, playback_infos[i].id.coreaudio, sizeof(selected_coreaudio_uid) - 1);
                    strncpy(selected_device_display_name, playback_infos[i].name, sizeof(selected_device_display_name) - 1);
#endif
                    break;
                }
            }
        }
    }

#if defined(__APPLE__)
    if (out_cfg->bitperfect && out_cfg->bitperfect_candidate) {
        if (player_init_coreaudio_ioproc(p,
                                         selected_coreaudio_uid[0] ? selected_coreaudio_uid : NULL,
                                         selected_device_display_name[0] ? selected_device_display_name : device_name,
                                         out_cfg) != 0) {
            AV_DEBUG("CoreAudio IOProc unavailable: %s", p->bitperfect_reason);
            return -1;
        }

        p->output_format = out_cfg->ma_format;
        p->output_sample_rate = out_cfg->sample_rate;
        p->output_channels = out_cfg->channels;
        p->output_av_format = out_cfg->av_format;
        p->output_bit_depth = out_cfg->bit_depth > 0 ? out_cfg->bit_depth : ma_format_bit_depth(out_cfg->ma_format);
        return 0;
    }
#endif

    if (ma_device_init(&p->ma_ctx, &cfg, &p->device) != MA_SUCCESS) {
        return -1;
    }
    p->device_init = 1;

#if defined(__APPLE__)
    const char *active_uid = selected_coreaudio_uid[0] ? selected_coreaudio_uid : p->device.playback.id.coreaudio;
    if (active_uid && active_uid[0]) {
        strncpy(p->device_uid, active_uid, sizeof(p->device_uid) - 1);
    }
    if (selected_device_display_name[0]) {
        strncpy(p->device_display_name, selected_device_display_name, sizeof(p->device_display_name) - 1);
    } else if (device_name && device_name[0]) {
        strncpy(p->device_display_name, device_name, sizeof(p->device_display_name) - 1);
    } else {
        strncpy(p->device_display_name, "Default Output", sizeof(p->device_display_name) - 1);
    }
    AudioObjectID active_device = kAudioObjectUnknown;
    if (coreaudio_device_for_uid(active_uid, &active_device) == 0) {
        av_device_info_t caps;
        memset(&caps, 0, sizeof(caps));
        coreaudio_fill_device_caps(active_device, &caps);
        strncpy(p->device_transport, caps.transport, sizeof(p->device_transport) - 1);
        p->device_physical_formats = caps.physical_format_count;
        p->device_exclusive_formats = caps.exclusive_format_count;
        p->device_min_sample_rate = caps.min_sample_rate;
        p->device_max_sample_rate = caps.max_sample_rate;
        p->device_max_bit_depth = caps.max_bit_depth;
        p->device_channels = caps.channels;
        if (caps.active_format[0]) {
            strncpy(p->dac_format, caps.active_format, sizeof(p->dac_format) - 1);
        }
    }
    if (out_cfg->exclusive) {
        const char *uid = active_uid;
        AudioObjectID hog_device = kAudioObjectUnknown;
        if (coreaudio_device_for_uid(uid, &hog_device) == 0) {
            if (coreaudio_set_system_volume_full(hog_device) == 0) {
                AV_DEBUG("CoreAudio system volume set to 100%% for uid=%s", uid);
            } else {
                AV_DEBUG("CoreAudio system volume not writable for uid=%s", uid);
            }
            if (coreaudio_set_hog_mode(hog_device, 1) == 0) {
                p->coreaudio_hog_device = hog_device;
                p->exclusive_active = 1;
                AV_DEBUG("CoreAudio hog mode active for uid=%s", uid);
            } else {
                p->exclusive_active = 0;
                AV_DEBUG("CoreAudio hog mode unavailable for uid=%s", uid);
            }
        } else {
            p->exclusive_active = 0;
            AV_DEBUG("CoreAudio device lookup failed for uid=%s", uid ? uid : "<unknown>");
        }
    } else {
        p->exclusive_active = 0;
    }
#else
    p->exclusive_active = out_cfg->exclusive ? 1 : 0;
#endif

    p->output_format = out_cfg->ma_format;
    p->output_sample_rate = out_cfg->sample_rate;
    p->output_channels = out_cfg->channels;
    p->output_av_format = out_cfg->av_format;
    p->output_bit_depth = out_cfg->bit_depth > 0 ? out_cfg->bit_depth : ma_format_bit_depth(out_cfg->ma_format);

    if (ma_device_start(&p->device) != MA_SUCCESS) {
        player_release_exclusive(p);
        ma_device_uninit(&p->device);
        p->device_init = 0;
        return -1;
    }
    AV_DEBUG("device started exclusive_active=%d output=%s/%dHz/%dch",
             p->exclusive_active,
             ma_get_format_name(p->output_format),
             p->output_sample_rate,
             p->output_channels);
    return 0;
}

static int output_config_matches_current(const av_player_t *p, const output_config_t *cfg) {
    return p->output_sample_rate == cfg->sample_rate &&
           p->output_channels == cfg->channels &&
           p->output_format == cfg->ma_format &&
           (p->exclusive_requested || p->bitperfect_requested) == cfg->exclusive &&
           p->bitperfect_requested == cfg->bitperfect &&
           p->dop_active == cfg->dop;
}

static output_config_t current_output_config(const av_player_t *p) {
    output_config_t cfg;
    memset(&cfg, 0, sizeof(cfg));
    cfg.sample_rate = p->output_sample_rate > 0 ? p->output_sample_rate : AVPLAYER_DEFAULT_SAMPLE_RATE;
    cfg.channels = p->output_channels > 0 ? p->output_channels : AVPLAYER_DEFAULT_CHANNELS;
    cfg.ma_format = p->output_format != ma_format_unknown ? p->output_format : ma_format_f32;
    cfg.av_format = p->output_av_format != AV_SAMPLE_FMT_NONE ? p->output_av_format : av_sample_fmt_from_ma_format(cfg.ma_format);
    cfg.bit_depth = p->output_bit_depth > 0 ? p->output_bit_depth : ma_format_bit_depth(cfg.ma_format);
    cfg.exclusive = p->exclusive_requested || p->bitperfect_requested;
    cfg.bitperfect = p->bitperfect_requested;
    cfg.bitperfect_candidate = p->bitperfect_active;
    cfg.dop = p->dop_active;
    cfg.dop_carrier_rate = p->dop_carrier_rate;
    cfg.dsd_bit_rate = p->dsd_rate;
    strncpy(cfg.reason, p->bitperfect_reason, sizeof(cfg.reason) - 1);
    return cfg;
}

static void output_config_make_app_pcm_fallback(output_config_t *cfg, const char *reason) {
    if (!cfg) return;
    cfg->sample_rate = AVPLAYER_DEFAULT_SAMPLE_RATE;
    cfg->channels = AVPLAYER_DEFAULT_CHANNELS;
    cfg->av_format = AV_SAMPLE_FMT_FLT;
    cfg->ma_format = ma_format_f32;
    cfg->bit_depth = 32;
    cfg->bitperfect_candidate = 0;
    cfg->dop = 0;
    cfg->dsd_bit_rate = 0;
    cfg->dop_carrier_rate = 0;
    snprintf(cfg->reason, sizeof(cfg->reason), "%s",
             reason && reason[0] ? reason : "bit-perfect output unavailable");
}

static int player_apply_output_config(av_player_t *p, output_config_t *cfg) {
    if (ring_reinit(&p->ring, cfg->sample_rate, cfg->channels, cfg->ma_format) != 0) {
        p->bitperfect_active = 0;
        set_bitperfect_reason(p, "failed to allocate output buffer");
        return -1;
    }

    if (player_init_device(p, p->device_name, cfg) != 0) {
        if (cfg->bitperfect && cfg->bitperfect_candidate) {
            char fallback_reason[sizeof(p->bitperfect_reason)];
            snprintf(fallback_reason, sizeof(fallback_reason), "%s",
                     p->bitperfect_reason[0] ? p->bitperfect_reason : "bit-perfect output unavailable");
            AV_DEBUG("bit-perfect output unavailable; falling back to app PCM: %s",
                     fallback_reason);
            output_config_make_app_pcm_fallback(cfg, fallback_reason);
            if (ring_reinit(&p->ring, cfg->sample_rate, cfg->channels, cfg->ma_format) != 0) {
                p->bitperfect_active = 0;
                set_bitperfect_reason(p, "failed to allocate fallback output buffer");
                return -1;
            }
            if (player_init_device(p, p->device_name, cfg) == 0) {
                goto device_ready;
            }
        }
        p->bitperfect_active = 0;
        if (p->bitperfect_reason[0] == '\0') {
            set_bitperfect_reason(p, cfg->exclusive ? "exclusive output device unavailable" : "output device unavailable");
        }
        return -1;
    }

device_ready:
    if (!cfg->bitperfect) {
        p->bitperfect_active = 0;
        p->dop_active = 0;
        set_bitperfect_reason(p, cfg->exclusive ? "bit-perfect off" : "exclusive mode off");
        strncpy(p->signal_status, cfg->exclusive ? "Exclusive Mode" : "Shared Output",
                sizeof(p->signal_status) - 1);
    } else if (!cfg->bitperfect_candidate) {
        p->bitperfect_active = 0;
        p->dop_active = 0;
        set_bitperfect_reason(p, cfg->reason);
        strncpy(p->signal_status, "Not Bit-Perfect", sizeof(p->signal_status) - 1);
    } else if (!p->exclusive_active) {
        p->bitperfect_active = 0;
        p->dop_active = 0;
        set_bitperfect_reason(p, "exclusive/hog mode unavailable");
        strncpy(p->signal_status, "Not Bit-Perfect", sizeof(p->signal_status) - 1);
    } else {
        p->bitperfect_active = 1;
        p->dop_active = cfg->dop ? 1 : 0;
        if (p->dop_active) {
            p->dsd_rate = cfg->dsd_bit_rate;
            p->dop_carrier_rate = cfg->dop_carrier_rate;
        }
        set_bitperfect_reason(p, "");
        strncpy(p->signal_status, cfg->dop ? "Bit-Perfect DoP" : "Bit-Perfect",
                sizeof(p->signal_status) - 1);
    }

    AV_DEBUG("output config applied exclusive_requested=%d exclusive_active=%d bitperfect_requested=%d bitperfect_active=%d reason=%s",
             p->exclusive_requested,
             p->exclusive_active,
             p->bitperfect_requested,
             p->bitperfect_active,
             p->bitperfect_reason);

    return 0;
}

static void player_refresh_media_info(av_player_t *p, const decoder_t *d) {
    memset(&p->media_info, 0, sizeof(p->media_info));
    p->source_is_dsd = 0;
    p->dsd_rate = 0;
    p->dop_carrier_rate = 0;
    if (d && d->codec_ctx && d->codec_ctx->codec) {
        strncpy(p->media_info.codec,
                d->codec_ctx->codec->name,
                sizeof(p->media_info.codec) - 1);
        int channels = decoder_channel_count(d);
        p->media_info.channels = channels;
        int src_bits = source_bit_depth_from_decoder(d);
        p->source_is_dsd = decoder_is_dsd(d);
        p->dsd_rate = p->source_is_dsd ? decoder_dsd_bit_rate(d) : 0;
        p->dop_carrier_rate = p->source_is_dsd ? decoder_dop_carrier_rate(d) : 0;
        if (d->wv_dsd && d->wv_average_bitrate > 0) {
            p->media_info.bitrate = d->wv_average_bitrate;
        }
        if (p->source_is_dsd) {
            strncpy(p->media_info.sample_format, "dsd", sizeof(p->media_info.sample_format) - 1);
            p->media_info.sample_rate = p->dsd_rate;
        } else {
            const char *sample_fmt = av_get_sample_fmt_name(d->codec_ctx->sample_fmt);
            if (sample_fmt) {
                strncpy(p->media_info.sample_format,
                        sample_fmt,
                        sizeof(p->media_info.sample_format) - 1);
            }
            p->media_info.sample_rate = d->codec_ctx->sample_rate;
        }
        if (p->source_is_dsd) {
            const char *dsd_label = d->wv_dsd ? "WavPack DSD" : "DSD";
            snprintf(p->media_info.source_format, sizeof(p->media_info.source_format),
                     "%s %.4f MHz / %dch",
                     dsd_label,
                     p->dsd_rate > 0 ? (double)p->dsd_rate / 1000000.0 : 0.0,
                     channels);
            if (d->wv_dsd) {
                snprintf(p->media_info.decode_path, sizeof(p->media_info.decode_path),
                         "libwavpack native DSD -> %s",
                         p->dop_active ? "DoP carrier" : "PCM fallback");
            } else {
                snprintf(p->media_info.decode_path, sizeof(p->media_info.decode_path),
                         "%s -> %s",
                         d->codec_ctx->codec->name,
                         p->dop_active ? "DoP carrier" : "PCM fallback");
            }
        } else {
            snprintf(p->media_info.source_format, sizeof(p->media_info.source_format),
                     "%.1f kHz / %d-bit / %dch",
                     d->codec_ctx->sample_rate > 0 ? (double)d->codec_ctx->sample_rate / 1000.0 : 0.0,
                     src_bits,
                     channels);
            snprintf(p->media_info.decode_path, sizeof(p->media_info.decode_path),
                     "%s -> %s PCM",
                     d->codec_ctx->codec->name,
                     p->bitperfect_active ? "Integer" : "App");
        }
    }
    strncpy(p->media_info.output_format, ma_format_label(p->output_format),
            sizeof(p->media_info.output_format) - 1);
    p->media_info.output_sample_rate = p->output_sample_rate;
    p->media_info.output_channels = p->output_channels;
    if (p->dop_active) {
        snprintf(p->media_info.output_path, sizeof(p->media_info.output_path),
                 "DoP carrier / %.1f kHz / 32-bit / %dch",
                 p->output_sample_rate > 0 ? (double)p->output_sample_rate / 1000.0 : 0.0,
                 p->output_channels);
    } else {
        snprintf(p->media_info.output_path, sizeof(p->media_info.output_path),
                 "%s / %.1f kHz / %dch",
                 p->output_format == ma_format_f32 ? "Float PCM" : "Integer PCM",
                 p->output_sample_rate > 0 ? (double)p->output_sample_rate / 1000.0 : 0.0,
                 p->output_channels);
    }
    p->media_info.exclusive_requested = p->exclusive_requested;
    p->media_info.exclusive_active = p->exclusive_active;
    p->media_info.bitperfect_requested = p->bitperfect_requested;
    p->media_info.bitperfect_active = p->bitperfect_active;
    strncpy(p->media_info.bitperfect_reason, p->bitperfect_reason,
            sizeof(p->media_info.bitperfect_reason) - 1);
    strncpy(p->media_info.playback_path, p->playback_path,
            sizeof(p->media_info.playback_path) - 1);
    strncpy(p->media_info.signal_status, p->signal_status,
            sizeof(p->media_info.signal_status) - 1);
    strncpy(p->media_info.device_name, p->device_display_name,
            sizeof(p->media_info.device_name) - 1);
    strncpy(p->media_info.device_uid, p->device_uid,
            sizeof(p->media_info.device_uid) - 1);
    strncpy(p->media_info.device_transport, p->device_transport,
            sizeof(p->media_info.device_transport) - 1);
    strncpy(p->media_info.dac_format, p->dac_format,
            sizeof(p->media_info.dac_format) - 1);
    p->media_info.output_mixable = p->output_mixable;
    p->media_info.device_physical_formats = p->device_physical_formats;
    p->media_info.device_exclusive_formats = p->device_exclusive_formats;
    p->media_info.device_min_sample_rate = p->device_min_sample_rate;
    p->media_info.device_max_sample_rate = p->device_max_sample_rate;
    p->media_info.device_max_bit_depth = p->device_max_bit_depth;
    p->media_info.device_channels = p->device_channels;
    p->media_info.source_is_dsd = p->source_is_dsd;
    p->media_info.dsd_rate = p->dsd_rate;
    p->media_info.dop_carrier_rate = p->dop_carrier_rate;
}

// --------------------------------------------------------------------------
// Public API implementation
// --------------------------------------------------------------------------

av_player_t *av_player_create(void) {
    av_player_t *p = (av_player_t *)calloc(1, sizeof(av_player_t));
    if (!p) return NULL;
    if (ring_init(&p->ring, AVPLAYER_DEFAULT_SAMPLE_RATE, AVPLAYER_DEFAULT_CHANNELS, ma_format_f32) != 0) {
        free(p);
        return NULL;
    }
    p->volume = 1.0f;
    p->output_format = ma_format_f32;
    p->output_sample_rate = AVPLAYER_DEFAULT_SAMPLE_RATE;
    p->output_channels = AVPLAYER_DEFAULT_CHANNELS;
    p->output_av_format = AV_SAMPLE_FMT_FLT;
    p->output_bit_depth = 32;
    p->output_mixable = 1;
    p->dop_active = 0;
    p->dop_marker_phase = 0;
    strncpy(p->playback_path, "SharedMiniaudio", sizeof(p->playback_path) - 1);
    strncpy(p->signal_status, "Shared Output", sizeof(p->signal_status) - 1);
    strncpy(p->dac_format, "f32 Mixable 48000 Hz 2ch", sizeof(p->dac_format) - 1);
    set_bitperfect_reason(p, "exclusive mode off");
#if defined(__APPLE__)
    p->coreaudio_hog_device = kAudioObjectUnknown;
    p->coreaudio_io_device = kAudioObjectUnknown;
    p->coreaudio_io_stream = kAudioObjectUnknown;
    p->coreaudio_io_proc = NULL;
    p->coreaudio_original_format_valid = 0;
    p->coreaudio_io_active = 0;
#endif
    atomic_store(&p->state, AVPLAYER_STATE_STOPPED);
    atomic_store(&p->l_peak, -INFINITY);
    atomic_store(&p->r_peak, -INFINITY);
    atomic_store(&p->l_rms,  -INFINITY);
    atomic_store(&p->r_rms,  -INFINITY);
    ma_mutex_init(&p->decoder_lock);
    ma_mutex_init(&p->filter_lock);
    return p;
}

int av_player_init(av_player_t *p, const char *device_name, int exclusive) {
    output_config_t out_cfg;
    memset(&out_cfg, 0, sizeof(out_cfg));
    out_cfg.sample_rate = AVPLAYER_DEFAULT_SAMPLE_RATE;
    out_cfg.channels = AVPLAYER_DEFAULT_CHANNELS;
    out_cfg.av_format = AV_SAMPLE_FMT_FLT;
    out_cfg.ma_format = ma_format_f32;
    out_cfg.bit_depth = 32;
    out_cfg.exclusive = exclusive ? 1 : 0;
    out_cfg.bitperfect = 0;
    out_cfg.bitperfect_candidate = 0;

    p->exclusive_requested = exclusive ? 1 : 0;
    if (device_name && device_name[0] != '\0') {
        strncpy(p->device_name, device_name, sizeof(p->device_name) - 1);
    } else {
        p->device_name[0] = '\0';
    }

    if (ma_context_init(NULL, 0, NULL, &p->ma_ctx) != MA_SUCCESS) {
        return -1;
    }
    p->ma_ctx_init = 1;
    return player_init_device(p, device_name, &out_cfg);
}

void av_player_destroy(av_player_t *p) {
    if (!p) return;
    atomic_store(&p->state, AVPLAYER_STATE_STOPPED);
    player_release_exclusive(p);
    if (p->device_init) {
        ma_device_uninit(&p->device);
    }
    if (p->ma_ctx_init) {
        ma_context_uninit(&p->ma_ctx);
    }
    decoder_free(p->dec);
    decoder_free(p->next_dec);
    ring_free(&p->ring);
    ma_mutex_uninit(&p->decoder_lock);
    ma_mutex_uninit(&p->filter_lock);
    // Clean up EQ banks
    for (int b = 0; b < 2; b++) {
        if (p->eq_banks[b].initialized) {
            for (int i = 0; i < p->eq_banks[b].num_bands; i++)
                ma_peak2_uninit(&p->eq_banks[b].filters[i], NULL);
        }
    }
    free(p);
}

int av_player_open(av_player_t *p, const char *url, double start_time)
{
    // Stop current playback
    atomic_store(&p->state, AVPLAYER_STATE_STOPPED);
    ring_clear(&p->ring);
    atomic_store(&p->eof_reached, 0);
    atomic_store(&p->next_consumed, 0);
    atomic_store(&p->pending_track_change, 0);
    atomic_store_explicit((_Atomic long long *)&p->frames_played_total, 0LL, memory_order_relaxed);
    atomic_store_explicit((_Atomic long long *)&p->ring.frames_written_total, 0LL, memory_order_relaxed);
    atomic_store_explicit((_Atomic double *)&p->position_offset, 0.0, memory_order_relaxed);
    atomic_store_explicit((_Atomic long long *)&p->position_clock_ref, 0LL, memory_order_relaxed);
    memset(&p->bitrate_hist, 0, sizeof(p->bitrate_hist));

    // Acquire decoder_lock to prevent racing with av_player_open_next which
    // also reads/frees next_dec under this lock.
    ma_mutex_lock(&p->decoder_lock);
    decoder_t *old_next = p->next_dec;
    p->next_dec = NULL;
    ma_mutex_unlock(&p->decoder_lock);
    decoder_free(old_next);

    decoder_free(p->dec);
    p->dec = NULL;

    p->last_icy_title[0] = '\0';

    decoder_t *d = decoder_alloc();
    if (!d) return AVERROR(ENOMEM);

    int ret = decoder_open(d, url);
    if (ret < 0) { decoder_free(d); return ret; }

    output_config_t out_cfg;
    choose_output_config(p, d, &out_cfg);
    if (player_apply_output_config(p, &out_cfg) != 0) {
        decoder_free(d);
        return -1;
    }

    if (out_cfg.dop) {
        decoder_clear_dop_state(d);
        p->dop_marker_phase = 0;
    } else {
        ret = decoder_build_filter_graph(d, &out_cfg);
        if (ret < 0) { decoder_free(d); return ret; }
    }

    player_refresh_media_info(p, d);

    atomic_store_explicit((_Atomic double *)&p->duration, d->duration, memory_order_relaxed);
    atomic_store_explicit((_Atomic double *)&p->time_pos, 0.0, memory_order_relaxed);

    p->dec = d;

    // Compute and apply RG gain from the newly opened track's tags.
    atomic_store(&p->rg_switch_pending, 0);
    atomic_store_explicit(&p->rg_gain,
        compute_rg_gain(d, p->rg_mode, p->rg_prevent_clip, p->rg_preamp_db),
        memory_order_relaxed);

    if (start_time > 0.0) {
        av_player_seek(p, start_time);
    }

    atomic_store(&p->state, AVPLAYER_STATE_PLAYING);
    return 0;
}

int av_player_open_next(av_player_t *p, const char *url)
{
    // Atomically take ownership of the old next decoder so that the decode
    // goroutine cannot simultaneously free it during a gapless swap.
    ma_mutex_lock(&p->decoder_lock);
    decoder_t *old_next = p->next_dec;
    p->next_dec = NULL;
    ma_mutex_unlock(&p->decoder_lock);

    decoder_free(old_next);  // free outside the lock — can be slow
    atomic_store(&p->next_consumed, 0);

    if (!url || url[0] == '\0') return 0;

    decoder_t *d = decoder_alloc();
    if (!d) return AVERROR(ENOMEM);

    int ret = decoder_open(d, url);
    if (ret < 0) { decoder_free(d); return ret; }

    // Publish the new decoder atomically.
    ma_mutex_lock(&p->decoder_lock);
    p->next_dec = d;
    ma_mutex_unlock(&p->decoder_lock);
    return 0;
}

void av_player_stop(av_player_t *p) {
    atomic_store(&p->state, AVPLAYER_STATE_STOPPED);
    ring_clear(&p->ring);
    player_release_exclusive(p);
    atomic_store(&p->eof_reached, 0);
    atomic_store(&p->pending_track_change, 0);
    decoder_free(p->dec);   p->dec      = NULL;

    ma_mutex_lock(&p->decoder_lock);
    decoder_t *old_next = p->next_dec;
    p->next_dec = NULL;
    ma_mutex_unlock(&p->decoder_lock);
    decoder_free(old_next);
    atomic_store_explicit((_Atomic double *)&p->time_pos, 0.0, memory_order_relaxed);
    atomic_store_explicit((_Atomic double *)&p->duration,  0.0, memory_order_relaxed);
}

void av_player_pause(av_player_t *p) {
    if (atomic_load(&p->state) == AVPLAYER_STATE_PLAYING) {
        atomic_store(&p->state, AVPLAYER_STATE_PAUSED);
    }
}

void av_player_resume(av_player_t *p) {
    if (atomic_load(&p->state) == AVPLAYER_STATE_PAUSED) {
        atomic_store(&p->state, AVPLAYER_STATE_PLAYING);
    }
}

int av_player_seek(av_player_t *p, double seconds) {
    if (!p->dec) return -1;
    if (p->dop_active && p->dec->wv_dsd && p->dec->wv_ctx) {
        int64_t sample = (int64_t)(seconds * (double)p->dec->wv_dsd_byte_rate);
        if (!WavpackSeekSample64(p->dec->wv_ctx, sample)) return -1;
        p->dec->wv_samples_unpacked = sample;
        decoder_clear_dop_state(p->dec);
        p->dop_marker_phase = 0;
        ring_clear(&p->ring);
        atomic_store(&p->eof_reached, 0);
        atomic_store(&p->pending_track_change, 0);
        long long played_now = atomic_load_explicit(
            (_Atomic long long *)&p->frames_played_total, memory_order_relaxed);
        atomic_store_explicit((_Atomic double *)&p->position_offset, seconds, memory_order_relaxed);
        atomic_store_explicit((_Atomic long long *)&p->position_clock_ref, played_now, memory_order_relaxed);
        atomic_store_explicit((_Atomic long long *)&p->ring.frames_written_total,
                              played_now, memory_order_relaxed);
        memset(&p->bitrate_hist, 0, sizeof(p->bitrate_hist));
        atomic_store_explicit((_Atomic double *)&p->time_pos, seconds, memory_order_relaxed);
        return 0;
    }

    AVFormatContext *fc = p->dec->fmt_ctx;

    int64_t ts = (int64_t)(seconds * AV_TIME_BASE);
    int ret = avformat_seek_file(fc, -1, INT64_MIN, ts, INT64_MAX, 0);
    if (ret < 0) return ret;

    avcodec_flush_buffers(p->dec->codec_ctx);

    if (p->dop_active) {
        decoder_clear_dop_state(p->dec);
        p->dop_marker_phase = 0;
    } else if (p->dec->buffersrc_ctx && p->dec->buffersink_ctx) {
        // Flush filter graph by draining and sending NULL frame.
        (void)av_buffersrc_add_frame_flags(p->dec->buffersrc_ctx, NULL, AV_BUFFERSRC_FLAG_PUSH);
        av_frame_unref(p->dec->filt_frame);
        while (av_buffersink_get_frame(p->dec->buffersink_ctx, p->dec->filt_frame) >= 0) {
            av_frame_unref(p->dec->filt_frame);
        }
        // Reinit the filter graph to avoid corruption.
        output_config_t out_cfg = current_output_config(p);
        decoder_build_filter_graph(p->dec, &out_cfg);
    }

    // Discard any pending frame from before the seek.
    av_frame_free(&p->dec->pending_frame);

    ring_clear(&p->ring);
    atomic_store(&p->eof_reached, 0);
    atomic_store(&p->pending_track_change, 0);
    // Reset position clock to the seek target.
    long long played_now = atomic_load_explicit(
        (_Atomic long long *)&p->frames_played_total, memory_order_relaxed);
    atomic_store_explicit((_Atomic double *)&p->position_offset, seconds, memory_order_relaxed);
    atomic_store_explicit((_Atomic long long *)&p->position_clock_ref, played_now, memory_order_relaxed);
    // Realign frames_written_total to frames_played_total so post-seek bitrate
    // history entries (frame_pos >= played_now) are immediately visible to the
    // lookup in av_player_get_media_info without waiting for the gap to drain.
    atomic_store_explicit((_Atomic long long *)&p->ring.frames_written_total,
                          played_now, memory_order_relaxed);
    memset(&p->bitrate_hist, 0, sizeof(p->bitrate_hist));
    atomic_store(&p->rg_switch_pending, 0);
    return 0;
}

void av_player_set_volume(av_player_t *p, float volume) {
    p->volume = volume;
}

void av_player_set_replay_gain(av_player_t *p, int mode, int prevent_clip, double preamp_db) {
    p->rg_mode         = mode;
    p->rg_prevent_clip = prevent_clip;
    p->rg_preamp_db    = preamp_db;
    // Clear any pending track-boundary switch; mode change applies immediately.
    atomic_store(&p->rg_switch_pending, 0);
    atomic_store_explicit(&p->rg_gain,
        compute_rg_gain(p->dec, mode, prevent_clip, preamp_db),
        memory_order_release);
}

void av_player_set_eq(av_player_t *p, const av_eq_band_t *bands, int num_bands, double preamp_db) {
    if (num_bands > AVPLAYER_MAX_EQ_BANDS)
        num_bands = AVPLAYER_MAX_EQ_BANDS;

    int active = atomic_load_explicit(&p->eq_active_idx, memory_order_relaxed);
    int target = 1 - active;
    eq_bank_t *bank = &p->eq_banks[target];

    // Uninit stale filters in the target bank (deferred cleanup from previous swap)
    if (bank->initialized) {
        for (int i = 0; i < bank->num_bands; i++)
            ma_peak2_uninit(&bank->filters[i], NULL);
        bank->initialized = 0;
        bank->num_bands = 0;
    }

    bank->preamp = (preamp_db != 0.0) ? (float)pow(10.0, preamp_db / 20.0) : 1.0f;

    if (num_bands == 0 || !bands) {
        bank->num_bands = 0;
        atomic_store_explicit(&p->eq_active_idx, target, memory_order_release);
        return;
    }

    for (int i = 0; i < num_bands; i++) {
        ma_peak2_config cfg = ma_peak2_config_init(
            ma_format_f32, (ma_uint32)p->output_channels, (ma_uint32)p->output_sample_rate,
            bands[i].gain_db, bands[i].q, bands[i].frequency);
        if (ma_peak2_init(&cfg, NULL, &bank->filters[i]) != MA_SUCCESS) {
            // Roll back filters already initialised in this bank
            for (int j = 0; j < i; j++)
                ma_peak2_uninit(&bank->filters[j], NULL);
            bank->num_bands = 0;
            return;  // keep old EQ active
        }
    }
    bank->num_bands = num_bands;
    bank->initialized = 1;

    atomic_store_explicit(&p->eq_active_idx, target, memory_order_release);
}

int av_player_get_state(av_player_t *p) {
    return atomic_load(&p->state);
}

double av_player_get_position(av_player_t *p) {
    double offset = atomic_load_explicit((_Atomic double *)&p->position_offset, memory_order_relaxed);
    long long ref  = atomic_load_explicit((_Atomic long long *)&p->position_clock_ref, memory_order_relaxed);
    long long played = atomic_load_explicit((_Atomic long long *)&p->frames_played_total, memory_order_relaxed);
    int sample_rate = p->output_sample_rate > 0 ? p->output_sample_rate : AVPLAYER_DEFAULT_SAMPLE_RATE;
    double result = offset + (double)(played - ref) / (double)sample_rate;
    return result < 0.0 ? 0.0 : result;
}

double av_player_get_duration(av_player_t *p) {
    return atomic_load_explicit((_Atomic double *)&p->duration, memory_order_relaxed);
}

int av_player_buffered_frames(av_player_t *p) {
    return ring_avail(&p->ring);
}

void av_player_get_peaks(av_player_t *p,
                         double *l_peak, double *r_peak,
                         double *l_rms,  double *r_rms)
{
    *l_peak = atomic_load_explicit((_Atomic double *)&p->l_peak, memory_order_relaxed);
    *r_peak = atomic_load_explicit((_Atomic double *)&p->r_peak, memory_order_relaxed);
    *l_rms  = atomic_load_explicit((_Atomic double *)&p->l_rms,  memory_order_relaxed);
    *r_rms  = atomic_load_explicit((_Atomic double *)&p->r_rms,  memory_order_relaxed);
}

void av_player_set_peaks_enabled(av_player_t *p, int enabled) {
    p->peaks_enabled = enabled;
    if (!enabled) {
        atomic_store_explicit((_Atomic double *)&p->l_peak, -INFINITY, memory_order_relaxed);
        atomic_store_explicit((_Atomic double *)&p->r_peak, -INFINITY, memory_order_relaxed);
        atomic_store_explicit((_Atomic double *)&p->l_rms,  -INFINITY, memory_order_relaxed);
        atomic_store_explicit((_Atomic double *)&p->r_rms,  -INFINITY, memory_order_relaxed);
    }
}

// Check whether the ICY StreamTitle in the current decoder's metadata has changed
// since the last call.  If it has, copies the new title into buf (up to buflen-1
// bytes, NUL-terminated) and returns 1.  Returns 0 if unchanged or unavailable.
// Only call from the decode goroutine.
int av_player_check_icy_title(av_player_t *p, char *buf, int buflen) {
    if (buflen <= 0) return 0;
    const char *title = "";
    if (p->dec && p->dec->fmt_ctx) {
        AVDictionaryEntry *e = av_dict_get(p->dec->fmt_ctx->metadata, "StreamTitle", NULL, 0);
        if (e && e->value) title = e->value;
    }
    if (strcmp(title, p->last_icy_title) != 0) {
        strncpy(p->last_icy_title, title, sizeof(p->last_icy_title) - 1);
        p->last_icy_title[sizeof(p->last_icy_title) - 1] = '\0';
        strncpy(buf, title, buflen - 1);
        buf[buflen - 1] = '\0';
        return 1;
    }
    return 0;
}

void av_player_get_media_info(av_player_t *p, av_media_info_t *info) {
    *info = p->media_info;

    // Return the bitrate entry whose frame position is closest to (but not
    // exceeding) frames_played_total, so the displayed bitrate tracks the
    // audio currently being written to the sound card rather than the most
    // recently decoded packet (~ring-buffer-duration ahead).
    long long played = atomic_load_explicit(
        (_Atomic long long *)&p->frames_played_total, memory_order_relaxed);

    bitrate_history_t *h = &p->bitrate_hist;
    int wi = atomic_load_explicit(&h->write_idx, memory_order_acquire);

    int best_bitrate = atomic_load(&p->cur_bitrate); // fallback if history empty
    for (int i = 0; i < BITRATE_HISTORY_SIZE; i++) {
        int idx = (wi - 1 - i + BITRATE_HISTORY_SIZE) % BITRATE_HISTORY_SIZE;
        if (h->entries[idx].frame_pos > 0 && h->entries[idx].frame_pos <= played) {
            best_bitrate = h->entries[idx].bitrate;
            break;
        }
    }

    info->bitrate = best_bitrate;
}

int av_player_list_devices(av_device_info_t *devices, int max_devices) {
    // This function can be called before player init; create a temporary context
    ma_context ctx;
    if (ma_context_init(NULL, 0, NULL, &ctx) != MA_SUCCESS) return 0;

    ma_device_info *playback_infos;
    ma_uint32        playback_count;
    int              count = 0;

    if (ma_context_get_devices(&ctx, &playback_infos, &playback_count, NULL, NULL) == MA_SUCCESS) {
        for (ma_uint32 i = 0; i < playback_count && count < max_devices; i++) {
            strncpy(devices[count].name, playback_infos[i].name, 255);
            strncpy(devices[count].description, playback_infos[i].name, 255);
#if defined(__APPLE__)
            strncpy(devices[count].uid, playback_infos[i].id.coreaudio, sizeof(devices[count].uid) - 1);
            AudioObjectID device_id = kAudioObjectUnknown;
            if (coreaudio_device_for_uid(playback_infos[i].id.coreaudio, &device_id) == 0) {
                coreaudio_fill_device_caps(device_id, &devices[count]);
            }
#endif
            count++;
        }
    }

    ma_context_uninit(&ctx);
    return count;
}

int av_player_set_device(av_player_t *p, const char *device_name) {
    if (device_name && device_name[0] != '\0') {
        strncpy(p->device_name, device_name, sizeof(p->device_name) - 1);
        p->device_name[sizeof(p->device_name) - 1] = '\0';
    } else {
        p->device_name[0] = '\0';
    }

    output_config_t cfg = p->dec ? (output_config_t){0} : current_output_config(p);
    if (p->dec) {
        choose_output_config(p, p->dec, &cfg);
    }
    int ret = player_apply_output_config(p, &cfg);
    if (ret == 0 && p->dec) {
        if (cfg.dop) {
            decoder_clear_dop_state(p->dec);
            p->dop_marker_phase = 0;
        } else {
            ret = decoder_build_filter_graph(p->dec, &cfg);
        }
    }
    player_refresh_media_info(p, p->dec);
    return ret;
}

int av_player_set_exclusive(av_player_t *p, int exclusive) {
    AV_DEBUG("set exclusive=%d previous_exclusive=%d bitperfect_requested=%d",
             exclusive ? 1 : 0,
             p->exclusive_requested,
             p->bitperfect_requested);
    p->exclusive_requested = exclusive ? 1 : 0;
    if (!p->exclusive_requested) {
        p->bitperfect_requested = 0;
    }

    output_config_t cfg;
    if (p->dec) {
        choose_output_config(p, p->dec, &cfg);
    } else {
        memset(&cfg, 0, sizeof(cfg));
        cfg.sample_rate = AVPLAYER_DEFAULT_SAMPLE_RATE;
        cfg.channels = AVPLAYER_DEFAULT_CHANNELS;
        cfg.av_format = AV_SAMPLE_FMT_FLT;
        cfg.ma_format = ma_format_f32;
        cfg.exclusive = p->exclusive_requested || p->bitperfect_requested;
        cfg.bitperfect = p->bitperfect_requested;
        cfg.bitperfect_candidate = 0;
        strncpy(cfg.reason,
                cfg.bitperfect ? "no active stream" : (cfg.exclusive ? "bit-perfect off" : "exclusive mode off"),
                sizeof(cfg.reason) - 1);
    }

    int ret = player_apply_output_config(p, &cfg);
    if (ret == 0 && p->dec) {
        if (cfg.dop) {
            decoder_clear_dop_state(p->dec);
            p->dop_marker_phase = 0;
        } else {
            ret = decoder_build_filter_graph(p->dec, &cfg);
        }
    }
    player_refresh_media_info(p, p->dec);
    return ret;
}

int av_player_set_bitperfect(av_player_t *p, int bitperfect) {
    AV_DEBUG("set bitperfect=%d previous_bitperfect=%d exclusive_requested=%d",
             bitperfect ? 1 : 0,
             p->bitperfect_requested,
             p->exclusive_requested);
    p->bitperfect_requested = bitperfect ? 1 : 0;
    if (p->bitperfect_requested) {
        p->exclusive_requested = 1;
    }

    output_config_t cfg;
    if (p->dec) {
        choose_output_config(p, p->dec, &cfg);
    } else {
        memset(&cfg, 0, sizeof(cfg));
        cfg.sample_rate = AVPLAYER_DEFAULT_SAMPLE_RATE;
        cfg.channels = AVPLAYER_DEFAULT_CHANNELS;
        cfg.av_format = AV_SAMPLE_FMT_FLT;
        cfg.ma_format = ma_format_f32;
        cfg.exclusive = p->exclusive_requested || p->bitperfect_requested;
        cfg.bitperfect = p->bitperfect_requested;
        cfg.bitperfect_candidate = 0;
        strncpy(cfg.reason,
                cfg.bitperfect ? "no active stream" : (cfg.exclusive ? "bit-perfect off" : "exclusive mode off"),
                sizeof(cfg.reason) - 1);
    }

    int ret = player_apply_output_config(p, &cfg);
    if (ret == 0 && p->dec) {
        if (cfg.dop) {
            decoder_clear_dop_state(p->dec);
            p->dop_marker_phase = 0;
        } else {
            ret = decoder_build_filter_graph(p->dec, &cfg);
        }
    }
    player_refresh_media_info(p, p->dec);
    return ret;
}

// --------------------------------------------------------------------------
// Decode step — called from Go goroutine in a tight loop
// --------------------------------------------------------------------------

static uint8_t reverse_bits8(uint8_t v) {
    v = (uint8_t)(((v & 0xf0) >> 4) | ((v & 0x0f) << 4));
    v = (uint8_t)(((v & 0xcc) >> 2) | ((v & 0x33) << 2));
    v = (uint8_t)(((v & 0xaa) >> 1) | ((v & 0x55) << 1));
    return v;
}

static uint8_t dop_dsd_byte(const decoder_t *d, uint8_t value) {
    return decoder_dsd_is_lsb_first(d) ? value : reverse_bits8(value);
}

static uint8_t dop_packet_byte(const decoder_t *d,
                               const AVPacket *pkt,
                               int channel,
                               int byte_index,
                               int channels,
                               int per_channel_bytes) {
    if (decoder_dsd_is_planar(d)) {
        return pkt->data[channel * per_channel_bytes + byte_index];
    }
    return pkt->data[byte_index * channels + channel];
}

static int dop_packet_output_frames(const decoder_t *d, const AVPacket *pkt, int channels) {
    if (!d || !pkt || channels <= 0 || channels > AVPLAYER_MAX_DOP_CHANNELS) return -1;
    if (pkt->size <= 0 || (pkt->size % channels) != 0) return -1;
    int per_channel_bytes = pkt->size / channels;
    int frames = INT32_MAX;
    for (int ch = 0; ch < channels; ch++) {
        int total = per_channel_bytes + (d->dop_pending_valid[ch] ? 1 : 0);
        int ch_frames = total / 2;
        if (ch_frames < frames) frames = ch_frames;
    }
    return frames == INT32_MAX ? 0 : frames;
}

static int32_t dop_pack_s32(uint8_t marker, uint8_t dsd0, uint8_t dsd1) {
    // 32-bit DoP: zero padding in the low byte, 16 DSD bits, marker in the MSB.
    return (int32_t)(((uint32_t)marker << 24) |
                     ((uint32_t)dsd1 << 16) |
                     ((uint32_t)dsd0 << 8));
}

static int dop_write_packet(av_player_t *p, decoder_t *d, const AVPacket *pkt) {
    int channels = d->codec_ctx->ch_layout.nb_channels;
    if (channels <= 0 || channels > AVPLAYER_MAX_DOP_CHANNELS) return AVERROR_INVALIDDATA;
    if (pkt->size <= 0 || (pkt->size % channels) != 0) return AVERROR_INVALIDDATA;

    int frames = dop_packet_output_frames(d, pkt, channels);
    if (frames < 0) return AVERROR_INVALIDDATA;
    int per_channel_bytes = pkt->size / channels;
    if (frames == 0) {
        for (int ch = 0; ch < channels && per_channel_bytes > 0; ch++) {
            d->dop_pending_byte[ch] =
                dop_packet_byte(d, pkt, ch, 0, channels, per_channel_bytes);
            d->dop_pending_valid[ch] = 1;
        }
        return 0;
    }
    if (ring_space(&p->ring) < frames) return AVERROR(EAGAIN);

    int channel_pos[AVPLAYER_MAX_DOP_CHANNELS] = {0};
    int32_t *out = (int32_t *)malloc((size_t)frames * (size_t)channels * sizeof(int32_t));
    if (!out) return AVERROR(ENOMEM);

    for (int frame = 0; frame < frames; frame++) {
        uint8_t marker = (p->dop_marker_phase & 1) ? 0xfa : 0x05;
        for (int ch = 0; ch < channels; ch++) {
            uint8_t b0;
            if (d->dop_pending_valid[ch]) {
                b0 = d->dop_pending_byte[ch];
                d->dop_pending_valid[ch] = 0;
            } else {
                b0 = dop_packet_byte(d, pkt, ch, channel_pos[ch]++, channels, per_channel_bytes);
            }
            uint8_t b1 = dop_packet_byte(d, pkt, ch, channel_pos[ch]++, channels, per_channel_bytes);
            out[frame * channels + ch] = dop_pack_s32(marker,
                                                      dop_dsd_byte(d, b0),
                                                      dop_dsd_byte(d, b1));
        }
        p->dop_marker_phase ^= 1;
    }

    for (int ch = 0; ch < channels; ch++) {
        if (channel_pos[ch] < per_channel_bytes) {
            d->dop_pending_byte[ch] =
                dop_packet_byte(d, pkt, ch, channel_pos[ch], channels, per_channel_bytes);
            d->dop_pending_valid[ch] = 1;
        }
    }

    int written = ring_write(&p->ring, out, frames);
    free(out);
    return written == frames ? 0 : AVERROR(EAGAIN);
}

static int dop_pending_any(const decoder_t *d, int channels) {
    if (!d || channels <= 0) return 0;
    for (int ch = 0; ch < channels && ch < AVPLAYER_MAX_DOP_CHANNELS; ch++) {
        if (d->dop_pending_valid[ch]) return 1;
    }
    return 0;
}

static int dop_write_wavpack_samples(av_player_t *p,
                                     decoder_t *d,
                                     const int32_t *samples,
                                     int samples_per_channel) {
    int channels = decoder_channel_count(d);
    if (!samples || samples_per_channel < 0 ||
        channels <= 0 || channels > AVPLAYER_MAX_DOP_CHANNELS) {
        return AVERROR_INVALIDDATA;
    }

    int frames = INT32_MAX;
    for (int ch = 0; ch < channels; ch++) {
        int total = samples_per_channel + (d->dop_pending_valid[ch] ? 1 : 0);
        int ch_frames = total / 2;
        if (ch_frames < frames) frames = ch_frames;
    }
    if (frames == INT32_MAX) frames = 0;

    if (frames == 0) {
        if (samples_per_channel > 0) {
            for (int ch = 0; ch < channels; ch++) {
                d->dop_pending_byte[ch] = (uint8_t)(samples[ch] & 0xff);
                d->dop_pending_valid[ch] = 1;
            }
        }
        return 0;
    }
    if (ring_space(&p->ring) < frames) return AVERROR(EAGAIN);

    int channel_pos[AVPLAYER_MAX_DOP_CHANNELS] = {0};
    int32_t *out = (int32_t *)malloc((size_t)frames * (size_t)channels * sizeof(int32_t));
    if (!out) return AVERROR(ENOMEM);

    for (int frame = 0; frame < frames; frame++) {
        uint8_t marker = (p->dop_marker_phase & 1) ? 0xfa : 0x05;
        for (int ch = 0; ch < channels; ch++) {
            uint8_t b0;
            if (d->dop_pending_valid[ch]) {
                b0 = d->dop_pending_byte[ch];
                d->dop_pending_valid[ch] = 0;
            } else {
                b0 = (uint8_t)(samples[channel_pos[ch] * channels + ch] & 0xff);
                channel_pos[ch]++;
            }
            uint8_t b1 = (uint8_t)(samples[channel_pos[ch] * channels + ch] & 0xff);
            channel_pos[ch]++;
            out[frame * channels + ch] = dop_pack_s32(marker,
                                                      dop_dsd_byte(d, b0),
                                                      dop_dsd_byte(d, b1));
        }
        p->dop_marker_phase ^= 1;
    }

    for (int ch = 0; ch < channels; ch++) {
        if (channel_pos[ch] < samples_per_channel) {
            d->dop_pending_byte[ch] =
                (uint8_t)(samples[channel_pos[ch] * channels + ch] & 0xff);
            d->dop_pending_valid[ch] = 1;
        }
    }

    int written = ring_write(&p->ring, out, frames);
    free(out);
    return written == frames ? 0 : AVERROR(EAGAIN);
}

static int av_player_decode_wavpack_dop_step(av_player_t *p, decoder_t *d) {
    int channels = decoder_channel_count(d);
    if (!d->wv_ctx || channels <= 0 || channels > AVPLAYER_MAX_DOP_CHANNELS ||
        d->wv_dsd_byte_rate <= 0) {
        return AVPLAYER_DECODE_ERROR;
    }

    if (atomic_load(&p->eof_reached)) {
        if (dop_pending_any(d, channels)) {
            AV_DEBUG("dropping incomplete trailing WavPack DSD byte at EOF");
            decoder_clear_dop_state(d);
        }
        return handle_eof_and_gapless(p);
    }

    int space = ring_space(&p->ring);
    if (space < 1) return AVPLAYER_DECODE_RING_FULL;

    int pending = dop_pending_any(d, channels);
    int request_samples = space * 2 - pending;
    if (request_samples <= 0) return AVPLAYER_DECODE_RING_FULL;
    if (request_samples > 8192) request_samples = 8192;

    int32_t *samples = (int32_t *)malloc((size_t)request_samples *
                                         (size_t)channels *
                                         sizeof(int32_t));
    if (!samples) return AVPLAYER_DECODE_ERROR;

    uint32_t got = WavpackUnpackSamples(d->wv_ctx, samples, (uint32_t)request_samples);
    if (got == 0) {
        free(samples);
        atomic_store(&p->eof_reached, 1);
        return AVPLAYER_DECODE_EOF;
    }

    d->wv_samples_unpacked += (long long)got;
    atomic_store_explicit((_Atomic double *)&p->time_pos,
                          (double)d->wv_samples_unpacked / (double)d->wv_dsd_byte_rate,
                          memory_order_relaxed);
    if (d->wv_average_bitrate > 0) {
        atomic_store(&p->cur_bitrate, d->wv_average_bitrate);
        p->media_info.bitrate = d->wv_average_bitrate;
    }

    int ret = dop_write_wavpack_samples(p, d, samples, (int)got);
    free(samples);
    if (ret == AVERROR(EAGAIN)) return AVPLAYER_DECODE_RING_FULL;
    return ret == 0 ? AVPLAYER_DECODE_OK : AVPLAYER_DECODE_ERROR;
}

static void update_packet_bitrate(av_player_t *p, decoder_t *d, const AVPacket *pkt) {
    if (!p || !d || !pkt || pkt->duration <= 0) return;
    AVRational tb = d->fmt_ctx->streams[d->audio_stream_idx]->time_base;
    double dur_secs = (double)pkt->duration * av_q2d(tb);
    if (dur_secs <= 0) return;

    int bps = (int)((double)pkt->size * 8.0 / dur_secs);
    atomic_store(&p->cur_bitrate, bps);
    p->media_info.bitrate = bps;

    bitrate_history_t *h = &p->bitrate_hist;
    int wi = atomic_load_explicit(&h->write_idx, memory_order_relaxed);
    h->entries[wi].frame_pos = atomic_load_explicit(
        (_Atomic long long *)&p->ring.frames_written_total, memory_order_relaxed);
    h->entries[wi].bitrate = bps;
    atomic_store_explicit(&h->write_idx, (wi + 1) % BITRATE_HISTORY_SIZE,
                          memory_order_release);
}

static int swap_to_next_decoder(av_player_t *p, output_config_t *next_cfg, int fill) {
    decoder_t *old_dec = NULL;

    decoder_t *next = NULL;
    ma_mutex_lock(&p->decoder_lock);
    next = p->next_dec;
    ma_mutex_unlock(&p->decoder_lock);
    if (!next) return 0;

    if (next_cfg->dop) {
        decoder_clear_dop_state(next);
    } else if (decoder_build_filter_graph(next, next_cfg) < 0) {
        return AVPLAYER_DECODE_ERROR;
    }

    ma_mutex_lock(&p->decoder_lock);
    old_dec = p->dec;
    p->dec = next;
    p->next_dec = NULL;
    ma_mutex_unlock(&p->decoder_lock);

    decoder_free(old_dec);
    atomic_store(&p->eof_reached, 0);
    atomic_store(&p->next_consumed, 1);

    atomic_store_explicit((_Atomic double *)&p->duration, p->dec->duration, memory_order_relaxed);
    atomic_store_explicit((_Atomic double *)&p->time_pos, 0.0, memory_order_relaxed);

    player_refresh_media_info(p, p->dec);

    long long fp = atomic_load_explicit(
        (_Atomic long long *)&p->frames_played_total, memory_order_relaxed);
    p->track_change_threshold = fp + (long long)fill;
    atomic_store(&p->pending_track_change, 1);
    atomic_store_explicit(&p->rg_gain_pending,
        compute_rg_gain(p->dec, p->rg_mode, p->rg_prevent_clip, p->rg_preamp_db),
        memory_order_relaxed);
    atomic_store_explicit(&p->rg_switch_threshold,
        (long long)p->track_change_threshold, memory_order_relaxed);
    atomic_store_explicit(&p->rg_switch_pending, 1, memory_order_release);
    return AVPLAYER_DECODE_OK;
}

static int handle_eof_and_gapless(av_player_t *p) {
    decoder_t *next = NULL;
    ma_mutex_lock(&p->decoder_lock);
    next = p->next_dec;
    ma_mutex_unlock(&p->decoder_lock);

    if (next) {
        output_config_t next_cfg;
        choose_output_config(p, next, &next_cfg);
        int same_output = output_config_matches_current(p, &next_cfg);
        int fill = atomic_load(&p->ring.fill);

        if (!same_output && fill > 0) {
            return AVPLAYER_DECODE_RING_FULL;
        }

        if (!same_output) {
            if (player_apply_output_config(p, &next_cfg) != 0) {
                return AVPLAYER_DECODE_ERROR;
            }
            atomic_store_explicit((_Atomic long long *)&p->frames_played_total, 0LL, memory_order_relaxed);
            atomic_store_explicit((_Atomic long long *)&p->ring.frames_written_total, 0LL, memory_order_relaxed);
            atomic_store_explicit((_Atomic double *)&p->position_offset, 0.0, memory_order_relaxed);
            atomic_store_explicit((_Atomic long long *)&p->position_clock_ref, 0LL, memory_order_relaxed);
            memset(&p->bitrate_hist, 0, sizeof(p->bitrate_hist));
            p->dop_marker_phase = 0;
            fill = 0;
        }

        int swap_ret = swap_to_next_decoder(p, &next_cfg, fill);
        if (swap_ret < 0) return swap_ret;
        return AVPLAYER_DECODE_RING_FULL;
    }

    if (ring_avail(&p->ring) == 0) {
        atomic_store(&p->state, AVPLAYER_STATE_STOPPED);
        return AVPLAYER_DECODE_STOPPED;
    }
    return AVPLAYER_DECODE_RING_FULL;
}

static int av_player_decode_dop_step(av_player_t *p, decoder_t *d) {
    if (d->wv_dsd) {
        return av_player_decode_wavpack_dop_step(p, d);
    }

    if (d->dop_pending_pkt && d->dop_pending_pkt->size > 0) {
        int pending_ret = dop_write_packet(p, d, d->dop_pending_pkt);
        if (pending_ret == AVERROR(EAGAIN)) return AVPLAYER_DECODE_RING_FULL;
        av_packet_unref(d->dop_pending_pkt);
        return pending_ret == 0 ? AVPLAYER_DECODE_OK : AVPLAYER_DECODE_ERROR;
    }

    if (ring_space(&p->ring) < 512) {
        return AVPLAYER_DECODE_RING_FULL;
    }

    if (atomic_load(&p->eof_reached)) {
        return handle_eof_and_gapless(p);
    }

    int ret = av_read_frame(d->fmt_ctx, d->pkt);
    if (ret == AVERROR_EOF) {
        atomic_store(&p->eof_reached, 1);
        return AVPLAYER_DECODE_EOF;
    }
    if (ret < 0) {
        return AVPLAYER_DECODE_ERROR;
    }

    if (d->pkt->stream_index != d->audio_stream_idx) {
        av_packet_unref(d->pkt);
        return AVPLAYER_DECODE_OK;
    }

    if (d->pkt->pts != AV_NOPTS_VALUE) {
        AVRational tb = d->fmt_ctx->streams[d->audio_stream_idx]->time_base;
        atomic_store_explicit((_Atomic double *)&p->time_pos,
                              (double)d->pkt->pts * av_q2d(tb),
                              memory_order_relaxed);
    }
    update_packet_bitrate(p, d, d->pkt);

    int frames = dop_packet_output_frames(d, d->pkt, d->codec_ctx->ch_layout.nb_channels);
    if (frames < 0) {
        av_packet_unref(d->pkt);
        return AVPLAYER_DECODE_ERROR;
    }
    if (ring_space(&p->ring) < frames) {
        if (av_packet_ref(d->dop_pending_pkt, d->pkt) < 0) {
            av_packet_unref(d->pkt);
            return AVPLAYER_DECODE_ERROR;
        }
        av_packet_unref(d->pkt);
        return AVPLAYER_DECODE_RING_FULL;
    }

    ret = dop_write_packet(p, d, d->pkt);
    av_packet_unref(d->pkt);
    if (ret == AVERROR(EAGAIN)) return AVPLAYER_DECODE_RING_FULL;
    return ret == 0 ? AVPLAYER_DECODE_OK : AVPLAYER_DECODE_ERROR;
}

// Attempt to write one filtered frame to the ring buffer.
//
// If d->pending_frame is set, we retry that frame first (it didn't fit last
// time).  Otherwise we pull the next frame from the filter sink.
//
// Returns:
//   0              — a frame was successfully written
//   AVERROR(EAGAIN) — ring full or no frame available from sink yet
//   other negative  — real error
static int drain_sink(av_player_t *p, decoder_t *d) {
    AVFrame *ff;

    if (d->pending_frame) {
        // Retry previously-stalled frame.
        ff = d->pending_frame;
    } else {
        // Pull next frame from filter graph — hold filter_lock so a concurrent
        // filter rebuild cannot free/replace buffersink_ctx mid-call.
        ff = d->filt_frame;
        ma_mutex_lock(&p->filter_lock);
        int ret = av_buffersink_get_frame(d->buffersink_ctx, ff);
        AVRational tb = (ff->pts != AV_NOPTS_VALUE)
                        ? av_buffersink_get_time_base(d->buffersink_ctx)
                        : (AVRational){0, 1};
        ma_mutex_unlock(&p->filter_lock);

        if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF) {
            return AVERROR(EAGAIN);  // nothing ready
        }
        if (ret < 0) return ret;

        // Update time position from this frame's pts.
        if (ff->pts != AV_NOPTS_VALUE) {
            double t = (double)ff->pts * av_q2d(tb);
            atomic_store_explicit((_Atomic double *)&p->time_pos, t, memory_order_relaxed);
        }
    }

    // Try to write to ring (all-or-nothing).
    int written = ring_write(&p->ring, ff->data[0], ff->nb_samples);
    if (written == 0) {
        // Ring is full.  Save this frame to retry next decode step.
        if (!d->pending_frame) {
            // Move filt_frame into pending_frame (take ownership).
            d->pending_frame = av_frame_clone(ff);
            av_frame_unref(d->filt_frame);
        }
        // else: pending_frame already holds it, nothing to do.
        return AVERROR(EAGAIN);
    }

    // Frame written successfully; discard it.
    if (d->pending_frame) {
        av_frame_free(&d->pending_frame);  // sets to NULL
    } else {
        av_frame_unref(d->filt_frame);
    }
    return 0;
}

int av_player_decode_step(av_player_t *p) {
    // Check if a delayed track-change signal is ready to fire.
    if (atomic_load_explicit(&p->pending_track_change, memory_order_relaxed)) {
        long long played = atomic_load_explicit(
            (_Atomic long long *)&p->frames_played_total, memory_order_relaxed);
        if (played >= p->track_change_threshold) {
            atomic_store(&p->pending_track_change, 0);
            // Reset position clock so the new track starts from 0.
            atomic_store_explicit((_Atomic double *)&p->position_offset, 0.0, memory_order_relaxed);
            atomic_store_explicit((_Atomic long long *)&p->position_clock_ref, (long long)played, memory_order_relaxed);
            return AVPLAYER_DECODE_NEXT_READY;
        }
    }

    if (atomic_load(&p->state) == AVPLAYER_STATE_STOPPED) {
        return AVPLAYER_DECODE_STOPPED;
    }

    // If paused, check if ring buffer still has data (don't decode more)
    if (atomic_load(&p->state) == AVPLAYER_STATE_PAUSED) {
        return AVPLAYER_DECODE_RING_FULL;  // tell Go to sleep briefly
    }

    decoder_t *d = p->dec;
    if (!d) return AVPLAYER_DECODE_STOPPED;

    if (p->dop_active) {
        return av_player_decode_dop_step(p, d);
    }

    // If we have a pending frame (ring was full last time), try to flush it first.
    // If the ring is still too full, tell Go to sleep briefly.
    if (d->pending_frame) {
        int ret = drain_sink(p, d);
        if (ret == 0) return AVPLAYER_DECODE_OK;
        // pending_frame still set → ring still full
        return AVPLAYER_DECODE_RING_FULL;
    }

    // Also drain whatever the filter sink has ready (from the previous packet).
    {
        int ret = drain_sink(p, d);
        if (ret == 0) return AVPLAYER_DECODE_OK;
        // EAGAIN from sink means no frame ready — proceed to read a new packet.
        // But if the ring is nearly full, don't bother reading more right now.
        if (ring_space(&p->ring) < 512) {
            return AVPLAYER_DECODE_RING_FULL;
        }
    }

    // Check EOF already reached.  By this point pending_frame is NULL (the
    // check above returned early if it was set), so all old-track samples are
    // in the ring.  Same-format next tracks can be swapped before the ring
    // drains for gapless playback. Different-format tracks must wait until
    // the old ring drains so the host device can be reconfigured.
    if (atomic_load(&p->eof_reached)) {
        return handle_eof_and_gapless(p);
    }

    // Read and decode a packet
    int ret = av_read_frame(d->fmt_ctx, d->pkt);
    if (ret == AVERROR_EOF) {
        // Signal EOF to the decoder
        avcodec_send_packet(d->codec_ctx, NULL);
        atomic_store(&p->eof_reached, 1);

        // Drain decoder
        AVFrame *frame = d->frame;
        while (avcodec_receive_frame(d->codec_ctx, frame) >= 0) {
            ma_mutex_lock(&p->filter_lock);
            (void)av_buffersrc_add_frame_flags(d->buffersrc_ctx, frame, AV_BUFFERSRC_FLAG_PUSH);
            ma_mutex_unlock(&p->filter_lock);
            av_frame_unref(frame);
        }
        // Flush filter graph
        ma_mutex_lock(&p->filter_lock);
        (void)av_buffersrc_add_frame_flags(d->buffersrc_ctx, NULL, AV_BUFFERSRC_FLAG_PUSH);
        ma_mutex_unlock(&p->filter_lock);
        // Drain sink into ring
        while (drain_sink(p, d) == 0) {}

        return AVPLAYER_DECODE_EOF;
    }
    if (ret < 0) {
        return AVPLAYER_DECODE_ERROR;
    }

    if (d->pkt->stream_index != d->audio_stream_idx) {
        av_packet_unref(d->pkt);
        return AVPLAYER_DECODE_OK;
    }

    // Update instantaneous bitrate
    if (d->pkt->duration > 0) {
        AVRational tb = d->fmt_ctx->streams[d->audio_stream_idx]->time_base;
        double dur_secs = (double)d->pkt->duration * av_q2d(tb);
        if (dur_secs > 0) {
            int bps = (int)((double)d->pkt->size * 8.0 / dur_secs);
            atomic_store(&p->cur_bitrate, bps);
            p->media_info.bitrate = bps;

            // Push (frame_pos, bitrate) to history ring so GetMediaInfo can
            // return the bitrate matching the currently playing audio rather
            // than the most recently decoded packet.
            bitrate_history_t *h = &p->bitrate_hist;
            int wi = atomic_load_explicit(&h->write_idx, memory_order_relaxed);
            h->entries[wi].frame_pos = atomic_load_explicit(
                (_Atomic long long *)&p->ring.frames_written_total, memory_order_relaxed);
            h->entries[wi].bitrate = bps;
            atomic_store_explicit(&h->write_idx, (wi + 1) % BITRATE_HISTORY_SIZE,
                                  memory_order_release);
        }
    }

    ret = avcodec_send_packet(d->codec_ctx, d->pkt);
    av_packet_unref(d->pkt);
    if (ret < 0 && ret != AVERROR(EAGAIN)) return AVPLAYER_DECODE_ERROR;

    // Receive all decoded frames
    AVFrame *frame = d->frame;
    while ((ret = avcodec_receive_frame(d->codec_ctx, frame)) == 0) {
        ma_mutex_lock(&p->filter_lock);
        int r2 = av_buffersrc_add_frame_flags(d->buffersrc_ctx, frame, AV_BUFFERSRC_FLAG_PUSH);
        ma_mutex_unlock(&p->filter_lock);
        av_frame_unref(frame);
        if (r2 < 0) break;

        // Drain as many filtered frames as will fit in the ring right now.
        // Stop (leave pending_frame set) if the ring is full.
        while (drain_sink(p, d) == 0) {
            // Keep draining until sink is empty or ring is full.
        }
        // If we have a pending frame, the ring is full — stop reading more packets.
        if (d->pending_frame) break;
    }

    return AVPLAYER_DECODE_OK;
}
