// Floating format bar: a Notion-style pill that hovers over a text selection
// with the inline formats. Included at file scope; the editor half lives in
// format_bar_darwin.h.

#pragma once

@class CREditor;

static const CGFloat CRFormatBarHeight = 34;
static const CGFloat CRFormatBarButton = 28;
static const CGFloat CRFormatBarGap = 2;
static const CGFloat CRFormatBarPad = 4;
static const CGFloat CRFormatBarSeparator = 9;
// The color palette unfolds under the pill: a text row and a background row,
// each "default" plus the palette.
static const CGFloat CRPaletteSwatch = 22;
static const CGFloat CRPaletteGap = 4;
static const CGFloat CRPaletteHeight = 112;

// CRFormatBarPanel never becomes key: the text view keeps the selection and
// the first responder role while a button applies a command.
@interface CRFormatBarPanel : NSPanel
@end
@implementation CRFormatBarPanel
- (BOOL)canBecomeKey { return NO; }
- (BOOL)canBecomeMain { return NO; }
@end

@interface CRFormatBarView : NSView
@property(nonatomic, weak) CREditor *owner;
@property(nonatomic, copy) NSSet *active; // commands currently applied
@property(nonatomic) NSInteger hovered;   // item index, -1 none
@property(nonatomic) BOOL paletteOpen;
@property(nonatomic) NSInteger hoveredSwatch; // 0...count text, then background; -1 none
// The selection's uniform colors (nil when unset or mixed).
@property(nonatomic, copy) NSString *currentColor;
@property(nonatomic, copy) NSString *currentBackground;
- (NSRect)swatchRect:(NSInteger)swatch;
- (NSString *)commandForSwatch:(NSInteger)swatch;
@end

@implementation CRFormatBarView {
    NSTrackingArea *_tracking;
    NSArray<NSDictionary *> *_items;
    NSUInteger _divider; // index of the separator item
}

