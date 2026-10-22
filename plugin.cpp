#include "include/yamlstar_plugin.h"
#include "rymlshim.h"
#include ".cache/native-manifest.h"

#include <array>
#include <cstdlib>
#include <cstring>
#include <map>
#include <stdexcept>
#include <string>

namespace {

struct InputError : std::runtime_error {
    using std::runtime_error::runtime_error;
};

bool valid_utf8(const std::string &text)
{
    const auto *bytes = reinterpret_cast<const unsigned char*>(text.data());
    for(size_t i = 0; i < text.size(); )
    {
        const unsigned char first = bytes[i++];
        if(first < 0x80)
            continue;
        size_t extra;
        unsigned char low = 0x80, high = 0xbf;
        if(first >= 0xc2 && first <= 0xdf)
            extra = 1;
        else if(first >= 0xe0 && first <= 0xef)
        {
            extra = 2;
            if(first == 0xe0) low = 0xa0;
            if(first == 0xed) high = 0x9f;
        }
        else if(first >= 0xf0 && first <= 0xf4)
        {
            extra = 3;
            if(first == 0xf0) low = 0x90;
            if(first == 0xf4) high = 0x8f;
        }
        else
            return false;
        if(extra > text.size() - i || bytes[i] < low || bytes[i] > high)
            return false;
        ++i;
        while(--extra)
        {
            if(bytes[i] < 0x80 || bytes[i] > 0xbf)
                return false;
            ++i;
        }
    }
    return true;
}

// Match the previous JSON-compatible EDN quoting, including embedded NULs,
// HTML characters, and Unicode line separators.
std::string quote_edn(const std::string &text)
{
    if(!valid_utf8(text))
        throw std::runtime_error("event string is not UTF-8");
    static const char hex[] = "0123456789abcdef";
    std::string out = "\"";
    for(size_t i = 0; i < text.size(); ++i)
    {
        const auto c = static_cast<unsigned char>(text[i]);
        switch(c)
        {
        case '"': out += "\\\""; break;
        case '\\': out += "\\\\"; break;
        case '\b': out += "\\b"; break;
        case '\f': out += "\\f"; break;
        case '\n': out += "\\n"; break;
        case '\r': out += "\\r"; break;
        case '\t': out += "\\t"; break;
        default:
            if(c < 0x20 || c == '<' || c == '>' || c == '&')
            {
                out += "\\u00";
                out += hex[c >> 4];
                out += hex[c & 15];
            }
            else if(c == 0xe2 && i + 2 < text.size() &&
                    static_cast<unsigned char>(text[i + 1]) == 0x80 &&
                    (static_cast<unsigned char>(text[i + 2]) == 0xa8 ||
                     static_cast<unsigned char>(text[i + 2]) == 0xa9))
            {
                out += text[i + 2] == static_cast<char>(0xa8) ?
                    "\\u2028" : "\\u2029";
                i += 2;
            }
            else
                out += static_cast<char>(c);
        }
    }
    return out + '"';
}

int hex_value(char c)
{
    if(c >= '0' && c <= '9') return c - '0';
    if(c >= 'a' && c <= 'f') return c - 'a' + 10;
    if(c >= 'A' && c <= 'F') return c - 'A' + 10;
    return -1;
}

using Tags = std::map<std::string, std::string>;

Tags default_tags()
{
    return {{"!!", "tag:yaml.org,2002:"}, {"!", "!"}};
}

std::string resolve_tag(std::string tag, const Tags &handles)
{
    if(tag.compare(0, 2, "!<") == 0 && tag.back() == '>')
        tag = tag.substr(2, tag.size() - 3);
    else if(!tag.empty() && tag.front() == '<' && tag.back() == '>')
        tag = tag.substr(1, tag.size() - 2);
    else if(!tag.empty() && tag.front() == '!')
    {
        const size_t end = tag.find('!', 1);
        const std::string handle = end == std::string::npos ?
            "!" : tag.substr(0, end + 1);
        const auto found = handles.find(handle);
        if(found == handles.end())
            throw std::runtime_error("undefined tag handle " + quote_edn(handle));
        tag = found->second + tag.substr(handle.size());
    }
    std::string out;
    for(size_t i = 0; i < tag.size(); ++i)
    {
        if(tag[i] != '%')
        {
            out += tag[i];
            continue;
        }
        if(i + 2 >= tag.size() || hex_value(tag[i + 1]) < 0 ||
           hex_value(tag[i + 2]) < 0)
            throw std::runtime_error("invalid tag: invalid URL escape");
        out += static_cast<char>(hex_value(tag[i + 1]) * 16 +
                                 hex_value(tag[i + 2]));
        i += 2;
    }
    return out;
}

enum Field { Value, Style, Anchor, Tag, Name, Version, FieldCount };

struct Event {
    const char *kind = nullptr;
    std::array<std::string, FieldCount> strings;
    std::array<bool, FieldCount> present{};
    int flow = -1;
    bool explicit_document = false;

