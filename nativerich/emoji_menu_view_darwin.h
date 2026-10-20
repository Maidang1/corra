// Emoji menu panel and list: typing ":" opens a shortcode picker beside the
// caret. The editor half lives in emoji_menu_darwin.h. Included at file scope.

#pragma once

@class CREditor;

static const CGFloat CREmojiRowHeight = 32;
static const CGFloat CREmojiWidth = 250;
static const CGFloat CREmojiHeaderHeight = 28;
static const CGFloat CREmojiFooterHeight = 22;
static const NSInteger CREmojiMaxRows = 8;

@interface CREmojiPanel : NSPanel
@end
@implementation CREmojiPanel
- (BOOL)canBecomeKey { return NO; }
- (BOOL)canBecomeMain { return NO; }
@end

@interface CREmojiMenuView : NSView
@property(nonatomic, weak) CREditor *owner;
@property(nonatomic, copy) NSArray *items; // @{name, emoji}
@property(nonatomic) NSInteger selected;
@property(nonatomic) NSInteger hovered;
@property(nonatomic) NSInteger offset;
@end

@implementation CREmojiMenuView
- (instancetype)initWithOwner:(CREditor *)owner {
    self = [super initWithFrame:NSMakeRect(0, 0, CREmojiWidth, CREmojiHeaderHeight + CREmojiRowHeight + CREmojiFooterHeight)];
    self.owner = owner;
    _hovered = -1;
    return self;
}
- (BOOL)isFlipped { return YES; }
- (NSInteger)visibleRows { return MIN(CREmojiMaxRows, (NSInteger)self.items.count); }
- (CGFloat)contentHeight {
    NSInteger rows = MAX(1, self.visibleRows);
    CGFloat height = CREmojiHeaderHeight + rows * CREmojiRowHeight + CREmojiFooterHeight;
    if (!self.items.count) return height;
    return height;
}
- (void)setItems:(NSArray *)items {
    _items = [items copy];
    _offset = 0;
    [self setFrameSize:NSMakeSize(CREmojiWidth, [self contentHeight])];
}
- (void)keepSelectedVisible {
    NSInteger rows = [self visibleRows];
    if (_selected < _offset) _offset = _selected;
    else if (_selected >= _offset + rows) _offset = _selected - rows + 1;
    _offset = MAX(0, MIN(_offset, (NSInteger)self.items.count - rows));
}
- (void)setSelected:(NSInteger)selected {
    _selected = selected;
    [self keepSelectedVisible];
    [self setNeedsDisplay:YES];
}
- (void)setOffset:(NSInteger)offset {
    NSInteger max = MAX(0, (NSInteger)self.items.count - [self visibleRows]);
    _offset = MAX(0, MIN(offset, max));
    [self setNeedsDisplay:YES];
}
- (void)scrollWheel:(NSEvent *)event {
    if (event.scrollingDeltaY == 0) return;
    self.offset = _offset + (event.scrollingDeltaY > 0 ? -1 : 1);
}
- (NSInteger)itemAtPoint:(NSPoint)point {
    CGFloat y = point.y - CREmojiHeaderHeight;
    if (y < 0) return -1;
    NSInteger row = _offset + (NSInteger)(y / CREmojiRowHeight);
    return row >= 0 && row < (NSInteger)self.items.count ? row : -1;
}
- (void)drawRect:(NSRect)dirty {
    BOOL dark = self.owner.dark;
    NSRect bounds = self.bounds;
    NSBezierPath *outline = [NSBezierPath bezierPathWithRoundedRect:NSInsetRect(bounds, 0.5, 0.5) xRadius:10 yRadius:10];
    [outline addClip];
    [(dark ? [NSColor colorWithWhite:0.16 alpha:1] : [NSColor whiteColor]) setFill];
    [NSBezierPath fillRect:bounds];
    [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.12] setStroke];
    [outline setLineWidth:1];
    [outline stroke];
    NSColor *ink = dark ? [NSColor colorWithWhite:0.94 alpha:1] : [NSColor colorWithWhite:0.16 alpha:1];
    NSColor *muted = dark ? [NSColor colorWithWhite:0.60 alpha:1] : [NSColor colorWithWhite:0.45 alpha:1];
    NSDictionary *headerAttrs = @{NSFontAttributeName: [NSFont systemFontOfSize:11 weight:NSFontWeightSemibold], NSForegroundColorAttributeName: muted};
    [@"表情" drawAtPoint:NSMakePoint(12, 9) withAttributes:headerAttrs];
    NSString *filter = [self.owner emojiMenuFilterText];
    if (filter.length) {
        NSString *query = [NSString stringWithFormat:@":%@", filter];
        NSSize size = [query sizeWithAttributes:headerAttrs];
        [query drawAtPoint:NSMakePoint(CREmojiWidth - size.width - 12, 9) withAttributes:headerAttrs];
    }
    if (!self.items.count) {
        NSDictionary *attrs = @{NSFontAttributeName: [NSFont systemFontOfSize:12], NSForegroundColorAttributeName: muted};
        NSString *text = @"无匹配表情";
        NSSize size = [text sizeWithAttributes:attrs];
        [text drawAtPoint:NSMakePoint((bounds.size.width - size.width) / 2, CREmojiHeaderHeight + CREmojiRowHeight / 2 - size.height / 2) withAttributes:attrs];
    }
    NSInteger last = MIN((NSInteger)self.items.count, _offset + [self visibleRows]);
    for (NSInteger index = _offset; index < last; index++) {
        NSDictionary *item = self.items[index];
        CGFloat top = CREmojiHeaderHeight + (index - _offset) * CREmojiRowHeight;
        NSRect rowRect = NSMakeRect(4, top, bounds.size.width - 8, CREmojiRowHeight);
        if (index == self.selected) {
            [[NSColor.controlAccentColor colorWithAlphaComponent:dark ? 0.24 : 0.12] setFill];
            [[NSBezierPath bezierPathWithRoundedRect:rowRect xRadius:6 yRadius:6] fill];
        } else if (index == self.hovered) {
            [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.06] setFill];
            [[NSBezierPath bezierPathWithRoundedRect:rowRect xRadius:6 yRadius:6] fill];
        }
        NSString *glyph = item[@"emoji"] ?: @"";
        NSDictionary *glyphAttrs = @{NSFontAttributeName: [NSFont systemFontOfSize:17]};
        [glyph drawAtPoint:NSMakePoint(12, NSMidY(rowRect) - 11) withAttributes:glyphAttrs];
        NSString *name = [NSString stringWithFormat:@":%@:", item[@"name"] ?: @""];
        NSDictionary *nameAttrs = @{NSFontAttributeName: [NSFont systemFontOfSize:12.5],
                                    NSForegroundColorAttributeName: index == self.selected ? ink : muted};
        [name drawAtPoint:NSMakePoint(46, NSMidY(rowRect) - 8) withAttributes:nameAttrs];
    }
    NSDictionary *footerAttrs = @{NSFontAttributeName: [NSFont systemFontOfSize:10.5], NSForegroundColorAttributeName: muted};
    NSString *footer = @"↑↓ 选择 · ⏎ 插入 · esc 关闭";
    NSSize footerSize = [footer sizeWithAttributes:footerAttrs];
    [footer drawAtPoint:NSMakePoint((bounds.size.width - footerSize.width) / 2, bounds.size.height - CREmojiFooterHeight + 5) withAttributes:footerAttrs];
}
- (void)updateTrackingAreas {
    [super updateTrackingAreas];
    [self removeAllToolTips];
}
- (void)mouseExited:(NSEvent *)event { self.hovered = -1; }
- (void)mouseMoved:(NSEvent *)event { self.hovered = [self itemAtPoint:[self convertPoint:event.locationInWindow fromView:nil]]; }
- (void)mouseDown:(NSEvent *)event {
    NSInteger index = [self itemAtPoint:[self convertPoint:event.locationInWindow fromView:nil]];
    if (index >= 0 && index < (NSInteger)self.items.count) [self.owner emojiMenuChoose:self.items[index]];
}
- (void)setHovered:(NSInteger)hovered { _hovered = hovered; [self setNeedsDisplay:YES]; }
@end
