#import "GopeedTaskEventForwarder.h"

#import <Libgopeed/Libgopeed.h>

@interface GopeedTaskEventForwarder () <LibgopeedTaskEventListener>

@property(nonatomic, strong) FlutterMethodChannel *channel;

@end

@interface GopeedInvokeResultForwarder : NSObject <LibgopeedInvokeResultListener>

@property(nonatomic, copy, nullable) FlutterResult result;

- (instancetype)initWithResult:(FlutterResult)result;

@end

@implementation GopeedInvokeResultForwarder

- (instancetype)initWithResult:(FlutterResult)result {
  self = [super init];
  if (self) {
    _result = [result copy];
  }
  return self;
}

- (void)onResult:(int64_t)requestID
         success:(BOOL)success
         payload:(NSString * _Nullable)payload {
  FlutterResult result = self.result;
  self.result = nil;
  if (result == nil) {
    return;
  }
  dispatch_async(dispatch_get_main_queue(), ^{
    if (success) {
      result(payload ?: @"");
    } else {
      result([FlutterError errorWithCode:@"ERROR"
                                 message:payload ?: @"InvokeAsync failed"
                                 details:nil]);
    }
  });
}

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

void GopeedInvokeAsyncWithResult(
    NSString *method,
    NSString *path,
    NSString *query,
    NSString *body,
    int64_t requestID,
    FlutterResult result) {
  GopeedInvokeResultForwarder *forwarder =
      [[GopeedInvokeResultForwarder alloc] initWithResult:result];
  LibgopeedInvokeAsync(method, path, query, body, requestID, forwarder);
}
