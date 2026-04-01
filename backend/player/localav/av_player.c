// av_player.c — FFmpeg + miniaudio audio player
// One translation unit: defines miniaudio implementation here.

#define MA_IMPLEMENTATION
#define MA_NO_FLAC  // Use FFmpeg for FLAC decoding instead
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

#include <stdlib.h>
#include <string.h>
#include <stdio.h>
#include <math.h>
#include <stdatomic.h>

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
    int              audio_stream_idx;
    double           duration;      // seconds; 0 if unknown
    double           rg_gain_db;
    int              rg_prevent_clip;
    char             eq_filter[4096];
    char             url[2048];
} decoder_t;

// SPSC ring buffer — producer: decode goroutine, consumer: miniaudio callback
typedef struct {
    float          *buf;         // [AVPLAYER_RING_FRAMES * AVPLAYER_CHANNELS]
    int             cap;         // in frames = AVPLAYER_RING_FRAMES
    atomic_int      write_idx;   // next frame to write (mod cap)
    atomic_int      read_idx;    // next frame to read  (mod cap)
    atomic_int      fill;        // frames available to read
} ring_buf_t;

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
};

// --------------------------------------------------------------------------
// Ring buffer helpers
// --------------------------------------------------------------------------

static void ring_init(ring_buf_t *r) {
    r->cap = AVPLAYER_RING_FRAMES;
    r->buf = (float *)malloc(r->cap * AVPLAYER_CHANNELS * sizeof(float));
    atomic_store(&r->write_idx, 0);
    atomic_store(&r->read_idx,  0);
    atomic_store(&r->fill,      0);
}

