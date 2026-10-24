#include "yamlstar_plugin.h"

#include <dlfcn.h>
#include <pthread.h>
#include <stdio.h>
#include <stdlib.h>
#include <string.h>

#ifndef PLUGIN_EXTENSION
#define PLUGIN_EXTENSION "so"
#endif

#ifndef PLUGIN_VERSION
#define PLUGIN_VERSION "0.1.5"
#endif

typedef uint64_t (*abi_fn)(void);
typedef int32_t (*manifest_fn)(uint8_t **, size_t *);
typedef int32_t (*parse_fn)(const uint8_t *, size_t, const uint8_t *,
                           size_t, uint8_t **, size_t *);
typedef void (*free_fn)(uint8_t *);

struct api {
    parse_fn transform;
    free_fn free_output;
};

static void fail(const char *message) {
    fprintf(stderr, "%s\n", message);
    exit(1);
}

static char *take_output(
    struct api *api,
    uint8_t *output,
    size_t length
) {
    char *text = malloc(length + 1);
    if (text == NULL) {
        fail("malloc failed");
    }
    memcpy(text, output, length);
    text[length] = '\0';
    api->free_output(output);
    return text;
}

static char *parse(struct api *api, const char *input, int32_t *status) {
    static const char options[] = "{}";
    uint8_t *output = NULL;
    size_t length = 0;
    *status = api->transform(
        (const uint8_t *) input,
        strlen(input),
        (const uint8_t *) options,
        strlen(options),
        &output,
        &length);
    if (*status == 0 && (length < 16 || memcmp(output, "[{", 2))) {
        fail("invalid EDN event response");
    }
    return take_output(api, output, length);
}

static void *parse_repeatedly(void *argument) {
    struct api *api = argument;
    for (int attempt = 0; attempt < 10; attempt++) {
        int32_t status;
        char *output = parse(api, "value: true # comment\n", &status);
        if (status != 0 || memcmp(output, "[{", 2) != 0) {
            fail("concurrent parse failed");
        }
        free(output);
        uint8_t *binary = NULL;
        size_t length = 0;
        status = api->transform((const uint8_t *) "a: b", 4,
                                   (const uint8_t *) "{}", 2,
                                   &binary, &length);
        if (status != 0 || length < 16 || memcmp(binary, "[{", 2)) {
            fail("concurrent binary parse failed");
        }
        api->free_output(binary);
    }
    return NULL;
}

int main(void) {
    const char *directory = getenv("YAMLSTAR_LIBRARY_PATH");
    if (directory == NULL) {
        fail("YAMLSTAR_LIBRARY_PATH is not set");
    }
    char path[4096];
    snprintf(path, sizeof(path),
             "%s/libyamlstar-plugin-parser-rapidyaml.%s",
             directory, PLUGIN_EXTENSION);
    void *handle = dlopen(path, RTLD_NOW | RTLD_LOCAL);
    if (handle == NULL) {
        fail(dlerror());
    }

    abi_fn abi = (abi_fn) dlsym(handle, "yamlstar_plugin_v2_abi");
    manifest_fn manifest = (manifest_fn) dlsym(
        handle, "yamlstar_plugin_v2_manifest");
    struct api api = {
        .transform = (parse_fn) dlsym(
            handle, "yamlstar_plugin_v2_transform"),
        .free_output = (free_fn) dlsym(
            handle, "yamlstar_plugin_v2_free"),
    };
    if (abi == NULL || manifest == NULL
        || api.transform == NULL
        || api.free_output == NULL) {
        fail("plugin ABI symbol is missing");
    }
    if (dlsym(handle, "yamlstar_plugin_v1_parse_binary") != NULL) {
        fail("plugin still exports ABI v1 parsing");
    }
    if (dlsym(handle, "crosscall2") != NULL) {
        fail("plugin still embeds a Go runtime");
    }
    if (abi() != 2) {
        fail("plugin ABI version is not 2");
    }

    uint8_t *manifest_output = NULL;
    size_t manifest_length = 0;
    if (manifest(&manifest_output, &manifest_length) != 0) {
        fail("manifest call failed");
    }
    char *manifest_text = take_output(
        &api, manifest_output, manifest_length);
    if (strstr(manifest_text, ":api \"yaml-parser\"") == NULL) {
        fail("manifest API is incorrect");
    }
    if (strstr(manifest_text, ":version \"" PLUGIN_VERSION "\"")
        == NULL) {
        fail("manifest version is incorrect");
    }
    free(manifest_text);
    if (manifest(NULL, &manifest_length) != 2
        || manifest(&manifest_output, NULL) != 2) {
        fail("manifest accepted a null output pointer");
    }

    int32_t status;
    char *output = parse(
        &api,
        "a: true// comment\nb: foo// not a comment\n"
        "c: foo // comment\nd: http://not-a-comment.com\n",
        &status);
    if (status != 0 || memcmp(output, "[{", 2) != 0) {
        fail("parse result is incorrect");
    }
    free(output);

    output = parse(&api, "[unterminated", &status);
    if (status != 1 || strstr(output, "missing terminating") == NULL) {
        fail("parse error result is incorrect");
    }
    free(output);

    uint8_t *bad_output = NULL;
    size_t bad_length = 0;
    status = api.transform(NULL, 1, NULL, 0, &bad_output, &bad_length);
    if (status != 2) {
        fail("binary ABI accepted a nil input with nonzero length");
    }
    api.free_output(bad_output);

    static const uint8_t invalid_utf8[] = {0xff};
    status = api.transform(invalid_utf8, sizeof(invalid_utf8), NULL, 0,
                           &bad_output, &bad_length);
    if (status != 1) {
        fail("invalid UTF-8 did not return a parse error");
    }
    api.free_output(bad_output);
    static const char unknown_options[] = "{:unknown true}";
    status = api.transform(NULL, 0, (const uint8_t *)unknown_options,
                           sizeof(unknown_options) - 1,
                           &bad_output, &bad_length);
    if (status != 1) {
        fail("unknown options did not return a configuration error");
    }
    api.free_output(bad_output);

    api.free_output(NULL);
    status = api.transform(NULL, 0, NULL, 0, &bad_output, &bad_length);
    if (status != 0 || bad_length < 16) {
        fail("empty input failed");
    }
    api.free_output(bad_output);
    status = api.transform(NULL, 0, NULL, 0, NULL, &bad_length);
    if (status != 2) {
        fail("null output pointer accepted");
    }
    static const char invalid[] = "[untermi\rated";
    status = api.transform((const uint8_t *)invalid, sizeof(invalid) - 1,
                             NULL, 0, &bad_output, &bad_length);
    if (status == 0 || bad_length == 0) {
        fail("assertion regression did not return an error");
    }
    api.free_output(bad_output);

    pthread_t threads[4];
    for (int index = 0; index < 4; index++) {
        if (pthread_create(&threads[index], NULL,
                           parse_repeatedly, &api) != 0) {
            fail("pthread_create failed");
        }
    }
    for (int index = 0; index < 4; index++) {
        if (pthread_join(threads[index], NULL) != 0) {
            fail("pthread_join failed");
        }
    }

    puts("shared plugin ABI test passed");
    return 0;
}
