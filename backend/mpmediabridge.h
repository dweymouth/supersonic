/**
 * mpmediabridge.h
 * 
 * This file provides a C bridge to the Objective-C framework for macOS media playback.
 * It offers a simplified interface to interact with the MPNowPlayingInfoCenter and other
 * related media playback functionalities in macOS without dealing directly with 
 * Objective-C code.
 */

#include <AppKit/AppKit.h>
#include <MediaPlayer/MediaPlayer.h>

/**
 * Updates the "Now Playing" information on macOS for media playback
 * using the MPNowPlayingInfoCenter API to set the metadata 
 * for the currently playing media in the system's "Now Playing" interface.
 */
void setNowPlayingInfo(const char *title, const char *artist);