static void ring_free(ring_buf_t *r) {
    free(r->buf);
    r->buf = NULL;
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

// Write exactly n_frames of interleaved f32 stereo.
// Returns n_frames on success, 0 if there is not enough space (all-or-nothing).
// Never clips or drops samples — the caller must retry with the same data.
static int ring_write(ring_buf_t *r, const float *src, int n_frames) {
    if (ring_space(r) < n_frames) return 0;

    int wi  = atomic_load(&r->write_idx);
    int ch  = AVPLAYER_CHANNELS;
    int cap = r->cap;

    // How many frames fit before the wrap-around point?
    int first  = cap - wi;             // frames from wi to end of buffer
    if (first > n_frames) first = n_frames;
    int second = n_frames - first;     // frames that wrap to start of buffer

    memcpy(&r->buf[wi * ch], src, first * ch * sizeof(float));
    if (second > 0) {
        memcpy(&r->buf[0], src + first * ch, second * ch * sizeof(float));
    }

    atomic_store(&r->write_idx, (wi + n_frames) % cap);
    atomic_fetch_add(&r->fill, n_frames);
    return n_frames;
}

// Read up to n_frames into dst.  Returns number of frames actually read.
static int ring_read(ring_buf_t *r, float *dst, int n_frames) {
    int avail   = ring_avail(r);
    int to_read = (n_frames < avail) ? n_frames : avail;
    if (to_read == 0) return 0;

    int ri  = atomic_load(&r->read_idx);
    int ch  = AVPLAYER_CHANNELS;
    int cap = r->cap;

    int first  = cap - ri;
    if (first > to_read) first = to_read;
    int second = to_read - first;

    memcpy(dst, &r->buf[ri * ch], first * ch * sizeof(float));
    if (second > 0) {
        memcpy(dst + first * ch, &r->buf[0], second * ch * sizeof(float));
    }

    atomic_store(&r->read_idx, (ri + to_read) % cap);
    atomic_fetch_sub(&r->fill, to_read);
    return to_read;
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
    if (!d->pkt || !d->frame || !d->filt_frame) {
        av_packet_free(&d->pkt);
        av_frame_free(&d->frame);
        av_frame_free(&d->filt_frame);
        free(d);
        return NULL;
    }
    return d;
}

static void decoder_free(decoder_t *d) {
    if (!d) return;
    if (d->filter_graph) avfilter_graph_free(&d->filter_graph);
    if (d->codec_ctx)    avcodec_free_context(&d->codec_ctx);
    if (d->fmt_ctx)      avformat_close_input(&d->fmt_ctx);
    av_packet_free(&d->pkt);
    av_frame_free(&d->frame);
    av_frame_free(&d->filt_frame);
    av_frame_free(&d->pending_frame);  // safe if NULL
    free(d);
}

// Build filter graph: abuffer → [astats] → [EQ] → [volume] → aformat → aresample → abuffersink
// eq_filter: comma-separated avfilter fragment, or NULL/""
// rg_gain_db: volume adjustment in dB (0 = no change)
// rg_prevent_clip: non-zero → add alimiter at 0 dBFS
// peaks_enabled: non-zero → prepend astats filter
static int decoder_build_filter_graph(decoder_t *d,
                                      int peaks_enabled,
                                      const char *eq_filter,
                                      double rg_gain_db,
                                      int rg_prevent_clip)
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

    // --- abuffersink (fixed output: f32 interleaved, stereo, 48 kHz) ---
    AVFilterContext *sink_ctx = NULL;
    ret = avfilter_graph_create_filter(&sink_ctx,
                                       avfilter_get_by_name("abuffersink"),
                                       "out", NULL, NULL, graph);
    if (ret < 0) goto fail;

    // Output format constraints are enforced via the aformat filter in the chain
    // below rather than via deprecated av_opt_set_int_list on the sink.

    // --- Build filter chain string ---
    // We use avfilter_graph_parse_ptr to assemble the full chain.
    char chain[8192];
    chain[0] = '\0';

    // astats (peak measurement)
    if (peaks_enabled) {
        strncat(chain, "astats=metadata=1:reset=1:measure_overall=none,",
                sizeof(chain) - strlen(chain) - 1);
    }

    // User EQ bands
    if (eq_filter && eq_filter[0] != '\0') {
        strncat(chain, eq_filter, sizeof(chain) - strlen(chain) - 1);
        strncat(chain, ",", sizeof(chain) - strlen(chain) - 1);
    }

    // Volume (preamp / ReplayGain)
    if (fabs(rg_gain_db) > 0.001) {
        char vol[64];
        snprintf(vol, sizeof(vol), "volume=volume=%0.3fdB,", rg_gain_db);
        strncat(chain, vol, sizeof(chain) - strlen(chain) - 1);
    }

    // Prevent clipping with a simple limiter
    if (rg_prevent_clip && fabs(rg_gain_db) > 0.001) {
        strncat(chain, "alimiter=level_in=1:level_out=1:limit=0.9999:attack=1:release=5:level=disabled,",
                sizeof(chain) - strlen(chain) - 1);
    }

    // Resample to fixed output rate + convert to interleaved f32 stereo
    strncat(chain, "aresample=" AVPLAYER_SAMPLE_RATE_STR ",", sizeof(chain) - strlen(chain) - 1);
    strncat(chain, "aformat=sample_fmts=flt:channel_layouts=stereo",
            sizeof(chain) - strlen(chain) - 1);

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
    return 0;

fail:
    avcodec_free_context(&d->codec_ctx);
    avformat_close_input(&d->fmt_ctx);
    return ret;
}

// Read ReplayGain metadata from format context tags.
// Returns 0 if nothing found.
static double read_rg_gain(AVFormatContext *fmt_ctx, int use_album) {
    const char *track_key = "REPLAYGAIN_TRACK_GAIN";
    const char *album_key = "REPLAYGAIN_ALBUM_GAIN";
    AVDictionaryEntry *e = NULL;

    if (use_album) {
        e = av_dict_get(fmt_ctx->metadata, album_key, NULL, AV_DICT_IGNORE_SUFFIX);
        if (!e) e = av_dict_get(fmt_ctx->metadata, "replaygain_album_gain", NULL, AV_DICT_IGNORE_SUFFIX);
    }
    if (!e) {
        e = av_dict_get(fmt_ctx->metadata, track_key, NULL, AV_DICT_IGNORE_SUFFIX);
        if (!e) e = av_dict_get(fmt_ctx->metadata, "replaygain_track_gain", NULL, AV_DICT_IGNORE_SUFFIX);
    }
    if (!e) return 0.0;
    // value like "-6.23 dB" — atof stops at non-numeric
    return atof(e->value);
}

