//go:build darwin

#import "mpmediabridge.h"

/**
 * C bridge registering callbacks for media playback events using the native CommandCenter API.
 */
void register_os_remote_commands() {
    MPRemoteCommandCenter *commandCenter = [MPRemoteCommandCenter sharedCommandCenter];
    [commandCenter.playCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        os_remote_command_callback(PLAY, 0);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [commandCenter.pauseCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        os_remote_command_callback(PAUSE, 0);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [commandCenter.togglePlayPauseCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        os_remote_command_callback(TOGGLE, 0);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [commandCenter.stopCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        os_remote_command_callback(STOP, 0);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [commandCenter.nextTrackCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        os_remote_command_callback(NEXT_TRACK, 0);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [commandCenter.previousTrackCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        os_remote_command_callback(PREVIOUS_TRACK, 0);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [commandCenter.changePlaybackPositionCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        MPChangePlaybackPositionCommandEvent *positionChangeEvent = (MPChangePlaybackPositionCommandEvent *)event;
        double posSecDouble = positionChangeEvent.positionTime;
        int posSeconds = (int)posSecDouble; 
        os_remote_command_callback(SEEK, posSeconds);
        return MPRemoteCommandHandlerStatusSuccess;
    }];
}

/**
 * C bridge setting "Now Playing" information on macOS for media playback using the native APIs.
 */
void set_os_now_playing_info(const char *title, const char *artist, const char *coverArtFileURL, double trackDuration) {
    NSString *coverArtLocationString = [NSString stringWithUTF8String:coverArtFileURL];
    NSURL *coverArtURL = [NSURL URLWithString:coverArtLocationString];
    NSImage *coverArtImage = [[NSImage alloc] initWithContentsOfURL:coverArtURL];

    MPMediaItemArtwork *coverArt = [[MPMediaItemArtwork alloc] initWithBoundsSize:coverArtImage.size requestHandler:^NSImage * _Nonnull(CGSize size) {
        return coverArtImage;
    }];

    MPNowPlayingInfoCenter *infoCenter = [MPNowPlayingInfoCenter defaultCenter];
    NSDictionary *nowPlayingInfo = @{
        MPMediaItemPropertyTitle: [NSString stringWithUTF8String:title],
        MPMediaItemPropertyArtist: [NSString stringWithUTF8String:artist],
        MPMediaItemPropertyArtwork: coverArt,
        MPMediaItemPropertyPlaybackDuration: @(trackDuration) // Expects 'NSNumber'
    };
    
    infoCenter.nowPlayingInfo = nowPlayingInfo;
}

/**
 * C bridge updating the OS playback position.
 * creates a mutable copy of the immutable dictionary and writes it back with updated position.
 */
void update_os_now_playing_info_position(double positionSeconds) {
    MPNowPlayingInfoCenter *infoCenter = [MPNowPlayingInfoCenter defaultCenter];
    NSMutableDictionary *updatedInfo = [infoCenter.nowPlayingInfo mutableCopy];
    updatedInfo[MPNowPlayingInfoPropertyElapsedPlaybackTime] = @(positionSeconds);
    infoCenter.nowPlayingInfo = [updatedInfo copy];
}

/**
 * C bridge setting the OS playback state to 'playing'.
 */
void set_os_playback_state_playing() {
    MPNowPlayingInfoCenter *infoCenter = [MPNowPlayingInfoCenter defaultCenter];
    infoCenter.playbackState = MPNowPlayingPlaybackStatePlaying;
}

/**
 * C bridge setting the OS playback state to 'paused'.
 */
void set_os_playback_state_paused() {
    MPNowPlayingInfoCenter *infoCenter = [MPNowPlayingInfoCenter defaultCenter];
    infoCenter.playbackState = MPNowPlayingPlaybackStatePaused;
}

/**
 * C bridge setting the OS playback state to 'stopped'.
 */
void set_os_playback_state_stopped() {
    MPNowPlayingInfoCenter *infoCenter = [MPNowPlayingInfoCenter defaultCenter];
    infoCenter.playbackState = MPNowPlayingPlaybackStateStopped;
}
