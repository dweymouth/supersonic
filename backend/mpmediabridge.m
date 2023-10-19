#import "mpmediabridge.h"

/**
 * C bridge registering callbacks for media playback events using the native CommandCenter API.
 */
void register_os_remote_commands() {
    MPRemoteCommandCenter *commandCenter = [MPRemoteCommandCenter sharedCommandCenter];
    [commandCenter.playCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        os_remote_command_callback(PLAY);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [commandCenter.pauseCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        os_remote_command_callback(PAUSE);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [commandCenter.togglePlayPauseCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        os_remote_command_callback(TOGGLE);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [commandCenter.stopCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        os_remote_command_callback(STOP);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [commandCenter.nextTrackCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        os_remote_command_callback(NEXT_TRACK);
        return MPRemoteCommandHandlerStatusSuccess;
    }];

    [commandCenter.previousTrackCommand addTargetWithHandler:^MPRemoteCommandHandlerStatus(MPRemoteCommandEvent * _Nonnull event) {
        os_remote_command_callback(PREVIOUS_TRACK);
        return MPRemoteCommandHandlerStatusSuccess;
    }];
}

/**
 * C bridge setting "Now Playing" information on macOS for media playback using the native APIs.
 */
void set_os_now_playing_info(const char *title, const char *artist, const char *coverArtFileURL) {
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
        MPMediaItemPropertyArtwork: coverArt 
    };
    
    infoCenter.nowPlayingInfo = nowPlayingInfo;
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