// --------------------------------------------------------------------------
// miniaudio callback
// --------------------------------------------------------------------------

static void ma_data_callback(ma_device *device, void *output, const void *input, ma_uint32 frame_count) {
    (void)input;
    av_player_t *p = (av_player_t *)device->pUserData;
    float *out = (float *)output;

    if (atomic_load(&p->state) != AVPLAYER_STATE_PLAYING) {
        // Silence output when paused or stopped
        memset(out, 0, frame_count * AVPLAYER_CHANNELS * sizeof(float));
        return;
    }

    int got = ring_read(&p->ring, out, (int)frame_count);
    // Apply volume
    float vol = p->volume;
    if (vol < 0.9999f || vol > 1.0001f) {
        for (int i = 0; i < got * AVPLAYER_CHANNELS; i++) {
            out[i] *= vol;
        }
    }
    // Fill remainder with silence
    if (got < (int)frame_count) {
        memset(out + got * AVPLAYER_CHANNELS, 0,
               (frame_count - got) * AVPLAYER_CHANNELS * sizeof(float));
    }
}

// --------------------------------------------------------------------------
// Device management helpers
// --------------------------------------------------------------------------

static int player_init_device(av_player_t *p, const char *device_name, int exclusive) {
    if (p->device_init) {
        ma_device_uninit(&p->device);
        p->device_init = 0;
    }

    ma_device_config cfg = ma_device_config_init(ma_device_type_playback);
    cfg.playback.format   = ma_format_f32;
    cfg.playback.channels = AVPLAYER_CHANNELS;
    cfg.sampleRate        = AVPLAYER_SAMPLE_RATE;
    cfg.dataCallback      = ma_data_callback;
    cfg.pUserData         = p;
    cfg.periodSizeInMilliseconds = 10;

    if (exclusive) {
        cfg.playback.shareMode = ma_share_mode_exclusive;
    }

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
                    break;
                }
            }
        }
    }

    if (ma_device_init(&p->ma_ctx, &cfg, &p->device) != MA_SUCCESS) {
        return -1;
    }
    p->device_init = 1;

    if (ma_device_start(&p->device) != MA_SUCCESS) {
        ma_device_uninit(&p->device);
        p->device_init = 0;
        return -1;
    }
    return 0;
}

// --------------------------------------------------------------------------
// Public API implementation
// --------------------------------------------------------------------------

av_player_t *av_player_create(void) {
    av_player_t *p = (av_player_t *)calloc(1, sizeof(av_player_t));
    if (!p) return NULL;
    ring_init(&p->ring);
    p->volume = 1.0f;
    atomic_store(&p->state, AVPLAYER_STATE_STOPPED);
    atomic_store(&p->l_peak, -INFINITY);
    atomic_store(&p->r_peak, -INFINITY);
    atomic_store(&p->l_rms,  -INFINITY);
    atomic_store(&p->r_rms,  -INFINITY);
    return p;
}

int av_player_init(av_player_t *p, const char *device_name, int exclusive) {
    if (ma_context_init(NULL, 0, NULL, &p->ma_ctx) != MA_SUCCESS) {
        return -1;
    }
    p->ma_ctx_init = 1;
    return player_init_device(p, device_name, exclusive);
}

void av_player_destroy(av_player_t *p) {
    if (!p) return;
    atomic_store(&p->state, AVPLAYER_STATE_STOPPED);
    if (p->device_init) {
        ma_device_uninit(&p->device);
    }
    if (p->ma_ctx_init) {
        ma_context_uninit(&p->ma_ctx);
    }
    decoder_free(p->dec);
    decoder_free(p->next_dec);
    ring_free(&p->ring);
    free(p);
}

