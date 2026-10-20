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
            [NSBezierPath fillRect:NSMakeRect(NSMidX(rect), 9, 1, bounds.size.height - 18)];
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
        NSImage *symbol = CRTintedSymbol(item[@"icon"], 13, NSFontWeightMedium, active ? accent : idle);
        if (!symbol) continue;
        NSSize size = symbol.size;
        [symbol drawInRect:NSMakeRect(NSMidX(rect) - size.width / 2, NSMidY(rect) - size.height / 2, size.width, size.height)
                  fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1 respectFlipped:YES hints:nil];
    }
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
}
- (NSString *)view:(NSView *)view stringForToolTip:(NSToolTipTag)tag point:(NSPoint)point userData:(void *)data {
    NSInteger index = [self itemAtPoint:point];
    return index >= 0 ? _items[index][@"title"] : nil;
}
- (void)mouseMoved:(NSEvent *)event { self.hovered = [self itemAtPoint:[self convertPoint:event.locationInWindow fromView:nil]]; }
- (void)mouseExited:(NSEvent *)event { self.hovered = -1; }
- (void)mouseDown:(NSEvent *)event {
    NSInteger index = [self itemAtPoint:[self convertPoint:event.locationInWindow fromView:nil]];
    if (index >= 0) [self.owner formatBarApply:_items[index][@"command"]];
}
- (void)setActive:(NSSet *)active { _active = [active copy]; [self setNeedsDisplay:YES]; }
- (void)setHovered:(NSInteger)hovered { _hovered = hovered; [self setNeedsDisplay:YES]; }
@end
