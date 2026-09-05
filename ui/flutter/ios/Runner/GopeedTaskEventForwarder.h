#import <Flutter/Flutter.h>

NS_ASSUME_NONNULL_BEGIN

@interface GopeedTaskEventForwarder : NSObject

- (instancetype)initWithChannel:(FlutterMethodChannel *)channel NS_DESIGNATED_INITIALIZER;
- (instancetype)init NS_UNAVAILABLE;

@end

FOUNDATION_EXPORT void GopeedSubscribeTaskEventsWithForwarder(
    int64_t mask,
    GopeedTaskEventForwarder * _Nullable listener);

NS_ASSUME_NONNULL_END