int av_player_open(av_player_t *p, const char *url, double start_time,
                   const char *eq_filter, double rg_gain_db, int rg_prevent_clip)
{
    // Stop current playback
    atomic_store(&p->state, AVPLAYER_STATE_STOPPED);
    ring_clear(&p->ring);
    atomic_store(&p->eof_reached, 0);
    atomic_store(&p->next_consumed, 0);

    decoder_free(p->next_dec);
    p->next_dec = NULL;
    decoder_free(p->dec);
    p->dec = NULL;

    decoder_t *d = decoder_alloc();
    if (!d) return AVERROR(ENOMEM);

    int ret = decoder_open(d, url);
    if (ret < 0) { decoder_free(d); return ret; }

    // Apply ReplayGain from tags if gain not explicitly provided
    // (rg_gain_db == 0 means use tags; nonzero means caller already computed it)
    strncpy(d->eq_filter, eq_filter ? eq_filter : "", sizeof(d->eq_filter) - 1);
    d->rg_gain_db      = rg_gain_db;
    d->rg_prevent_clip = rg_prevent_clip;

    ret = decoder_build_filter_graph(d, p->peaks_enabled,
                                     d->eq_filter, d->rg_gain_db, d->rg_prevent_clip);
    if (ret < 0) { decoder_free(d); return ret; }

    // Populate media info
    memset(&p->media_info, 0, sizeof(p->media_info));
    strncpy(p->media_info.codec,
            d->codec_ctx->codec->name,
            sizeof(p->media_info.codec) - 1);
    p->media_info.sample_rate = d->codec_ctx->sample_rate;
    p->media_info.channels    = d->codec_ctx->ch_layout.nb_channels;

    atomic_store_explicit((_Atomic double *)&p->duration, d->duration, memory_order_relaxed);
    atomic_store_explicit((_Atomic double *)&p->time_pos, 0.0, memory_order_relaxed);

    p->dec = d;

    if (start_time > 0.0) {
        av_player_seek(p, start_time);
    }

    atomic_store(&p->state, AVPLAYER_STATE_PLAYING);
    return 0;
}

int av_player_open_next(av_player_t *p, const char *url,
                        const char *eq_filter, double rg_gain_db, int rg_prevent_clip)
{
    decoder_free(p->next_dec);
    p->next_dec = NULL;
    atomic_store(&p->next_consumed, 0);

    if (!url || url[0] == '\0') return 0;

    decoder_t *d = decoder_alloc();
    if (!d) return AVERROR(ENOMEM);

    int ret = decoder_open(d, url);
    if (ret < 0) { decoder_free(d); return ret; }

    strncpy(d->eq_filter, eq_filter ? eq_filter : "", sizeof(d->eq_filter) - 1);
    d->rg_gain_db      = rg_gain_db;
    d->rg_prevent_clip = rg_prevent_clip;

    ret = decoder_build_filter_graph(d, p->peaks_enabled,
                                     d->eq_filter, d->rg_gain_db, d->rg_prevent_clip);
    if (ret < 0) { decoder_free(d); return ret; }

    p->next_dec = d;
    return 0;
}

