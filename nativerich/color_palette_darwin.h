// Text and background colors (Notion's palette). Documents store the light
// theme's hex in <span style="color:…">; the editor maps a palette hex to its
// dark variant in dark mode, so colored notes read well in both themes.
// Included at file scope.

#pragma once

typedef struct {
    const char *name;  // CSS color name it stands in for ("red")
    const char *label; // menu label
    uint32_t text, textDark, background, backgroundDark;
} CRPaletteColor;

static const CRPaletteColor CRPalette[] = {
    {"gray",   "灰色", 0x787774, 0x9B9B9B, 0xF1F1EF, 0x373737},
    {"brown",  "棕色", 0x9F6B53, 0xBA856F, 0xF4EEEE, 0x4A3228},
    {"orange", "橙色", 0xD9730D, 0xC77D48, 0xFBECDD, 0x5C3B23},
    {"yellow", "黄色", 0xCB912F, 0xCA9849, 0xFBF3DB, 0x564328},
    {"green",  "绿色", 0x448361, 0x529E72, 0xEDF3EC, 0x243D30},
    {"blue",   "蓝色", 0x337EA9, 0x379AD3, 0xE7F3F8, 0x143A4E},
    {"purple", "紫色", 0x9065B0, 0x9D68D3, 0xF6F3F9, 0x3C2D49},
    {"pink",   "粉色", 0xC14C8A, 0xD15796, 0xFAF1F5, 0x4E2C3C},
    {"red",    "红色", 0xD44C47, 0xDF5452, 0xFDEBEC, 0x522E2A},
};
static const NSUInteger CRPaletteCount = sizeof(CRPalette) / sizeof(CRPalette[0]);

static NSColor *CRColorFromRGB(uint32_t rgb) {
    return [NSColor colorWithSRGBRed:((rgb >> 16) & 0xFF) / 255.0 green:((rgb >> 8) & 0xFF) / 255.0 blue:(rgb & 0xFF) / 255.0 alpha:1];
}

// CRPaletteHex is the value a document stores for a palette entry.
static NSString *CRPaletteHex(NSUInteger index, BOOL background) {
    if (index >= CRPaletteCount) return @"";
    uint32_t rgb = background ? CRPalette[index].background : CRPalette[index].text;
    return [NSString stringWithFormat:@"#%06x", rgb];
}

// CRParseHex reads "#rgb" or "#rrggbb"; NO for anything else.
static BOOL CRParseHex(NSString *value, uint32_t *rgb) {
    if (![value hasPrefix:@"#"] || (value.length != 4 && value.length != 7)) return NO;
    uint32_t result = 0;
    for (NSUInteger i = 1; i < value.length; i++) {
        unichar c = [value characterAtIndex:i];
        uint32_t digit;
        if (c >= '0' && c <= '9') digit = c - '0';
        else if (c >= 'a' && c <= 'f') digit = c - 'a' + 10;
        else if (c >= 'A' && c <= 'F') digit = c - 'A' + 10;
        else return NO;
        result = result * 16 + digit;
        if (value.length == 4) result = result * 16 + digit; // #abc is #aabbcc
    }
    *rgb = result;
    return YES;
}

// CRPaletteIndex finds the palette entry a stored value names: its hex (text
// or background) or the CSS name it stands in for. NSNotFound otherwise.
static NSUInteger CRPaletteIndex(NSString *value) {
    if (![value isKindOfClass:NSString.class] || !value.length) return NSNotFound;
    NSString *lower = value.lowercaseString;
    if ([lower isEqual:@"grey"]) lower = @"gray";
    uint32_t rgb = 0;
    BOOL hex = CRParseHex(lower, &rgb);
    for (NSUInteger i = 0; i < CRPaletteCount; i++) {
        if (hex ? (CRPalette[i].text == rgb || CRPalette[i].background == rgb) : [lower isEqual:@(CRPalette[i].name)]) return i;
    }
    return NSNotFound;
}

// CRResolvedColor is the color to draw a stored value with, or nil when the
// value is not one the editor can read.
static NSColor *CRResolvedColor(NSString *value, BOOL background, BOOL dark) {
    NSUInteger index = CRPaletteIndex(value);
    if (index != NSNotFound) {
        const CRPaletteColor *c = &CRPalette[index];
        return CRColorFromRGB(background ? (dark ? c->backgroundDark : c->background) : (dark ? c->textDark : c->text));
    }
    uint32_t rgb = 0;
    if ([value isKindOfClass:NSString.class] && CRParseHex(value, &rgb)) {
        NSColor *color = CRColorFromRGB(rgb);
        // A foreign light background would glare on the dark canvas.
        return background && dark ? [color colorWithAlphaComponent:0.35] : color;
    }
    return nil;
}
