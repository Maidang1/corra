// Slash menu panel and list view: a Notion-style block picker. The editor
// half lives in slash_menu_darwin.h and drives these classes through
// CREditor's slash* methods. Included at file scope, outside @implementation.

#pragma once

// MARK: - Constants

static const CGFloat CRSlashRowHeight = 30;
static const CGFloat CRSlashPad = 6;
static const CGFloat CRSlashWidth = 268;
static const NSInteger CRSlashMaxRows = 9;

// MARK: - Panel

// CRSlashPanel never takes key status: the editor keeps the focus, so typed
// characters keep filtering the query and Enter/Escape reach the text view.
@class CREditor;

@interface CRSlashPanel : NSPanel
@end
@implementation CRSlashPanel
- (BOOL)canBecomeKey { return NO; }
- (BOOL)canBecomeMain { return NO; }
@end

// MARK: - List view

@interface CRSlashMenuView : NSView
@property(nonatomic, weak) CREditor *owner;
@property(nonatomic, copy) NSArray *items; // filtered item dictionaries
@property(nonatomic) NSInteger selected;   // highlighted item index, -1 none
@property(nonatomic) NSInteger hovered;    // mouse hover item index, -1 none
@property(nonatomic) NSInteger offset;     // first visible display row
@end

@implementation CRSlashMenuView {
    NSTrackingArea *_tracking;
    NSArray<NSDictionary *> *_rows; // headers and items, in display order
}