void av_player_stop(av_player_t *p) {
    atomic_store(&p->state, AVPLAYER_STATE_STOPPED);
    ring_clear(&p->ring);
    atomic_store(&p->eof_reached, 0);
    decoder_free(p->dec);   p->dec      = NULL;
    decoder_free(p->next_dec); p->next_dec = NULL;
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
    AVFormatContext *fc = p->dec->fmt_ctx;
    int stream_idx = p->dec->audio_stream_idx;

    int64_t ts = (int64_t)(seconds * AV_TIME_BASE);
    int ret = avformat_seek_file(fc, -1, INT64_MIN, ts, INT64_MAX, 0);
    if (ret < 0) return ret;

    avcodec_flush_buffers(p->dec->codec_ctx);

    // Flush filter graph by draining and sending NULL frame
    (void)av_buffersrc_add_frame_flags(p->dec->buffersrc_ctx, NULL, AV_BUFFERSRC_FLAG_PUSH);
    av_frame_unref(p->dec->filt_frame);
    while (av_buffersink_get_frame(p->dec->buffersink_ctx, p->dec->filt_frame) >= 0) {
        av_frame_unref(p->dec->filt_frame);
    }
    // Reinit the filter graph to avoid corruption
    decoder_build_filter_graph(p->dec, p->peaks_enabled,
                                p->dec->eq_filter, p->dec->rg_gain_db, p->dec->rg_prevent_clip);

    // Discard any pending frame from before the seek.
    av_frame_free(&p->dec->pending_frame);

    ring_clear(&p->ring);
    atomic_store(&p->eof_reached, 0);
    atomic_store_explicit((_Atomic double *)&p->time_pos, seconds, memory_order_relaxed);
    return 0;
}

void av_player_set_volume(av_player_t *p, float volume) {
    p->volume = volume;
}

int av_player_set_filters(av_player_t *p,
                          const char *eq_filter,
                          double rg_gain_db,
                          int rg_prevent_clip)
{
    if (!p->dec) return -1;
    strncpy(p->dec->eq_filter, eq_filter ? eq_filter : "", sizeof(p->dec->eq_filter) - 1);
    p->dec->rg_gain_db      = rg_gain_db;
    p->dec->rg_prevent_clip = rg_prevent_clip;
    return decoder_build_filter_graph(p->dec, p->peaks_enabled,
                                      p->dec->eq_filter, rg_gain_db, rg_prevent_clip);
}

int av_player_get_state(av_player_t *p) {
    return atomic_load(&p->state);
}

double av_player_get_position(av_player_t *p) {
    return atomic_load_explicit((_Atomic double *)&p->time_pos, memory_order_relaxed);
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
    if (p->dec) {
        decoder_build_filter_graph(p->dec, enabled,
                                   p->dec->eq_filter, p->dec->rg_gain_db, p->dec->rg_prevent_clip);
    }
}

void av_player_get_media_info(av_player_t *p, av_media_info_t *info) {
    *info = p->media_info;
    info->bitrate = atomic_load(&p->cur_bitrate);
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
            count++;
        }
    }

    ma_context_uninit(&ctx);
    return count;
}

int av_player_set_device(av_player_t *p, const char *device_name) {
    int state = atomic_load(&p->state);
    // Stop the device, reinit with new device, restart
    ma_device_uninit(&p->device);
    p->device_init = 0;
    int ret = player_init_device(p, device_name, 0);
    // Restore play state (the audio callback will handle the ring buffer)
    (void)state;
    return ret;
}

int av_player_set_exclusive(av_player_t *p, int exclusive) {
    ma_device_uninit(&p->device);
    p->device_init = 0;
    return player_init_device(p, NULL, exclusive);
}

// --------------------------------------------------------------------------
// Decode step — called from Go goroutine in a tight loop
// --------------------------------------------------------------------------

