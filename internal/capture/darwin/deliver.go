//go:build darwin

package darwin

/*
#include <stdint.h>
#include <stdlib.h>
*/
import "C"

import (
	"time"
	"unsafe"
)

//export goDeliverFrame
func goDeliverFrame(_ unsafe.Pointer, bgra *C.uchar, length C.size_t, width C.uint32_t, height C.uint32_t, ptsNs C.int64_t, errorMsg *C.char) {
	if errorMsg != nil {
		return
	}
	if bgra == nil || length == 0 {
		return
	}

	w := int(width)
	h := int(height)
	bytesPerRow := w * 4
	dataLen := int(length)
	raw := C.GoBytes(unsafe.Pointer(bgra), C.int(dataLen))

	var data []byte
	if dataLen == bytesPerRow*h {
		data = raw
	} else {
		rowBytes := dataLen / h
		if rowBytes < bytesPerRow {
			return
		}
		data = make([]byte, bytesPerRow*h)
		for y := 0; y < h; y++ {
			copy(data[y*bytesPerRow:(y+1)*bytesPerRow], raw[y*rowBytes:y*rowBytes+bytesPerRow])
		}
	}

	activeMu.Lock()
	cap := activeCapturer
	activeMu.Unlock()
	if cap == nil {
		return
	}
	cap.deliverFrame(data, w, h, time.Unix(0, int64(ptsNs)))
}