- (instancetype)initWithOwner:(CREditor *)owner {
    self = [super initWithFrame:NSMakeRect(0, 0, CRSlashWidth, CRSlashRowHeight + 2 * CRSlashPad)];
    self.owner = owner;
    _hovered = -1;
    return self;
}
- (BOOL)isFlipped { return YES; }
- (NSInteger)visibleRows { return MIN(CRSlashMaxRows, (NSInteger)_rows.count); }
- (void)setItems:(NSArray *)items {
    _items = [items copy];
    NSMutableArray *rows = [NSMutableArray array];
    NSString *section = nil;
    for (NSUInteger i = 0; i < items.count; i++) {
        NSDictionary *item = items[i];
        NSString *next = item[@"section"];
        if (next.length && ![next isEqual:section]) {
            [rows addObject:@{@"header": next}];
            section = next;
        }
        [rows addObject:@{@"index": @(i)}];
    }
    _rows = [rows copy];
    _offset = 0;
    [self setFrameSize:NSMakeSize(CRSlashWidth, MAX(1, [self visibleRows]) * CRSlashRowHeight + 2 * CRSlashPad)];
}
// rowForItem maps an item index to its display row, skipping headers.
- (NSInteger)rowForItem:(NSInteger)item {
    for (NSInteger row = 0; row < (NSInteger)_rows.count; row++) {
        NSNumber *index = _rows[row][@"index"];
        if (index && index.integerValue == item) return row;
    }
    return -1;
}
// keepSelectedVisible scrolls the window of rows so the highlight is never
// hidden; the palette is taller than the panel.
- (void)keepSelectedVisible {
    NSInteger rows = [self visibleRows];
    NSInteger row = [self rowForItem:_selected];
    if (row >= 0) {
        if (row < _offset) _offset = row;
        else if (row >= _offset + rows) _offset = row - rows + 1;
    }
    _offset = MAX(0, MIN(_offset, (NSInteger)_rows.count - rows));
}
- (void)setOffset:(NSInteger)offset {
    NSInteger max = MAX(0, (NSInteger)_rows.count - [self visibleRows]);
    _offset = MAX(0, MIN(offset, max));
    [self setNeedsDisplay:YES];
}
- (void)scrollWheel:(NSEvent *)event {
    CGFloat delta = event.scrollingDeltaY;
    if (delta == 0) return;
    self.offset = _offset + (delta > 0 ? -1 : 1);
}
// rowAtPoint returns the item index under point, or -1 on a header or gap.
- (NSInteger)rowAtPoint:(NSPoint)point {
    NSInteger row = _offset + (NSInteger)((point.y - CRSlashPad) / CRSlashRowHeight);
    if (row < _offset || row >= (NSInteger)_rows.count) return -1;
    NSNumber *index = _rows[row][@"index"];
    return index ? index.integerValue : -1;
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
    // Explicit ink colors: the shell's theme can differ from the system's, so
    // semantic label colors would read wrong in a forced dark/light window.
    NSColor *ink = dark ? [NSColor colorWithWhite:0.94 alpha:1] : [NSColor colorWithWhite:0.16 alpha:1];
    NSColor *muted = dark ? [NSColor colorWithWhite:0.60 alpha:1] : [NSColor colorWithWhite:0.45 alpha:1];
    if (!_items.count) {
        NSDictionary *attrs = @{NSFontAttributeName: [NSFont systemFontOfSize:12], NSForegroundColorAttributeName: muted};
        NSString *text = @"无匹配结果";
        NSSize size = [text sizeWithAttributes:attrs];
        [text drawAtPoint:NSMakePoint((bounds.size.width - size.width) / 2, (bounds.size.height - size.height) / 2) withAttributes:attrs];
        return;
    }
    NSInteger last = MIN((NSInteger)_rows.count, _offset + [self visibleRows]);
    for (NSInteger row = _offset; row < last; row++) {
        NSDictionary *entry = _rows[row];
        CGFloat top = CRSlashPad + (row - _offset) * CRSlashRowHeight;
        NSString *header = entry[@"header"];
        if (header) {
            NSDictionary *attrs = @{NSFontAttributeName: [NSFont systemFontOfSize:11 weight:NSFontWeightSemibold], NSForegroundColorAttributeName: muted};
            NSSize size = [header sizeWithAttributes:attrs];
            [header drawAtPoint:NSMakePoint(CRSlashPad + 4, top + (CRSlashRowHeight - size.height) / 2) withAttributes:attrs];
            continue;
        }
        NSInteger index = [entry[@"index"] integerValue];
        NSDictionary *item = self.items[index];
        NSRect rowRect = NSMakeRect(4, top, bounds.size.width - 8, CRSlashRowHeight);
        if (index == self.selected) {
            [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:dark ? 0.10 : 0.07] setFill];
            [[NSBezierPath bezierPathWithRoundedRect:rowRect xRadius:6 yRadius:6] fill];
        } else if (index == self.hovered) {
            [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.05] setFill];
            [[NSBezierPath bezierPathWithRoundedRect:rowRect xRadius:6 yRadius:6] fill];
        }
        // Icon: a quiet rounded tile carrying an SF Symbol (or an H1..H6 glyph).
        NSRect iconBox = NSMakeRect(NSMinX(rowRect) + 6, NSMidY(rowRect) - 10, 20, 20);
        [[(dark ? [NSColor whiteColor] : [NSColor blackColor]) colorWithAlphaComponent:0.08] setFill];
        [[NSBezierPath bezierPathWithRoundedRect:iconBox xRadius:5 yRadius:5] fill];
        NSString *glyph = item[@"glyph"];
        if (glyph.length) {
            NSDictionary *glyphAttrs = @{NSFontAttributeName: [NSFont systemFontOfSize:10.5 weight:NSFontWeightSemibold], NSForegroundColorAttributeName: muted};
            NSSize glyphSize = [glyph sizeWithAttributes:glyphAttrs];
            [glyph drawAtPoint:NSMakePoint(NSMidX(iconBox) - glyphSize.width / 2, NSMidY(iconBox) - glyphSize.height / 2) withAttributes:glyphAttrs];
        }
        NSImage *symbol = nil;
        for (NSString *name in item[@"icons"]) {
            symbol = CRTintedSymbol(name, 12, NSFontWeightMedium, muted);
            if (symbol) break;
        }
        if (symbol) {
            NSSize size = symbol.size;
            [symbol drawInRect:NSMakeRect(NSMidX(iconBox) - size.width / 2, NSMidY(iconBox) - size.height / 2, size.width, size.height)
                      fromRect:NSZeroRect operation:NSCompositingOperationSourceOver fraction:1 respectFlipped:YES hints:nil];
        }
        NSDictionary *titleAttrs = @{NSFontAttributeName: [NSFont systemFontOfSize:13], NSForegroundColorAttributeName: ink};
        NSString *title = item[@"title"] ?: @"";
        NSMutableAttributedString *richTitle = [[NSMutableAttributedString alloc] initWithString:title attributes:titleAttrs];
        // Bold the part of the title the filter matched, like a command palette.
        NSString *filter = [self.owner slashFilterText];
        if (filter.length && title.length) {
            NSRange match = [title.lowercaseString rangeOfString:filter.lowercaseString];
            if (match.location != NSNotFound && NSMaxRange(match) <= richTitle.length) {
                NSColor *accent = dark ? [NSColor colorWithRed:0.45 green:0.66 blue:1 alpha:1]
                                       : [NSColor colorWithRed:0.12 green:0.44 blue:0.92 alpha:1];
                [richTitle addAttribute:NSForegroundColorAttributeName value:accent range:match];
                [richTitle addAttribute:NSFontAttributeName value:[NSFont systemFontOfSize:13 weight:NSFontWeightSemibold] range:match];
            }
        }
        NSSize titleSize = [richTitle size];
        [richTitle drawAtPoint:NSMakePoint(NSMaxX(iconBox) + 8, NSMidY(rowRect) - titleSize.height / 2)];
        NSString *subtitle = item[@"subtitle"] ?: @"";
        if (subtitle.length) {
            NSMutableParagraphStyle *right = [NSMutableParagraphStyle new];
            right.alignment = NSTextAlignmentRight;
            NSDictionary *subtitleAttrs = @{NSFontAttributeName: [NSFont systemFontOfSize:11], NSForegroundColorAttributeName: muted, NSParagraphStyleAttributeName: right};
            [subtitle drawInRect:NSMakeRect(NSMinX(rowRect) + 96, NSMidY(rowRect) - 7, NSMaxX(rowRect) - 102, 14) withAttributes:subtitleAttrs];
        }
    }
}
- (void)updateTrackingAreas {
    [super updateTrackingAreas];
    if (_tracking) [self removeTrackingArea:_tracking];
    _tracking = [[NSTrackingArea alloc] initWithRect:self.bounds options:NSTrackingMouseEnteredAndExited | NSTrackingMouseMoved | NSTrackingActiveAlways owner:self userInfo:nil];
    [self addTrackingArea:_tracking];
}
- (void)mouseExited:(NSEvent *)event { self.hovered = -1; }
- (void)mouseMoved:(NSEvent *)event { self.hovered = [self rowAtPoint:[self convertPoint:event.locationInWindow fromView:nil]]; }
- (void)mouseDown:(NSEvent *)event {
    NSInteger index = [self rowAtPoint:[self convertPoint:event.locationInWindow fromView:nil]];
    if (index >= 0 && index < (NSInteger)self.items.count) [self.owner slashChoose:self.items[index]];
}
- (void)setSelected:(NSInteger)selected { _selected = selected; [self keepSelectedVisible]; [self setNeedsDisplay:YES]; }
- (void)setHovered:(NSInteger)hovered { _hovered = hovered; [self setNeedsDisplay:YES]; }
@end
