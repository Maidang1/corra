// Link menu panel and list: typing "[[" opens note suggestions beside the
// caret. The editor half lives in link_menu_darwin.h. Included at file scope.

#pragma once

@class CREditor;

static const CGFloat CRLinkRowHeight = 38;
static const CGFloat CRLinkWidth = 300;
static const CGFloat CRLinkHeaderHeight = 28;
static const CGFloat CRLinkFooterHeight = 22;
static const NSInteger CRLinkMaxRows = 8;

@interface CRLinkPanel : NSPanel
@end
@implementation CRLinkPanel
- (BOOL)canBecomeKey { return NO; }
- (BOOL)canBecomeMain { return NO; }
@end

@interface CRLinkMenuView : NSView
@property(nonatomic, weak) CREditor *owner;
@property(nonatomic, copy) NSArray *items; // @{title, path}
@property(nonatomic) NSInteger selected;
@property(nonatomic) NSInteger hovered;
@property(nonatomic) NSInteger offset;
@end

@implementation CRLinkMenuView
- (instancetype)initWithOwner:(CREditor *)owner {
    self = [super initWithFrame:NSMakeRect(0, 0, CRLinkWidth, CRLinkHeaderHeight + CRLinkRowHeight + CRLinkFooterHeight)];
    self.owner = owner;
    _hovered = -1;
    return self;
}
- (BOOL)isFlipped { return YES; }
- (NSInteger)visibleRows { return MIN(CRLinkMaxRows, (NSInteger)self.items.count); }
- (void)setItems:(NSArray *)items {
    _items = [items copy];
    _offset = 0;
    NSInteger rows = MAX(1, self.visibleRows);
    [self setFrameSize:NSMakeSize(CRLinkWidth, CRLinkHeaderHeight + rows * CRLinkRowHeight + CRLinkFooterHeight)];
}
- (void)keepSelectedVisible {
    NSInteger rows = [self visibleRows];
    if (_selected < _offset) _offset = _selected;
    else if (_selected >= _offset + rows) _offset = _selected - rows + 1;
    _offset = MAX(0, MIN(_offset, (NSInteger)self.items.count - rows));
}
- (void)setSelected:(NSInteger)selected { _selected = selected; [self keepSelectedVisible]; [self setNeedsDisplay:YES]; }
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
    CGFloat y = point.y - CRLinkHeaderHeight;
    if (y < 0) return -1;
    NSInteger row = _offset + (NSInteger)(y / CRLinkRowHeight);
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
    NSString *header = [self.owner linkMenuHeadingMode] ? @"标题锚点" : @"链接文档";
    [header drawAtPoint:NSMakePoint(12, 9) withAttributes:headerAttrs];
    NSString *filter = [self.owner linkMenuFilterText];
    if (filter.length) {
        NSString *query = [NSString stringWithFormat:@"[[%@", filter];
        NSSize size = [query sizeWithAttributes:headerAttrs];
        [query drawAtPoint:NSMakePoint(CRLinkWidth - size.width - 12, 9) withAttributes:headerAttrs];
    }
    if (!self.items.count) {
        NSDictionary *attrs = @{NSFontAttributeName: [NSFont systemFontOfSize:12], NSForegroundColorAttributeName: muted};
        NSString *text = [self.owner linkMenuHeadingMode] ? @"没有匹配的标题" : @"没有匹配的文档";
        NSSize size = [text sizeWithAttributes:attrs];
        [text drawAtPoint:NSMakePoint((bounds.size.width - size.width) / 2, CRLinkHeaderHeight + CRLinkRowHeight / 2 - size.height / 2) withAttributes:attrs];
    }
    NSInteger last = MIN((NSInteger)self.items.count, _offset + [self visibleRows]);
    for (NSInteger index = _offset; index < last; index++) {
        NSDictionary *item = self.items[index];
        CGFloat top = CRLinkHeaderHeight + (index - _offset) * CRLinkRowHeight;
        NSRect rowRect = NSMakeRect(4, top, bounds.size.width - 8, CRLinkRowHeight);
        if (index == self.selected) {
            [[NSColor.controlAccentColor colorWithAlphaComponent:dark ? 0.24 : 0.12] setFill];
            [[NSBezierPath bezierPathWithRoundedRect:rowRect xRadius:6 yRadius:6] fill];
        } else if (index == self.hovered) {
            [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.06] setFill];
            [[NSBezierPath bezierPathWithRoundedRect:rowRect xRadius:6 yRadius:6] fill];
        }
        NSDictionary *titleAttrs = @{NSFontAttributeName: [NSFont systemFontOfSize:13], NSForegroundColorAttributeName: ink};
        NSString *title = item[@"title"] ?: @"";
        NSSize titleSize = [title sizeWithAttributes:titleAttrs];
        [title drawAtPoint:NSMakePoint(12, NSMidY(rowRect) - titleSize.height / 2) withAttributes:titleAttrs];
        NSString *path = item[@"path"] ?: @"";
        NSMutableParagraphStyle *right = [NSMutableParagraphStyle new];
        right.alignment = NSTextAlignmentRight;
        right.lineBreakMode = NSLineBreakByTruncatingMiddle;
        NSDictionary *pathAttrs = @{NSFontAttributeName: [NSFont systemFontOfSize:10.5], NSForegroundColorAttributeName: muted,
                                    NSParagraphStyleAttributeName: right};
        [path drawInRect:NSMakeRect(NSMinX(rowRect) + 140, NSMidY(rowRect) - 7, NSMaxX(rowRect) - 146, 14) withAttributes:pathAttrs];
    }
    NSDictionary *footerAttrs = @{NSFontAttributeName: [NSFont systemFontOfSize:10.5], NSForegroundColorAttributeName: muted};
    NSString *footer = @"↑↓ 选择 · ⏎ 插入 · esc 关闭";
    NSSize footerSize = [footer sizeWithAttributes:footerAttrs];
    [footer drawAtPoint:NSMakePoint((bounds.size.width - footerSize.width) / 2, bounds.size.height - CRLinkFooterHeight + 5) withAttributes:footerAttrs];
}
- (void)mouseExited:(NSEvent *)event { self.hovered = -1; }
- (void)mouseMoved:(NSEvent *)event { self.hovered = [self itemAtPoint:[self convertPoint:event.locationInWindow fromView:nil]]; }
- (void)mouseDown:(NSEvent *)event {
    NSInteger index = [self itemAtPoint:[self convertPoint:event.locationInWindow fromView:nil]];
    if (index >= 0 && index < (NSInteger)self.items.count) [self.owner linkMenuChoose:self.items[index]];
}
- (void)setHovered:(NSInteger)hovered { _hovered = hovered; [self setNeedsDisplay:YES]; }
@end