// Update peak values from a filtered frame's metadata
static void update_peaks(av_player_t *p, AVFrame *frame) {
    if (!p->peaks_enabled) return;
    AVDictionaryEntry *e;

    e = av_dict_get(frame->metadata, "lavfi.astats.1.Peak_level", NULL, 0);
    if (e) atomic_store_explicit((_Atomic double *)&p->l_peak, atof(e->value), memory_order_relaxed);

    e = av_dict_get(frame->metadata, "lavfi.astats.2.Peak_level", NULL, 0);
    if (e) atomic_store_explicit((_Atomic double *)&p->r_peak, atof(e->value), memory_order_relaxed);

    e = av_dict_get(frame->metadata, "lavfi.astats.1.RMS_level", NULL, 0);
    if (e) atomic_store_explicit((_Atomic double *)&p->l_rms, atof(e->value), memory_order_relaxed);

    e = av_dict_get(frame->metadata, "lavfi.astats.2.RMS_level", NULL, 0);
    if (e) atomic_store_explicit((_Atomic double *)&p->r_rms, atof(e->value), memory_order_relaxed);
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
        // Pull next frame from filter graph.
        ff = d->filt_frame;
        int ret = av_buffersink_get_frame(d->buffersink_ctx, ff);
        if (ret == AVERROR(EAGAIN) || ret == AVERROR_EOF) {
            return AVERROR(EAGAIN);  // nothing ready
        }
        if (ret < 0) return ret;

        // Update time position from this frame's pts.
        if (ff->pts != AV_NOPTS_VALUE) {
            AVRational tb = av_buffersink_get_time_base(d->buffersink_ctx);
            double t = (double)ff->pts * av_q2d(tb);
            atomic_store_explicit((_Atomic double *)&p->time_pos, t, memory_order_relaxed);
        }
        update_peaks(p, ff);
    }

    // Try to write to ring (all-or-nothing).
    int written = ring_write(&p->ring, (const float *)ff->data[0], ff->nb_samples);
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
    if (atomic_load(&p->state) == AVPLAYER_STATE_STOPPED) {
        return AVPLAYER_DECODE_STOPPED;
    }

    // If paused, check if ring buffer still has data (don't decode more)
    if (atomic_load(&p->state) == AVPLAYER_STATE_PAUSED) {
        return AVPLAYER_DECODE_RING_FULL;  // tell Go to sleep briefly
    }

    decoder_t *d = p->dec;
    if (!d) return AVPLAYER_DECODE_STOPPED;

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
    // in the ring.  Swap to the next decoder immediately — before the ring
    // drains — so the ring stays filled and there is no audible gap.
    if (atomic_load(&p->eof_reached)) {
        if (p->next_dec) {
            // Gapless: swap in next decoder while ring still has audio data.
            decoder_free(p->dec);
            p->dec = p->next_dec;
            p->next_dec = NULL;
            atomic_store(&p->eof_reached, 0);
            atomic_store(&p->next_consumed, 1);

            // Update duration / time
            atomic_store_explicit((_Atomic double *)&p->duration, p->dec->duration, memory_order_relaxed);
            atomic_store_explicit((_Atomic double *)&p->time_pos, 0.0, memory_order_relaxed);

            // Update media info
            memset(&p->media_info, 0, sizeof(p->media_info));
            strncpy(p->media_info.codec, p->dec->codec_ctx->codec->name, sizeof(p->media_info.codec) - 1);
            p->media_info.sample_rate = p->dec->codec_ctx->sample_rate;
            p->media_info.channels    = p->dec->codec_ctx->ch_layout.nb_channels;

            return AVPLAYER_DECODE_NEXT_READY;  // Go should invoke OnTrackChange
        }
        // No next track: wait for ring to drain before signalling stopped.
        if (ring_avail(&p->ring) == 0) {
            atomic_store(&p->state, AVPLAYER_STATE_STOPPED);
            return AVPLAYER_DECODE_STOPPED;
        }
        return AVPLAYER_DECODE_RING_FULL;
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
            (void)av_buffersrc_add_frame_flags(d->buffersrc_ctx, frame, AV_BUFFERSRC_FLAG_PUSH);
            av_frame_unref(frame);
        }
        // Flush filter graph
        (void)av_buffersrc_add_frame_flags(d->buffersrc_ctx, NULL, AV_BUFFERSRC_FLAG_PUSH);
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
        }
    }

    ret = avcodec_send_packet(d->codec_ctx, d->pkt);
    av_packet_unref(d->pkt);
    if (ret < 0 && ret != AVERROR(EAGAIN)) return AVPLAYER_DECODE_ERROR;

    // Receive all decoded frames
    AVFrame *frame = d->frame;
    while ((ret = avcodec_receive_frame(d->codec_ctx, frame)) == 0) {
        int r2 = av_buffersrc_add_frame_flags(d->buffersrc_ctx, frame, AV_BUFFERSRC_FLAG_PUSH);
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
