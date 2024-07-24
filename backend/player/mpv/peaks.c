#include <mpv/client.h>
#include <stdlib.h>
#include <string.h>

int mpv_get_peaks(mpv_handle* handle, double* lPeak, double* rPeak, double* lRMS, double* rRMS) {
    mpv_node result;
    int ret = mpv_get_property(handle, "af-metadata/astats", MPV_FORMAT_NODE, &result);
    if (ret != MPV_ERROR_SUCCESS) {
        return ret;
    }
    if (result.format != MPV_FORMAT_NODE_MAP) {
        mpv_free_node_contents(&result);
        return MPV_ERROR_PROPERTY_FORMAT;
    }

    int found = 0;
    for (int i = 0; found < 4 && i < result.u.list->num; i++) {
        if (strcmp("lavfi.astats.1.Peak_level", result.u.list->keys[i]) == 0) {
            if (result.u.list->values[i].format != MPV_FORMAT_STRING) {
                ret = MPV_ERROR_PROPERTY_FORMAT;
                break;
            }
            *lPeak = atof(result.u.list->values[i].u.string);
            found++;
        }
        if (strcmp("lavfi.astats.2.Peak_level", result.u.list->keys[i]) == 0) {
            if (result.u.list->values[i].format != MPV_FORMAT_STRING) {
                ret = MPV_ERROR_PROPERTY_FORMAT;
                break;
            }
            *rPeak = atof(result.u.list->values[i].u.string);
            found++;
        }
        if (strcmp("lavfi.astats.1.RMS_level", result.u.list->keys[i]) == 0) {
            if (result.u.list->values[i].format != MPV_FORMAT_STRING) {
                ret = MPV_ERROR_PROPERTY_FORMAT;
                break;
            }
            *lRMS = atof(result.u.list->values[i].u.string);
            found++;
        }
        if (strcmp("lavfi.astats.2.RMS_level", result.u.list->keys[i]) == 0) {
            if (result.u.list->values[i].format != MPV_FORMAT_STRING) {
                ret = MPV_ERROR_PROPERTY_FORMAT;
                break;
            }
            *rRMS = atof(result.u.list->values[i].u.string);
            found++;
        }
    }

    mpv_free_node_contents(&result);
    return ret;
}
