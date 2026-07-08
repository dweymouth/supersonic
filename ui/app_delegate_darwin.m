//go:build darwin

#import <AppKit/AppKit.h>

extern void appReopened(void);
extern void appShouldTerminate(void);

@interface ReopenDelegate : NSObject<NSApplicationDelegate>
@property (nonatomic, strong) id<NSApplicationDelegate> wrapped;
@end


@implementation ReopenDelegate

- (NSApplicationTerminateReply)applicationShouldTerminate:(NSApplication *)sender {
    // Set the quitting flag before GLFW's delegate fires close requests for
    // each window, so our close intercept knows to close rather than hide.
    appShouldTerminate();
    if ([self.wrapped respondsToSelector:_cmd]) {
        return [self.wrapped applicationShouldTerminate:sender];
    }
    return NSTerminateNow;
}

- (BOOL)applicationShouldHandleReopen:(NSApplication *)app
                    hasVisibleWindows:(BOOL)hasVisible {
    if (!hasVisible) {
        appReopened();
    }
    if ([self.wrapped respondsToSelector:_cmd]) {
        return [self.wrapped applicationShouldHandleReopen:app
                                        hasVisibleWindows:hasVisible];
    }
    return YES;
}

- (BOOL)respondsToSelector:(SEL)sel {
    return [super respondsToSelector:sel] ||
           [self.wrapped respondsToSelector:sel];
}

- (id)forwardingTargetForSelector:(SEL)sel {
    return [self.wrapped respondsToSelector:sel] ? self.wrapped : nil;
}

@end

void installReopenDelegate(void) {
    ReopenDelegate *d = [ReopenDelegate new];
    d.wrapped = [NSApp delegate];
    [NSApp setDelegate:d];
}