- (instancetype)initWithOwner:(CREditor *)owner {
    _items = @[
        @{@"command": @"bold", @"icon": @"bold", @"title": @"粗体  ⌘B"},
        @{@"command": @"italic", @"icon": @"italic", @"title": @"斜体  ⌘I"},
        @{@"command": @"underline", @"icon": @"underline", @"title": @"下划线  ⌘U"},
        @{@"command": @"strike", @"icon": @"strikethrough", @"title": @"删除线  ⌘⇧X"},
        @{@"command": @"code", @"icon": @"chevron.left.forwardslash.chevron.right", @"title": @"行内代码  ⌘⇧`"},
        @{@"command": @"highlight", @"icon": @"highlighter", @"title": @"高亮  ⌘⇧H"},
        @{@"command": @"palette", @"title": @"文字颜色与背景色"},
        @{@"command": @"link", @"icon": @"link", @"title": @"链接  ⌘K"},
        @{@"separator": @YES},
        @{@"command": @"clear", @"icon": @"eraser", @"title": @"清除格式  ⌘\\"},
    ];
    for (NSUInteger i = 0; i < _items.count; i++) {
        if (_items[i][@"separator"]) _divider = i;
    }
    CGFloat width = 2 * CRFormatBarPad + CRFormatBarSeparator;
    for (NSDictionary *item in _items) {
        width += item[@"separator"] ? 0 : CRFormatBarButton + CRFormatBarGap;
    }
    self = [super initWithFrame:NSMakeRect(0, 0, width - CRFormatBarGap, CRFormatBarHeight)];
    if (self) {
        self.owner = owner;
        _hovered = -1;
        _hoveredSwatch = -1;
    }
    return self;
}
- (BOOL)isFlipped { return YES; }
- (NSRect)rectForItem:(NSInteger)index {
    CGFloat x = CRFormatBarPad;
    for (NSInteger i = 0; i < (NSInteger)_items.count; i++) {
        NSDictionary *item = _items[i];
        if (item[@"separator"]) {
            if (i == index) return NSMakeRect(x, 0, CRFormatBarSeparator, CRFormatBarHeight);
            x += CRFormatBarSeparator;
            continue;
        }
        NSRect rect = NSMakeRect(x, (CRFormatBarHeight - CRFormatBarButton) / 2 + 2, CRFormatBarButton, CRFormatBarButton - 2);
        if (i == index) return rect;
        x += CRFormatBarButton + CRFormatBarGap;
    }
    return NSZeroRect;
}
// Swatch i < 1+CRPaletteCount is a text color (0 = default); the rest are
// backgrounds in the same order.
- (NSRect)swatchRect:(NSInteger)swatch {
    NSInteger per = 1 + (NSInteger)CRPaletteCount;
    if (!_paletteOpen || swatch < 0 || swatch >= 2 * per) return NSZeroRect;
    NSInteger row = swatch / per, column = swatch % per;
    CGFloat rowWidth = per * CRPaletteSwatch + (per - 1) * CRPaletteGap;
    CGFloat x = (self.bounds.size.width - rowWidth) / 2 + column * (CRPaletteSwatch + CRPaletteGap);
    CGFloat y = CRFormatBarHeight + 8 + 14 + 6 + row * (CRPaletteSwatch + 10 + 14 + 6);
    return NSMakeRect(x, y, CRPaletteSwatch, CRPaletteSwatch);
}
- (NSString *)commandForSwatch:(NSInteger)swatch {
    NSInteger per = 1 + (NSInteger)CRPaletteCount;
    if (swatch < 0 || swatch >= 2 * per) return nil;
    BOOL background = swatch >= per;
    NSInteger column = swatch % per;
    NSString *value = column ? CRPaletteHex(column - 1, background) : @"";
    return [(background ? @"background:" : @"color:") stringByAppendingString:value];
}
- (NSInteger)swatchAtPoint:(NSPoint)point {
    for (NSInteger i = 0; i < 2 * (1 + (NSInteger)CRPaletteCount); i++) {
        if (NSPointInRect(point, NSInsetRect([self swatchRect:i], -2, -2))) return i;
    }
    return -1;
}
- (void)setPaletteOpen:(BOOL)open {
    _paletteOpen = open;
    _hoveredSwatch = -1;
    [self setFrameSize:NSMakeSize(self.frame.size.width, CRFormatBarHeight + (open ? CRPaletteHeight : 0))];
    [self updateTrackingAreas];
    [self setNeedsDisplay:YES];
}
- (void)drawPaletteWithDark:(BOOL)dark accent:(NSColor *)accent {
    NSRect bounds = self.bounds;
    [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.08] setFill];
    [NSBezierPath fillRect:NSMakeRect(8, CRFormatBarHeight, bounds.size.width - 16, 1)];
    NSDictionary *labelAttrs = @{NSFontAttributeName: [NSFont systemFontOfSize:11 weight:NSFontWeightMedium],
                                 NSForegroundColorAttributeName: dark ? [NSColor colorWithWhite:0.6 alpha:1] : [NSColor colorWithWhite:0.45 alpha:1]};
    NSColor *ink = dark ? [NSColor colorWithWhite:0.9 alpha:1] : [NSColor colorWithWhite:0.2 alpha:1];
    NSInteger per = 1 + (NSInteger)CRPaletteCount;
    for (NSInteger row = 0; row < 2; row++) {
        NSRect first = [self swatchRect:row * per];
        [(row ? @"背景颜色" : @"文字颜色") drawAtPoint:NSMakePoint(NSMinX(first), NSMinY(first) - 6 - 14) withAttributes:labelAttrs];
        for (NSInteger column = 0; column < per; column++) {
            NSInteger swatch = row * per + column;
            NSRect rect = [self swatchRect:swatch];
            NSString *value = column ? CRPaletteHex(column - 1, row == 1) : nil;
            NSString *current = row ? self.currentBackground : self.currentColor;
            BOOL selected = column ? [value isEqual:current] : current == nil;
            NSBezierPath *shape = [NSBezierPath bezierPathWithRoundedRect:NSInsetRect(rect, 0.5, 0.5) xRadius:5 yRadius:5];
            NSColor *fill = row && column ? CRResolvedColor(value, YES, dark) : (dark ? [NSColor colorWithWhite:0.22 alpha:1] : NSColor.whiteColor);
            [fill setFill];
            [shape fill];
            [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:swatch == self.hoveredSwatch ? 0.35 : 0.14] setStroke];
            shape.lineWidth = 1;
            [shape stroke];
            if (selected) {
                NSBezierPath *ring = [NSBezierPath bezierPathWithRoundedRect:NSInsetRect(rect, -2.5, -2.5) xRadius:7 yRadius:7];
                ring.lineWidth = 1.5;
                [accent setStroke];
                [ring stroke];
            }
            // Every swatch previews itself on an "A", as Notion's menu does.
            NSColor *letter = !row && column ? CRResolvedColor(value, NO, dark) : ink;
            NSDictionary *attrs = @{NSFontAttributeName: [NSFont systemFontOfSize:12.5 weight:NSFontWeightSemibold], NSForegroundColorAttributeName: letter};
            NSSize size = [@"A" sizeWithAttributes:attrs];
            [@"A" drawAtPoint:NSMakePoint(NSMidX(rect) - size.width / 2, NSMidY(rect) - size.height / 2) withAttributes:attrs];
        }
    }
}
- (NSInteger)itemAtPoint:(NSPoint)point {
    for (NSInteger i = 0; i < (NSInteger)_items.count; i++) {
        if (_items[i][@"separator"]) continue;
        if (NSPointInRect(point, [self rectForItem:i])) return i;
    }
    return -1;
}
- (void)drawRect:(NSRect)dirty {
    BOOL dark = self.owner.dark;
    NSRect bounds = self.bounds;
    NSBezierPath *outline = [NSBezierPath bezierPathWithRoundedRect:NSInsetRect(bounds, 0.5, 0.5) xRadius:9 yRadius:9];
    [outline addClip];
    [(dark ? [NSColor colorWithWhite:0.17 alpha:1] : [NSColor whiteColor]) setFill];
    [NSBezierPath fillRect:bounds];
    [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.12] setStroke];
    [outline setLineWidth:1];
    [outline stroke];
    NSColor *idle = dark ? [NSColor colorWithWhite:0.84 alpha:1] : [NSColor colorWithWhite:0.32 alpha:1];
    NSColor *accent = dark ? [NSColor colorWithRed:0.45 green:0.66 blue:1 alpha:1] : [NSColor colorWithRed:0.12 green:0.44 blue:0.92 alpha:1];
    for (NSInteger i = 0; i < (NSInteger)_items.count; i++) {
        NSDictionary *item = _items[i];
        NSRect rect = [self rectForItem:i];
        if (item[@"separator"]) {
            [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.10] setFill];
            [NSBezierPath fillRect:NSMakeRect(NSMidX(rect), 9, 1, CRFormatBarHeight - 18)];
            continue;
        }
        BOOL active = [self.active containsObject:item[@"command"]];
        if (active) {
            [[accent colorWithAlphaComponent:dark ? 0.24 : 0.14] setFill];
            [[NSBezierPath bezierPathWithRoundedRect:rect xRadius:6 yRadius:6] fill];
        } else if (i == self.hovered) {
            [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.07] setFill];
            [[NSBezierPath bezierPathWithRoundedRect:rect xRadius:6 yRadius:6] fill];
        }
        if ([item[@"command"] isEqual:@"palette"]) {
            // "A" over a bar in the selection's color, Notion's color button.
            NSColor *bar = CRResolvedColor(self.currentColor, NO, dark) ?: CRResolvedColor(self.currentBackground, YES, dark) ?: idle;
            NSDictionary *attrs = @{NSFontAttributeName: [NSFont systemFontOfSize:13.5 weight:NSFontWeightSemibold], NSForegroundColorAttributeName: active ? accent : idle};
            NSSize size = [@"A" sizeWithAttributes:attrs];
            [@"A" drawAtPoint:NSMakePoint(NSMidX(rect) - size.width / 2, NSMidY(rect) - size.height / 2 - 1.5) withAttributes:attrs];
            [bar setFill];
            [[NSBezierPath bezierPathWithRoundedRect:NSMakeRect(NSMidX(rect) - 6, NSMidY(rect) + 6.5, 12, 2.5) xRadius:1.25 yRadius:1.25] fill];
            continue;
        }
        NSImage *symbol = CRTintedSymbol(item[@"icon"], 13, NSFontWeightMedium, active ? accent : idle);
        if (!symbol) continue;
        NSSize size = symbol.size;
        [symbol drawInRect:NSMakeRect(NSMidX(rect) - size.width / 2, NSMidY(rect) - size.height / 2, size.width, size.height)
                  fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1 respectFlipped:YES hints:nil];
    }
    if (_paletteOpen) [self drawPaletteWithDark:dark accent:accent];
}
- (void)updateTrackingAreas {
    [super updateTrackingAreas];
    if (_tracking) [self removeTrackingArea:_tracking];
    _tracking = [[NSTrackingArea alloc] initWithRect:self.bounds options:NSTrackingMouseEnteredAndExited | NSTrackingMouseMoved | NSTrackingActiveAlways owner:self userInfo:nil];
    [self addTrackingArea:_tracking];
    [self removeAllToolTips];
    for (NSInteger i = 0; i < (NSInteger)_items.count; i++) {
        if (_items[i][@"separator"]) continue;
        [self addToolTipRect:[self rectForItem:i] owner:self userData:NULL];
    }
    for (NSInteger i = 0; i < 2 * (1 + (NSInteger)CRPaletteCount); i++) {
        if (_paletteOpen) [self addToolTipRect:[self swatchRect:i] owner:self userData:NULL];
    }
}
- (NSString *)view:(NSView *)view stringForToolTip:(NSToolTipTag)tag point:(NSPoint)point userData:(void *)data {
    NSInteger index = [self itemAtPoint:point];
    if (index >= 0) return _items[index][@"title"];
    NSInteger swatch = [self swatchAtPoint:point], per = 1 + (NSInteger)CRPaletteCount;
    if (swatch < 0) return nil;
    NSString *name = swatch % per ? @(CRPalette[swatch % per - 1].label) : @"默认";
    return [NSString stringWithFormat:@"%@%@", swatch < per ? @"文字：" : @"背景：", name];
}
- (void)mouseMoved:(NSEvent *)event {
    NSPoint point = [self convertPoint:event.locationInWindow fromView:nil];
    self.hovered = [self itemAtPoint:point];
    NSInteger swatch = [self swatchAtPoint:point];
    if (swatch != _hoveredSwatch) { _hoveredSwatch = swatch; [self setNeedsDisplay:YES]; }
}
- (void)mouseExited:(NSEvent *)event { self.hovered = -1; _hoveredSwatch = -1; }
- (void)mouseDown:(NSEvent *)event {
    NSPoint point = [self convertPoint:event.locationInWindow fromView:nil];
    NSInteger index = [self itemAtPoint:point];
    if (index >= 0) { [self.owner formatBarApply:_items[index][@"command"]]; return; }
    NSString *command = [self commandForSwatch:[self swatchAtPoint:point]];
    if (command) [self.owner formatBarApply:command];
}
- (void)setActive:(NSSet *)active { _active = [active copy]; [self setNeedsDisplay:YES]; }
- (void)setHovered:(NSInteger)hovered { _hovered = hovered; [self setNeedsDisplay:YES]; }
@end
