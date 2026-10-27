#import "native/sc_capture.h"

#import <AppKit/AppKit.h>
#import <CoreMedia/CoreMedia.h>
#import <CoreVideo/CoreVideo.h>
#import <ScreenCaptureKit/ScreenCaptureKit.h>
#import <dispatch/dispatch.h>

static SCStream *g_stream = nil;
static id g_output = nil;

@interface SCFrameOutput : NSObject <SCStreamOutput>
@property(nonatomic, assign) sc_frame_callback callback;
@property(nonatomic, assign) void *userData;
@end

@implementation SCFrameOutput

- (void)stream:(SCStream *)stream
    didOutputSampleBuffer:(CMSampleBufferRef)sampleBuffer
                   ofType:(SCStreamOutputType)type {
    if (type != SCStreamOutputTypeScreen || self.callback == NULL) {
        return;
    }
    CVImageBufferRef imageBuffer = CMSampleBufferGetImageBuffer(sampleBuffer);
    if (imageBuffer == NULL) {
        return;
    }

    CVPixelBufferLockBaseAddress(imageBuffer, kCVPixelBufferLock_ReadOnly);
    size_t width = CVPixelBufferGetWidth(imageBuffer);
    size_t height = CVPixelBufferGetHeight(imageBuffer);
    size_t bytesPerRow = CVPixelBufferGetBytesPerRow(imageBuffer);
    uint8_t *base = CVPixelBufferGetBaseAddress(imageBuffer);
    if (base == NULL) {
        CVPixelBufferUnlockBaseAddress(imageBuffer, kCVPixelBufferLock_ReadOnly);
        return;
    }

    size_t length = bytesPerRow * height;
    CMTime pts = CMSampleBufferGetPresentationTimeStamp(sampleBuffer);
    int64_t pts_ns = (int64_t)(CMTimeGetSeconds(pts) * 1e9);

    CVPixelBufferUnlockBaseAddress(imageBuffer, kCVPixelBufferLock_ReadOnly);

    self.callback(self.userData, base, length, (uint32_t)width, (uint32_t)height, pts_ns, NULL);
}

@end

static void sc_copy_error(NSError *error, char *errbuf, size_t errlen) {
    if (errbuf == NULL || errlen == 0) {
        return;
    }
    errbuf[0] = '\0';
    if (error == nil) {
        return;
    }
    NSString *msg = error.localizedDescription ?: @"unknown error";
    strncpy(errbuf, msg.UTF8String, errlen - 1);
    errbuf[errlen - 1] = '\0';
}

int sc_list_displays(sc_display_info_t *out, int max_out, char *errbuf, size_t errlen) {
    if (out == NULL || max_out <= 0) {
        if (errbuf && errlen > 0) {
            strncpy(errbuf, "invalid arguments", errlen - 1);
        }
        return -1;
    }

    __block int count = 0;
    __block NSError *blockError = nil;
    dispatch_semaphore_t sem = dispatch_semaphore_create(0);

    [SCShareableContent getShareableContentWithCompletionHandler:^(SCShareableContent *content, NSError *error) {
        if (error != nil) {
            blockError = error;
            dispatch_semaphore_signal(sem);
            return;
        }
        for (SCDisplay *display in content.displays) {
            if (count >= max_out) {
                break;
            }
            sc_display_info_t *slot = &out[count];
            slot->display_id = (uint32_t)display.displayID;
            slot->width = (int32_t)display.width;
            slot->height = (int32_t)display.height;
            snprintf(slot->name, sizeof(slot->name), "Display %u", slot->display_id);
            count++;
        }
        dispatch_semaphore_signal(sem);
    }];

    dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);
    if (blockError != nil) {
        sc_copy_error(blockError, errbuf, errlen);
        return -1;
    }
    return count;
}

int sc_stream_start(
    uint32_t display_id,
    int width,
    int height,
    int fps,
    sc_frame_callback callback,
    void *user_data,
    char *errbuf,
    size_t errlen) {
    if (callback == NULL || fps <= 0) {
        if (errbuf && errlen > 0) {
            strncpy(errbuf, "invalid stream arguments", errlen - 1);
        }
        return -1;
    }

    sc_stream_stop();

    __block NSError *blockError = nil;
    dispatch_semaphore_t sem = dispatch_semaphore_create(0);

    [SCShareableContent getShareableContentWithCompletionHandler:^(SCShareableContent *content, NSError *error) {
        if (error != nil) {
            blockError = error;
            dispatch_semaphore_signal(sem);
            return;
        }

        SCDisplay *target = nil;
        for (SCDisplay *display in content.displays) {
            if (display.displayID == display_id) {
                target = display;
                break;
            }
        }
        if (target == nil && content.displays.count > 0) {
            target = content.displays.firstObject;
        }
        if (target == nil) {
            blockError = [NSError errorWithDomain:@"sc_capture" code:2 userInfo:@{NSLocalizedDescriptionKey : @"no displays available"}];
            dispatch_semaphore_signal(sem);
            return;
        }

        SCContentFilter *filter = [[SCContentFilter alloc] initWithDisplay:target excludingWindows:@[]];
        SCStreamConfiguration *config = [[SCStreamConfiguration alloc] init];
        if (width > 0 && height > 0) {
            config.width = width;
            config.height = height;
        } else {
            config.width = (size_t)target.width;
            config.height = (size_t)target.height;
        }
        config.pixelFormat = kCVPixelFormatType_32BGRA;
        config.showsCursor = YES;
        config.minimumFrameInterval = CMTimeMake(1, (int32_t)fps);
        config.queueDepth = 3;

        SCStream *stream = [[SCStream alloc] initWithFilter:filter configuration:config delegate:nil];
        SCFrameOutput *output = [[SCFrameOutput alloc] init];
        output.callback = callback;
        output.userData = user_data;

        NSError *addError = nil;
        [stream addStreamOutput:output type:SCStreamOutputTypeScreen sampleHandlerQueue:dispatch_get_main_queue() error:&addError];
        if (addError != nil) {
            blockError = addError;
            dispatch_semaphore_signal(sem);
            return;
        }

        [stream startCaptureWithCompletionHandler:^(NSError *startError) {
            if (startError != nil) {
                blockError = startError;
            } else {
                g_stream = stream;
                g_output = output;
            }
            dispatch_semaphore_signal(sem);
        }];
    }];

    dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);

    if (blockError != nil) {
        sc_copy_error(blockError, errbuf, errlen);
        return -1;
    }
    return 0;
}

void sc_stream_stop(void) {
    if (g_stream != nil) {
        dispatch_semaphore_t sem = dispatch_semaphore_create(0);
        SCStream *stream = g_stream;
        g_stream = nil;
        g_output = nil;
        [stream stopCaptureWithCompletionHandler:^(NSError *error) {
            dispatch_semaphore_signal(sem);
        }];
        dispatch_semaphore_wait(sem, DISPATCH_TIME_FOREVER);
    }
}
