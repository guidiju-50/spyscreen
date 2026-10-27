#include "native/bridge.h"

extern void goDeliverFrame(
    void *userData,
    const uint8_t *bgra,
    size_t length,
    uint32_t width,
    uint32_t height,
    int64_t pts_ns,
    char *error_msg
);

static void deliver_bridge(
    void *userData,
    const uint8_t *bgra,
    size_t length,
    uint32_t width,
    uint32_t height,
    int64_t pts_ns,
    const char *error_msg
) {
    goDeliverFrame(userData, bgra, length, width, height, pts_ns, (char *)error_msg);
}

sc_frame_callback sc_get_frame_callback(void) { return deliver_bridge; }
