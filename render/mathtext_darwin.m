#import <Foundation/Foundation.h>
#import <CoreText/CoreText.h>
#import <CoreGraphics/CoreGraphics.h>
#include "mathtext_darwin.h"

static void appendMathPath(void *info, const CGPathElement *element) {
    NSMutableString *output = (__bridge NSMutableString *)info;
    const CGPoint *p = element->points;
    switch (element->type) {
        case kCGPathElementMoveToPoint: [output appendFormat:@"M%.8g %.8g", p[0].x, p[0].y]; break;
        case kCGPathElementAddLineToPoint: [output appendFormat:@"L%.8g %.8g", p[0].x, p[0].y]; break;
        case kCGPathElementAddQuadCurveToPoint: [output appendFormat:@"Q%.8g %.8g %.8g %.8g", p[0].x, p[0].y, p[1].x, p[1].y]; break;
        case kCGPathElementAddCurveToPoint: [output appendFormat:@"C%.8g %.8g %.8g %.8g %.8g %.8g", p[0].x, p[0].y, p[1].x, p[1].y, p[2].x, p[2].y]; break;
        case kCGPathElementCloseSubpath: [output appendString:@"Z"]; break;
    }
}

char *corraMathTextPath(const char *text, double size, int traits) {
    @autoreleasepool {
        NSString *string = [NSString stringWithUTF8String:text];
        if (!string) return NULL;
        CTFontRef font = CTFontCreateWithName(CFSTR("TimesNewRomanPSMT"), size, NULL);
        CTFontSymbolicTraits mask = ((traits & 1) ? kCTFontBoldTrait : 0) | ((traits & 2) ? kCTFontItalicTrait : 0);
        if (mask) {
            CTFontRef styled = CTFontCreateCopyWithSymbolicTraits(font, size, NULL, mask, mask);
            if (styled) { CFRelease(font); font = styled; }
        }
        NSAttributedString *attributed = [[[NSAttributedString alloc] initWithString:string attributes:@{(__bridge NSString *)kCTFontAttributeName: (__bridge id)font}] autorelease];
        CTLineRef line = CTLineCreateWithAttributedString((__bridge CFAttributedStringRef)attributed);
        CFRelease(font);
        NSMutableString *output = [NSMutableString string];
        bool valid = true;
        for (id value in (__bridge NSArray *)CTLineGetGlyphRuns(line)) {
            CTRunRef run = (__bridge CTRunRef)value;
            CTFontRef runFont = CFDictionaryGetValue(CTRunGetAttributes(run), kCTFontAttributeName);
            CFIndex count = CTRunGetGlyphCount(run);
            for (CFIndex i = 0; i < count; i++) {
                CGGlyph glyph;
                CGPoint position;
                CTRunGetGlyphs(run, CFRangeMake(i, 1), &glyph);
                CTRunGetPositions(run, CFRangeMake(i, 1), &position);
                if (!glyph) { valid = false; break; }
                // CoreText outlines use Y-up; SVG text uses Y-down. Keep the
                // surrounding MathJax transforms untouched for scripts/fractions.
                CGAffineTransform transform = CGAffineTransformMake(1, 0, 0, -1, position.x, -position.y);
                CGPathRef path = CTFontCreatePathForGlyph(runFont, glyph, &transform);
                if (path) {
                    CGPathApply(path, (__bridge void *)output, appendMathPath);
                    CGPathRelease(path);
                } else {
                    CFIndex index;
                    CTRunGetStringIndices(run, CFRangeMake(i, 1), &index);
                    if (index < string.length && ![[NSCharacterSet whitespaceAndNewlineCharacterSet] characterIsMember:[string characterAtIndex:index]]) valid = false;
                }
            }
            if (!valid) break;
        }
        CFRelease(line);
        return valid ? strdup(output.UTF8String) : NULL;
    }
}
