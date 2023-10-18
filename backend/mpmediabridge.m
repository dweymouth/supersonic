#import "mpmediabridge.h"

/**
 * Native Objective-C function for setting "Now Playing" information on macOS for media playback using the native APIs.
 */
void setNowPlayingInfoNative(const char *title, const char *artist, double trackDuration, NSImage *artworkImage, double elapsedTime) {

    // TODO: the remove command center logic needs to move
    MPRemoteCommandCenter *commandCenter = [MPRemoteCommandCenter sharedCommandCenter];
    [commandCenter.playCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        // TODO: Handle play command
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [commandCenter.pauseCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        // TODO: Handle pause command
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [commandCenter.togglePlayPauseCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        // TODO: Handle toggle command
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    MPNowPlayingInfoCenter *infoCenter = [MPNowPlayingInfoCenter defaultCenter];

    NSDictionary *nowPlayingInfo = @{
        MPMediaItemPropertyTitle: [NSString stringWithUTF8String:title],
        MPMediaItemPropertyArtist: [NSString stringWithUTF8String:artist],
        MPMediaItemPropertyPlaybackDuration: @(trackDuration),
        MPNowPlayingInfoPropertyElapsedPlaybackTime: @(elapsedTime)
    };
    
    infoCenter.nowPlayingInfo = nowPlayingInfo;
    infoCenter.playbackState = MPNowPlayingPlaybackStatePlaying;
}

/**
 * 'C' bridge function for 'setNowPlayingInfoNative'
 */
void setNowPlayingInfo(const char *title, const char *artist) {
    const char *imagePath = "/path/to/image.png";
    double trackDuration = 300.0;
    double elapsedTime = 120.0;

    setNowPlayingInfoNative(title, artist, trackDuration, NULL, elapsedTime);
}