    void set(Field field, const std::string &value)
    {
        strings[field] = value;
        present[field] = true;
    }

    void append(std::string *output) const
    {
        static const char *fields[] = {
            "value", "style", "anchor", "tag", "name", "version"};
        *output += "{:event " + quote_edn(kind);
        for(size_t i = 0; i < FieldCount; ++i)
            if(present[i])
                *output += std::string(" :") + fields[i] + " " +
                    quote_edn(strings[i]);
        if(flow >= 0)
            *output += flow ? " :flow true" : " :flow false";
        if(explicit_document)
            *output += " :explicit true";
        *output += "} ";
    }
};

std::string event_text(const RymlParseResult &result, size_t *index)
{
    if(*index + 2 >= result.events_len)
        throw std::runtime_error("truncated rapidyaml event");
    const int32_t bits = result.events[*index];
    const int32_t offset = result.events[++*index];
    const int32_t length = result.events[++*index];
    const bool arena = (bits & RYML_AREN) != 0;
    const size_t size = arena ? result.arena_len : result.source_len;
    const char *region = arena ? result.arena : result.source;
    if(offset < 0 || length < 0 || static_cast<size_t>(offset) > size ||
       static_cast<size_t>(length) > size - static_cast<size_t>(offset))
        throw std::runtime_error("rapidyaml string offset out of bounds");
    return length ? std::string(region + offset, length) : std::string();
}

std::string encode_events(const RymlParseResult &result)
{
    std::string output = "[", anchor, tag, version, handle;
    bool has_anchor = false, has_tag = false;
    Tags tags = default_tags();
    for(size_t i = 0; i < result.events_len; ++i)
    {
        const int32_t bits = result.events[i];
        const std::string text = bits & RYML_WSTR ?
            event_text(result, &i) : std::string();
        Event event;
        const auto all = [bits](int32_t mask) { return (bits & mask) == mask; };
        bool node = false;
        if(all(RYML_BSTR)) event.kind = "stream_start";
        else if(all(RYML_ESTR)) event.kind = "stream_end";
        else if(all(RYML_BDOC))
        {
            event.kind = "document_start";
            if(!version.empty())
            {
                event.set(Version, version);
                version.clear();
            }
            event.explicit_document = (bits & RYML_EXPL) != 0;
        }
        else if(all(RYML_EDOC))
        {
            event.kind = "document_end";
            event.explicit_document = (bits & RYML_EXPL) != 0;
            tags = default_tags();
        }
        else if(all(RYML_BMAP)) { event.kind = "mapping_start"; node = true; }
        else if(all(RYML_EMAP)) event.kind = "mapping_end";
        else if(all(RYML_BSEQ)) { event.kind = "sequence_start"; node = true; }
        else if(all(RYML_ESEQ)) event.kind = "sequence_end";
        else if(bits & RYML_YAML) { version = text; continue; }
        else if(bits & RYML_TAGH) { handle = text; continue; }
        else if(bits & RYML_TAGP) { tags[handle] = text; handle.clear(); continue; }
        else if(bits & RYML_ANCH) { anchor = text; has_anchor = true; continue; }
        else if(bits & RYML_TAG) { tag = resolve_tag(text, tags); has_tag = true; continue; }
        else if(bits & RYML_ALIA)
        {
            event.kind = "alias";
            event.set(Name, text);
        }
        else if(bits & RYML_SCLR)
        {
            event.kind = "scalar";
            event.set(Value, text);
            node = true;
            if(bits & RYML_SQUO) event.set(Style, "single");
            else if(bits & RYML_DQUO) event.set(Style, "double");
            else if(bits & RYML_LITL) event.set(Style, "literal");
            else if(bits & RYML_FOLD) event.set(Style, "folded");
        }
        else
            throw std::runtime_error("unknown rapidyaml event");
        if(node && !(bits & RYML_SCLR))
            event.flow = (bits & RYML_FLOW) != 0;
        if(node)
        {
            if(has_anchor) { event.set(Anchor, anchor); has_anchor = false; }
            if(has_tag) { event.set(Tag, tag); has_tag = false; }
        }
        event.append(&output);
    }
    if(has_anchor || has_tag || !version.empty() || !handle.empty())
        throw std::runtime_error("incomplete rapidyaml metadata");
    return output + ']';
}

struct ParseResult {
    RymlParseResult value;
    ~ParseResult() { yamlstar_ryml_free_result(&value); }
};

std::string parse_edn(const std::string &input, const std::string &options)
{
    std::string compact;
    for(char c : options)
        if(c != ' ' && c != '\t' && c != '\r' && c != '\n' && c != ',')
            compact += c;
    if(!compact.empty() && compact != "{}")
        throw InputError("rapidyaml does not support parser options");
    if(!valid_utf8(input))
        throw InputError("input is not UTF-8");
    std::string normalized;
    normalized.reserve(input.size());
    for(size_t i = 0; i < input.size(); ++i)
    {
        if(input[i] != '\r') normalized += input[i];
        else
        {
            normalized += '\n';
            if(i + 1 < input.size() && input[i + 1] == '\n') ++i;
        }
    }
    ParseResult result{yamlstar_ryml_parse_events(
        normalized.data(), normalized.size())};
    if(result.value.ok <= 0)
    {
        const std::string message = result.value.error ? result.value.error :
            "rapidyaml internal allocation failure";
        if(result.value.ok < 0) throw std::runtime_error(message);
        throw InputError(message);
    }
    return encode_events(result.value);
}

int32_t write_output(const std::string &text, uint8_t **output, size_t *length,
                     int32_t status = 0)
{
    if(!output || !length) return 2;
    *output = nullptr;
    *length = 0;
    if(!text.empty())
    {
        *output = static_cast<uint8_t*>(malloc(text.size()));
        if(!*output) return 2;
        memcpy(*output, text.data(), text.size());
    }
    *length = text.size();
    return status;
}

} // namespace

