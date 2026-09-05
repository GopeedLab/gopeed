#import "GopeedTaskEventForwarder.h"

#import <Libgopeed/Libgopeed.h>

@interface GopeedTaskEventForwarder () <LibgopeedTaskEventListener>

@property(nonatomic, strong) FlutterMethodChannel *channel;

@end

@implementation GopeedTaskEventForwarder

- (instancetype)initWithChannel:(FlutterMethodChannel *)channel {
  self = [super init];
  if (self) {
    _channel = channel;
  }
  return self;
}

- (void)onTaskEvent:(NSString * _Nullable)payload {
  NSString *arguments = payload ?: @"";
  FlutterMethodChannel *channel = self.channel;
  dispatch_async(dispatch_get_main_queue(), ^{
    [channel invokeMethod:@"taskEvent" arguments:arguments];
  });
}

@end

void GopeedSubscribeTaskEventsWithForwarder(
    int64_t mask,
    GopeedTaskEventForwarder * _Nullable listener) {
  LibgopeedSubscribeTaskEvents(mask, listener);
}
