#import "GopeedTaskEventForwarder.h"

#import <Libgopeed/Libgopeed.h>

#import "Runner-Swift.h"

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


@interface GopeedNativeInvokeResultForwarder
    : NSObject <LibgopeedInvokeResultListener>

@property(nonatomic, copy, nullable)
    GopeedNativeInvokeCompletion completion;

- (instancetype)initWithCompletion:
    (GopeedNativeInvokeCompletion)completion;

@end

@implementation GopeedNativeInvokeResultForwarder

- (instancetype)initWithCompletion:
    (GopeedNativeInvokeCompletion)completion {
  self = [super init];
  if (self) {
    _completion = [completion copy];
  }
  return self;
}

- (void)onResult:(int64_t)requestID
         success:(BOOL)success
         payload:(NSString * _Nullable)payload {
  GopeedNativeInvokeCompletion completion =
      self.completion;
  self.completion = nil;

  if (completion == nil) {
    return;
  }

  completion(success, payload ?: @"");
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

  NSData *data =
      [arguments dataUsingEncoding:NSUTF8StringEncoding];

  NSDictionary *json = nil;

  if (data) {
    json =
        [NSJSONSerialization JSONObjectWithData:data
                                        options:0
                                          error:nil];
  }

  NSString *type = json[@"type"];
  NSString *taskID = json[@"taskId"];

  BOOL continuedProcessingHandlesTask = NO;

  // iOS 26 system continued-processing Live Activity.
  if (@available(iOS 26.0, *)) {
    GopeedContinuedProcessingManager *manager =
        [GopeedContinuedProcessingManager shared];

    [manager handleTaskEventPayload:arguments];

    if (taskID.length > 0) {
      continuedProcessingHandlesTask =
          [manager isHandlingTaskId:taskID];
    }
  }

  // When BGCPT is active, let Apple's system Live Activity
  // represent progress. Otherwise keep using our existing
  // custom Gopeed ActivityKit implementation.
  if (!continuedProcessingHandlesTask) {
    [[GopeedLiveActivityManager shared]
        handleTaskEventPayload:arguments];
  }

  // Flutter currently understands only done/error.
  BOOL flutterEvent =
      [type isEqualToString:@"task.done"] ||
      [type isEqualToString:@"task.error"];

  if (!flutterEvent) {
    return;
  }

  FlutterMethodChannel *channel = self.channel;

  dispatch_async(dispatch_get_main_queue(), ^{
    [channel invokeMethod:@"taskEvent"
                arguments:arguments];
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

void GopeedInvokeAsyncNative(
    NSString *method,
    NSString *path,
    NSString *query,
    NSString *body,
    GopeedNativeInvokeCompletion completion) {
  if (completion == nil) {
    return;
  }

  GopeedNativeInvokeResultForwarder *forwarder =
      [[GopeedNativeInvokeResultForwarder alloc]
          initWithCompletion:completion];

  LibgopeedInvokeAsync(
      method,
      path,
      query,
      body,
      0,
      forwarder);
}