extern "C" uint64_t yamlstar_plugin_v2_abi(void) { return 2; }

extern "C" void yamlstar_plugin_v2_free(uint8_t *output) { free(output); }

extern "C" int32_t yamlstar_plugin_v2_manifest(uint8_t **output, size_t *length)
{
    if(!output || !length) return 2;
    *output = nullptr;
    *length = 0;
    try { return write_output(plugin_manifest, output, length); }
    catch(...) { return 2; }
}

extern "C" int32_t yamlstar_plugin_v2_transform(
    const uint8_t *input, size_t input_length,
    const uint8_t *options, size_t options_length,
    uint8_t **output, size_t *output_length)
{
    if(!output || !output_length) return 2;
    *output = nullptr;
    *output_length = 0;
    try
    {
        const size_t max_length = INT32_MAX / 4;
        if((!input && input_length) || (!options && options_length) ||
           input_length > max_length || options_length > max_length)
            return write_output("invalid input buffer", output, output_length, 2);
        const std::string result = parse_edn(
            std::string(input ? reinterpret_cast<const char*>(input) : "", input_length),
            std::string(options ? reinterpret_cast<const char*>(options) : "", options_length));
        return write_output(result, output, output_length);
    }
    catch(const InputError &error)
    {
        try { return write_output(error.what(), output, output_length, 1); }
        catch(...) { return 2; }
    }
    catch(const std::exception &error)
    {
        try { return write_output(error.what(), output, output_length, 2); }
        catch(...) { return 2; }
    }
    catch(...) { return 2; }
}
