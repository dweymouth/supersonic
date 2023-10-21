//go:build darwin

/**
 * mpmediabridge.h
 * 
 * This file provides a C bridge to the Objective-C framework for macOS media playback & events.
 * It offers a simplified interface to interact with the MPNowPlayingInfoCenter and other
 * related media playback functionalities in macOS without dealing directly with Objective-C code.
 */

#include <AppKit/AppKit.h>
#include <MediaPlayer/MediaPlayer.h>

/**
* OS remote command enumeration, accepted by 'os_remote_command_callback'.
*/
typedef enum {
    PLAY,
    PAUSE,
    STOP,
    TOGGLE,
    NEXT_TRACK,
    PREVIOUS_TRACK,
    SEEK
} Command;

/**
* registers the 'os_remote_command_callback' to receive OS media commands. 
*/
void register_os_remote_commands();

/**
* Go-backed callback to static function that is called when OS remote commands are received.
* If a value is anticipated with the specified command, the 'value' argument will be non-zero.
*/
void os_remote_command_callback(Command command, double value);

/**
 * Updates the "Now Playing" information on macOS for media playback
 * using the MPNowPlayingInfoCenter API to set the metadata 
 * for the currently playing media in the system's "Now Playing" interface.
 */
void set_os_now_playing_info(const char *title, const char *artist, const char *coverArtFileURL, double trackDuration);
void update_os_now_playing_info_position(double positionSeconds);

/**
 * Setter functions for updating the global playback state.
 */
void set_os_playback_state_playing();
void set_os_playback_state_paused();
void set_os_playback_state_stopped();
