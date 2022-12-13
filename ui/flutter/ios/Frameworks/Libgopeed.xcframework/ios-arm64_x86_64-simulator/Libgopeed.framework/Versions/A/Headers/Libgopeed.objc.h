// Objective-C API for talking to github.com/monkeyWie/gopeed/bind/mobile Go package.
//   gobind -lang=objc github.com/monkeyWie/gopeed/bind/mobile
//
// File is generated by gobind. Do not edit.

#ifndef __Libgopeed_H__
#define __Libgopeed_H__

@import Foundation;
#include "ref.h"
#include "Universe.objc.h"


FOUNDATION_EXPORT BOOL LibgopeedStart(NSString* _Nullable cfg, long* _Nullable ret0_, NSError* _Nullable* _Nullable error);

FOUNDATION_EXPORT void LibgopeedStop(void);

#endif
