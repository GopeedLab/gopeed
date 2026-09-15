#import <Flutter/Flutter.h>

NS_ASSUME_NONNULL_BEGIN

@interface GopeedTaskEventForwarder : NSObject

- (instancetype)initWithChannel:(FlutterMethodChannel *)channel NS_DESIGNATED_INITIALIZER;
- (instancetype)init NS_UNAVAILABLE;

@end

FOUNDATION_EXPORT void GopeedSubscribeTaskEventsWithForwarder(
    int64_t mask,
    GopeedTaskEventForwarder * _Nullable listener);


typedef void (^GopeedNativeInvokeCompletion)(
    BOOL success,
    NSString *payload);

FOUNDATION_EXPORT void GopeedInvokeAsyncNative(
    NSString *method,
    NSString *path,
    NSString *query,
    NSString *body,
    GopeedNativeInvokeCompletion completion);

FOUNDATION_EXPORT void GopeedInvokeAsyncWithResult(
    NSString *method,
    NSString *path,
    NSString *query,
    NSString *body,
    int64_t requestID,
    FlutterResult result);

NS_ASSUME_NONNULL_END
