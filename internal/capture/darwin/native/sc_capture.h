#ifndef SC_CAPTURE_H
#define SC_CAPTURE_H

#include <stddef.h>
#include <stdint.h>

typedef struct {
    uint32_t display_id;
    int32_t width;
    int32_t height;
    char name[256];
} sc_display_info_t;

typedef void (*sc_frame_callback)(
    void *user_data,
    const uint8_t *bgra,
    size_t length,
    uint32_t width,
    uint32_t height,
    int64_t pts_ns,
    const char *error_msg
);

int sc_list_displays(sc_display_info_t *out, int max_out, char *errbuf, size_t errlen);

int sc_stream_start(
    uint32_t display_id,
    int width,
    int height,
    int fps,
    sc_frame_callback callback,
    void *user_data,
    char *errbuf,
    size_t errlen
);

void sc_stream_stop(void);

#endif
